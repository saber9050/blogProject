package chat

import (
	"reflect"
	"testing"
	"time"

	"blog/internal/model/entity"
)

// 这组测试用反射保证「实体有的字段，DTO 里同名字段也要被搬过去」。
//
// 为什么值得单独写：字段映射是纯体力活，漏一个**不会编译报错、不会 panic**，
// 只会让接口静默少返回一个字段。Steps 就这么漏过一次——
// 数据明明落库了，前端却显示「未检索资料」，排查绕了好几轮才定位到。
//
// 做法：给实体填满非零值，逐个字段在 DTO 里找同名项，断言值相等。
// 以后 DTO 新增字段（名字与实体一致）会自动被这条测试覆盖，忘了映射立刻红。

func TestToMessageItemCopiesAllFields(t *testing.T) {
	src := fullChatMessage()
	dst := toMessageItem(src)

	checkMappedFields(t, src, dst, map[string]bool{
		// CreatedAt 在 DTO 里是格式化后的 string，类型不同，单独比对
		"CreatedAt": true,
	})
	if dst.CreatedAt != src.CreatedAt.Format(timeLayout) {
		t.Fatalf("CreatedAt 映射错误: %q", dst.CreatedAt)
	}
}

func TestToAdminMessageItemCopiesAllFields(t *testing.T) {
	src := fullChatMessage()
	dst := toAdminMessageItem(src)

	checkMappedFields(t, src, dst, map[string]bool{
		"CreatedAt": true,
	})
	if dst.CreatedAt != src.CreatedAt.Format(timeLayout) {
		t.Fatalf("CreatedAt 映射错误: %q", dst.CreatedAt)
	}
}

// fullChatMessage 构造一个「每个字段都非零」的实体。
//
// 全非零是关键：零值字段映射漏了也看不出来，必须靠非零值才能区分
// 「映射正确」和「恰好是零值」。
// （Refused 与 NoRetrieval 同时为 true 在业务上不该出现，这里只为了让两个字段都非零。）
func fullChatMessage() *entity.ChatMessage {
	return &entity.ChatMessage{
		ID:        7,
		SessionID: 9,
		Role:      entity.ChatRoleAssistant,
		Content:   "回答",
		Model:     "deepseek-chat",
		Refs: []entity.ChatRefChunk{{
			ChunkID: 1, ArticleID: 2, SectionIndex: 3, HeadingPath: "a > b", RRFScore: 0.5,
		}},
		ArticleRefs: []entity.ChatArticleRef{{ArticleID: 4, Title: "引用文章"}},
		Steps: []entity.ChatStep{{
			Round: 1, Tool: searchToolName, Query: "表单",
			Hits: 3, Chars: 100, Ms: 20,
			Articles: []string{"文章A"}, Degraded: true, Note: "降级",
		}},
		PromptTokens:     10,
		CompletionTokens: 20,
		Error:            "err",
		Interrupted:      true,
		Refused:          true,
		NoRetrieval:      true,
		CreatedAt:        time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC),
	}
}

// checkMappedFields 逐个字段比对：实体有、DTO 也有同名导出字段时，值必须相等。
//
// DTO 里没有的字段（SessionID、PromptTokens 之类）直接跳过——那是设计上不对外，
// 不是漏映射。
func checkMappedFields(t *testing.T, src, dst any, skip map[string]bool) {
	t.Helper()
	sv := reflect.ValueOf(src).Elem()
	dv := reflect.ValueOf(dst).Elem()
	st := sv.Type()

	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if !f.IsExported() || skip[f.Name] {
			continue
		}
		df := dv.FieldByName(f.Name)
		if !df.IsValid() {
			continue
		}
		want := sv.Field(i).Interface()
		got := df.Interface()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("字段 %s 没有被映射：DTO=%#v，实体=%#v", f.Name, got, want)
		}
	}
}
