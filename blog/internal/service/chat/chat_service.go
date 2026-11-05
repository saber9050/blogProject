package chat

import (
	stderrors "errors"
	"strings"
	"sync"
	"unicode/utf8"

	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/chat"
	llmconfigSvc "blog/internal/service/llmconfig"
	"blog/pkg/errors"
	"blog/pkg/llmclient"
	"blog/pkg/logger"

	"go.uber.org/zap"
)

// timeLayout 沿用项目其它 response 的格式，前端直接 new Date() 能解析。
const timeLayout = "2006-01-02T15:04:05Z"

// maxTitleFieldRunes 标题字段的存储上限，与 chat_sessions.title 的 varchar(120) 对齐。
// 超过会被 PG 拒绝（真到这一步就是 500，而不是友好的 400），所以在服务层先挡。
// ⚠️ 与 titleSnippetRunes（自动生成标题时截前 30 字）是两个不同的数，别合并。
const maxTitleFieldRunes = 120

type chatService struct {
	repo   repo.Repository
	llmCfg llmconfigSvc.Service
	// agents 按模型配置 id 缓存 agent（构建一次，跨请求复用）。
	agents *agentBuilder

	// embedClient 复用的向量化客户端。每次检索都新建 http.Client 会让连接无法复用，
	// 高频问答时容易耗掉本地端口。
	embedMu     sync.Mutex
	embedClient *llmclient.Client
	embedKey    string
}

// NewChatService 创建助手服务。
//
// 依赖只有两个：会话仓储 + 模型配置服务。
// ⚠️ 刻意**不注入 articleService**：article 服务的 GetDetail 每次都会 IncrementViews，
// AI 侧读文章若走它，用户每问一句文章浏览量就涨一次，统计直接失真。
// 文章校验用仓储里的只读查询（FindArticle），一次 select 两列。
func NewChatService(r repo.Repository, llmCfg llmconfigSvc.Service) Service {
	svc := &chatService{repo: r, llmCfg: llmCfg}
	// agent 的检索工具在调用时才用到 service，所以把自己传进去：
	// 工具本身只依赖「按 scope + query 检索」这一个能力（见 contextSearcher）。
	svc.agents = newAgentBuilder(llmCfg, svc)
	return svc
}

// ===== Context =====

func (s *chatService) Context(userID, articleID uint) (*response.ChatContextResponse, error) {
	article, err := s.mustPublishedArticle(articleID)
	if err != nil {
		return nil, err
	}

	models, err := s.activeChatModels()
	if err != nil {
		return nil, err
	}

	sessions, err := s.repo.ListSessionsByArticle(userID, articleID, MaxSessionsPerArticle)
	if err != nil {
		logger.Error("读取会话列表失败", zap.Uint("user_id", userID),
			zap.Uint("article_id", articleID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取会话列表失败", err)
	}

	modelName := make(map[uint]string, len(models))
	for _, m := range models {
		modelName[m.ID] = m.Model
	}

	items := make([]*response.ChatSessionItem, 0, len(sessions))
	for _, sess := range sessions {
		items = append(items, toSessionItem(sess, modelName))
	}

	resp := &response.ChatContextResponse{
		ArticleID:      article.ID,
		ArticleTitle:   article.Title,
		MaxRefArticles: MaxRefArticles,
		Models:         models,
		Sessions:       items,
		RAGReady:       s.ragReady(article),
	}

	if len(models) == 0 {
		// 这不是错误：后台还没配对话模型时，前台面板要能正常打开并说明原因，
		// 而不是弹一个红色报错。返回 200 + enabled:false 是刻意的。
		resp.Enabled = false
		resp.Reason = "尚未配置可用的对话模型，请联系管理员"
	} else {
		resp.Enabled = true
		resp.DefaultModelID = s.defaultModelID(models, sessions)
		if len(sessions) > 0 {
			resp.LastSessionID = sessions[0].ID
		}
	}
	return resp, nil
}

// defaultModelID 优先沿用上次那个会话的模型，否则用列表第一个。
//
// 为什么要沿用：用户上次在 A 会话里挑了便宜的模型，下次进来又被重置成默认的，
// 每次都要重新挑一遍。但如果那个模型已经被禁用/删除，就安静地回退到列表第一个——
// 此时「保持上次选择」已经不可能，硬要沿用只会报错。
func (s *chatService) defaultModelID(models []*response.ChatModelItem, sessions []*entity.ChatSession) uint {
	if len(models) == 0 {
		return 0
	}
	if len(sessions) > 0 {
		want := sessions[0].ModelConfigID
		for _, m := range models {
			if m.ID == want {
				return m.ID
			}
		}
	}
	return models[0].ID
}

// ragReady 本文能否走向量检索。
//
// 三个条件缺一不可，且**按从便宜到贵的顺序判断**：
//  1. vector_status=3（读文章时顺手带出来的列，零成本）
//  2. 向量模型可用（一次配置查询，且会解密密钥）
//  3. article_chunks 表存在（一次 to_regclass）
//
// 任一不满足只影响「资料质量」，不影响功能可用性——降级链会退到关键词检索或全文。
func (s *chatService) ragReady(a *repo.ArticleBrief) bool {
	if a.VectorStatus != entity.VectorStatusDone {
		return false
	}
	if _, _, err := s.llmCfg.ResolveEmbedding(); err != nil {
		return false
	}
	ready, err := s.repo.VectorTableReady()
	if err != nil {
		// 查不到表就当不可用：宁可降级到全文，也不要带着不确定去发一条必然失败的 SQL
		logger.Warn("检查向量表是否存在失败，本次按不可用处理", zap.Error(err))
		return false
	}
	return ready
}

// ===== 会话 CRUD =====

func (s *chatService) CreateSession(userID uint, req *request.CreateChatSessionRequest) (*response.ChatSessionItem, error) {
	article, err := s.mustPublishedArticle(req.ArticleID)
	if err != nil {
		return nil, err
	}

	models, err := s.activeChatModels()
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, errors.New(errors.CodeServiceUnavailable, "尚未配置可用的对话模型")
	}

	modelID := req.ModelConfigID
	if modelID == 0 {
		modelID = models[0].ID
	} else if !containsModel(models, modelID) {
		// 传了就必须合法：可能是后台刚把这个模型禁用，或前端缓存了旧 id。
		// 不接受「传错就静默用默认」——那样用户以为在用 A 模型，实际是 B。
		return nil, errors.New(errors.CodeBadRequest, "指定的模型不可用（不存在、已禁用或不是对话模型）")
	}

	sess := &entity.ChatSession{
		ArticleID:     article.ID,
		UserID:        userID,
		ModelConfigID: modelID,
		Title:         "",
		MessageCount:  0,
	}
	if err := s.repo.CreateSession(sess); err != nil {
		logger.Error("创建会话失败", zap.Uint("user_id", userID),
			zap.Uint("article_id", article.ID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "创建会话失败", err)
	}

	name := make(map[uint]string, len(models))
	for _, m := range models {
		name[m.ID] = m.Model
	}
	return toSessionItem(sess, name), nil
}

func (s *chatService) ListMessages(userID, sessionID uint, beforeID uint, limit int) (*response.ChatMessageListResponse, error) {
	if _, err := s.mustOwnSession(userID, sessionID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	// 多取一条用来判断 has_more，避免再发一次 count
	msgs, err := s.repo.ListMessages(sessionID, beforeID, limit+1)
	if err != nil {
		logger.Error("读取消息列表失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取消息失败", err)
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}

	// 仓储给的是 id DESC（最新在前），接口要正序（旧 → 新），就地反转
	list := make([]*response.ChatMessageItem, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		list = append(list, toMessageItem(msgs[i]))
	}
	return &response.ChatMessageListResponse{List: list, HasMore: hasMore}, nil
}

func (s *chatService) UpdateSession(userID, sessionID uint, req *request.UpdateChatSessionRequest) error {
	if _, err := s.mustOwnSession(userID, sessionID); err != nil {
		return err
	}
	if req.Title == nil && req.ModelConfigID == nil {
		return errors.New(errors.CodeBadRequest, "没有要更新的内容")
	}

	fields := make(map[string]any, 2)
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if utf8.RuneCountInString(title) > maxTitleFieldRunes {
			return errors.New(errors.CodeBadRequest, "标题过长")
		}
		fields["title"] = title
	}
	if req.ModelConfigID != nil {
		models, err := s.activeChatModels()
		if err != nil {
			return err
		}
		// 0 也走「取默认」，与新建会话保持同一套语义
		if *req.ModelConfigID == 0 {
			if len(models) == 0 {
				return errors.New(errors.CodeServiceUnavailable, "尚未配置可用的对话模型")
			}
			fields["model_config_id"] = models[0].ID
		} else {
			if !containsModel(models, *req.ModelConfigID) {
				return errors.New(errors.CodeBadRequest, "指定的模型不可用（不存在、已禁用或不是对话模型）")
			}
			fields["model_config_id"] = *req.ModelConfigID
		}
		// 换模型不影响历史消息：旧消息保留各自生成时的 model，
		// 新消息才用新模型。这是「会话级绑定」而不是「全局绑定」的意义。
	}

	if err := s.repo.UpdateSession(sessionID, fields); err != nil {
		if isNotFound(err) {
			return errors.New(errors.CodeNotFound, "会话不存在")
		}
		logger.Error("更新会话失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "更新会话失败", err)
	}
	return nil
}

func (s *chatService) DeleteSession(userID, sessionID uint) error {
	if _, err := s.mustOwnSession(userID, sessionID); err != nil {
		return err
	}
	if err := s.repo.DeleteSession(sessionID); err != nil {
		// 并发下两次删除：第一次成功后第二次会 RowsAffected=0，这里当 404 处理即可，
		// 对用户来说「它已经没了」和「刚被你删掉」是一回事。
		if isNotFound(err) {
			return errors.New(errors.CodeNotFound, "会话不存在")
		}
		logger.Error("删除会话失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "删除会话失败", err)
	}
	return nil
}

// ===== 候选文章 =====

func (s *chatService) ListArticleCandidates(keyword string, excludeID uint, limit int) ([]*response.ChatArticleCandidateItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	list, err := s.repo.SearchPublishedArticles(keyword, excludeID, limit)
	if err != nil {
		logger.Error("搜索候选文章失败", zap.String("keyword", keyword), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "搜索文章失败", err)
	}
	items := make([]*response.ChatArticleCandidateItem, 0, len(list))
	for _, a := range list {
		items = append(items, &response.ChatArticleCandidateItem{ID: a.ID, Title: a.Title})
	}
	return items, nil
}

// ===== 内部工具 =====

// mustPublishedArticle 取文章并校验已发布。
//
// 「不存在」「已软删」「还是草稿」三种情况对外统一 404：前台用户不该知道
// 「这篇草稿存在但你不能聊」——那等于泄露了草稿的存在性。
func (s *chatService) mustPublishedArticle(id uint) (*repo.ArticleBrief, error) {
	if id == 0 {
		return nil, errors.New(errors.CodeBadRequest, "文章 id 无效")
	}
	a, err := s.repo.FindArticle(id)
	if err != nil {
		logger.Error("读取文章失败", zap.Uint("article_id", id), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取文章失败", err)
	}
	if a == nil || a.Status != 1 {
		return nil, errors.New(errors.CodeNotFound, "文章不存在或未发布")
	}
	return a, nil
}

// mustOwnSession 取会话并校验归属。不属于当前用户一律 404（不是 403）：
// 403 等于告诉对方「这个会话存在，只是不给你看」，可以被用来枚举 id。
func (s *chatService) mustOwnSession(userID, sessionID uint) (*entity.ChatSession, error) {
	sess, err := s.repo.FindSession(sessionID, userID)
	if err != nil {
		logger.Error("读取会话失败", zap.Uint("session_id", sessionID), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取会话失败", err)
	}
	if sess == nil {
		return nil, errors.New(errors.CodeNotFound, "会话不存在")
	}
	return sess, nil
}

// activeChatModels 启用的对话模型（type=1 且 status=1）。
//
// 只取 id + model，不返回 base_url / api_key——前台没有理由知道这些。
func (s *chatService) activeChatModels() ([]*response.ChatModelItem, error) {
	list, err := s.llmCfg.ListByType(1)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ChatModelItem, 0, len(list))
	for _, m := range list {
		if m.Status != 1 {
			continue
		}
		out = append(out, &response.ChatModelItem{ID: m.ID, Model: m.Model})
	}
	return out, nil
}

func containsModel(models []*response.ChatModelItem, id uint) bool {
	for _, m := range models {
		if m.ID == id {
			return true
		}
	}
	return false
}

func toSessionItem(sess *entity.ChatSession, modelName map[uint]string) *response.ChatSessionItem {
	item := &response.ChatSessionItem{
		ID:            sess.ID,
		Title:         sess.Title,
		ModelConfigID: sess.ModelConfigID,
		MessageCount:  sess.MessageCount,
		CreatedAt:     sess.CreatedAt.Format(timeLayout),
	}
	// 会话绑定的模型可能已被删除/禁用，此时不给名字（前端显示「模型已下线」），
	// 而不是编一个「未知模型」让用户以为是 bug。
	item.Model = modelName[sess.ModelConfigID]
	if sess.LastMessageAt != nil {
		t := sess.LastMessageAt.Format(timeLayout)
		item.LastMessageAt = &t
	}
	return item
}

func toMessageItem(m *entity.ChatMessage) *response.ChatMessageItem {
	return &response.ChatMessageItem{
		ID:          m.ID,
		Role:        m.Role,
		Content:     m.Content,
		Model:       m.Model,
		Refs:        m.Refs,
		ArticleRefs: m.ArticleRefs,
		Interrupted: m.Interrupted,
		Refused:     m.Refused,
		Error:       m.Error,
		CreatedAt:   m.CreatedAt.Format(timeLayout),
	}
}

// isNotFound 判断仓储层的「记录不存在」。用 errors.Is 而不是 ==：
// gorm 在不同路径上可能把 ErrRecordNotFound 包一层，直接比较会在某条分支上失效。
func isNotFound(err error) bool {
	return stderrors.Is(err, repo.ErrNotFound)
}
