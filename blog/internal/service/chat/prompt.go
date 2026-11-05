package chat

import (
	"strings"

	"blog/internal/model/entity"

	"github.com/cloudwego/eino/schema"
)

// tutorInstruction agent 的系统提示词。
//
// 三条硬约束，每条都对应一类真实失败：
//  1. 「资料里没有就说没有」——不写它，模型会用通用知识编一段，用户以为是文章里写的
//  2. 「不要照抄用户原话当检索词」——原句里全是「为什么」「怎么」，关键词路会被污染
//  3. 「无关问题不调工具」——不写它，模型会对着「今天天气」检索三轮，白烧 token
//
// ⚠️ 这里**绝不能**出现任何文章标题、id、用户名。Instruction 是 agent 级共享配置，
// agent 按模型缓存、跨用户复用，拼进请求级数据就是串号（见 scope.go 的注释）。
// 当前文章与引用文章通过首条 user 消息的前缀传给模型（buildMessages）。
const tutorInstruction = `你是「博客文章答疑」助手，只解答与用户正在阅读的文章相关的问题。

规则：
1. 需要文章内容时，调用 search_article_context 检索：
   - query 用关键名词短语，不要整句照抄用户原话（去掉「为什么」「怎么」这类疑问词）；
   - 一次没查到时，换同义词或更具体的术语再查。
2. 拿到检索结果后判断资料是否足够：
   - 足够：基于资料回答，并说明内容出自哪篇文章；
   - 资料里没有：明确说「这篇文章没有提到」，不要编造；
   - 明显不足：换词再检索，仍不足则如实说明。
3. 如果问题与文章主题无关（闲聊、天气、让写代码、与主题无关的提问），
   直接用一句话说明你只回答与文章内容相关的问题，并且**不要调用检索工具**。
   区分清楚两种情况：「文章里没提到」是正常回答，「问题和文章无关」才是拒绝。
4. 回答用简体中文、Markdown 格式；代码块标注语言；避免冗长的客套。`

// buildMessages 组装发给 agent 的消息数组。
//
// 为什么是 Run(msgs) 而不是 Query(string)：Query 只接受一个字符串，
// 历史一条都传不进去，多轮对话直接变成「每句都是第一句」——而且不会报错，
// 只有真正连着追问几次才会发现模型完全不记得上文。
func buildMessages(history []*schema.Message, question, articleTitle string, refTitles []string) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(history)+1)
	msgs = append(msgs, history...)
	msgs = append(msgs, &schema.Message{
		Role:    schema.User,
		Content: userPrefix(articleTitle, refTitles) + question,
	})
	return msgs
}

// userPrefix 首条消息的范围前缀，告诉模型「在什么范围内」作答。
//
// 这是请求级数据的唯一合法载体之一（另一个是 ctx 里的 scope，给 tool 用）。
// 放在消息里而不是 Instruction 里，是因为每个请求都不一样，而 Instruction 是共享的。
func userPrefix(articleTitle string, refTitles []string) string {
	var b strings.Builder
	b.WriteString("[当前文章：《")
	b.WriteString(articleTitle)
	b.WriteString("》")
	if len(refTitles) > 0 {
		b.WriteString("；本轮引用：")
		for i, t := range refTitles {
			if i > 0 {
				b.WriteString("、")
			}
			b.WriteString("《")
			b.WriteString(t)
			b.WriteString("》")
		}
	}
	b.WriteString("]\n\n")
	return b.String()
}

// historyToMessages 把库里的消息转成 agent 输入格式。
//
// 两类消息会被跳过：
//   - error 非空且 content 为空的（失败消息，没有内容，进上下文只会干扰）
//   - role 不是 user/assistant 的（脏数据兜底，正常不会有）
//
// 被用户中断的消息保留已生成的部分：用户很可能接着那半句继续问。
func historyToMessages(list []*entity.ChatMessage) []*schema.Message {
	out := make([]*schema.Message, 0, len(list))
	for _, m := range list {
		if m.Content == "" {
			continue
		}
		var role schema.RoleType
		switch m.Role {
		case entity.ChatRoleUser:
			role = schema.User
		case entity.ChatRoleAssistant:
			role = schema.Assistant
		default:
			continue
		}
		out = append(out, &schema.Message{Role: role, Content: m.Content})
	}
	return out
}
