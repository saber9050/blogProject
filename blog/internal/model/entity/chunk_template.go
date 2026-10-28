package entity

// 切分模板参数边界。放在实体包里是因为校验发生在 service 层，
// 但上下限属于「这个字段本身是什么」，改的时候两边要一起看。
const (
	MinChunkSizeLimit     = 100  // 块大小下限（字符）
	MaxChunkSizeLimit     = 4000 // 块大小上限（字符）
	MinEmbeddingDimension = 128  // 向量维度下限
	MaxEmbeddingDimension = 4096 // 向量维度上限
)

// ChunkTemplate 文章切分模板
//
// 模板的差异只体现在三个数值参数上，不做「策略枚举」（段落/句子/定长）：
// 递归分隔符切分本身就会按 段落→换行→句号→硬切 的顺序降级，
// 再叠一层策略选择只会让实现和测试都翻倍，而收益看不出来。
//
// ⚠️ 只有 chunk_size 保留 gorm 的 default tag，其余四个字段都不加。
//
// GORM 在 Create 时会把「零值」替换成 default 指定的值，于是：
//   - status=0（禁用）  → 被静默改写成 1（启用）
//   - strip_html=false  → 被静默改写成 true
//   - chunk_overlap=0   → 被静默改写成 80
//   - min_chunk_size=0  → 被静默改写成 100
//
// 这四个的零值都是合法且有意义的配置，静默改写等于让用户配不出想要的效果，
// 而且现象极难查（界面显示已保存，库里存的是别的值）。默认值一律由 service
// 层显式赋值。AIModelConfig.Status 当年踩的是同一个坑。
type ChunkTemplate struct {
	BaseEntity
	Name         string `gorm:"type:varchar(64);not null;unique;comment:模板名称" json:"name"`
	Description  string `gorm:"type:varchar(255);comment:用途说明" json:"description"`
	ChunkSize    int    `gorm:"not null;default:800;comment:目标块大小(字符)" json:"chunk_size"`
	ChunkOverlap int    `gorm:"not null;comment:相邻块重叠(字符)" json:"chunk_overlap"`
	MinChunkSize int    `gorm:"not null;comment:过小碎块合并阈值(字符)" json:"min_chunk_size"`
	StripHTML    bool   `gorm:"not null;comment:是否剥离HTML标签" json:"strip_html"`
	Status       int8   `gorm:"type:smallint;not null;comment:状态:1启用,0禁用" json:"status"`
}

func (ChunkTemplate) TableName() string {
	return "chunk_templates"
}
