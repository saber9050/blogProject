package chat

import (
	"strings"
	"testing"

	repo "blog/internal/repository/chat"
)

// 混合检索是本模块技术含量最高、也最容易错的一块，但它的核心逻辑（融合、配额、
// 截断、清洗）全是纯函数——不需要数据库也不需要模型就能验。
// 这里覆盖的都是「写错了不会报错、只会让答案变差」的地方。

func TestCleanQuery(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"为什么 Redis 用 Stream？", "Redis 用 Stream"},
		// 中文没有空格分词，剥离疑问词后会留下粘连（"消费组是"）。
		// 不做逐字精修：trgm 是 3-gram 匹配，粘连一两个字影响很小，
		// 而过度清洗（比如把「是」「的」也删掉）反而会把有效词拆碎。
		{"这个消费组是怎么工作的", "消费组是 工作的"},
		{"Stream 消费组", "Stream 消费组"},
		{"为什么", ""}, // 洗完只剩疑问词 → 空（关键词路整条跳过）
		{"？？？", ""}, // 只剩标点
		{"Redis，发布订阅。", "Redis 发布订阅"},
	}
	for _, c := range cases {
		if got := cleanQuery(c.in); got != c.want {
			t.Errorf("cleanQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToPGIntArray(t *testing.T) {
	if got := toPGIntArray(nil); got != "{}" {
		t.Fatalf("空列表应为 {}，实际 %q", got)
	}
	if got := toPGIntArray([]uint{12, 8, 3}); got != "{12,8,3}" {
		t.Fatalf("实际 %q", got)
	}
}

func TestFuseRRF(t *testing.T) {
	sem := []*repo.RetrievedChunk{
		{ID: 1, ArticleID: 12},
		{ID: 2, ArticleID: 12},
	}
	kw := []*repo.RetrievedChunk{
		{ID: 2, ArticleID: 12},
		{ID: 3, ArticleID: 12},
	}

	got := fuseRRF(sem, kw)
	if len(got) != 3 {
		t.Fatalf("应融合出 3 个候选，实际 %d", len(got))
	}
	// id=2 两路都命中（rank1 关键词 + rank2 语义），分数应最高
	if got[0].chunk.ID != 2 {
		t.Fatalf("两路都命中的应排第一，实际第一是 %d", got[0].chunk.ID)
	}
	// 单路第一名 (1/61) 应高于单路第二名 (1/62)
	if got[1].chunk.ID != 1 {
		t.Fatalf("单路 rank1 应排第二，实际 %d", got[1].chunk.ID)
	}
}

func TestApplyArticleQuota(t *testing.T) {
	// 构造：当前文章 12 有 6 个候选，文章 8 有 3 个，RRF 分从高到低
	fused := []*fusedChunk{
		{chunk: &repo.RetrievedChunk{ID: 1, ArticleID: 12}, rrf: 0.9},
		{chunk: &repo.RetrievedChunk{ID: 2, ArticleID: 12}, rrf: 0.8},
		{chunk: &repo.RetrievedChunk{ID: 3, ArticleID: 12}, rrf: 0.7},
		{chunk: &repo.RetrievedChunk{ID: 4, ArticleID: 12}, rrf: 0.6},
		{chunk: &repo.RetrievedChunk{ID: 5, ArticleID: 12}, rrf: 0.5},
		{chunk: &repo.RetrievedChunk{ID: 6, ArticleID: 12}, rrf: 0.4},
		{chunk: &repo.RetrievedChunk{ID: 7, ArticleID: 8}, rrf: 0.35},
		{chunk: &repo.RetrievedChunk{ID: 8, ArticleID: 8}, rrf: 0.3},
		{chunk: &repo.RetrievedChunk{ID: 9, ArticleID: 8}, rrf: 0.25},
	}
	sc := &Scope{CurrentArticleID: 12}

	got := applyArticleQuota(fused, sc, fusionTopN)

	var cur, other int
	for _, f := range got {
		switch f.chunk.ArticleID {
		case 12:
			cur++
		case 8:
			other++
		}
	}
	// 当前文章 ≤4、其它 ≤2 —— 这是「本文为主、引用为辅」的硬约束
	if cur != currentArticleMaxSegments {
		t.Errorf("当前文章应取 %d 段，实际 %d", currentArticleMaxSegments, cur)
	}
	if other != otherArticleMaxSegments {
		t.Errorf("引用文章应取 %d 段，实际 %d", otherArticleMaxSegments, other)
	}
}

func TestApplyArticleQuotaSkipsSeen(t *testing.T) {
	// 第二轮检索必须排除第一轮已返回的块，否则模型看到重复内容会误判「资料就这些」
	fused := []*fusedChunk{
		{chunk: &repo.RetrievedChunk{ID: 1, ArticleID: 12}, rrf: 0.9},
		{chunk: &repo.RetrievedChunk{ID: 2, ArticleID: 12}, rrf: 0.8},
	}
	sc := &Scope{CurrentArticleID: 12, Seen: map[uint]struct{}{1: {}}}

	got := applyArticleQuota(fused, sc, fusionTopN)
	if len(got) != 1 || got[0].chunk.ID != 2 {
		t.Fatalf("应只剩 id=2，实际 %v", got)
	}
}

func TestAssembleContextPrefersCurrentArticle(t *testing.T) {
	sc := &Scope{CurrentArticleID: 12}
	sections := []*section{
		{articleID: 8, articleTitle: "引用文章", headingPath: "A", content: "引用的内容", rrf: 0.9,
			hitChunk: &repo.RetrievedChunk{ID: 100, ArticleID: 8}},
		{articleID: 12, articleTitle: "本文", headingPath: "B", content: "本文的内容", rrf: 0.1,
			hitChunk: &repo.RetrievedChunk{ID: 200, ArticleID: 12}},
	}

	text, refs := assembleContext(sc, sections, totalMaxChars)

	// RRF 分低的当前文章必须排在前面：它是主体，引用只是补充
	if !strings.HasPrefix(text, "[1] 《本文》") {
		t.Fatalf("当前文章应排在第一位，实际:\n%s", text)
	}
	if len(refs) != 2 || refs[0].ChunkID != 200 {
		t.Fatalf("refs 顺序应与资料一致，实际 %+v", refs)
	}
}

func TestAssembleContextRespectsBudget(t *testing.T) {
	sc := &Scope{CurrentArticleID: 12}
	big := strings.Repeat("字", 2000)
	sections := make([]*section, 0, 6)
	for i := 0; i < 6; i++ {
		sections = append(sections, &section{
			articleID: 12, articleTitle: "本文", headingPath: "小节", content: big, rrf: 1 - float64(i)*0.1,
			hitChunk: &repo.RetrievedChunk{ID: uint(i + 1), ArticleID: 12},
		})
	}

	text, _ := assembleContext(sc, sections, 6000)
	// 单次上限 2500 字 + 4 段，两个都要生效
	if n := strings.Count(text, "[1] ") + strings.Count(text, "[2] ") + strings.Count(text, "[3] ") +
		strings.Count(text, "[4] ") + strings.Count(text, "[5] "); n > singleCallMaxSegments {
		t.Fatalf("段数超过单次上限 %d，实际 %d", singleCallMaxSegments, n)
	}
	if len([]rune(text)) > singleCallMaxChars+200 {
		t.Fatalf("字符数明显超过单次上限 %d，实际 %d", singleCallMaxChars, len([]rune(text)))
	}
}

func TestStripHeadingPrefix(t *testing.T) {
	content := "Redis > 二，Stream > 消费组\n正文第一段"
	if got := stripHeadingPrefix(content, true); got != content {
		t.Fatalf("第一块应保留路径，实际 %q", got)
	}
	if got := stripHeadingPrefix(content, false); got != "正文第一段" {
		t.Fatalf("后续块应剥掉重复路径，实际 %q", got)
	}
	// 没有路径前缀的块不该被误删首行
	plain := "第一段就是正文\n第二段"
	if got := stripHeadingPrefix(plain, false); got != plain {
		t.Fatalf("无路径前缀时不应删首行，实际 %q", got)
	}
}
