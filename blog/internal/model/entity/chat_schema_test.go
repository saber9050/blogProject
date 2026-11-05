package entity

import (
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// 这组测试不连数据库，只解析 GORM 的 schema 定义。
//
// 存在的意义：模块 05 的表结构有两条「一旦写错就不会报错、但线上行为全错」的约定——
//   ① 不能出现 deleted_at（有的话 db.Delete 静默变软删，而查询都不带 deleted_at 条件）
//   ② 两个 jsonb 切片字段必须真的被当成普通列，而不是被 GORM 误判成 has-many 关联
// （关联会让 AutoMigrate 去建外键、让 Create 顺带写子表，问题会以「插一条消息却报
// 找不到关联表」这种面目出现）。这两条都只能在 schema 层验证，测一次省很多排查时间。

func parseEntity(t *testing.T, m interface{}) *schema.Schema {
	t.Helper()
	s, err := schema.Parse(m, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("解析 %T 的 schema 失败: %v", m, err)
	}
	return s
}

// indexFields 把索引的字段按 priority 排好，返回 "列名" 或 "列名:DESC" 的序列。
// ParseIndexes 内部已按 Priority 排序，这里只是转成人眼可比的形式。
func indexFields(idx *schema.Index) []string {
	out := make([]string, 0, len(idx.Fields))
	for _, f := range idx.Fields {
		name := f.Field.DBName
		if f.Sort != "" {
			// GORM 解析出来的 Sort 是小写（desc），DDL 里是大写 DESC；统一成大写便于断言
			name += ":" + strings.ToUpper(f.Sort)
		}
		out = append(out, name)
	}
	return out
}

// lookupIndex 按名字取索引，没有就一起把「实际有哪些」打出来，便于对照。
func lookupIndex(t *testing.T, s *schema.Schema, name string) *schema.Index {
	t.Helper()
	idx := s.LookIndex(name)
	if idx == nil {
		names := make([]string, 0)
		for _, i := range s.ParseIndexes() {
			names = append(names, i.Name)
		}
		t.Fatalf("缺少索引 %s，实际有 %v", name, names)
	}
	return idx
}

func assertFields(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("索引字段数量不符：got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("索引字段顺序不符：got %v, want %v", got, want)
		}
	}
}

func TestChatSessionSchema(t *testing.T) {
	s := parseEntity(t, &ChatSession{})

	if s.Table != "chat_sessions" {
		t.Fatalf("表名不符：%s", s.Table)
	}

	// ① 绝不能有 deleted_at
	if f := s.LookUpField("DeletedAt"); f != nil {
		t.Fatal("chat_sessions 不应有 DeletedAt 字段：删除是硬删，加了会让 db.Delete 静默变软删")
	}

	// ② user_id 必须 NOT NULL（游客已不支持，为空就说明写数据时漏了归属）
	if f := s.LookUpField("UserID"); f == nil || !f.NotNull {
		t.Fatal("chat_sessions.user_id 必须是 NOT NULL")
	}

	// ③ 两个索引的字段与顺序（顺序错了索引就白建：前缀不匹配等于全表扫）
	assertFields(t, indexFields(lookupIndex(t, s, "idx_chat_sessions_user_article")),
		[]string{"user_id", "article_id", "last_message_at:DESC"})
	assertFields(t, indexFields(lookupIndex(t, s, "idx_chat_sessions_article")),
		[]string{"article_id", "last_message_at:DESC"})
}

func TestChatMessageSchema(t *testing.T) {
	s := parseEntity(t, &ChatMessage{})

	if s.Table != "chat_messages" {
		t.Fatalf("表名不符：%s", s.Table)
	}
	if f := s.LookUpField("DeletedAt"); f != nil {
		t.Fatal("chat_messages 不应有 DeletedAt 字段")
	}
	// 消息不可编辑，不该有 updated_at
	if f := s.LookUpField("UpdatedAt"); f != nil {
		t.Fatal("chat_messages 不应有 UpdatedAt 字段：消息没有「被修改」的语义")
	}

	// jsonb 字段不能被解析成关联关系
	for _, name := range []string{"Refs", "ArticleRefs"} {
		if _, ok := s.Relationships.Relations[name]; ok {
			t.Fatalf("%s 被 GORM 误判成了关联关系，必须带 serializer:json 才会按普通列处理", name)
		}
		f := s.LookUpField(name)
		if f == nil {
			t.Fatalf("%s 字段不存在", name)
		}
		if f.DataType == "" || string(f.DataType) != "jsonb" {
			t.Fatalf("%s 的列类型应为 jsonb，实际 %q", name, f.DataType)
		}
		if f.Serializer == nil {
			t.Fatalf("%s 缺少 serializer:json，切片字段会被当关联处理", name)
		}
	}

	// (session_id, id)：按会话倒序分页靠它
	assertFields(t, indexFields(lookupIndex(t, s, "idx_chat_messages_session")),
		[]string{"session_id", "id"})
}

func TestChatEntitiesInAllModels(t *testing.T) {
	tables := make(map[string]bool)
	for _, t2 := range AllTables() {
		tables[t2] = true
	}
	for _, want := range []string{"chat_sessions", "chat_messages"} {
		if !tables[want] {
			t.Fatalf("%s 不在 AllModels() 里，AutoMigrate 不会建这张表", want)
		}
	}
}
