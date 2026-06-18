package article

import (
	"blog/internal/model/dto/response"
	repo "blog/internal/repository/article"
	"blog/pkg/database"
	"blog/pkg/errors"
	"blog/pkg/logger"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	viewKeyPrefix = "article:view:"  // Redis 浏览量计数器 key 前缀
	viewTTL       = 30 * time.Minute // Redis key 过期时间
	flushInterval = 5 * time.Minute  // 定时刷盘间隔
)

type articleService struct {
	articleRepo repo.ArticleRepository
}

// NewArticleService 创建文章服务实例
func NewArticleService(articleRepo repo.ArticleRepository) ArticleService {
	return &articleService{articleRepo: articleRepo}
}

// ListArticles 文章列表
func (s *articleService) ListArticles(params *repo.ArticleListParams) (*response.ArticleListResponse, error) {
	articles, total, err := s.articleRepo.ListArticles(params)
	if err != nil {
		return nil, fmt.Errorf("查询文章列表失败: %w", err)
	}

	if len(articles) == 0 {
		return &response.ArticleListResponse{
			List:     []response.ArticleListItem{},
			Total:    total,
			Page:     params.Page,
			PageSize: params.PageSize,
		}, nil
	}

	// 提取文章ID列表
	articleIDs := make([]uint, len(articles))
	for i, a := range articles {
		articleIDs[i] = a.ID
	}

	// 批量获取点赞数、评论数
	likeCounts, _ := s.articleRepo.GetLikeCounts(articleIDs)
	commentCounts, _ := s.articleRepo.GetCommentCounts(articleIDs)

	// 批量获取用户点赞状态（仅登录用户）
	var likedMap map[uint]bool
	if params.UserID != 0 {
		likedMap, _ = s.articleRepo.BatchLikeStatus(params.UserID, articleIDs)
	}
	if likedMap == nil {
		likedMap = make(map[uint]bool)
	}

	// 组装列表项
	list := make([]response.ArticleListItem, len(articles))
	for i, a := range articles {
		isLiked := false
		if v, ok := likedMap[a.ID]; ok {
			isLiked = v
		}
		list[i] = response.ArticleListItem{
			ID:           a.ID,
			Title:        a.Title,
			Summary:      a.Summary,
			CoverURL:     a.CoverURL,
			AuthorID:     a.UserID,
			AuthorName:   "", // 需 JOIN 用户表，这里留空，实际使用时可在 repository 层 JOIN
			AuthorAvatar: "",
			Category:     response.CategoryInfo{}, // 同上
			Tags:         []response.TagInfo{},
			ViewCount:    a.Views,
			LikeCount:    likeCounts[a.ID],
			CommentCount: commentCounts[a.ID],
			IsLiked:      isLiked,
			CreatedAt:    a.CreatedAt,
		}
	}

	return &response.ArticleListResponse{
		List:     list,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// GetArticleDetail 文章详情
func (s *articleService) GetArticleDetail(articleID, userID uint) (*response.ArticleDetail, error) {
	article, err := s.articleRepo.FindByID(articleID)
	if err != nil {
		return nil, fmt.Errorf("查询文章详情失败: %w", err)
	}
	if article == nil {
		return nil, errors.ErrResourceNotFound
	}

	// 获取点赞数、评论数
	likeCounts, _ := s.articleRepo.GetLikeCounts([]uint{articleID})
	commentCounts, _ := s.articleRepo.GetCommentCounts([]uint{articleID})

	// 获取用户点赞状态
	isLiked := false
	if userID != 0 {
		likedMap, _ := s.articleRepo.BatchLikeStatus(userID, []uint{articleID})
		isLiked = likedMap[articleID]
	}

	// 获取上一篇/下一篇
	prev, next, _ := s.articleRepo.GetPrevAndNext(articleID)

	detail := &response.ArticleDetail{
		ArticleListItem: response.ArticleListItem{
			ID:           article.ID,
			Title:        article.Title,
			Summary:      article.Summary,
			CoverURL:     article.CoverURL,
			AuthorID:     article.UserID,
			AuthorName:   "",
			AuthorAvatar: "",
			Category:     response.CategoryInfo{},
			Tags:         []response.TagInfo{},
			ViewCount:    article.Views,
			LikeCount:    likeCounts[articleID],
			CommentCount: commentCounts[articleID],
			IsLiked:      isLiked,
			CreatedAt:    article.CreatedAt,
		},
		Content: article.Content,
	}
	if prev != nil {
		detail.PrevArticle = &response.ArticleNavLink{ID: prev.ID, Title: prev.Title}
	}
	if next != nil {
		detail.NextArticle = &response.ArticleNavLink{ID: next.ID, Title: next.Title}
	}

	return detail, nil
}

// LikeArticle 点赞
func (s *articleService) LikeArticle(articleID, userID uint) error {
	// 先检查文章是否存在
	article, err := s.articleRepo.FindByID(articleID)
	if err != nil {
		return fmt.Errorf("查询文章失败: %w", err)
	}
	if article == nil {
		return errors.ErrResourceNotFound
	}

	if err := s.articleRepo.LikeArticle(articleID, userID); err != nil {
		// 唯一索引冲突 → 已点赞
		if isDuplicateError(err) {
			return errors.ErrResourceAlreadyExists
		}
		return fmt.Errorf("点赞失败: %w", err)
	}
	return nil
}

// UnlikeArticle 取消点赞
func (s *articleService) UnlikeArticle(articleID, userID uint) error {
	if err := s.articleRepo.UnlikeArticle(articleID, userID); err != nil {
		return errors.ErrResourceNotFound // 未点赞
	}
	return nil
}

// BatchLikeStatus 批量查询点赞状态
func (s *articleService) BatchLikeStatus(userID uint, articleIDs []uint) (*response.LikeStatusMap, error) {
	likedMap, err := s.articleRepo.BatchLikeStatus(userID, articleIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询点赞状态失败: %w", err)
	}
	return &response.LikeStatusMap{LikedMap: likedMap}, nil
}

// RecordView 记录阅读（Redis INCR）
func (s *articleService) RecordView(articleID uint) error {
	redisClient := database.GetRedis()
	if redisClient == nil {
		return nil // Redis 不可用时静默跳过
	}

	key := fmt.Sprintf("%s%d", viewKeyPrefix, articleID)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisClient.Incr(ctx, key).Err(); err != nil {
		logger.Warn("Redis INCR 浏览量失败", zap.Uint("articleID", articleID), zap.Error(err))
		return err
	}

	// 设置过期时间（仅首次设置）
	redisClient.Expire(ctx, key, viewTTL)
	return nil
}

// FlushViewCounts 定时刷新浏览量到 MySQL
func (s *articleService) FlushViewCounts() error {
	redisClient := database.GetRedis()
	if redisClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// SCAN 遍历所有 article:view:* key
	var cursor uint64
	viewMap := make(map[uint]int)

	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, viewKeyPrefix+"*", 100).Result()
		if err != nil {
			logger.Warn("Redis SCAN 失败", zap.Error(err))
			return err
		}

		for _, key := range keys {
			articleIDStr := strings.TrimPrefix(key, viewKeyPrefix)
			articleID, parseErr := strconv.ParseUint(articleIDStr, 10, 64)
			if parseErr != nil {
				continue
			}

			countStr, getErr := redisClient.GetDel(ctx, key).Result()
			if getErr != nil {
				continue
			}
			count, convErr := strconv.Atoi(countStr)
			if convErr != nil || count <= 0 {
				continue
			}
			viewMap[uint(articleID)] += count
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(viewMap) == 0 {
		return nil
	}

	if err := s.articleRepo.IncrementViews(viewMap); err != nil {
		logger.Error("刷盘浏览量失败", zap.Error(err))
		return err
	}

	logger.Info(fmt.Sprintf("浏览量刷盘完成，共 %d 篇文章", len(viewMap)))
	return nil
}

// ListCategories 获取所有分类
func (s *articleService) ListCategories() ([]response.CategoryInfo, error) {
	categories, err := s.articleRepo.ListCategories()
	if err != nil {
		return nil, fmt.Errorf("获取分类列表失败: %w", err)
	}
	result := make([]response.CategoryInfo, len(categories))
	for i, c := range categories {
		result[i] = response.CategoryInfo{ID: c.ID, Name: c.CategoryName}
	}
	return result, nil
}

// ListTags 获取所有标签
func (s *articleService) ListTags() ([]response.TagInfo, error) {
	tags, err := s.articleRepo.ListTags()
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}
	result := make([]response.TagInfo, len(tags))
	for i, t := range tags {
		result[i] = response.TagInfo{ID: t.ID, Name: t.TagName}
	}
	return result, nil
}

// isDuplicateError 判断是否为唯一索引冲突错误
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "duplicate key")
}
