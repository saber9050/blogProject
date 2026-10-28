// Package chunker 把文章正文切成适合向量化的块。
//
// 这里刻意做成纯函数：不碰数据库、不发 HTTP、不读配置。
// 切分是整个向量化链路里唯一「参数敏感、结果难验证」的一步，
// 只有它是纯函数，才能用单测把边界情况全部钉死。
package chunker

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"blog/internal/model/entity"
)

// 默认参数。块大小取 800 字：一篇 5000 字的文章切成 6~7 块，
// 每块都在主流 embedding 模型的输入上限（512~8192 token）之内，
// 又不会把一整个章节压成一块导致检索粒度太粗。
const (
	DefaultChunkSize    = 800
	DefaultChunkOverlap = 80
	DefaultMinChunkSize = 100

	// MinContentChars 正文少于这个字数就不切。
	//
	// 不是拍脑袋定的：实测本地 19 篇文章里有 6 篇正文就是 `<p>1</p>`
	// 这种测试残留，纯文本 1 个字。这类内容拿去 embedding 会产出一个
	// 几乎不含信息的向量，既浪费一次 API 调用，又会在库里留一行垃圾，
	// 将来做检索时它还可能被召回。宁可明确报错让人知道「这篇没内容」。
	MinContentChars = 10
)

// 切分失败的两类原因。分开放是因为给用户的提示不同：
// 一个是「你没写内容」，一个是「内容太短」，后者往往不符合直觉。
var (
	ErrEmptyContent    = errors.New("文章正文为空")
	ErrContentTooShort = errors.New("文章正文过少，无法向量化")
)

// Params 切分参数
type Params struct {
	ChunkSize    int  // 目标块大小（字符/rune）
	ChunkOverlap int  // 相邻块重叠字符数
	MinChunkSize int  // 小于该值的块并入相邻块
	StripHTML    bool // 是否先把 HTML 转成纯文本
}

// Result 一个切分块
type Result struct {
	Content   string
	CharCount int // rune 数，不是字节数
}

// ParamsFromTemplate 用切分模板构造参数
func ParamsFromTemplate(t *entity.ChunkTemplate) Params {
	if t == nil {
		return Params{
			ChunkSize:    DefaultChunkSize,
			ChunkOverlap: DefaultChunkOverlap,
			MinChunkSize: DefaultMinChunkSize,
			StripHTML:    true,
		}
	}
	return Params{
		ChunkSize:    t.ChunkSize,
		ChunkOverlap: t.ChunkOverlap,
		MinChunkSize: t.MinChunkSize,
		StripHTML:    t.StripHTML,
	}
}

// Split 把正文切成块。
//
// 返回的三类错误要分开处理，因为给用户的提示完全不同：
//   - ErrEmptyContent    正文是空的（编辑器清空后存的是 `<p><br></p>`）
//   - ErrContentTooShort 正文不足 MinContentChars
//   - 其它               不应该发生
func Split(content string, p Params) ([]Result, error) {
	p = p.withDefaults()

	text := ExtractText(content, p.StripHTML)
	if text == "" {
		return nil, ErrEmptyContent
	}
	if n := runeLen(text); n < MinContentChars {
		return nil, fmt.Errorf("%w（当前 %d 字，至少 %d 字）", ErrContentTooShort, n, MinContentChars)
	}

	chunks := splitRecursive(text, 0, p.ChunkSize)
	chunks = mergeSmall(chunks, p.MinChunkSize, p.ChunkSize+p.MinChunkSize)
	chunks = applyOverlap(chunks, p.ChunkOverlap)

	out := make([]Result, 0, len(chunks))
	for _, c := range chunks {
		if c == "" {
			continue
		}
		out = append(out, Result{Content: c, CharCount: runeLen(c)})
	}
	return out, nil
}

// ExtractText 按参数把正文转成纯文本。单独导出是因为 service 层在报错时
// 需要知道正文长度（判断「空」还是「太短」），不该为此把切分再跑一遍。
func ExtractText(content string, stripHTML bool) string {
	if stripHTML {
		return ToPlainText(content)
	}
	// 不剥标签时也要清洗零宽字符和多余空白，否则块首块尾会带看不见的脏字符
	return normalizeSpace(content)
}

// withDefaults 兜住非法参数。
//
// 不做报错而是静默修正：参数合法性由 chunktemplate 服务在写入时校验，
// 走到这里说明数据已经过校验，剩下的只可能是历史脏数据。让它跑出结果
// 比让整个向量化任务失败更有用——失败原因里出现「chunk_size 不能为 0」
// 对排查毫无帮助。
func (p Params) withDefaults() Params {
	if p.ChunkSize <= 0 {
		p.ChunkSize = DefaultChunkSize
	}
	if p.ChunkOverlap < 0 {
		p.ChunkOverlap = 0
	}
	// 重叠不能超过块大小的一半，否则相邻块大部分重复，切了等于没切
	if p.ChunkOverlap > p.ChunkSize/2 {
		p.ChunkOverlap = p.ChunkSize / 2
	}
	if p.MinChunkSize < 0 {
		p.MinChunkSize = 0
	}
	if p.MinChunkSize > p.ChunkSize {
		p.MinChunkSize = p.ChunkSize
	}
	return p
}

// 块级标签闭合后应该换段。
//
// 只列真正负责分段语义的标签，不加 span/a/strong 这类行内标签——
// 按行内标签换段会把一句话切碎。
//
// td/th 虽然不算「块级」，但必须换行：表格里 `<td>enctype</td><td>file</td>`
// 直接拼起来会变成 "enctypefile"，实测真实文章里就出现了
// "enctypeapplication/x-www-form-urlencoded" 这种粘在一起的内容。
var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style|noscript)[^>]*>.*?</(script|style|noscript)\s*>`)
	reBr          = regexp.MustCompile(`(?i)<br\s*/?>`)
	reBlockEnd    = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr|td|th|blockquote|pre|section|article|ul|ol|table|thead|tbody|figure|figcaption|dd|dt)\s*>`)
	reAnyTag      = regexp.MustCompile(`(?s)<[^>]+>`)
)

// zeroWidth 富文本里常见的零宽字符。它们不显示但占字符数，
// 会让块首出现看不见的脏字符、也会让 CharCount 虚高。实测真实文章块首就有 \u200b。
var zeroWidth = strings.NewReplacer(
	"\u200b", "", // 零宽空格
	"\u200c", "", // 零宽不连字
	"\u200d", "", // 零宽连字
	"\u2060", "", // 词连接符
	"\ufeff", "", // BOM
)

// ToPlainText 把文章正文的 HTML 转成保留段落边界的纯文本。
//
// 不能复用 llm_service 里的 stripHTMLTags：它把 `\s+` 压成单个空格，
// 段落边界全丢，切出来的块毫无语义（整篇文章变成一行）。
//
// 步骤顺序有讲究：
//  1. 先整段删除 script/style（连同内容）。如果先剥标签，脚本里的代码会残留。
//  2. 再反转义 HTML 实体？不行——必须放在剥标签之后。
//     否则 `&lt;script&gt;` 会被反转义成真标签，紧接着被当成标签删掉。
func ToPlainText(raw string) string {
	if raw == "" {
		return ""
	}
	s := reScriptStyle.ReplaceAllString(raw, "")
	s = reBr.ReplaceAllString(s, "\n")
	s = reBlockEnd.ReplaceAllString(s, "\n\n")
	s = reAnyTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return normalizeSpace(s)
}

// normalizeSpace 归一化空白，但保留段落/换行的结构。
//
// 逐行处理：行内的连续空白（含全角空格、制表符）压成一个半角空格，
// 行首尾去空白；再统一 \r\n，最后把 3 个以上连续换行压成 2 个（一个空行）。
func normalizeSpace(s string) string {
	if s == "" {
		return ""
	}
	s = zeroWidth.Replace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// 不换行空格常常出现在富文本里，先统一成普通空格
	s = strings.ReplaceAll(s, "\u00a0", " ")

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		// Fields 按 unicode 空白切分，再用单空格拼回去
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	s = strings.Join(lines, "\n")

	// 连续 3 个以上换行压成 2 个
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}

// separators 切分优先级，从「语义最强」到「最弱」。
//
// 英文句号后面带空格，是为了不把 3.14、example.com 这类切开。
// 全角标点不需要，中文里句号后面通常没有空格。
//
// 局限：按 `。` 切完如果每块都已达标就直接返回，不会再用 `！` 切。
// 这个损失可以接受——两者都是句子边界，切在哪一侧都合理。
var separators = []string{
	"\n\n", // 段落
	"\n",   // 行
	"。",    // 中文句末
	"！",
	"？",
	"；",
	". ", // 英文句末（带空格）
	"! ",
	"? ",
	"; ",
	"，", // 中文逗号
	", ",
	" ", // 最后按空格
}

// splitRecursive 按分隔符优先级递归切分，直到每块都不超过 size。
func splitRecursive(text string, sepIndex, size int) []string {
	if runeLen(text) <= size {
		return []string{text}
	}
	if sepIndex >= len(separators) {
		// 所有分隔符都试过了（比如一长串没有空格的连续正文），只能按字数硬切
		return hardSplit(text, size)
	}

	sep := separators[sepIndex]
	parts := strings.Split(text, sep)
	if len(parts) == 1 {
		// 这个分隔符在文本里不存在，直接降级，省掉一轮无谓的循环
		return splitRecursive(text, sepIndex+1, size)
	}

	var chunks []string
	var buf strings.Builder
	bufLen := 0
	flush := func() {
		if bufLen > 0 {
			if c := strings.TrimSpace(buf.String()); c != "" {
				chunks = append(chunks, c)
			}
			buf.Reset()
			bufLen = 0
		}
	}

	sepLen := runeLen(sep)
	for _, part := range parts {
		partLen := runeLen(part)

		if partLen > size {
			// 这一段本身超长，用更细的分隔符继续切
			flush()
			chunks = append(chunks, splitRecursive(part, sepIndex+1, size)...)
			continue
		}
		if bufLen > 0 && bufLen+partLen > size {
			flush()
		}
		buf.WriteString(part)
		buf.WriteString(sep)
		bufLen += partLen + sepLen
	}
	flush()

	return chunks
}

// hardSplit 按字符数硬切。用 rune 切片而不是字节切片，
// 否则中文会被切成半个字符（JSON 里变成 U+FFFD 替换字符）。
func hardSplit(text string, size int) []string {
	r := []rune(text)
	var out []string
	for i := 0; i < len(r); i += size {
		end := i + size
		if end > len(r) {
			end = len(r)
		}
		out = append(out, string(r[i:end]))
	}
	return out
}

// mergeSmall 把过小的块并入相邻块。
//
// 碎块主要在两种情况下产生：标题独占一行（「一、背景」）、列表项很短。
// 几个字的块拿去 embedding 是纯浪费——向量里几乎不含信息，
// 却占掉一次 API 调用和一行存储。
//
// maxSize 是合并后的硬上限。不加这个限制的话，「大段正文 + 一串短列表项」
// 会让同一块被反复追加：实测真实文章里出现过 904 字的块（目标 800）。
// 超限时宁可留一个碎块，也不让块大小失控——块大小是模板的承诺，
// 悄悄涨 50% 比多一个碎块更难排查。
func mergeSmall(chunks []string, minSize, maxSize int) []string {
	if minSize <= 0 || len(chunks) < 2 {
		return chunks
	}

	out := make([]string, 0, len(chunks))
	for _, c := range chunks {
		if len(out) > 0 && runeLen(c) < minSize {
			prev := out[len(out)-1]
			// +2 是拼接用的 "\n\n"
			if runeLen(prev)+runeLen(c)+2 <= maxSize {
				out[len(out)-1] = prev + "\n\n" + c
				continue
			}
		}
		out = append(out, c)
	}

	// 首块太小的话前面没有块可并，只能并给后一块
	if len(out) > 1 && runeLen(out[0]) < minSize &&
		runeLen(out[0])+runeLen(out[1])+2 <= maxSize {
		out[1] = out[0] + "\n\n" + out[1]
		out = out[1:]
	}
	return out
}

// applyOverlap 给相邻块加上前一块的尾部内容。
//
// 目的是不让一个完整句子正好卡在边界上、两边都读不到上下文。
// 现在没有检索出口看不出效果，但检索质量强依赖这个重叠——
// 等做检索时再补就要全量重新向量化，代价高得多。
//
// 注意：重叠会让块长度超过 ChunkSize，这是刻意的（上限 = ChunkSize + Overlap）。
func applyOverlap(chunks []string, overlap int) []string {
	if overlap <= 0 || len(chunks) < 2 {
		return chunks
	}
	out := make([]string, len(chunks))
	out[0] = chunks[0]
	for i := 1; i < len(chunks); i++ {
		prev := []rune(chunks[i-1])
		take := overlap
		if take > len(prev) {
			take = len(prev)
		}
		out[i] = string(prev[len(prev)-take:]) + chunks[i]
	}
	return out
}

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}
