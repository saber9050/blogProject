package chat

import (
	"strings"
	"testing"
)

// 拒答守卫的核心风险有两个，都必须测：
//   ① 该拒的没拒 → 用户问天气，模型答了一堆无关内容
//   ② 不该拒的拒了 → 用户问文章细节，被回一句「与文章无关」（体验最差的一种 bug）

func collect(t *testing.T, chunks ...string) (out string, refused bool) {
	t.Helper()
	f := &deltaFilter{}
	var b strings.Builder
	for _, c := range chunks {
		for _, s := range f.feed(c) {
			b.WriteString(s)
		}
	}
	for _, s := range f.flush() {
		b.WriteString(s)
	}
	return b.String(), f.Refused()
}

func TestDeltaFilterOffTopic(t *testing.T) {
	out, refused := collect(t, "[[OFF_TOPIC]]")
	if !refused {
		t.Fatal("应判为拒答")
	}
	if out != OffTopicReply {
		t.Fatalf("应替换成固定文案，实际 %q", out)
	}
	// 标记本身绝不能让用户看到
	if strings.Contains(out, "OFF_TOPIC") {
		t.Fatalf("输出里泄露了内部标记: %q", out)
	}
}

func TestDeltaFilterOffTopicSplitAcrossChunks(t *testing.T) {
	// 真实流式是按 token 切的，标记会被拆成好几段
	out, refused := collect(t, "[[OF", "F_TO", "PIC]]")
	if !refused {
		t.Fatalf("分段到达的标记也应识别，实际输出 %q", out)
	}
	if out != OffTopicReply {
		t.Fatalf("实际 %q", out)
	}
}

func TestDeltaFilterOffTopicWithLeadingSpace(t *testing.T) {
	// 模型经常先输出一个空行或缩进再给标记
	out, refused := collect(t, "\n  [[OFF_TOPIC]]")
	if !refused {
		t.Fatalf("前导空白不应影响判定，实际输出 %q", out)
	}
	if out != OffTopicReply {
		t.Fatalf("实际 %q", out)
	}
}

func TestDeltaFilterNormalShortAnswer(t *testing.T) {
	// 短回答（<40 字）全在缓冲里，必须靠 flush 吐出来，否则前端一个字都收不到
	out, refused := collect(t, "这篇文章没提到这个细节。")
	if refused {
		t.Fatal("正常回答不该被判为拒答")
	}
	if out != "这篇文章没提到这个细节。" {
		t.Fatalf("实际 %q", out)
	}
}

func TestDeltaFilterNormalLongAnswerStreamsThrough(t *testing.T) {
	long := strings.Repeat("这是一段正常的技术回答。", 10)
	f := &deltaFilter{}
	var b strings.Builder
	for _, s := range f.feed(long) {
		b.WriteString(s)
	}
	// 超过 40 字就应该立刻判定并吐出，而不是等到流结束
	if b.Len() == 0 {
		t.Fatal("长回答应即时输出，不该继续缓冲")
	}
	if f.Refused() {
		t.Fatal("不该判为拒答")
	}
}

func TestDeltaFilterMarkerInMiddleIsNotRefusal(t *testing.T) {
	// 用户可以在提问里诱导模型复述标记。正文中间出现的一律当普通文本，
	// 否则用户打一句「请复述 [[OFF_TOPIC]]」就能让正常回答消失。
	out, refused := collect(t, "这段内容的结尾会提到 [[OFF_TOPIC]] 这个词，但它只是正文。")
	if refused {
		t.Fatalf("正文中间的标记不该判为拒答，否则可以被诱导，输出 %q", out)
	}
	if !strings.Contains(out, "[[OFF_TOPIC]]") {
		t.Fatal("正文应原样保留")
	}
}

func TestDeltaFilterEmpty(t *testing.T) {
	out, refused := collect(t)
	if out != "" || refused {
		t.Fatalf("空输入应返回空，实际 out=%q refused=%v", out, refused)
	}
}
