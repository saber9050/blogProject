package chunker

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// mustSplit 测试辅助：切分并断言无错误（不传文章标题）
func mustSplit(t *testing.T, content string, p Params) []Result {
	t.Helper()
	got, err := Split(content, "", p)
	if err != nil {
		t.Fatalf("Split 返回错误：%v", err)
	}
	return got
}

// TestSplitEmptyContent 空内容必须返回 ErrEmptyContent。
//
// 如果这里返回一个内容为 "" 的块，向量化会拿着空文本去调 embedding，
// 多数服务报参数错误，而错误信息完全看不出根因。
// 清洗后为空的几种形态都要认出来：空串、纯空白、纯零宽字符、只有分隔线。
func TestSplitEmptyContent(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"空字符串", ""},
		{"只有空白", "   \n\n\t  "},
		{"只有零宽字符", "\u200b\u200b\ufeff"},
		{"只有分隔线", "---"},
		{"只有图片", "![](http://example.com/a.png)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Split(c.content, "", Params{ChunkSize: 800, MinChunkSize: 100, StripMarkup: true})
			if !errors.Is(err, ErrEmptyContent) {
				t.Fatalf("期望 ErrEmptyContent，实际 err=%v got=%+v", err, got)
			}
			if len(got) != 0 {
				t.Fatalf("期望 0 块，实际 %d 块", len(got))
			}
		})
	}
}

// TestSplitRejectsTooShort 正文太短要拒绝，不能产出一个无意义的向量。
//
// 依据来自真实数据：本地 19 篇文章里有 6 篇正文就是 "1" 这种
// 测试残留。这类内容拿去 embedding 既不报错也没价值，只会在库里留垃圾行。
func TestSplitRejectsTooShort(t *testing.T) {
	cases := []string{
		"1",
		"123456789", // 9 字，差一个字
		"哦",
	}
	for _, c := range cases {
		_, err := Split(c, "", Params{ChunkSize: 800, MinChunkSize: 0, StripMarkup: true})
		if !errors.Is(err, ErrContentTooShort) {
			t.Fatalf("%q 期望 ErrContentTooShort，实际 %v", c, err)
		}
		// 错误信息要带上当前字数，方便用户判断是不是写漏了
		if !strings.Contains(err.Error(), "字") {
			t.Fatalf("%q 的错误信息应含字数提示，实际：%s", c, err.Error())
		}
	}

	// 刚好 10 字应该通过（边界值不能卡掉）
	got := mustSplit(t, strings.Repeat("字", MinContentChars),
		Params{ChunkSize: 800, MinChunkSize: 10, StripMarkup: true})
	if len(got) != 1 {
		t.Fatalf("恰好 %d 字应通过，实际 %d 块", MinContentChars, len(got))
	}
}

func TestSplitShortContent(t *testing.T) {
	md := "这是一段**很短**的正文，只有一句话。"
	got := mustSplit(t, md, Params{ChunkSize: 800, MinChunkSize: 100, StripMarkup: true})

	if len(got) != 1 {
		t.Fatalf("期望 1 块，实际 %d 块", len(got))
	}
	// 加粗标记要去掉，文字保留
	want := "这是一段很短的正文，只有一句话。"
	if got[0].Content != want {
		t.Fatalf("内容不一致\n期望：%q\n实际：%q", want, got[0].Content)
	}
	if got[0].CharCount != utf8.RuneCountInString(want) {
		t.Fatalf("CharCount 期望 %d，实际 %d", utf8.RuneCountInString(want), got[0].CharCount)
	}
}

// TestHardSplitLongTextWithoutPunctuation 超长无标点文本按字数硬切。
//
// 这是最容易出问题的一条：如果按字节切，中文会被切成半个字符，
// 存进数据库后变成 U+FFFD 替换字符，看起来是「乱码」。
// overlap 设 0，这样每块长度就是上限，好断言。
func TestHardSplitLongTextWithoutPunctuation(t *testing.T) {
	text := strings.Repeat("字", 2500)
	got := mustSplit(t, text, Params{ChunkSize: 800, ChunkOverlap: 0, MinChunkSize: 100, StripMarkup: false})

	if len(got) != 4 {
		t.Fatalf("2500 字按 800 切，期望 4 块，实际 %d 块", len(got))
	}
	total := 0
	for i, r := range got {
		if r.CharCount > 800 {
			t.Fatalf("第 %d 块 %d 字，超过上限 800", i, r.CharCount)
		}
		if !utf8.ValidString(r.Content) {
			t.Fatalf("第 %d 块不是合法 UTF-8", i)
		}
		if strings.ContainsRune(r.Content, utf8.RuneError) {
			t.Fatalf("第 %d 块含替换字符（说明按字节切碎了中文）", i)
		}
		total += r.CharCount
	}
	// 拼回去应该一字不差
	if total != 2500 {
		t.Fatalf("总字数 %d，期望 2500（有内容丢失）", total)
	}
}

func TestSplitByParagraph(t *testing.T) {
	p1 := "第一段" + strings.Repeat("甲", 197)
	p2 := "第二段" + strings.Repeat("乙", 197)
	p3 := "第三段" + strings.Repeat("丙", 197)
	md := p1 + "\n\n" + p2 + "\n\n" + p3

	got := mustSplit(t, md, Params{ChunkSize: 300, ChunkOverlap: 0, MinChunkSize: 50, StripMarkup: true})

	if len(got) != 3 {
		t.Fatalf("期望 3 块（每段一块），实际 %d 块", len(got))
	}
	markers := []struct{ own, other string }{
		{"第一段", "第二段"},
		{"第二段", "第三段"},
		{"第三段", "第一段"},
	}
	for i, m := range markers {
		if !strings.HasPrefix(got[i].Content, m.own) {
			t.Fatalf("第 %d 块应以 %q 开头，实际开头：%.20q", i, m.own, got[i].Content)
		}
		if strings.Contains(got[i].Content, m.other) {
			t.Fatalf("第 %d 块跨段了：包含了 %q", i, m.other)
		}
	}
}

// TestOverlapApplied 相邻块之间有重叠。
func TestOverlapApplied(t *testing.T) {
	first := strings.Repeat("甲", 100)
	second := strings.Repeat("乙", 100)
	text := first + "。" + second + "。"

	withOverlap := mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 10, MinChunkSize: 0, StripMarkup: false})
	noOverlap := mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 0, MinChunkSize: 0, StripMarkup: false})

	if len(withOverlap) < 2 || len(noOverlap) < 2 {
		t.Fatalf("应至少切出 2 块，实际 overlap=%d / no-overlap=%d", len(withOverlap), len(noOverlap))
	}

	if strings.Contains(noOverlap[1].Content, "甲") {
		t.Fatalf("overlap=0 时第二块不该含第一块内容：%.40q", noOverlap[1].Content)
	}

	prev := []rune(withOverlap[0].Content)
	wantPrefix := string(prev[len(prev)-10:])
	if !strings.HasPrefix(withOverlap[1].Content, wantPrefix) {
		t.Fatalf("第二块开头应为第一块结尾 %q\n实际开头：%q",
			wantPrefix, string([]rune(withOverlap[1].Content)[:10]))
	}
	if withOverlap[1].CharCount <= noOverlap[1].CharCount {
		t.Fatalf("有重叠时第二块应更长：%d vs %d", withOverlap[1].CharCount, noOverlap[1].CharCount)
	}
}

// TestMergeSmallChunks 小节内的碎块被合并。
//
// 结构感知切分之后，「小标题单独成块」这个问题不存在了（标题就是边界）。
// 但小节内部仍会产生碎块：一段短说明后面跟一大段正文，按字数切完尾巴会剩几十字。
// 几个字的块拿去 embedding 是纯浪费——向量里几乎没有信息，却占一次 API 调用。
func TestMergeSmallChunks(t *testing.T) {
	// 450 字硬切成 200 / 200 / 50，尾巴 50 字低于 minSize，应当并进前一块
	md := "## 小结\n\n" + strings.Repeat("设", 450)
	got := mustSplit(t, md, Params{ChunkSize: 200, ChunkOverlap: 0, MinChunkSize: 100, StripMarkup: true})

	if len(got) == 0 {
		t.Fatal("不该返回空结果")
	}
	for i, r := range got {
		if r.CharCount < 100 {
			t.Fatalf("第 %d 块只有 %d 字，小于 min_chunk_size=100，未被合并：%q",
				i, r.CharCount, r.Content)
		}
	}
	if len(got) != 2 {
		t.Fatalf("450 字按 200 切应该得到 2 块（尾块被合并），实际 %d 块", len(got))
	}
	for _, r := range got {
		if !strings.HasPrefix(r.Content, "小结") {
			t.Fatalf("块首缺少标题路径：%q", r.Content)
		}
	}
}

// TestMergeRespectsMaxSize 碎块合并不能突破硬上限。
//
// 真实数据里出现过 904 字的块（目标 800）：一个接近满的块后面跟着一串
// 短列表项，每次合并都在往上加，最后涨了 13%。块大小是模板的承诺，
// 悄悄涨一半比多一个碎块更难排查，所以宁可留碎块。
func TestMergeRespectsMaxSize(t *testing.T) {
	const (
		size    = 200
		minSize = 50
		maxSize = size + minSize // 与 Split 内部的算法一致
	)

	// 195 字的正文 + 20 个短标题，全部想并进第一块的话会达到 295 字
	var b strings.Builder
	b.WriteString(strings.Repeat("正", size-5))
	for i := 0; i < 20; i++ {
		b.WriteString("\n\n小标题")
	}

	got := mustSplit(t, b.String(), Params{
		ChunkSize: size, ChunkOverlap: 0, MinChunkSize: minSize, StripMarkup: true,
	})

	if len(got) == 0 {
		t.Fatal("不该返回空")
	}
	for i, r := range got {
		if r.CharCount > maxSize {
			t.Fatalf("第 %d 块 %d 字，超过硬上限 %d", i, r.CharCount, maxSize)
		}
	}
	// 内容不能丢
	joined := ""
	for _, r := range got {
		joined += r.Content
	}
	if strings.Count(joined, "小标题") != 20 {
		t.Fatalf("合并后小标题数量不对：期望 20，实际 %d", strings.Count(joined, "小标题"))
	}
}

// TestParagraphBreakPreserved 段落边界不能被压平。
//
// Markdown 里空行就是段落边界，切分优先按 "\n\n" 分。压平了就只能按字数硬切，
// 块边界会落在句子中间。
func TestParagraphBreakPreserved(t *testing.T) {
	md := "第一段\n\n第二段\n\n第三段"
	if got := MarkdownToPlainText(md); got != md {
		t.Fatalf("段落边界丢失\n期望：%q\n实际：%q", md, got)
	}

	// 单换行（行内断行）也要保留，只是不构成段落
	got := MarkdownToPlainText("第一行\n第二行")
	want := "第一行\n第二行"
	if got != want {
		t.Fatalf("单换行处理错误\n期望：%q\n实际：%q", want, got)
	}
}

// TestZeroWidthRemoved 零宽字符要被清掉。
//
// 实测真实文章的第一块开头就有一个 \u200b：不显示、但占字符数，
// 会让块首出现看不见的脏字符，也会让 CharCount 虚高。
func TestZeroWidthRemoved(t *testing.T) {
	got := MarkdownToPlainText("\u200b正文\u200b内容\ufeff。")
	want := "正文内容。"
	if got != want {
		t.Fatalf("零宽字符未被清除\n期望：%q\n实际：%q", want, got)
	}

	// 整个内容只有零宽字符，应当视为空
	if s := MarkdownToPlainText("\u200b\u200b"); s != "" {
		t.Fatalf("只有零宽字符应转成空串，实际 %q", s)
	}
}

// TestParamsDefaults 非法参数被兜底而不是报错。
//
// 参数校验在 chunktemplate 服务层，走到这里说明数据已过校验，
// 剩下的只可能是历史脏数据。让它跑出结果比让向量化失败更有用。
func TestParamsDefaults(t *testing.T) {
	text := strings.Repeat("字", 2000)

	// ChunkSize 为 0 → 用默认 800
	got := mustSplit(t, text, Params{ChunkSize: 0, ChunkOverlap: 0, MinChunkSize: 0})
	if len(got) == 0 {
		t.Fatal("ChunkSize=0 时应 fallback 到默认值，不该返回空")
	}
	for i, r := range got {
		if r.CharCount > DefaultChunkSize {
			t.Fatalf("第 %d 块 %d 字，超过默认上限 %d", i, r.CharCount, DefaultChunkSize)
		}
	}

	// ChunkOverlap 大于一半 → 被截到一半，不会无限叠加
	// 上限再加 2：重叠与本块之间有一个 "\n\n" 分隔符（见 applyOverlap）
	got = mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 500, MinChunkSize: 0})
	for i, r := range got {
		if r.CharCount > 100+50+2 {
			t.Fatalf("第 %d 块 %d 字，超过上限 ChunkSize+ChunkSize/2+2=152", i, r.CharCount)
		}
	}

	// MinChunkSize 大于 ChunkSize → 被截到 ChunkSize
	got = mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 0, MinChunkSize: 9999})
	if len(got) == 0 {
		t.Fatal("MinChunkSize 越界时不该返回空")
	}
}

// TestDefaultParams 生产参数已固化，改这三个数会影响全部已切好的块，
// 用例在这里钉死，改常量时必须同步改测试——等于强制停下来想一次。
func TestDefaultParams(t *testing.T) {
	got := DefaultParams()
	want := Params{ChunkSize: 800, ChunkOverlap: 80, MinChunkSize: 100, StripMarkup: true}
	if got != want {
		t.Fatalf("默认参数变化了：期望 %+v，实际 %+v", want, got)
	}
}

// TestExtractText 导出函数的两种模式。
func TestExtractText(t *testing.T) {
	md := "## 第一段\n\n第二段**加粗**"
	if got := ExtractText(md, true); got != "第一段\n\n第二段加粗" {
		t.Fatalf("stripMarkup=true 结果错误：%q", got)
	}
	// 不洗时标记原样保留，但零宽字符与多余空白仍要清掉
	if got := ExtractText("\u200b##  原样  ", false); got != "## 原样" {
		t.Fatalf("stripMarkup=false 结果错误：%q", got)
	}
}

// TestMarkdownToPlainText Markdown 清洗：去标记、留内容、段落边界不丢。
func TestMarkdownToPlainText(t *testing.T) {
	cases := []struct{ md, want string }{
		{"# 一级标题", "一级标题"},
		{"### 三级标题", "三级标题"},
		{"这是**加粗**的文本", "这是加粗的文本"},
		{"这是*斜体*的文本", "这是斜体的文本"},
		{"这是~~删除线~~文本", "这是删除线文本"},
		{"行内 `code` 示例", "行内 code 示例"},
		{"- 第一项\n- 第二项", "第一项\n第二项"},
		{"1. 第一步\n2. 第二步", "第一步\n第二步"},
		{"[点这里](https://example.com) 查看", "点这里 查看"},
		{"![示意图](https://example.com/a.png) 正文", "正文"},
		{"> 引用的一句话", "引用的一句话"},
		{"---", ""},
		{"表格：\n| 命令 | 作用 |\n|---|---|\n| XADD | 添加 |", "表格：\n| 命令 | 作用 |\n| XADD | 添加 |"},
	}
	for _, c := range cases {
		if got := MarkdownToPlainText(c.md); got != c.want {
			t.Errorf("MarkdownToPlainText(%q)\n期望：%q\n实际：%q", c.md, c.want, got)
		}
	}
}

// TestMarkdownCodeBlock 代码块内容要保留，且块内的标记不能被误删。
func TestMarkdownCodeBlock(t *testing.T) {
	md := "说明文字\n\n```go\n// 这是注释\nx := a * b\n```\n\n结束"
	got := MarkdownToPlainText(md)
	if !strings.Contains(got, "这是注释") {
		t.Fatalf("代码块内容丢失：%q", got)
	}
	if !strings.Contains(got, "a * b") {
		t.Fatalf("代码里的 * 被误删：%q", got)
	}
	if strings.Contains(got, "```") {
		t.Fatalf("围栏没去掉：%q", got)
	}
	// 段落边界要保住
	if !strings.Contains(got, "\n\n") {
		t.Fatalf("段落边界丢了：%q", got)
	}
}

// TestSplitBySection 块不跨小节，且带上标题路径。
func TestSplitBySection(t *testing.T) {
	md := "## 一、背景\n\n" + strings.Repeat("甲", 300) +
		"\n\n## 二、方法\n\n" + strings.Repeat("乙", 300) +
		"\n\n## 三、结论\n\n" + strings.Repeat("丙", 300)

	got := mustSplit(t, md, Params{ChunkSize: 800, ChunkOverlap: 0, MinChunkSize: 0, StripMarkup: true})
	if len(got) != 3 {
		t.Fatalf("三个小节应切成 3 块，实际 %d 块", len(got))
	}
	for i, want := range []string{"一、背景", "二、方法", "三、结论"} {
		if got[i].HeadingPath != want {
			t.Fatalf("第 %d 块路径期望 %q，实际 %q", i, want, got[i].HeadingPath)
		}
		if !strings.HasPrefix(got[i].Content, want) {
			t.Fatalf("第 %d 块开头应是路径：%q", i, got[i].Content)
		}
		// 不跨小节：不该出现别的段落标记
		other := []string{"二、方法", "三、结论", "一、背景"}[i]
		for j, o := range []string{"一、背景", "二、方法", "三、结论"} {
			if j != i && strings.Contains(got[i].Content, o) {
				t.Fatalf("第 %d 块混入了 %q", i, o)
			}
		}
		_ = other
	}
}

// TestHeadingPathWithDocTitle 路径带上文章标题，且只取两级小节。
func TestHeadingPathWithDocTitle(t *testing.T) {
	md := "## 一级\n\n正文一\n\n### 二级\n\n正文二\n\n#### 三级\n\n正文三"
	got, err := Split(md, "我的文章", Params{ChunkSize: 800, ChunkOverlap: 0, MinChunkSize: 0, StripMarkup: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("期望 3 块，实际 %d", len(got))
	}
	wants := []string{"我的文章 > 一级", "我的文章 > 一级 > 二级", "我的文章 > 二级 > 三级"}
	for i, w := range wants {
		if got[i].HeadingPath != w {
			t.Fatalf("第 %d 块路径期望 %q，实际 %q", i, w, got[i].HeadingPath)
		}
	}
}

// TestAtomicTable 表格不被切断，哪怕它超长。
func TestAtomicTable(t *testing.T) {
	var b strings.Builder
	b.WriteString("## 命令表\n\n")
	b.WriteString("| 命令 | 说明 |\n| --- | --- |\n")
	for i := 0; i < 60; i++ {
		b.WriteString("| XADD" + strings.Repeat("x", 20) + " | 向流末尾追加消息的说明文字 |\n")
	}

	got := mustSplit(t, b.String(), Params{ChunkSize: 200, ChunkOverlap: 0, MinChunkSize: 0, StripMarkup: true})
	if len(got) != 1 {
		t.Fatalf("整张表格应作为 1 块，实际 %d 块", len(got))
	}
	// 表格所有行都在同一块里
	if n := strings.Count(got[0].Content, "XADD"); n != 60 {
		t.Fatalf("表格被切断了：只保留 %d 行，期望 60 行", n)
	}
}

// TestAtomicCodeBlock 代码块不被切断。
func TestAtomicCodeBlock(t *testing.T) {
	var b strings.Builder
	b.WriteString("## 示例代码\n\n```go\n")
	for i := 0; i < 50; i++ {
		b.WriteString("func handler" + strings.Repeat("x", 10) + "() { return }\n")
	}
	b.WriteString("```\n")

	got := mustSplit(t, b.String(), Params{ChunkSize: 200, ChunkOverlap: 0, MinChunkSize: 0, StripMarkup: true})
	if len(got) != 1 {
		t.Fatalf("整个代码块应作为 1 块，实际 %d 块", len(got))
	}
	if n := strings.Count(got[0].Content, "func handler"); n != 50 {
		t.Fatalf("代码块被切断了：只保留 %d 行，期望 50 行", n)
	}
	if strings.Contains(got[0].Content, "```") {
		t.Fatalf("围栏没去掉：%.60q", got[0].Content)
	}
}

// TestNoHeadingFallsBack 没有标题的文章退化为递归切分，不能报错。
func TestNoHeadingFallsBack(t *testing.T) {
	md := strings.Repeat("这是一段没有标题的正文。", 100)
	got := mustSplit(t, md, Params{ChunkSize: 200, ChunkOverlap: 0, MinChunkSize: 0, StripMarkup: true})
	if len(got) < 2 {
		t.Fatalf("700 字按 200 切应得到多块，实际 %d 块", len(got))
	}
	for i, r := range got {
		if r.HeadingPath != "" {
			t.Fatalf("第 %d 块不该有路径，实际 %q", i, r.HeadingPath)
		}
		if r.SectionIndex != 0 {
			t.Fatalf("第 %d 块小节序号应为 0，实际 %d", i, r.SectionIndex)
		}
	}
}

// TestSectionIndexShared 同一小节切出的多块共享 section_index（父子块的基础）。
func TestSectionIndexShared(t *testing.T) {
	md := "## 长小节\n\n" + strings.Repeat("甲", 500) + "\n\n## 下一节\n\n" + strings.Repeat("乙", 100)
	got := mustSplit(t, md, Params{ChunkSize: 200, ChunkOverlap: 0, MinChunkSize: 0, StripMarkup: true})

	first := got[0].SectionIndex
	n := 0
	for _, r := range got {
		if r.HeadingPath == "长小节" {
			if r.SectionIndex != first {
				t.Fatalf("同一小节的多块 section_index 不一致：%d vs %d", first, r.SectionIndex)
			}
			n++
		}
	}
	if n < 2 {
		t.Fatalf("长小节应切成多块，实际 %d 块", n)
	}
	// 下一节必须是不同的序号
	for _, r := range got {
		if r.HeadingPath == "下一节" && r.SectionIndex == first {
			t.Fatal("不同小节不该共享 section_index")
		}
	}
}
