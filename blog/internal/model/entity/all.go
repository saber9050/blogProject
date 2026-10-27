package entity

// AllModels 返回需要 AutoMigrate 的全部实体，顺序即建表顺序。
//
// 应用启动、迁移工具、集成测试三处都从这里取，避免各写一份清单而漂移——
// 曾经 repostest 里就漏了 AIModelConfig，导致测试库和线上结构不一致。
//
// ⚠️ ArticleChunk 刻意不在这里。它的 embedding 列是 vector(n)，维度来自运行时配置
// （ai_model_configs.dimension），AutoMigrate 表达不了；更麻烦的是，一旦把 vector 列
// 交给 AutoMigrate，目标库没装 pgvector 扩展时整个应用都起不来（连登录都挂）。
// 它由 pkg/database/vector.go 的 EnsureSchema 手写 DDL 单独创建。
func AllModels() []interface{} {
	return []interface{}{
		// 用户相关
		&User{},

		// 文章相关
		&Article{},
		&Tag{},
		&TagArticle{},
		&Category{},
		&Comment{},
		&Like{},

		// 文章图片引用
		&ArticleImage{},

		// 关于页面
		&About{},

		// 大模型配置
		&AIModelConfig{},

		// 文章切分模板（article_chunks 见上，不在此列）
		&ChunkTemplate{},
	}
}

// AllTables 返回 AllModels 对应的表名，顺序与建表顺序一致。
func AllTables() []string {
	tables := make([]string, 0, len(AllModels()))
	for _, m := range AllModels() {
		tables = append(tables, tableNameOf(m))
	}
	return tables
}

// tableNameOf 取实体的表名。schema 解析失败只可能是代码写错，直接 panic 更早暴露。
func tableNameOf(m interface{}) string {
	namer, ok := m.(interface{ TableName() string })
	if !ok {
		panic("entity 缺少 TableName() 方法")
	}
	return namer.TableName()
}
