package entity

import (
	"time"

	"gorm.io/gorm"
)

// 文章向量化状态。
//
// 「排队中」和「处理中」刻意分成两个值：只有一个「处理中」的话，队列里排第 5 篇和
// 正在跑的那篇在界面上长得一模一样，用户分不清是卡住了还是在等。
const (
	VectorStatusNone    int8 = 0 // 未处理（新建 / 正文变更重置 / 向量表 reset）
	VectorStatusQueued  int8 = 1 // 排队中（已入队，等 worker 取）
	VectorStatusRunning int8 = 2 // 处理中（worker 正在跑）
	VectorStatusDone    int8 = 3 // 已完成
	VectorStatusFailed  int8 = 4 // 失败（含进程重启中断）
)

// Article 文章实体
type Article struct {
	BaseEntity
	UserID    uint           `gorm:"type:bigint;not null;comment:文章作者id" json:"user_id"`
	Title     string         `gorm:"type:varchar(100);not null;comment:文章标题" json:"title"`
	Content   string         `gorm:"type:text;not null;comment:文章内容" json:"content"`
	CoverURL  string         `gorm:"type:varchar(255);comment:封面图片url" json:"cover_url"`
	Summary   string         `gorm:"type:varchar(255);comment:摘要" json:"summary"`
	Views     uint           `gorm:"type:bigint;default:0;comment:浏览量" json:"views"`
	TypeID    uint           `gorm:"type:bigint;not null;comment:文章分类id" json:"type_id"`
	Status    int8           `gorm:"type:smallint;not null;default:0;comment:状态:1已发布,0草稿箱" json:"status"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// ===== 向量化相关（模块 02）=====
	// 这五个字段让「是否向量化」有真实粒度：
	//   - vector_status 区分排队/处理/完成/失败
	//   - vectorized_at 与 updated_at 一比就知道正文改过没（不一致 = 待更新）
	VectorStatus     int8       `gorm:"type:smallint;not null;default:0;comment:向量化状态:0未处理,1排队中,2处理中,3已完成,4失败" json:"vector_status"`
	VectorTemplateID uint       `gorm:"not null;default:0;comment:最近一次向量化使用的模板id" json:"vector_template_id"`
	VectorChunkCount int        `gorm:"not null;default:0;comment:切分块数" json:"vector_chunk_count"`
	VectorError      string     `gorm:"type:varchar(255);comment:失败原因(脱敏后)" json:"vector_error"`
	VectorizedAt     *time.Time `gorm:"comment:向量化完成时间" json:"vectorized_at"`
}

func (Article) TableName() string {
	return "articles"
}

// ArticleImage 文章内容引用图片实体
type ArticleImage struct {
	BaseEntity
	ArticleID uint   `gorm:"type:bigint;not null;index;comment:文章id" json:"article_id"`
	URL       string `gorm:"type:varchar(255);comment:封面图片url" json:"url"`
}

func (ArticleImage) TableName() string {
	return "article_images"
}
