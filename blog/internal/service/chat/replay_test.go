package chat

import (
	"strings"
	"testing"

	"blog/internal/model/entity"
)

// TestHistoryToMessagesSkipsNoRetrieval 零检索的助手消息不能进历史。
//
// 为什么要专门守这条：它的内容是「没能读取到文章内容」的固定提示
// （修复前的版本里甚至是模型凭标题编的）。留在上下文里，模型可能把上一轮的
// 编造当成自己说过的事实继续推演——幻觉会一轮轮累积，而表现完全正常，
// 只有对着原文才能发现。
func TestHistoryToMessagesSkipsNoRetrieval(t *testing.T) {
	list := []*entity.ChatMessage{
		{ID: 1, Role: entity.ChatRoleUser, Content: "这篇文章讲了什么？"},
		{ID: 2, Role: entity.ChatRoleAssistant, Content: "编的内容", NoRetrieval: true},
		{ID: 3, Role: entity.ChatRoleUser, Content: "那第三点呢？"},
	}

	msgs := historyToMessages(list)

	if len(msgs) != 2 {
		t.Fatalf("应只剩 2 条（跳过零检索的助手消息），实际 %d 条", len(msgs))
	}
	for _, m := range msgs {
		if strings.Contains(m.Content, "编的内容") {
			t.Fatal("零检索的助手内容泄漏进了历史")
		}
	}
}

// TestHistoryToMessagesKeepsRefused 跑题消息要保留。
//
// 与零检索相反：拒答文案进历史是有用的，它能让模型知道「这个边界刚被踩过」，
// 避免用户换个说法再问同一件事时又跑题一次。
func TestHistoryToMessagesKeepsRefused(t *testing.T) {
	list := []*entity.ChatMessage{
		{ID: 1, Role: entity.ChatRoleUser, Content: "今天天气怎么样？"},
		{ID: 2, Role: entity.ChatRoleAssistant, Content: OffTopicReply, Refused: true},
		{ID: 3, Role: entity.ChatRoleUser, Content: "那这篇文章讲了什么？"},
	}

	if got := historyToMessages(list); len(got) != 3 {
		t.Fatalf("拒答消息应保留，实际只剩 %d 条", len(got))
	}
}

// TestBuildReplayText 回放文本的拼装。
func TestBuildReplayText(t *testing.T) {
	t.Run("空列表返回空串", func(t *testing.T) {
		if got := buildReplayText(nil, 3000); got != "" {
			t.Fatalf("期望空串，实际 %q", got)
		}
	})

	t.Run("检索失败的空 content 被跳过", func(t *testing.T) {
		list := []*entity.ChatToolMessage{
			{Round: 1, Content: ""},
			{Round: 2, Content: "第一段资料"},
		}
		got := buildReplayText(list, 3000)
		if !strings.Contains(got, "第一段资料") {
			t.Fatalf("应包含第二轮的资料，实际 %q", got)
		}
	})

	t.Run("必须带上可能过时的说明", func(t *testing.T) {
		// 少了这句，模型会把上一轮的资料当「当前文章内容」直接复述，
		// 而文章可能已经改过——这是静默的错答。
		got := buildReplayText([]*entity.ChatToolMessage{{Round: 1, Content: "x"}}, 3000)
		if !strings.Contains(got, "过时") {
			t.Fatalf("缺少「可能过时」的说明：%q", got)
		}
		if !strings.Contains(got, "重新检索") {
			t.Fatalf("缺少「重新检索」的要求：%q", got)
		}
	})

	t.Run("总字数不超过预算", func(t *testing.T) {
		// 每段 1000 字，预算 2500：只能装下前两段（第二段被截）
		var list []*entity.ChatToolMessage
		for i := 0; i < 5; i++ {
			list = append(list, &entity.ChatToolMessage{
				Round:   i + 1,
				Content: strings.Repeat("a", 1000),
			})
		}
		got := buildReplayText(list, 2500)
		// 掐掉说明头之后，正文不应超过 2500 字
		body := got[strings.Index(got, "\n\n")+2:]
		if n := len([]rune(body)); n > 2500 {
			t.Fatalf("正文 %d 字，超过预算 2500", n)
		}
	})
}
