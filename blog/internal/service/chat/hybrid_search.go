package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"blog/internal/model/entity"
	repo "blog/internal/repository/chat"
	"blog/pkg/database"
	"blog/pkg/llmclient"
	"blog/pkg/logger"

	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

// 混合检索的全部可调参数。集中在这里，改数值不用翻代码。
const (
	// 每路 SQL 内的单篇限额。两路各 3 条，融合后单篇最多 6 条 —— 所以要再做一次配额（见下方）。
	perArticleSemantic = 3
	perArticleKeyword  = 3

	// 语义路下限：余弦相似度 0.30 以下是「字面都不搭」，纳入资料只会稀释上下文。
	semanticMinScore = 0.30
	// 关键词路下限：中文 3-gram 的 word_similarity 普遍偏低（常在 0.1~0.5），门槛要低。
	keywordMinScore = 0.10

	// RRF 的 k。60 是论文默认值；它压平头部差距，避免某一路的第一名独大。
	rrfK = 60
	// 融合后进入父块扩展的 chunk 数。
	fusionTopN = 8

	// 融合后的第二道配额：当前文章最多 4 段，其它每篇最多 2 段。
	// 只做 SQL 内的限额是不够的：两路各 3 条融合后单篇能到 6 条。
	currentArticleMaxSegments = 4
	otherArticleMaxSegments   = 2

	// 单次 tool 调用的输出上限。模型一轮看 4 段足够，多了反而稀释重点。
	singleCallMaxSegments = 4
	singleCallMaxChars    = 2500
	// 一节拼回后的长度上限，超了退化成「命中块 + 前后各一块」。
	sectionMaxChars = 2000
	// 整个请求累计的资料字数上限（多轮累加）。
	totalMaxChars = 6000
	// 全文降级的截断长度：当前文章放宽，引用文章从严。
	fullTextCurrentChars = 5000
	fullTextRefChars     = 1500
)

// SearchResult 一次检索的产物。
//
// 除了给模型的文本，还要把「命中几段、查了哪几篇、有没有降级」带回去——
// tool 要拿它填 Scope.Steps，让用户能在界面上看到完整的检索过程。
type SearchResult struct {
	Text     string
	Hits     int
	Articles []string
	Degraded bool
	Note     string
}

// fusedChunk 融合后的候选，带 RRF 分。
type fusedChunk struct {
	chunk *repo.RetrievedChunk
	rrf   float64
}

// Search 检索入口：两路召回 → RRF 融合 → 配额 → 父块扩展 → 预算装配 → 带出处的文本。
//
// 这个方法是 agent 的 tool 唯一调用的东西，与 agent 完全解耦：
// 可以单独 curl / 单测，不需要起模型。调检索参数时这一点非常重要——
// 不然每次改阈值都要跑一遍 LLM，既慢又不可复现。
func (s *chatService) Search(ctx context.Context, sc *Scope, rawQuery string) (SearchResult, error) {
	ids := sc.ArticleIDs()
	if len(ids) == 0 {
		return SearchResult{
			Text: "检索范围无效。请直接根据已有信息回答；如果没有任何信息，请如实告知无法回答。",
			Note: "检索范围为空（会话没有绑定文章）",
		}, nil
	}

	remaining := totalMaxChars - sc.UsedChars
	if remaining <= 0 {
		// 多轮累计已用完预算。返回一句「够了」而不是报错——
		// 报错会让模型以为检索坏了，可能改去编答案。
		return SearchResult{
			Text: "本次对话的资料已积累足够，请基于已有内容作答。",
			Note: "资料预算已用完",
		}, nil
	}

	clean := cleanQuery(rawQuery)
	idsLiteral := toPGIntArray(ids)

	var semRows, kwRows []*repo.RetrievedChunk

	// 语义路。向量模型不可用 / 调用失败都只降级，不中断——
	// 关键词路仍然能召回专有名词，用户不该因为向量服务抖动就完全不能问。
	if vec, err := s.embedQuery(ctx, clean); err != nil {
		logger.Warn("问题向量化失败，本次跳过语义检索", zap.Error(err))
		sc.Degraded = true
	} else {
		rows, err := s.repo.SemanticSearch(vec, idsLiteral, perArticleSemantic, semanticMinScore)
		if err != nil {
			logger.Warn("语义检索失败，本次仅用关键词路", zap.Error(err))
			sc.Degraded = true
		} else {
			semRows = rows
		}
	}

	// 关键词路。清洗后为空（用户只问了「为什么」）就整条跳过，
	// 拿空串去算相似度只会得到噪声。
	if clean != "" {
		if !database.TrgmReady() {
			// 启动自检已 warn 过一次，这里不再重复刷日志
			sc.Degraded = true
		} else {
			rows, err := s.repo.KeywordSearch(clean, idsLiteral, perArticleKeyword, keywordMinScore)
			if err != nil {
				logger.Warn("关键词检索失败，本次仅用语义路", zap.Error(err))
				sc.Degraded = true
			} else {
				kwRows = rows
			}
		}
	}

	fused := fuseRRF(semRows, kwRows)
	fused = applyArticleQuota(fused, sc, fusionTopN)

	if len(fused) == 0 {
		// 两路都没命中：可能是文章没向量化、也可能是问题确实无关。
		// 退回全文，让模型至少能看到文章主体。
		text, err := s.fullTextFallback(ctx, sc, remaining)
		if err != nil {
			return SearchResult{}, err
		}
		return SearchResult{Text: text, Hits: 0, Degraded: true, Note: "未命中切片，已退回全文"}, nil
	}

	sections, err := s.expandSections(ctx, sc, fused)
	if err != nil {
		logger.Warn("父块扩展失败，退化为直接使用命中块", zap.Error(err))
		sc.Degraded = true
		sections = directSections(fused)
	}

	text, refs := assembleContext(sc, sections, remaining)
	// 记录本轮实际用到的块，供落库与前端展示
	sc.Refs = append(sc.Refs, refs...)
	for _, r := range refs {
		sc.Seen[r.ChunkID] = struct{}{}
	}
	if text == "" {
		text, err := s.fullTextFallback(ctx, sc, remaining)
		if err != nil {
			return SearchResult{}, err
		}
		return SearchResult{Text: text, Hits: 0, Degraded: true, Note: "装配后为空，已退回全文"}, nil
	}
	sc.UsedChars += len([]rune(text))

	// 实际命中的文章（按资料里出现的顺序去重），用于前端展示「查了哪几篇」
	articles := make([]string, 0, len(refs))
	seenTitle := make(map[string]struct{}, len(refs))
	for _, r := range refs {
		for _, sec := range sections {
			if sec.hitChunk != nil && sec.hitChunk.ID == r.ChunkID && sec.articleTitle != "" {
				if _, ok := seenTitle[sec.articleTitle]; !ok {
					seenTitle[sec.articleTitle] = struct{}{}
					articles = append(articles, sec.articleTitle)
				}
			}
		}
	}

	note := ""
	if sc.Degraded {
		note = "部分检索路降级"
	}
	return SearchResult{
		Text:     text,
		Hits:     len(refs),
		Articles: articles,
		Degraded: sc.Degraded,
		Note:     note,
	}, nil
}

// ===== ① 两路召回 =====

// embedQuery 把清洗后的问题向量化。
//
// eino 的 Embedder 与项目已有的 llmclient.Embed 功能完全重合，这里复用后者：
// 它是模块 02 向量化在用的同一条路径，已经验证过维度、错误处理与超时行为。
// 为了「用新框架」再包一层 embedding 组件，只会多一处可能不一致的地方。
func (s *chatService) embedQuery(ctx context.Context, text string) (pgvector.Vector, error) {
	cfg, _, err := s.llmCfg.ResolveEmbedding()
	if err != nil {
		return pgvector.Vector{}, err
	}
	cli := s.embedder(cfg)
	vecs, err := cli.Embed(ctx, []string{text})
	if err != nil {
		return pgvector.Vector{}, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return pgvector.Vector{}, fmt.Errorf("向量模型返回空向量")
	}
	// ⚠️ llmclient.Embed 返回 [][]float32，直接给 pgvector.NewVector 是正确的。
	// 若哪天换成 eino 的 EmbedStrings（[][]float64），不做 float64→float32 转换会编译不过，
	// 而错误地转成别的类型则会得到「检索永远 0 命中」这种静默故障。
	return pgvector.NewVector(vecs[0]), nil
}

// fuseRRF 用 RRF 融合两路结果。
//
// 为什么不用加权求和（0.7*语义 + 0.3*关键词）：余弦相似度与 trgm 相似度虽然都在 0~1，
// 但分布完全不同（trgm 对短文本天然给高分，余弦对长文本更稳）。线性加权等于拍脑袋
// 定权重且要反复重调；RRF 只用排名，天然免疫量纲差异。
func fuseRRF(sem, kw []*repo.RetrievedChunk) []*fusedChunk {
	scores := make(map[uint]float64, len(sem)+len(kw))
	byID := make(map[uint]*repo.RetrievedChunk, len(sem)+len(kw))

	for _, list := range [][]*repo.RetrievedChunk{sem, kw} {
		for i, c := range list {
			if c == nil {
				continue
			}
			// 1/(k+rank)，rank 从 1 开始
			scores[c.ID] += 1.0 / float64(rrfK+i+1)
			byID[c.ID] = c
		}
	}

	out := make([]*fusedChunk, 0, len(scores))
	for id, sc := range scores {
		out = append(out, &fusedChunk{chunk: byID[id], rrf: sc})
	}
	// 两路都命中的文档分数天然是两条之和，这本身就是最强的质量信号，不需要额外加权
	sort.Slice(out, func(i, j int) bool {
		if out[i].rrf == out[j].rrf {
			// 分数相同时用 chunk id 兜底，保证结果稳定（否则每次顺序都可能不同，
			// 排障时会怀疑「同样是这个问题，为什么两次结果不一样」）
			return out[i].chunk.ID < out[j].chunk.ID
		}
		return out[i].rrf > out[j].rrf
	})
	return out
}

// applyArticleQuota 融合后的第二道配额。
//
// 只做 SQL 内的限额是不够的：两路各 3 条，融合后单篇最多能到 6 条，
// 8 篇里一篇就能占掉大半名额，其他文章一条都进不了最终资料。
func applyArticleQuota(fused []*fusedChunk, sc *Scope, limit int) []*fusedChunk {
	counts := make(map[uint]int, 8)
	out := make([]*fusedChunk, 0, limit)

	for _, f := range fused {
		if f == nil || f.chunk == nil {
			continue
		}
		id := f.chunk.ArticleID
		// 已经返回过的块不再入选：多轮检索时重复内容会让模型误判「资料就这些了」
		if _, seen := sc.Seen[f.chunk.ID]; seen {
			continue
		}
		maxSeg := otherArticleMaxSegments
		if id == sc.CurrentArticleID {
			maxSeg = currentArticleMaxSegments
		}
		if counts[id] >= maxSeg {
			continue
		}
		counts[id]++
		out = append(out, f)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// ===== ② 父块扩展 =====

// section 一节拼好的内容。
type section struct {
	articleID    uint
	articleTitle string
	headingPath  string
	sectionIndex int
	content      string
	chunkIDs     []uint
	rrf          float64
	hitChunk     *repo.RetrievedChunk
}

// expandSections 把命中的小块拼回整节（小块召回保精度，整节给模型保完整）。
//
// 去重键必须是 (article_id, section_index)：section_index 是**文章内**的小节序号，
// A 文的第 3 节和 B 文的第 3 节毫无关系。只用 section_index 去重会把两篇的内容
// 拼进同一段——这类 bug 肉眼极难发现，而且答案会莫名其妙地串味。
func (s *chatService) expandSections(ctx context.Context, sc *Scope, fused []*fusedChunk) ([]*section, error) {
	// 按 (article, section) 去重，保留 RRF 分最高的那个作为「命中」
	type key = repo.SectionKey
	order := make([]key, 0, len(fused))
	best := make(map[key]*fusedChunk, len(fused))
	for _, f := range fused {
		k := key{ArticleID: f.chunk.ArticleID, SectionIndex: f.chunk.SectionIndex}
		if cur, ok := best[k]; !ok {
			best[k] = f
			order = append(order, k)
		} else if f.rrf > cur.rrf {
			best[k] = f
		}
	}

	keys := make([]repo.SectionKey, 0, len(order))
	for _, k := range order {
		keys = append(keys, k)
	}

	rows, err := s.repo.ListSectionChunks(keys)
	if err != nil {
		return nil, err
	}

	// 按 key 分组，保持 chunk_index 升序（仓储已排序）
	grouped := make(map[key][]*repo.RetrievedChunk, len(keys))
	articleTitle := make(map[uint]string, len(keys))
	for _, r := range rows {
		k := key{ArticleID: r.ArticleID, SectionIndex: r.SectionIndex}
		grouped[k] = append(grouped[k], r)
		if r.ArticleTitle != "" {
			articleTitle[r.ArticleID] = r.ArticleTitle
		}
	}

	out := make([]*section, 0, len(order))
	for _, k := range order {
		hit := best[k]
		blocks := grouped[k]
		if len(blocks) == 0 {
			// 查不到整节（极少见：块刚好被重新向量化替换掉了），退化为命中块本身
			out = append(out, sectionFromChunk(hit))
			continue
		}
		out = append(out, buildSection(k, hit, blocks, articleTitle[k.ArticleID]))
	}
	return out, nil
}

// buildSection 拼一节正文。
//
// 拼接规则：
//   - 整节 ≤2000 字：用整节
//   - 超了：只取「命中块 + 前后各一块」，保证上下文连贯又不过量
//
// 每一块开头都带了「标题 > 上级 > 当前小节」的路径前缀，拼起来会重复三遍，
// 所以拼完要剥掉紧随其后的重复行（只保留第一块带的那个）。
func buildSection(k repo.SectionKey, hit *fusedChunk,
	blocks []*repo.RetrievedChunk, title string) *section {

	sec := &section{
		articleID:    k.ArticleID,
		articleTitle: title,
		sectionIndex: k.SectionIndex,
		headingPath:  hit.chunk.HeadingPath,
		rrf:          hit.rrf,
		hitChunk:     hit.chunk,
	}

	total := 0
	for _, b := range blocks {
		total += b.CharCount
	}

	use := blocks
	if total > sectionMaxChars {
		// 退化：命中块 + 前后各一块
		idx := 0
		for i, b := range blocks {
			if b.ID == hit.chunk.ID {
				idx = i
				break
			}
		}
		from := maxInt(idx-1, 0)
		to := idx + 2
		if to > len(blocks) {
			to = len(blocks)
		}
		use = blocks[from:to]
	}

	parts := make([]string, 0, len(use))
	for i, b := range use {
		text := stripHeadingPrefix(b.Content, i == 0)
		if text == "" {
			continue
		}
		parts = append(parts, text)
		sec.chunkIDs = append(sec.chunkIDs, b.ID)
	}
	sec.content = strings.Join(parts, "\n")
	return sec
}

func sectionFromChunk(f *fusedChunk) *section {
	return &section{
		articleID:    f.chunk.ArticleID,
		articleTitle: f.chunk.ArticleTitle,
		sectionIndex: f.chunk.SectionIndex,
		headingPath:  f.chunk.HeadingPath,
		rrf:          f.rrf,
		content:      f.chunk.Content,
		chunkIDs:     []uint{f.chunk.ID},
		hitChunk:     f.chunk,
	}
}

func directSections(fused []*fusedChunk) []*section {
	out := make([]*section, 0, len(fused))
	for _, f := range fused {
		if f != nil && f.chunk != nil {
			out = append(out, sectionFromChunk(f))
		}
	}
	return out
}

// ===== ③ 预算装配 =====

// assembleContext 按预算拼最终给模型的资料文本。
//
// 三条上限同时生效：单次 ≤4 段、≤2500 字、整个请求累计 ≤6000 字（remaining）。
// 「当前文章先装」是刻意的：它是主体，引用文章只是补充。
// 也不做「每篇保底」——引满 7 篇时均分每篇只剩 400 字，每篇都不完整，反而更难答。
func assembleContext(sc *Scope, sections []*section, remaining int) (string, []entity.ChatRefChunk) {
	// 当前文章优先，其余按 RRF 分降序
	sort.SliceStable(sections, func(i, j int) bool {
		ci := sections[i].articleID == sc.CurrentArticleID
		cj := sections[j].articleID == sc.CurrentArticleID
		if ci != cj {
			return ci
		}
		return sections[i].rrf > sections[j].rrf
	})

	var b strings.Builder
	refs := make([]entity.ChatRefChunk, 0, singleCallMaxSegments)
	used := 0
	n := 0
	// 单次预算还要再被「整个请求的剩余额度」压一次：第 5 轮检索时通常只剩几百字。
	budget := minInt(singleCallMaxChars, remaining)

	for _, sec := range sections {
		if n >= singleCallMaxSegments {
			break
		}
		if sec.content == "" {
			continue
		}
		body := truncateRunes(sec.content, sectionMaxChars)
		size := len([]rune(body))
		// 第一段无论多长都装：至少要给模型一点真材料，否则整次检索等于白做。
		// 后续段落装不下就跳过、继续看下一段（后面的可能更短），而不是直接 break。
		if n > 0 && used+size > budget {
			continue
		}

		n++
		used += size
		b.WriteString(fmt.Sprintf("[%d] 《%s》· %s\n%s\n\n",
			n, sec.articleTitle, sec.headingPath, body))

		if sec.hitChunk != nil {
			refs = append(refs, entity.ChatRefChunk{
				ChunkID:      sec.hitChunk.ID,
				ArticleID:    sec.articleID,
				SectionIndex: sec.sectionIndex,
				HeadingPath:  sec.headingPath,
				RRFScore:     sec.rrf,
			})
		}
	}
	return strings.TrimSpace(b.String()), refs
}

// ===== ④ 全文降级 =====

// fullTextFallback 检索完全没结果时的兜底：直接把正文截一段给模型。
//
// 触发场景：文章没向量化、没配向量模型、pg_trgm 不可用，或问题确实与全文无关。
// 不降级的话，这些文章一句都答不出来——而「刚发布的文章还没切块」是最常见的情况。
func (s *chatService) fullTextFallback(ctx context.Context, sc *Scope, remaining int) (string, error) {
	ids := sc.ArticleIDs()
	var b strings.Builder
	n := 0

	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		text, err := s.repo.ArticleText(id)
		if err != nil {
			logger.Warn("读取文章正文失败", zap.Uint("article_id", id), zap.Error(err))
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		limit := fullTextRefChars
		title := sc.RefTitles[id]
		if id == sc.CurrentArticleID {
			limit = fullTextCurrentChars
			title = sc.CurrentArticleTitle
		}
		body := truncateRunes(text, limit)
		size := len([]rune(body))
		if n > 0 && len([]rune(b.String()))+size > remaining {
			break
		}
		n++
		b.WriteString(fmt.Sprintf("[%d] 《%s》· 全文（未切分）\n%s\n\n", n, title, body))
	}

	sc.Degraded = true
	if n == 0 {
		return "没有检索到相关内容，也没有可用的文章正文。请如实告知用户无法回答这个问题。", nil
	}
	sc.UsedChars += len([]rune(b.String()))
	return strings.TrimSpace(b.String()), nil
}

// ===== 小工具 =====

// questionWords 疑问词/代词。关键词路必须洗掉它们：
// 「为什么」在任何技术文章里都高频出现，不洗的话 trgm 会给一堆无关块打高分。
var questionWords = []string{
	"为什么", "是什么", "怎么样", "怎么", "如何", "哪些", "哪个", "什么",
	"这个", "那个", "这些", "那些", "是不是", "能不能", "可以", "请问",
	"我想知道", "一下", "关于", "以及", "还有", "吗", "呢", "吧",
}

// cleanQuery 清洗检索词。
//
// 模型给的 query 已经比用户原句干净得多（这是 agent 的天然优势），但仍要洗：
// 模型有时会照抄「为什么 Redis 用 Stream」这种带疑问词的短句。
func cleanQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	// 全角标点转半角空格，避免 similarity 被符号干扰
	replacer := strings.NewReplacer(
		"，", " ", "。", " ", "？", " ", "！", " ", "、", " ", "：", " ", "；", " ",
		"（", " ", "）", " ", "《", " ", "》", " ", "\n", " ", "\t", " ",
	)
	q = replacer.Replace(q)
	for _, w := range questionWords {
		q = strings.ReplaceAll(q, w, " ")
	}
	q = strings.Join(strings.Fields(q), " ")
	// 清洗后只剩一个字符的话，噪声远大于信息量
	if len([]rune(q)) < 2 {
		return ""
	}
	return q
}

// toPGIntArray 把 id 列表转成 PG 数组字面量 "{12,8}"。
//
// 不用 Go 切片直接传参：GORM + pgx 对 []uint 的数组绑定在不同版本上行为不一致
// （有时当成记录、有时报类型不匹配）。id 都是服务端校验过的正整数，拼接没有注入面。
func toPGIntArray(ids []uint) string {
	if len(ids) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// stripHeadingPrefix 处理块开头的标题路径前缀。
//
// 每块首行都是「标题 > 上级 > 当前小节」（参与 embedding 的召回信号）。
// 拼整节时：第一块保留（模型需要知道这是哪一节），后续块的首行如果与第一块相同就删掉，
// 否则会出现「标题 > 上级 > 当前小节」连刷三遍，凭空吃掉几十个 token。
func stripHeadingPrefix(content string, keepFirst bool) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	first, rest, found := strings.Cut(content, "\n")
	if !found {
		return content
	}
	if keepFirst {
		return content
	}
	// 只删「看着像标题路径」的首行：含 " > " 且不超过 200 字。
	// 正文段落里偶尔也会有 " > "，但一行 200 字以内还带这个符号的，基本都是路径。
	if strings.Contains(first, " > ") && len([]rune(first)) <= 200 {
		return strings.TrimSpace(rest)
	}
	return content
}

// embedder 按配置 id 缓存 llmclient，避免每次检索都新建 http.Client
// （不复用的连接会导致端口快速耗尽，高频问答时尤其明显）。
func (s *chatService) embedder(cfg llmclient.Config) *llmclient.Client {
	s.embedMu.Lock()
	defer s.embedMu.Unlock()
	// 用 BaseURL+Model 做 key：同一套配置才复用，改了配置自然换成新客户端
	key := cfg.BaseURL + "|" + cfg.Model + "|" + cfg.APIKey
	if s.embedClient != nil && s.embedKey == key {
		return s.embedClient
	}
	s.embedClient = llmclient.NewClient(cfg)
	s.embedKey = key
	return s.embedClient
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
