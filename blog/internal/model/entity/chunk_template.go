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
type ChunkTemplate struct {
	BaseEntity
	Name         string `gorm:"type:varchar(64);not null;unique;comment:模板名称" json:"name"`
	Description  string `gorm:"type:varchar(255);comment:用途说明" json:"description"`
	ChunkSize    int    `gorm:"not null;default:800;comment:目标块大小(字符)" json:"chunk_size"`
	ChunkOverlap int    `gorm:"not null;default:80;comment:相邻块重叠(字符)" json:"chunk_overlap"`
	MinChunkSize int    `gorm:"not null;default:100;comment:过小碎块合并阈值(字符)" json:"min_chunk_size"`
	StripHTML    bool   `gorm:"not null;default:true;comment:是否剥离HTML标签" json:"strip_html"`
	Status       int8   `gorm:"type:smallint;not null;default:1;comment:状态:1启用,0禁用" json:"status"`
}

func (ChunkTemplate) TableName() string {
	return "chunk_templates"
}
