package chunker

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"blog/internal/model/entity"
)

// mustSplit 测试辅助：切分并断言无错误
func mustSplit(t *testing.T, content string, p Params) []Result {
	t.Helper()
	got, err := Split(content, p)
	if err != nil {
		t.Fatalf("Split 返回错误：%v", err)
	}
	return got
}

// TestSplitEmptyContent 空内容必须返回 ErrEmptyContent。
//
// 编辑器清空正文后存的是 `<p><br></p>`，不是空字符串。如果这里返回一个
// 内容为 "" 的块，向量化会拿着空文本去调 embedding，多数服务报参数错误，
// 而错误信息完全看不出根因。
func TestSplitEmptyContent(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"空字符串", ""},
		{"编辑器空内容", "<p><br></p>"},
		{"只有标签", "<p></p><div></div>"},
		{"只有空白", "   \n\n\t  "},
		{"只有 script", "<script>var a=1</script>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Split(c.content, Params{ChunkSize: 800, MinChunkSize: 100, StripHTML: true})
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
// 依据来自真实数据：本地 19 篇文章里有 6 篇正文就是 `<p>1</p>` 这种
// 测试残留。这类内容拿去 embedding 既不报错也没价值，只会在库里留垃圾行。
func TestSplitRejectsTooShort(t *testing.T) {
	cases := []string{
		"<p>1</p>",
		"<p>123456789</p>", // 9 字，差一个字
		"<p>哦</p>",
	}
	for _, c := range cases {
		_, err := Split(c, Params{ChunkSize: 800, MinChunkSize: 0, StripHTML: true})
		if !errors.Is(err, ErrContentTooShort) {
			t.Fatalf("%q 期望 ErrContentTooShort，实际 %v", c, err)
		}
		// 错误信息要带上当前字数，方便用户判断是不是写漏了
		if !strings.Contains(err.Error(), "字") {
			t.Fatalf("%q 的错误信息应含字数提示，实际：%s", c, err.Error())
		}
	}

	// 刚好 10 字应该通过（边界值不能卡掉）
	got := mustSplit(t, "<p>"+strings.Repeat("字", MinContentChars)+"</p>",
		Params{ChunkSize: 800, MinChunkSize: 10, StripHTML: true})
	if len(got) != 1 {
		t.Fatalf("恰好 %d 字应通过，实际 %d 块", MinContentChars, len(got))
	}
}

// TestSplitShortContent 短内容只有一块，且内容与原文一致。
func TestSplitShortContent(t *testing.T) {
	html := `<p>这是一段很短的正文，只有一句话。</p>`
	got := mustSplit(t, html, Params{ChunkSize: 800, MinChunkSize: 100, StripHTML: true})

	if len(got) != 1 {
		t.Fatalf("期望 1 块，实际 %d 块", len(got))
	}
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
	got := mustSplit(t, text, Params{ChunkSize: 800, ChunkOverlap: 0, MinChunkSize: 100, StripHTML: false})

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

// TestSplitByParagraph 段落优先，不跨段。
//
// 每段 200 字、块大小 300：两块拼起来 400 > 300，所以一段必须独占一块。
func TestSplitByParagraph(t *testing.T) {
	p1 := "第一段" + strings.Repeat("甲", 197)
	p2 := "第二段" + strings.Repeat("乙", 197)
	p3 := "第三段" + strings.Repeat("丙", 197)
	html := "<p>" + p1 + "</p><p>" + p2 + "</p><p>" + p3 + "</p>"

	got := mustSplit(t, html, Params{ChunkSize: 300, ChunkOverlap: 0, MinChunkSize: 50, StripHTML: true})

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

// TestScriptStyleRemoved 脚本与样式连同内容整段删除。
//
// 只剥标签不删内容的话，CSS 选择器和 JS 变量名都会进向量，变成噪声。
func TestScriptStyleRemoved(t *testing.T) {
	html := `<style>.a{color:red}</style>` +
		`<p>正文第一部分。</p>` +
		`<script>var secretKey="不该出现的脚本内容";</script>` +
		`<p>正文第二部分。</p>`

	got := mustSplit(t, html, Params{ChunkSize: 800, MinChunkSize: 10, StripHTML: true})
	joined := ""
	for _, r := range got {
		joined += r.Content
	}

	for _, bad := range []string{"不该出现", "secretKey", "color:red", "var "} {
		if strings.Contains(joined, bad) {
			t.Fatalf("结果里残留了 %q：%s", bad, joined)
		}
	}
	for _, want := range []string{"正文第一部分", "正文第二部分"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("结果里缺少正文 %q：%s", want, joined)
		}
	}
}

// TestOverlapApplied 相邻块之间有重叠。
func TestOverlapApplied(t *testing.T) {
	first := strings.Repeat("甲", 100)
	second := strings.Repeat("乙", 100)
	text := first + "。" + second + "。"

	withOverlap := mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 10, MinChunkSize: 0, StripHTML: false})
	noOverlap := mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 0, MinChunkSize: 0, StripHTML: false})

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

// TestMergeSmallChunks 碎块被合并，结果里不该有小于阈值的块。
//
// 碎块来源很实际：小标题单独一行、列表项很短。几个字的块拿去 embedding
// 是纯浪费——向量里几乎没有信息，却占一次 API 调用和一行存储。
func TestMergeSmallChunks(t *testing.T) {
	html := "<p>一、背景</p>" +
		"<p>" + strings.Repeat("正", 300) + "</p>" +
		"<p>二、方法</p>"

	got := mustSplit(t, html, Params{ChunkSize: 200, ChunkOverlap: 0, MinChunkSize: 100, StripHTML: true})

	if len(got) == 0 {
		t.Fatal("不该返回空结果")
	}
	for i, r := range got {
		if r.CharCount < 100 {
			t.Fatalf("第 %d 块只有 %d 字，小于 min_chunk_size=100，未被合并：%q",
				i, r.CharCount, r.Content)
		}
	}
	joined := ""
	for _, r := range got {
		joined += r.Content
	}
	for _, want := range []string{"一、背景", "二、方法"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("合并后丢失了 %q", want)
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
	b.WriteString("<p>" + strings.Repeat("正", size-5) + "</p>")
	for i := 0; i < 20; i++ {
		b.WriteString("<p>小标题</p>")
	}

	got := mustSplit(t, b.String(), Params{
		ChunkSize: size, ChunkOverlap: 0, MinChunkSize: minSize, StripHTML: true,
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

// TestHTMLUnescapeAfterStrip 实体反转义必须发生在剥标签之后。
//
// 顺序错了就会踩这个坑：`&lt;script&gt;` 先被反转义成真标签，
// 紧接着被「删除所有标签」这一步当成标签删掉，正文里的「<script>」字样凭空消失。
func TestHTMLUnescapeAfterStrip(t *testing.T) {
	cases := []struct{ html, want string }{
		{"<p>a &lt; b</p>", "a < b"},
		{"<p>x &amp;&amp; y</p>", "x && y"},
		{"<p>&lt;script&gt; 这个字样要保留</p>", "<script> 这个字样要保留"},
		{"<p>引号 &quot;测试&quot;</p>", `引号 "测试"`},
		{"<p>不换行&nbsp;空格</p>", "不换行 空格"},
	}
	for _, c := range cases {
		got := ToPlainText(c.html)
		if got != c.want {
			t.Fatalf("ToPlainText(%q)\n期望：%q\n实际：%q", c.html, c.want, got)
		}
	}
}

// TestParagraphBreakPreserved 段落边界不能被压平。
//
// 现有 llm_service.stripHTMLTags 把 `\s+` 压成单个空格，整篇文章变成一行。
// 直接复用它的会切出一堆语义断裂的块，所以必须单独实现并锁住这个行为。
func TestParagraphBreakPreserved(t *testing.T) {
	html := "<p>第一段</p><p>第二段</p><p>第三段</p>"
	got := ToPlainText(html)
	want := "第一段\n\n第二段\n\n第三段"
	if got != want {
		t.Fatalf("段落边界丢失\n期望：%q\n实际：%q", want, got)
	}

	// <br> 只换行不分段
	got = ToPlainText("第一行<br>第二行")
	want = "第一行\n第二行"
	if got != want {
		t.Fatalf("<br> 处理错误\n期望：%q\n实际：%q", want, got)
	}
}

// TestZeroWidthRemoved 零宽字符要被清掉。
//
// 实测真实文章的第一块开头就有一个 \u200b：不显示、但占字符数，
// 会让块首出现看不见的脏字符，也会让 CharCount 虚高。
func TestZeroWidthRemoved(t *testing.T) {
	got := ToPlainText("<p>\u200b正文\u200b内容\ufeff。</p>")
	want := "正文内容。"
	if got != want {
		t.Fatalf("零宽字符未被清除\n期望：%q\n实际：%q", want, got)
	}

	// 整个内容只有零宽字符，应当视为空
	if s := ToPlainText("<p>\u200b\u200b</p>"); s != "" {
		t.Fatalf("只有零宽字符应转成空串，实际 %q", s)
	}
}

// TestTableCellSeparated 表格单元格之间要有分隔。
//
// 实测真实文章里出现过 "enctypeapplication/x-www-form-urlencoded" 这种
// 粘在一起的内容——`</td>` 不在分段列表里，相邻单元格被直接拼接了。
func TestTableCellSeparated(t *testing.T) {
	html := "<table><tr><td>enctype</td><td>说明</td></tr></table>"
	got := ToPlainText(html)

	if strings.Contains(got, "enctype说明") {
		t.Fatalf("单元格被拼接了：%q", got)
	}
	if !strings.Contains(got, "enctype") || !strings.Contains(got, "说明") {
		t.Fatalf("单元格内容丢失：%q", got)
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
	got = mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 500, MinChunkSize: 0})
	for i, r := range got {
		if r.CharCount > 100+50 {
			t.Fatalf("第 %d 块 %d 字，超过上限 ChunkSize+ChunkSize/2=150", i, r.CharCount)
		}
	}

	// MinChunkSize 大于 ChunkSize → 被截到 ChunkSize
	got = mustSplit(t, text, Params{ChunkSize: 100, ChunkOverlap: 0, MinChunkSize: 9999})
	if len(got) == 0 {
		t.Fatal("MinChunkSize 越界时不该返回空")
	}
}

// TestParamsFromTemplate 模板参数能正确映射，nil 模板走默认值。
func TestParamsFromTemplate(t *testing.T) {
	tpl := &entity.ChunkTemplate{
		ChunkSize: 600, ChunkOverlap: 60, MinChunkSize: 80, StripHTML: false,
	}
	got := ParamsFromTemplate(tpl)
	want := Params{ChunkSize: 600, ChunkOverlap: 60, MinChunkSize: 80, StripHTML: false}
	if got != want {
		t.Fatalf("期望 %+v，实际 %+v", want, got)
	}

	got = ParamsFromTemplate(nil)
	if got.ChunkSize != DefaultChunkSize || !got.StripHTML {
		t.Fatalf("nil 模板应走默认值，实际 %+v", got)
	}
}

// TestExtractText 导出函数的两种模式。
func TestExtractText(t *testing.T) {
	html := "<p>第一段</p><p>第二段</p>"
	if got := ExtractText(html, true); got != "第一段\n\n第二段" {
		t.Fatalf("stripHTML=true 结果错误：%q", got)
	}
	// 不剥标签时标签原样保留，但零宽字符仍要清掉
	if got := ExtractText("<p>\u200b原样</p>", false); got != "<p>原样</p>" {
		t.Fatalf("stripHTML=false 结果错误：%q", got)
	}
}
