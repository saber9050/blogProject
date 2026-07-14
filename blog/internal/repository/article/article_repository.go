package article

import (
	"blog/internal/model/entity"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"gorm.io/gorm"
)

// articleRepository 文章数据访问实现
type articleRepository struct {
	db *gorm.DB
}

// NewArticleRepository 创建文章仓储实例
func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db: db}
}

// FindByID 通过 ID 查找文章
func (r *articleRepository) FindByID(id uint) (*entity.Article, error) {
	var article entity.Article
	err := r.db.First(&article, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &article, nil
}

// ListPublic 前台获取已发布的文章列表
func (r *articleRepository) ListPublic(page, pageSize int, sort string, categoryID uint, tagIDs []uint, keyword string) ([]*entity.Article, int64, error) {
	var list []*entity.Article
	var total int64

	query := r.db.Model(&entity.Article{}).Where("status = ?", 1)

	if categoryID > 0 {
		query = query.Where("type_id = ?", categoryID)
	}
	if keyword != "" {
		query = query.Where("title LIKE ? OR summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 如果有标签筛选，需要子查询
	if len(tagIDs) > 0 {
		tagStr := strings.Trim(strings.Replace(fmt.Sprint(tagIDs), " ", ",", -1), "[]")
		subQuery := r.db.Table("tag_articles").
			Select("article_id").
			Where("tag_id IN (" + tagStr + ")").
			Group("article_id")
		query = query.Where("id IN (?)", subQuery)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	switch sort {
	case "popular":
		// 热度排序：点赞数×3 + 浏览量×1 + 评论数×2
		query = query.Order(`
			(COALESCE((SELECT COUNT(*) FROM likes WHERE likes.article_id = articles.id), 0) * 3 +
			views * 1 +
			COALESCE((SELECT COUNT(*) FROM comments WHERE comments.article_id = articles.id AND comments.deleted_at IS NULL), 0) * 2) DESC`)
	default:
		query = query.Order("created_at DESC")
	}

	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListAdmin 后台获取文章列表
func (r *articleRepository) ListAdmin(page, pageSize int, status *int, categoryID uint, tagIDs []uint, keyword string) ([]*entity.Article, int64, error) {
	var list []*entity.Article
	var total int64

	query := r.db.Model(&entity.Article{})

	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if categoryID > 0 {
		query = query.Where("type_id = ?", categoryID)
	}
	if keyword != "" {
		query = query.Where("title LIKE ? OR summary LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if len(tagIDs) > 0 {
		tagStr := strings.Trim(strings.Replace(fmt.Sprint(tagIDs), " ", ",", -1), "[]")
		subQuery := r.db.Table("tag_articles").
			Select("article_id").
			Where("tag_id IN (" + tagStr + ")").
			Group("article_id")
		query = query.Where("id IN (?)", subQuery)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Create 创建文章
func (r *articleRepository) Create(article *entity.Article) error {
	return r.db.Create(article).Error
}

// UpdateFields 更新文章的指定字段
func (r *articleRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&entity.Article{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除文章（软删除）
func (r *articleRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Article{}, id).Error
}

// IncrementViews 增加浏览量
func (r *articleRepository) IncrementViews(id uint) error {
	return r.db.Model(&entity.Article{}).Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).Error
}

// GetTagsByArticleID 获取文章关联的标签
func (r *articleRepository) GetTagsByArticleID(articleID uint) ([]*entity.Tag, error) {
	var tags []*entity.Tag
	err := r.db.Table("tags").
		Joins("JOIN tag_articles ON tags.id = tag_articles.tag_id").
		Where("tag_articles.article_id = ?", articleID).
		Where("tags.status = ?", 1).
		Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// SetArticleTags 设置文章标签（先删后插）
func (r *articleRepository) SetArticleTags(articleID uint, tagIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 删除旧关联
		if err := tx.Where("article_id = ?", articleID).Delete(&entity.TagArticle{}).Error; err != nil {
			return err
		}
		// 插入新关联
		for _, tagID := range tagIDs {
			ta := entity.TagArticle{
				ArticleID: articleID,
				TagID:     tagID,
			}
			if err := tx.Create(&ta).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindLiked 查询用户是否已点赞
func (r *articleRepository) FindLiked(articleID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&entity.Like{}).
		Where("article_id = ? AND user_id = ?", articleID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateLike 创建点赞
func (r *articleRepository) CreateLike(articleID, userID uint) error {
	like := entity.Like{
		ArticleID: articleID,
		UserID:    userID,
	}
	return r.db.Create(&like).Error
}

// DeleteLike 取消点赞
func (r *articleRepository) DeleteLike(articleID, userID uint) error {
	return r.db.Where("article_id = ? AND user_id = ?", articleID, userID).
		Delete(&entity.Like{}).Error
}

// CountLikes 获取文章点赞数
func (r *articleRepository) CountLikes(articleID uint) (int64, error) {
	var count int64
	err := r.db.Model(&entity.Like{}).
		Where("article_id = ?", articleID).
		Count(&count).Error
	return count, err
}

// CountComments 获取文章评论数（未删除的）
func (r *articleRepository) CountComments(articleID uint) (int64, error) {
	var count int64
	err := r.db.Model(&entity.Comment{}).
		Where("article_id = ? AND deleted_at IS NULL", articleID).
		Count(&count).Error
	return count, err
}

// GetStats 获取已发布文章的统计数据（文章数、总阅读量、总点赞量）
func (r *articleRepository) GetStats() (int64, int64, int64, error) {
	var result struct {
		ArticleCount int64 `gorm:"column:article_count"`
		TotalViews   int64 `gorm:"column:total_views"`
		TotalLikes   int64 `gorm:"column:total_likes"`
	}
	err := r.db.Model(&entity.Article{}).
		Select("COUNT(*) as article_count, COALESCE(SUM(views), 0) as total_views, (SELECT COUNT(*) FROM likes INNER JOIN articles a2 ON a2.id = likes.article_id WHERE a2.status = 1) as total_likes").
		Where("status = ?", 1).
		Scan(&result).Error
	if err != nil {
		return 0, 0, 0, err
	}
	return result.ArticleCount, result.TotalViews, result.TotalLikes, nil
}

// FindArticleImages 获取文章关联的所有图片
func (r *articleRepository) FindArticleImages(articleID uint) ([]entity.ArticleImage, error) {
	var images []entity.ArticleImage
	err := r.db.Where("article_id = ?", articleID).Find(&images).Error
	if err != nil {
		return nil, err
	}
	return images, nil
}

// CreateArticleImages 批量创建文章图片引用
func (r *articleRepository) CreateArticleImages(images []entity.ArticleImage) error {
	if len(images) == 0 {
		return nil
	}
	return r.db.Create(&images).Error
}

// DeleteArticleImages 批量删除文章图片引用（按 ID）
func (r *articleRepository) DeleteArticleImages(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Delete(&entity.ArticleImage{}, ids).Error
}

// DeleteArticleImagesByArticleID 删除文章的所有图片引用
func (r *articleRepository) DeleteArticleImagesByArticleID(articleID uint) error {
	return r.db.Where("article_id = ?", articleID).Delete(&entity.ArticleImage{}).Error
}

// TransferCategory 将一个分类下的所有文章转移到另一个分类，返回受影响行数
func (r *articleRepository) TransferCategory(fromTypeID, toTypeID uint) (int64, error) {
	result := r.db.Model(&entity.Article{}).Where("type_id = ?", fromTypeID).Update("type_id", toTypeID)
	return result.RowsAffected, result.Error
}

// GetRandomID 随机获取一篇已发布文章的 ID
func (r *articleRepository) GetRandomID() (uint, error) {
	var count int64
	if err := r.db.Model(&entity.Article{}).Where("status = ?", 1).Count(&count).Error; err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}

	offset := rand.Intn(int(count))
	var article entity.Article
	if err := r.db.Where("status = ?", 1).Offset(offset).Limit(1).Find(&article).Error; err != nil {
		return 0, err
	}
	return article.ID, nil
}
