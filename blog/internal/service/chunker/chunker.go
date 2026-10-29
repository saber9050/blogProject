// Package chunker 把文章正文切成适合向量化的块。
//
// 这里刻意做成纯函数：不碰数据库、不发 HTTP、不读配置。
// 切分是整个向量化链路里唯一「参数敏感、结果难验证」的一步，
// 只有它是纯函数，才能用单测把边界情况全部钉死。
package chunker

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
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
	ChunkSize    int // 目标块大小（字符/rune）
	ChunkOverlap int // 相邻块重叠字符数
	MinChunkSize int // 小于该值的块并入相邻块
	// StripMarkup 是否先把 Markdown 标记去掉、只留纯文本。
	// 标记（# ** | ``` 等）对 embedding 是噪声，生产环境恒为 true。
	StripMarkup bool
}

// Result 一个切分块
type Result struct {
	Content   string
	CharCount int // rune 数，不是字节数

	// HeadingPath 该块所属的标题路径，如 "redis发布订阅和Stream > 二，Stream"。
	// 会拼在 Content 最前面参与 embedding —— 中文技术博客的查询词往往就是小节标题，
	// 让标题进向量是最便宜的召回提升。同时单独返回一份供 UI 展示与按章节过滤。
	HeadingPath string
	// SectionIndex 小节序号（0 起）。同一小节切出的多块共享这个值，
	// 将来做「小块检索、整节喂模型」时靠它把块拼回整节。
	SectionIndex int
}

// DefaultParams 生产环境使用的切分参数。
//
// 不再由用户按模板配置：按字符数切分本身只有三个旋钮，调来调去差异很小，
// 做成可配置项的结果是「多了一个要维护的管理模块，块质量并没有变好」。
// 参数固化在代码里，要调整就改这里的常量并重新发布——这也让「某篇文章是
// 用哪套参数切的」不再是个需要记录的变量。
func DefaultParams() Params {
	return Params{
		ChunkSize:    DefaultChunkSize,
		ChunkOverlap: DefaultChunkOverlap,
		MinChunkSize: DefaultMinChunkSize,
		StripMarkup:  true,
	}
}

// Split 按 Markdown 结构切成块。
//
// 流程：解析标题 → 逐小节切分（超长才递归降级）→ 小节内合并碎块 → 加重叠 → 拼标题路径。
//
// 为什么不直接对全文递归切：小节边界是作者亲手划的主题边界，比任何字符分隔符
// 都准。实测旧方案在 14 篇真实文章上，80 块里有 29 块跨了小节。
//
// docTitle 用于拼标题路径（参与 embedding），传空则路径只含小节标题。
func Split(content, docTitle string, p Params) ([]Result, error) {
	p = p.withDefaults()

	// 整体校验用清洗后的全文：小节可能都合法但合计太短
	full := ExtractText(content, p.StripMarkup)
	if full == "" {
		return nil, ErrEmptyContent
	}
	if n := runeLen(full); n < MinContentChars {
		return nil, fmt.Errorf("%w（当前 %d 字，至少 %d 字）", ErrContentTooShort, n, MinContentChars)
	}

	sections := parseSections(content)
	out := make([]Result, 0, len(sections))
	for _, sec := range sections {
		path := buildHeadingPath(docTitle, sec.path)

		pieces := splitSectionBody(sec.body, p)
		if len(pieces) == 0 {
			// 只有标题没有正文的小节跳过：标题信息已经进了后续块的路径前缀，
			// 单独成块只会多出一堆几字长的碎块
			continue
		}
		pieces = mergeSmall(pieces, p.MinChunkSize, p.ChunkSize+p.MinChunkSize)
		pieces = applyOverlap(pieces, p.ChunkOverlap)

		for _, c := range pieces {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			full := c
			if path != "" {
				full = path + "\n\n" + c
			}
			out = append(out, Result{
				Content:      full,
				CharCount:    runeLen(full),
				HeadingPath:  path,
				SectionIndex: sec.index,
			})
		}
	}

	if len(out) == 0 {
		return nil, ErrEmptyContent
	}
	return out, nil
}

// section 一个 Markdown 小节：标题 + 标题下的正文
type section struct {
	index int
	level int // 0 表示「第一个标题之前的引子」
	title string
	path  []string // 祖先标题链（不含自己），最多保留两级
	body  string   // 原始 Markdown，尚未清洗
}

// reATXHeading 匹配 ATX 风格标题：`## 标题`（允许前面不超过 3 个空格）
var reATXHeading = regexp.MustCompile(`^[ \t]{0,3}(#{1,6})[ \t]+(.*)$`)

// headItem 标题栈里的一项
type headItem struct {
	level int
	title string
}

// parseSections 按标题把正文切成小节。
//
// 代码块里的 `#` 不算标题（`inFence` 判断）；`#` 一级标题当作文章标题的重复，
// 不进路径栈，否则路径里会出现 "标题 > 标题"。
func parseSections(md string) []section {
	var out []section
	var stack []headItem // 当前标题链
	cur := section{index: 0}
	inFence := false

	flush := func() {
		out = append(out, cur)
		cur = section{index: len(out), level: cur.level, path: cur.path}
	}

	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			cur.body += line + "\n"
			continue
		}
		if !inFence {
			if m := reATXHeading.FindStringSubmatch(line); m != nil {
				lv := len(m[1])
				title := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(m[2]), "#"))
				// 所有级别都进栈。一级标题（`#`）也当小节标题——
				// 不少文章就是用它分大节的（实测 #30 是 `# 一，发布订阅`），
				// 跳过它会让这些块丢掉小节信息。
				//
				// 按 level 弹栈而不是按栈深度：`## A` 后面紧跟 `## B` 时，
				// 两个都是第二级，B 要顶掉 A；用深度判断会因为 A 前面没有 `#`
				// 而只占了一层，导致 B 叠在 A 后面变成 "A > B"。
				for len(stack) > 0 && stack[len(stack)-1].level >= lv {
					stack = stack[:len(stack)-1]
				}
				stack = append(stack, headItem{level: lv, title: title})
				flush()
				cur.level = lv
				cur.title = title
				// path 含当前标题：buildHeadingPath 取最后两级，少了自己就只剩祖先
				titles := make([]string, len(stack))
				for i, it := range stack {
					titles[i] = it.title
				}
				cur.path = titles
				continue
			}
		}
		cur.body += line + "\n"
	}
	flush()
	return out
}

// buildHeadingPath 拼标题路径：文章标题 + 当前小节及其上一级。
//
// 只取两级小节是刻意的：路径会拼进块正文参与 embedding，太长会挤占 800 字的预算。
// 取「当前 + 上一级」而不是「最顶层两级」，保证当前标题一定在路径里——
// 三级标题下的小节若只取顶层，当前标题就丢了。
func buildHeadingPath(docTitle string, path []string) string {
	parts := make([]string, 0, 3)
	if t := strings.TrimSpace(docTitle); t != "" {
		parts = append(parts, t)
	}
	start := 0
	if len(path) > 2 {
		start = len(path) - 2
	}
	for _, t := range path[start:] {
		if t == "" {
			continue
		}
		// 文章正文第一行常是 `# 标题`，和 docTitle 重复，跳过
		if len(parts) == 1 && t == parts[0] {
			continue
		}
		parts = append(parts, t)
	}
	return strings.Join(parts, " > ")
}

// splitSectionBody 切一个小节的正文。
//
// 与全文切分的区别：表格和代码块是原子单位，永不切开——半张表格喂给 embedding
// 是纯噪声。原子段塞得进当前块就塞，塞不下就单独成块（允许超长）。
func splitSectionBody(body string, p Params) []string {
	paras := splitParagraphs(body)

	var chunks []string
	var buf []string
	bufLen := 0

	flush := func() {
		if len(buf) > 0 {
			c := strings.TrimSpace(strings.Join(buf, "\n\n"))
			if c != "" {
				chunks = append(chunks, c)
			}
			buf = nil
			bufLen = 0
		}
	}
	push := func(text string, n int) {
		if bufLen > 0 && bufLen+n > p.ChunkSize {
			flush()
		}
		buf = append(buf, text)
		bufLen += n + 2
	}

	for _, para := range paras {
		// 按参数决定要不要去标记：StripMarkup=false 时保留原始 Markdown
		// （调试、以及想在块里看结构的场景要用）
		var text string
		if p.StripMarkup {
			text = MarkdownToPlainText(para)
		} else {
			text = normalizeSpace(para)
		}
		if text == "" {
			continue
		}
		n := runeLen(text)

		if isAtomicBlock(para) {
			// 原子段：塞不下就单独成块，绝不切断
			if bufLen > 0 && bufLen+n <= p.ChunkSize {
				buf = append(buf, text)
				bufLen += n + 2
				continue
			}
			flush()
			chunks = append(chunks, text)
			continue
		}

		if n <= p.ChunkSize {
			push(text, n)
			continue
		}

		// 超长普通段：降级到字符分隔符递归切
		flush()
		for _, sub := range splitRecursive(text, 0, p.ChunkSize) {
			if c := strings.TrimSpace(sub); c != "" {
				chunks = append(chunks, c)
			}
		}
	}
	flush()
	return chunks
}

// splitParagraphs 按空行切段，但代码块内部的空行不算段落边界。
func splitParagraphs(body string) []string {
	var out []string
	var cur []string
	inFence := false

	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
		}
		if !inFence && t == "" {
			if len(cur) > 0 {
				out = append(out, strings.Join(cur, "\n"))
				cur = nil
			}
			continue
		}
		cur = append(cur, line)
	}
	if len(cur) > 0 {
		out = append(out, strings.Join(cur, "\n"))
	}
	return out
}

// isAtomicBlock 判断一段是否不可切分：代码块，或至少两行以 `|` 开头的表格。
func isAtomicBlock(para string) bool {
	t := strings.TrimSpace(para)
	if strings.HasPrefix(t, "```") {
		return true
	}
	lines := strings.Split(t, "\n")
	if len(lines) < 2 {
		return false
	}
	rows := 0
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "|") {
			rows++
		}
	}
	return rows >= 2
}

// ExtractText 按参数把正文转成纯文本。单独导出是因为 service 层在报错时
// 需要知道正文长度（判断「空」还是「太短」），不该为此把切分再跑一遍。
//
// 2026-10-29 起正文统一存 Markdown，所以只有一种清洗（不再判断 HTML）。
func ExtractText(content string, stripMarkup bool) string {
	if stripMarkup {
		return MarkdownToPlainText(content)
	}
	// 不去标记时也要清洗零宽字符和多余空白，否则块首块尾会带看不见的脏字符
	return normalizeSpace(content)
}

// Markdown 标记。顺序敏感：
//   - 代码块必须先抽出来占位，否则块内的 `#` `*` `|` 会被当成标记删掉
//   - 图片/链接要在强调之前处理，否则 `[**粗体**](url)` 里的括号会先被破坏
var (
	reMdCodeBlock = regexp.MustCompile("(?s)```[^\n]*\n(.*?)```")
	reMdImage     = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	reMdLink      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	reMdHeading   = regexp.MustCompile(`(?m)^[ \t]{0,3}#{1,6}[ \t]+`)
	reMdQuote     = regexp.MustCompile(`(?m)^[ \t]{0,3}>[ \t]?`)
	reMdHR        = regexp.MustCompile(`(?m)^[ \t]*([-*_][ \t]*){3,}$`)
	// 字符集只能用 [ \t] 不能用 \s：\s 含 \n，会让这个正则跨行吃掉下一行的开头
	// 末尾带 \n?：分隔行整行删掉，否则留下一个空行把表格拦腰截断
	reMdTableSep   = regexp.MustCompile(`(?m)^[ \t]*\|[ \t:|-]*\|[ \t]*\n?`)
	reMdBullet     = regexp.MustCompile(`(?m)^[ \t]*[-*+][ \t]+`)
	reMdOrdered    = regexp.MustCompile(`(?m)^[ \t]*\d+\.[ \t]+`)
	reMdStrike     = regexp.MustCompile(`~~([^~]+)~~`)
	reMdBold       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reMdItalic     = regexp.MustCompile(`\*([^*]+)\*`)
	reMdInlineCode = regexp.MustCompile("`([^`]*)`")
)

// MarkdownToPlainText 把 Markdown 正文转成保留段落结构的纯文本。
//
// 处理原则：**去掉标记，保留内容**。标题的 `#` 要去掉但标题文字要留；
// 链接的 `[text](url)` 留 text 丢掉 url（URL 对 embedding 没有语义价值）；
// 图片整段丢掉（`![x](url)` 里的 alt 通常为空或与正文无关）。
//
// 代码块里的内容**保留**：技术文章里代码本身就是正文的一部分，
// 而且里面的标识符对向量化有区分度。先抽出来占位是为了防止块内的
// `#` `*` `|` 被下面的正则误删。
func MarkdownToPlainText(md string) string {
	if md == "" {
		return ""
	}

	var codes []string
	s := reMdCodeBlock.ReplaceAllStringFunc(md, func(m string) string {
		body := reMdCodeBlock.FindStringSubmatch(m)
		if len(body) > 1 {
			codes = append(codes, strings.TrimRight(body[1], "\n"))
		} else {
			codes = append(codes, "")
		}
		return fmt.Sprintf("\x00%d\x00", len(codes)-1)
	})

	s = reMdImage.ReplaceAllString(s, "")
	s = reMdLink.ReplaceAllString(s, "$1")
	s = reMdHeading.ReplaceAllString(s, "")
	s = reMdQuote.ReplaceAllString(s, "")
	s = reMdHR.ReplaceAllString(s, "")
	s = reMdTableSep.ReplaceAllString(s, "")
	s = reMdBullet.ReplaceAllString(s, "")
	s = reMdOrdered.ReplaceAllString(s, "")
	s = reMdStrike.ReplaceAllString(s, "$1")
	s = reMdBold.ReplaceAllString(s, "$1")
	s = reMdItalic.ReplaceAllString(s, "$1")
	s = reMdInlineCode.ReplaceAllString(s, "$1")

	for i, c := range codes {
		s = strings.ReplaceAll(s, fmt.Sprintf("\x00%d\x00", i), c)
	}
	return normalizeSpace(s)
}

// withDefaults 兜住非法参数。
//
// 不做报错而是静默修正：参数现在固化在 DefaultParams 里，走到这里的非法值
// 只可能来自调用方手写的 Params（测试、临时脚本）。让它跑出结果比让整个
// 向量化任务失败更有用——失败原因里出现「chunk_size 不能为 0」对排查毫无帮助。
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

// zeroWidth 富文本里常见的零宽字符。它们不显示但占字符数，
// 会让块首出现看不见的脏字符、也会让 CharCount 虚高。实测真实文章块首就有 \u200b。
var zeroWidth = strings.NewReplacer(
	"\u200b", "", // 零宽空格
	"\u200c", "", // 零宽不连字
	"\u200d", "", // 零宽连字
	"\u2060", "", // 词连接符
	"\ufeff", "", // BOM
)

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
// 重叠与本块之间补一个空行：原来直接字符串拼接，上一块的尾巴和本块开头会粘成
// 一行（实测出现过「1.消息队列的相关命令| 命令 | 基本语法格式 |」这种），
// 把 markdown 表格的前半行和上一段的结尾黏在一起，喂给 embedding 就是噪声。
//
// 注意：重叠会让块长度超过 ChunkSize，这是刻意的（上限 = ChunkSize + Overlap + 2）。
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
		out[i] = string(prev[len(prev)-take:]) + "\n\n" + chunks[i]
	}
	return out
}

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}
