package articlechunk

import (
	"blog/internal/model/entity"

	"gorm.io/gorm"
)

// Repository 文章切分块数据访问接口
//
// 写方法显式接受 tx：向量化的「删旧块 + 插新块 + 回写文章状态」必须在同一个事务里，
// 而事务对象由 service 层的 db.Transaction 提供。repository 自己持有 db 的话，
// 这些方法就跑不到那个事务里，中途失败会留下「旧块删了、新块没写」的空档。
type Repository interface {
	// DeleteByArticle 删除某文章的全部块
	DeleteByArticle(tx *gorm.DB, articleID uint) error
	// BatchCreate 批量插入块
	BatchCreate(tx *gorm.DB, chunks []*entity.ArticleChunk) error
	// CountByArticle 某文章的块数
	CountByArticle(articleID uint) (int64, error)
}
