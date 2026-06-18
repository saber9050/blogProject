package article

import (
	"blog/internal/model/entity"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type articleRepository struct {
	db *gorm.DB
}

// NewArticleRepository 创建文章仓储实例
func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db: db}
}

// ListArticles 分页查询文章列表
func (r *articleRepository) ListArticles(params *ArticleListParams) ([]entity.Article, int64, error) {
	// 基础查询：未删除的文章
	baseQuery := r.db.Model(&entity.Article{}).Where("articles.deleted_at IS NULL")

	// 分类筛选
	if params.CategoryID != nil && *params.CategoryID != 0 {
		baseQuery = baseQuery.Where("articles.type_id = ?", *params.CategoryID)
	}

	// 标签筛选（多选：AND 语义）
	if len(params.TagIDs) > 0 {
		for _, tid := range params.TagIDs {
			subQuery := r.db.Model(&entity.TagArticle{}).
				Select("1").
				Where("tag_articles.article_id = articles.id AND tag_articles.tag_id = ?", tid)
			baseQuery = baseQuery.Where("EXISTS (?)", subQuery)
		}
	}

	// 标题搜索
	if params.Keyword != "" {
		baseQuery = baseQuery.Where("articles.title LIKE ?", "%"+params.Keyword+"%")
	}

	// 先查总数
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []entity.Article{}, 0, nil
	}

	// 排序
	sortQuery := baseQuery.Session(&gorm.Session{})
	switch params.Sort {
	case "popular":
		// 热度分排序：SQL 中实时计算
		sortQuery = r.applyHotScoreOrder(sortQuery)
	default:
		// 默认按创建时间倒序
		sortQuery = sortQuery.Order("articles.created_at DESC")
	}

	// 分页
	offset := (params.Page - 1) * params.PageSize
	var articles []entity.Article
	if err := sortQuery.Offset(offset).Limit(params.PageSize).Find(&articles).Error; err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// applyHotScoreOrder 热度排序
func (r *articleRepository) applyHotScoreOrder(query *gorm.DB) *gorm.DB {
	// 热度分 = (views*0.3 + likes*1.0 + comments*0.8) / POW(DATEDIFF(NOW(), created_at)+2, 1.5)
	likeSub := r.db.Model(&entity.Like{}).
		Select("COUNT(*)").
		Where("likes.article_id = articles.id")

	commentSub := r.db.Model(&entity.Comment{}).
		Select("COUNT(*)").
		Where("comments.article_id = articles.id AND comments.deleted_at IS NULL")

	hotExpr := fmt.Sprintf(
		"(articles.views * 0.3 + COALESCE((%s), 0) * 1.0 + COALESCE((%s), 0) * 0.8) / POW(DATEDIFF(NOW(), articles.created_at) + 2, 1.5)",
		r.db.ToSQL(func(tx *gorm.DB) *gorm.DB { return likeSub.Find(nil) }),
		r.db.ToSQL(func(tx *gorm.DB) *gorm.DB { return commentSub.Find(nil) }),
	)

	return query.Select("articles.*, " + hotExpr + " AS hot_score").Order("hot_score DESC")
}

// FindByID 根据 ID 查找文章
func (r *articleRepository) FindByID(id uint) (*entity.Article, error) {
	var article entity.Article
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&article).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &article, nil
}

// IncrementViews 批量增加文章浏览量
func (r *articleRepository) IncrementViews(viewMap map[uint]int) error {
	for articleID, count := range viewMap {
		if count <= 0 {
			continue
		}
		if err := r.db.Model(&entity.Article{}).
			Where("id = ?", articleID).
			UpdateColumn("views", gorm.Expr("views + ?", count)).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetPrevAndNext 获取上一篇和下一篇
func (r *articleRepository) GetPrevAndNext(id uint) (*PrevNextArticle, *PrevNextArticle, error) {
	var prev, next *PrevNextArticle

	// 上一篇：更早创建的文章
	var prevArticle entity.Article
	err := r.db.Select("id, title").
		Where("id < ? AND deleted_at IS NULL", id).
		Order("id DESC").
		First(&prevArticle).Error
	if err == nil {
		prev = &PrevNextArticle{ID: prevArticle.ID, Title: prevArticle.Title}
	} else if err != gorm.ErrRecordNotFound {
		return nil, nil, err
	}

	// 下一篇：更晚创建的文章
	var nextArticle entity.Article
	err = r.db.Select("id, title").
		Where("id > ? AND deleted_at IS NULL", id).
		Order("id ASC").
		First(&nextArticle).Error
	if err == nil {
		next = &PrevNextArticle{ID: nextArticle.ID, Title: nextArticle.Title}
	} else if err != gorm.ErrRecordNotFound {
		return nil, nil, err
	}

	return prev, next, nil
}

// LikeArticle 点赞
func (r *articleRepository) LikeArticle(articleID, userID uint) error {
	like := entity.Like{ArticleID: articleID, UserID: userID}
	return r.db.Create(&like).Error
}

// UnlikeArticle 取消点赞
func (r *articleRepository) UnlikeArticle(articleID, userID uint) error {
	result := r.db.Where("article_id = ? AND user_id = ?", articleID, userID).Delete(&entity.Like{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// BatchLikeStatus 批量查询用户是否点赞
func (r *articleRepository) BatchLikeStatus(userID uint, articleIDs []uint) (map[uint]bool, error) {
	if len(articleIDs) == 0 {
		return make(map[uint]bool), nil
	}

	var likes []entity.Like
	if err := r.db.Where("user_id = ? AND article_id IN ?", userID, articleIDs).Find(&likes).Error; err != nil {
		return nil, err
	}

	likedMap := make(map[uint]bool, len(articleIDs))
	for _, id := range articleIDs {
		likedMap[id] = false
	}
	for _, l := range likes {
		likedMap[l.ArticleID] = true
	}
	return likedMap, nil
}

// GetLikeCounts 批量获取点赞数
func (r *articleRepository) GetLikeCounts(articleIDs []uint) (map[uint]uint, error) {
	if len(articleIDs) == 0 {
		return make(map[uint]uint), nil
	}

	type countResult struct {
		ArticleID uint
		Count     uint
	}

	var results []countResult
	sql := "SELECT article_id, COUNT(*) AS count FROM likes WHERE article_id IN ? GROUP BY article_id"
	if err := r.db.Raw(sql, articleIDs).Scan(&results).Error; err != nil {
		return nil, err
	}

	countMap := make(map[uint]uint, len(articleIDs))
	for _, id := range articleIDs {
		countMap[id] = 0
	}
	for _, r := range results {
		countMap[r.ArticleID] = r.Count
	}
	return countMap, nil
}

// GetCommentCounts 批量获取评论数
func (r *articleRepository) GetCommentCounts(articleIDs []uint) (map[uint]uint, error) {
	if len(articleIDs) == 0 {
		return make(map[uint]uint), nil
	}

	type countResult struct {
		ArticleID uint
		Count     uint
	}

	var results []countResult
	sql := "SELECT article_id, COUNT(*) AS count FROM comments WHERE article_id IN ? AND deleted_at IS NULL GROUP BY article_id"
	if err := r.db.Raw(sql, articleIDs).Scan(&results).Error; err != nil {
		return nil, err
	}

	countMap := make(map[uint]uint, len(articleIDs))
	for _, id := range articleIDs {
		countMap[id] = 0
	}
	for _, r := range results {
		countMap[r.ArticleID] = r.Count
	}
	return countMap, nil
}

// ListCategories 获取所有分类
func (r *articleRepository) ListCategories() ([]entity.Category, error) {
	var categories []entity.Category
	err := r.db.Where("status = 1").Find(&categories).Error
	return categories, err
}

// ListTags 获取所有标签
func (r *articleRepository) ListTags() ([]entity.Tag, error) {
	var tags []entity.Tag
	err := r.db.Where("status = 1").Find(&tags).Error
	return tags, err
}

// parseNumericIDs 解析逗号分隔的ID字符串
func parseNumericIDs(s string) []uint {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var id uint
		if _, err := fmt.Sscanf(p, "%d", &id); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}
