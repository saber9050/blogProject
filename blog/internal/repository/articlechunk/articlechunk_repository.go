package articlechunk

import (
	"blog/internal/model/entity"

	"gorm.io/gorm"
)

const chunkTable = "article_chunks"

type articleChunkRepository struct {
	db *gorm.DB
}

// NewArticleChunkRepository 创建文章块仓储实例
func NewArticleChunkRepository(db *gorm.DB) Repository {
	return &articleChunkRepository{db: db}
}

func (r *articleChunkRepository) DeleteByArticle(tx *gorm.DB, articleID uint) error {
	// 用 Exec 而不是 tx.Delete：块表没有 DeletedAt，本该是硬删除，
	// 但走 ORM 的 Delete 要依赖「模型上没有软删除字段」这个隐含前提，
	// 万一以后有人给 ArticleChunk 加了 DeletedAt，这里会静默变成软删除——
	// 而 chunk 的语义是「整篇替换」，留软删除记录只会让数据越积越多。
	return tx.Exec("DELETE FROM "+chunkTable+" WHERE article_id = ?", articleID).Error
}

func (r *articleChunkRepository) BatchCreate(tx *gorm.DB, chunks []*entity.ArticleChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	// 批大小 100：每行含一个 vector(n) 列，太大容易撞 PG 的参数上限
	return tx.CreateInBatches(chunks, 100).Error
}

func (r *articleChunkRepository) CountByArticle(articleID uint) (int64, error) {
	var n int64
	if err := r.db.Table(chunkTable).Where("article_id = ?", articleID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
