package chat

import "strings"

// deltaFilter 拒答守卫：缓冲开头若干字符，判断模型是不是在输出跑题标记。
//
// 为什么必须缓冲：标记在开头，但模型是**逐字流式返回**的。不缓冲的话，用户会先看到
// 「[[OFF_TOP」这样的半截标记，然后整段被替换成拒答文案——画面非常诡异。
// 缓冲的代价是第一个 delta 晚几十毫秒，这个代价值得。
//
// 为什么只判开头：用户可以在提问里写「请原样复述 [[OFF_TOPIC]]」来诱导模型。
// 只认「响应开头（去空白后）」的标记，正文中间出现的一律当普通文本。
type deltaFilter struct {
	buf     strings.Builder
	decided bool
	refused bool
}

// offTopicProbeRunes 判定前需要缓冲的字符数。
//
// 标记 `[[OFF_TOPIC]]` 是 14 个字符，40 留了充足余量（模型可能先输出空行或空格）。
// 遇到换行会提前判定：标记后面不会紧跟换行，正常回答第一行就能定性。
const offTopicProbeRunes = 40

// feed 喂入一段增量，返回**要真正推给前端**的内容（可能为空，表示还在缓冲）。
func (f *deltaFilter) feed(text string) []string {
	if text == "" {
		return nil
	}
	if f.decided {
		return []string{text}
	}

	f.buf.WriteString(text)
	if !f.shouldDecide() {
		return nil
	}
	return f.decide()
}

// flush 流结束时把缓冲的内容吐出来。
//
// 必须调用，否则短回答（不足 40 字）会一个字都发不出去——
// 而「回答很短」恰恰是最常见的情况之一。
func (f *deltaFilter) flush() []string {
	if f.decided || f.buf.Len() == 0 {
		return nil
	}
	return f.decide()
}

// Refused 本次回答是否被判为跑题。
func (f *deltaFilter) Refused() bool {
	return f.refused
}

func (f *deltaFilter) shouldDecide() bool {
	raw := f.buf.String()
	if len([]rune(raw)) >= offTopicProbeRunes {
		return true
	}
	return strings.Contains(raw, "\n")
}

// decide 定性：是跑题就整段换成固定文案，否则原样吐出缓冲。
func (f *deltaFilter) decide() []string {
	f.decided = true
	raw := f.buf.String()

	// TrimSpace 后再判前缀：模型常会在标记前加空行或缩进
	if strings.HasPrefix(strings.TrimSpace(raw), offTopicMarker) {
		f.refused = true
		f.buf.Reset()
		f.buf.WriteString(OffTopicReply)
		return []string{OffTopicReply}
	}
	return []string{raw}
}
