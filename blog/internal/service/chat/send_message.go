package chat

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	"blog/pkg/errors"
	"blog/pkg/logger"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// maxSessionMessages 单会话消息总条数上限（user + assistant）。
	// 约等于 200 轮问答。⚠️ 不是「200 轮」——message_count 数的是**消息行**，
	// 一次问答产生 2 行，写成 200 会让实际只允许 100 轮。
	maxSessionMessages = 400
	// maxContentRunes 单条提问上限。与前端输入框的限制保持一致。
	maxContentRunes = 2000
	// maxRefArticleIDs 单次提问可额外引用的文章数（连同当前文章共 MaxRefArticles 篇）。
	maxRefArticleIDs = MaxRefArticles - 1
	// historyMessageLimit 带进上下文的历史消息条数（约 10 轮问答）。
	historyMessageLimit = 20
	// requestTimeout 整条 agent 链的总超时，包含所有检索轮次与模型生成。
	// 不用模型客户端的 TimeoutSec（那是单次 HTTP 的超时，管不住多轮 agent）。
	requestTimeout = 120 * time.Second
	// titleSnippetRunes 自动生成标题时取首问的前 30 个 rune。
	// 这是产品上的「一眼能看懂」长度，与字段上限（120）无关。
	titleSnippetRunes = 30
)

// SendMessage 一次提问的完整流程。
//
// sink 为 nil 表示调用方只要最终结果（非流式）；非 nil 时边跑边推事件。
// 两种模式共用下面的编排，唯一区别是事件往哪里送。
func (s *chatService) SendMessage(ctx context.Context, userID, sessionID uint,
	req *request.SendChatMessageRequest, sink StreamSink) (*response.SendChatMessageResponse, error) {

	sess, err := s.mustOwnSession(userID, sessionID)
	if err != nil {
		return nil, err
	}
	article, err := s.mustPublishedArticle(sess.ArticleID)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, errors.New(errors.CodeBadRequest, "提问内容不能为空")
	}
	if utf8.RuneCountInString(content) > maxContentRunes {
		return nil, errors.New(errors.CodeBadRequest, "提问过长，请控制在 2000 字以内")
	}

	// 会话总量上限。放在内容校验之后、落库之前：空消息不该消耗配额，
	// 而超限检查必须在写入之前，否则会留下一条「问了但注定失败」的消息。
	if sess.MessageCount >= maxSessionMessages {
		return nil, errors.New(errors.CodeConflict,
			"本会话已达上限（约 200 轮），请新建会话继续")
	}

	// 限流放在所有廉价校验之后：避免刷接口的人用空消息把配额刷掉
	if err := s.allow(ctx, userID); err != nil {
		return nil, err
	}

	// 引用文章校验必须在**落库之前**做完：不合法直接 400，
	// 不能先写入一条带着坏 article_refs 的消息再报错（那条消息会永远留在历史里）。
	articleRefs, err := s.validateArticleRefs(sess.ArticleID, req.ArticleIDs)
	if err != nil {
		return nil, err
	}

	modelID, modelFallback, err := s.resolveModel(sess.ModelConfigID)
	if err != nil {
		return nil, err
	}
	// 边界检查放在最前面：agent 构建要建 HTTP 客户端，这是本请求最贵的一步，
	// 早失败能省掉后面所有的落库与调用。不过上面已有校验，这里只防并发改配置。
	entry, err := s.agents.get(ctx, modelID)
	if err != nil {
		logger.Error("构建 agent 失败", zap.Uint("model_config_id", modelID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeServiceUnavailable, "当前模型不可用，请稍后重试", err)
	}

	// 落 user 消息（含本轮引用），article_refs 走 struct Create 才能吃到 serializer:json
	userMsg := &entity.ChatMessage{
		SessionID:   sessionID,
		Role:        entity.ChatRoleUser,
		Content:     content,
		ArticleRefs: articleRefs,
	}
	if err := s.repo.CreateMessage(userMsg); err != nil {
		logger.Error("写入用户消息失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "保存消息失败", err)
	}

	// 先建一条空的助手消息拿到 id：meta 事件必须在开流时就把 assistant_message_id 给出去，
	// 否则前端建不了气泡，只能等第一个 delta 才显示——那段时间用户看到的是「没反应」。
	asstMsg := &entity.ChatMessage{
		SessionID: sessionID,
		Role:      entity.ChatRoleAssistant,
		Model:     entry.modelName,
	}
	if err := s.repo.CreateMessage(asstMsg); err != nil {
		logger.Error("写入助手占位消息失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "保存消息失败", err)
	}

	if sink != nil {
		_ = sink.OnMeta(MetaEvent{
			SessionID:          sessionID,
			UserMessageID:      userMsg.ID,
			AssistantMessageID: asstMsg.ID,
			Model:              entry.modelName,
			ModelFallback:      modelFallback,
			ArticleRefs:        articleRefs,
		})
	}

	hist, err := s.loadHistory(sessionID, userMsg.ID)
	if err != nil {
		return nil, err
	}

	// 请求级范围只放这里和消息前缀，绝不进 Instruction（agent 是跨请求复用的）
	sc := &Scope{
		SessionID: sessionID,
		// 工具消息要挂到这条助手消息上，事后才能按 message_id 整组取出
		AssistantMessageID:  asstMsg.ID,
		CurrentArticleID:    article.ID,
		CurrentArticleTitle: article.Title,
		RefArticleIDs:       articleIDsOf(articleRefs),
		RefTitles:           titlesOf(articleRefs),
	}
	sc.EnsureSeen()

	// 超时挂在 ctx 上而不是模型客户端：这样它同时能中断工具执行、后续轮次和流式读取。
	runCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	runCtx = withScope(runCtx, sc)

	msgs := buildMessages(hist, content, article.Title, refTitlesInOrder(articleRefs))
	iter := entry.runner.Run(runCtx, msgs)
	res := consumeAgentStream(runCtx, iter, sink, sc)

	// 落库。⚠️ 即使中途失败也要走到这里：已生成的内容对用户有价值，
	// 整条丢弃等于既白烧 token 又白等时间（§6.7.1 的收尾规则）。
	fields := map[string]any{
		"content":           res.content,
		"prompt_tokens":     res.promptTokens,
		"completion_tokens": res.completionTokens,
		"interrupted":       res.interrupted,
		"refused":           res.refused,
		"no_retrieval":      res.noRetrieval,
	}
	if len(sc.Refs) > 0 {
		fields["refs"] = jsonbExpr(sc.Refs)
	}
	// 过程记录也要落库：只走 SSE 的话，用户刷新页面就再也看不到「它到底查了什么」，
	// 而事后追问「它为什么这么答」恰恰最需要看这个
	if len(sc.Steps) > 0 {
		fields["steps"] = jsonbExpr(sc.Steps)
	}
	if res.err != nil {
		fields["error"] = truncateRunes(res.err.Error(), 240)
		logger.Warn("助手回答中断", zap.Uint("session_id", sessionID),
			zap.Int("generated_runes", utf8.RuneCountInString(res.content)), zap.Error(res.err))
	}

	// 「模型一次都没检索就作答」单独告警。
	//
	// 这几乎总是配置问题而不是代码问题，但它的表现是「回答很快、看起来也正常」，
	// 不告警就永远不会有人发现——直到用户发现答案和文章对不上。
	// 常见原因按概率排：
	//   1. 模型不支持 function calling（deepseek-reasoner、部分本地模型）
	//   2. base_url 是第三方网关，请求里的 tools 字段被丢掉
	//   3. Instruction 被模型忽略（少见，但换模型后比例会变）
	if sc.ToolCalls == 0 && !res.refused {
		// 两种子情况要分开记，排查方向完全不同：
		//   noRetrieval=true  → 模型输出了正文但没调工具，内容已被替换
		//   noRetrieval=false → 模型一个字都没说（比如直接报错），无需替换
		logger.Warn("本次回答未发生任何检索（模型未调用检索工具）",
			zap.Uint("session_id", sessionID),
			zap.String("model", entry.modelName),
			zap.Bool("answer_replaced", res.noRetrieval),
			zap.Int("answer_runes", utf8.RuneCountInString(res.content)),
			zap.String("hint", "检查模型是否支持 function calling / 网关是否透传 tools / 工具名是否为空"))
	}
	if err := s.repo.UpdateMessage(asstMsg.ID, fields); err != nil {
		// 这里失败只记日志，不覆盖上面的错误：对用户来说回答已经收到了，
		// 落库失败属于运维问题，抛出去反而会让他以为白问了。
		logger.Error("回填助手消息失败", zap.Uint("message_id", asstMsg.ID), zap.Error(err))
	}

	s.afterMessage(sess, content, modelID, modelFallback, len(articleRefs))

	out := &response.SendChatMessageResponse{
		UserMessageID:      userMsg.ID,
		AssistantMessageID: asstMsg.ID,
		Content:            res.content,
		Model:              entry.modelName,
		ModelFallback:      modelFallback,
		Refs:               sc.Refs,
		Steps:              sc.Steps,
		ArticleRefs:        articleRefs,
		SearchedArticles:   sc.ArticleIDs(),
		PromptTokens:       res.promptTokens,
		CompletionTokens:   res.completionTokens,
		Refused:            res.refused,
		NoRetrieval:        res.noRetrieval,
		Interrupted:        res.interrupted,
		Rounds:             sc.Rounds,
		ToolCalls:          sc.ToolCalls,
		Degraded:           sc.Degraded,
	}

	// 有错误且一个字都没生成：返回错误让 controller 决定怎么表达。
	// 已经生成了内容的情况不返回错误——半截回答也是回答，用 error 字段标在库里即可。
	if res.err != nil && strings.TrimSpace(res.content) == "" {
		return out, errors.NewWithErr(errors.CodeServiceUnavailable, "模型调用失败，请稍后重试", res.err)
	}
	return out, nil
}

// afterMessage 一轮问答结束后的会话维护。
//
// 失败只 warn：这些都发生在回答已经推给用户之后，报错改变不了任何结果。
func (s *chatService) afterMessage(sess *entity.ChatSession, content string,
	modelID uint, fallback bool, refCount int) {

	// message_count 每次恒定 +2（user + assistant），与「消息总行数」的定义一致。
	if err := s.repo.TouchSession(sess.ID, 2, time.Now()); err != nil {
		logger.Warn("更新会话活跃时间失败", zap.Uint("session_id", sess.ID), zap.Error(err))
	}

	fields := map[string]any{}
	// 标题只在第一次提问时生成：前 30 个 rune，去换行。
	// 不用模型生成标题——多一次调用，收益远不够。
	if sess.Title == "" {
		fields["title"] = truncateRunes(strings.ReplaceAll(content, "\n", " "), titleSnippetRunes)
	}
	// 模型回退时把会话的选择改过来：否则下次又拿旧的失效 id 去解析，每次都要回退一次。
	if fallback {
		fields["model_config_id"] = modelID
	}
	if len(fields) == 0 {
		return
	}
	if err := s.repo.UpdateSession(sess.ID, fields); err != nil {
		logger.Warn("更新会话信息失败", zap.Uint("session_id", sess.ID), zap.Error(err))
	}
}

// validateArticleRefs 校验本轮的引用文章。
//
// 规则：去重 → 剔除当前文章 → 数量 ≤7 → 逐条「存在 + 已发布」。
// 任一条不合法就整体 400，**不做静默过滤**：用户明明选了 3 篇，其中 1 篇不能用，
// 悄悄只查 2 篇会让答案缺一块，而且用户永远不知道少了什么。
func (s *chatService) validateArticleRefs(currentID uint, ids []uint) ([]entity.ChatArticleRef, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	seen := make(map[uint]struct{}, len(ids))
	out := make([]entity.ChatArticleRef, 0, len(ids))
	for _, id := range ids {
		if id == 0 || id == currentID {
			continue // 当前文章是隐式的，不需要也不允许重复引用
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if len(out) >= maxRefArticleIDs {
			break
		}
		out = append(out, entity.ChatArticleRef{ArticleID: id})
	}

	// 超出上限要报错而不是截断：用户以为引了 8 篇，实际只查了 7 篇，答案必然缺。
	if len(seen) > maxRefArticleIDs {
		return nil, errors.New(errors.CodeBadRequest,
			"单次提问最多引用 7 篇文章（连同当前文章共 8 篇）")
	}

	for i := range out {
		a, err := s.repo.FindArticle(out[i].ArticleID)
		if err != nil {
			logger.Error("校验引用文章失败", zap.Uint("article_id", out[i].ArticleID), zap.Error(err))
			return nil, errors.NewWithErr(errors.CodeInternalError, "校验引用文章失败", err)
		}
		if a == nil || a.Status != 1 {
			return nil, errors.New(errors.CodeBadRequest, "引用的文章不存在或未发布")
		}
		out[i].Title = a.Title
	}
	return out, nil
}

// resolveModel 决定本次用哪个模型。
//
// 会话绑定的模型可能已被禁用 / 删除（管理员随时可能操作）。此时回退到首个启用的
// 对话模型，并通过 model_fallback 告知前端——不能静默换模型，用户有权知道
// 「这次回答不是你以为的那个模型生成的」。
func (s *chatService) resolveModel(current uint) (modelID uint, fallback bool, err error) {
	if current > 0 {
		if _, err := s.llmCfg.Resolve(current); err == nil {
			return current, false, nil
		}
		logger.Warn("会话绑定的模型不可用，尝试回退", zap.Uint("model_config_id", current))
	}
	models, err := s.activeChatModels()
	if err != nil {
		return 0, false, err
	}
	if len(models) == 0 {
		return 0, false, errors.New(errors.CodeServiceUnavailable, "尚未配置可用的对话模型")
	}
	return models[0].ID, current > 0, nil
}

// loadHistory 取当前提问之前的历史，转成 agent 输入。
//
// 从 beforeID 往前取而不是取「最近 N 条」：刚插入的 user + assistant 占位也会被
// 「最近 N 条」算进去，等于用两条空消息挤掉两条真实历史。
func (s *chatService) loadHistory(sessionID, beforeID uint) ([]*schema.Message, error) {
	list, err := s.repo.ListMessages(sessionID, beforeID, historyMessageLimit)
	if err != nil {
		logger.Error("读取历史消息失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取历史消息失败", err)
	}
	// 仓储给的是 id DESC，转成正序（旧 → 新）
	orded := make([]*entity.ChatMessage, 0, len(list))
	for i := len(list) - 1; i >= 0; i-- {
		orded = append(orded, list[i])
	}

	msgs := historyToMessages(orded)

	// 回放最近一轮检索到的资料原文。
	//
	// 为什么需要：历史里只有上一轮的**结论**（assistant 的 content），没有原文。
	// 用户追问「展开说说第 3 点」时，模型只能顺着自己的总结继续扩写——
	// 表现完全正常，但已经在编了。把原文还给它，「延续上下文」才有依据。
	//
	// 为什么只用文本注入、不用原生 tool 消息：OpenAI 协议要求 tool 消息紧跟带
	// tool_calls 的 assistant 消息、且靠 tool_call_id 配对，而我们的工具定义走
	// utils.InferTool，拿不到模型原始给的 id（落库字段只能留空）。伪造 id 有被
	// provider 拒绝（400）的风险，而 Qwen2.5-7B 这类小模型对原生 tool 协议的模板
	// 支持本来就不稳。文本注入零协议风险，模型照样能读。
	if replay := s.toolReplay(sessionID, beforeID); replay != "" {
		msgs = append(msgs, &schema.Message{Role: schema.System, Content: replay})
	}
	return msgs, nil
}

// toolReplayMaxChars 回放资料的总字数上限。
//
// 完整一轮可能装到 6000 字，而历史本身还有 2000 字预算；只回放一轮再砍到 3000，
// 才不至于把上下文顶爆。追问通常只需要「刚才那几段」，不需要完整档案。
const toolReplayMaxChars = 3000

// toolReplay 取最近一轮有检索记录的资料原文，拼成一段可注入的文本。
//
// 返回空串表示没有可回放的资料（新会话 / 上一轮没检索过），调用方直接跳过。
func (s *chatService) toolReplay(sessionID, beforeID uint) string {
	msgID, err := s.repo.LastMessageIDWithTools(sessionID, beforeID)
	if err != nil {
		logger.Warn("查询可回放的检索记录失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return ""
	}
	if msgID == 0 {
		return ""
	}
	list, err := s.repo.ListToolMessagesByMessage(msgID)
	if err != nil {
		logger.Warn("读取工具消息失败", zap.Uint("message_id", msgID), zap.Error(err))
		return ""
	}

	return buildReplayText(list, toolReplayMaxChars)
}

// buildReplayText 把工具消息拼成可注入的文本，总字数不超过 budget。
//
// 抽成纯函数是为了能直接单测：这段文本会直接进模型上下文，
// 写错了（比如忘了加「可能过时」的说明）不会报错，只会让模型把旧资料当现状复述。
func buildReplayText(list []*entity.ChatToolMessage, budget int) string {
	var b strings.Builder
	used := 0
	for _, t := range list {
		body := strings.TrimSpace(t.Content)
		if body == "" {
			continue // 检索失败的那一轮没有原文，跳过
		}
		// 段间分隔符也要算进预算：它同样是实际注入的字数，
		// 只在正文上做预算会让总量悄悄超出（2 段就多 4 个字）。
		remaining := budget - used
		if remaining <= 0 {
			break
		}
		if used > 0 {
			if remaining <= 2 {
				break
			}
			b.WriteString("\n\n")
			used += 2
			remaining -= 2
		}
		part := truncateRunes(body, remaining)
		if part == "" {
			break
		}
		b.WriteString(part)
		used += len([]rune(part))
	}
	if used == 0 {
		return ""
	}

	// 说明必须写清楚「这是快照、可能过时、要引用就重新查」：
	// 不写的话，模型会把上一轮的资料当成「当前文章内容」直接复述，
	// 而文章可能已经改过了。
	return "【上一轮检索到的资料原文（快照）】\n" +
		"以下内容是你上一轮回答时通过检索工具读到的，可能已经过时。" +
		"它只用来延续上下文，涉及文章事实时应当重新检索确认，不要直接当作当前事实复述。\n\n" +
		b.String()
}

// ===== 小工具 =====

func articleIDsOf(refs []entity.ChatArticleRef) []uint {
	ids := make([]uint, 0, len(refs))
	for _, r := range refs {
		ids = append(ids, r.ArticleID)
	}
	return ids
}

func titlesOf(refs []entity.ChatArticleRef) map[uint]string {
	m := make(map[uint]string, len(refs))
	for _, r := range refs {
		m[r.ArticleID] = r.Title
	}
	return m
}

func refTitlesInOrder(refs []entity.ChatArticleRef) []string {
	titles := make([]string, 0, len(refs))
	for _, r := range refs {
		titles = append(titles, r.Title)
	}
	return titles
}

// jsonbExpr 把 Go 值转成 jsonb 参数。
//
// GORM 的 Updates(map) 不会走字段上的 serializer:json —— 那是 struct 赋值时才生效的，
// map 里的值会被当成普通参数直接下发。直接塞 []ChatRefChunk 会得到
// 「unsupported type」或把切片拆成数组，所以显式序列化并 cast。
func jsonbExpr(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return gorm.Expr("?::jsonb", string(b))
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
