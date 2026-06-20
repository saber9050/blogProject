package article

import (
	"blog/internal/constant"
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/article"
	"blog/pkg/errors"
	minioPkg "blog/pkg/minio"
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

// articleService 文章服务实现
type articleService struct {
	articleRepo repo.ArticleRepository
	userRepo    interface {
		FindByID(id uint) (*entity.User, error)
	}
	categoryRepo interface {
		FindByID(id uint) (*entity.Category, error)
	}
	tagRepo interface {
		FindByIDs(ids []uint) ([]*entity.Tag, error)
	}
	minio *minioPkg.Client
}

// NewArticleService 创建文章服务实例
func NewArticleService(articleRepo repo.ArticleRepository) ArticleService {
	return &articleService{articleRepo: articleRepo}
}

// SetDeps 设置依赖（避免循环依赖）
func (s *articleService) SetDeps(userRepo interface {
	FindByID(id uint) (*entity.User, error)
}, categoryRepo interface {
	FindByID(id uint) (*entity.Category, error)
}, tagRepo interface {
	FindByIDs(ids []uint) ([]*entity.Tag, error)
}, minio *minioPkg.Client) {
	s.userRepo = userRepo
	s.categoryRepo = categoryRepo
	s.tagRepo = tagRepo
	s.minio = minio
}

// buildArticleResponse 构建文章响应对象
func (s *articleService) buildArticleResponse(article *entity.Article, userID uint) (*response.ArticleItem, error) {
	// 获取作者信息
	authorName := ""
	if s.userRepo != nil {
		user, err := s.userRepo.FindByID(article.UserID)
		if err == nil && user != nil {
			authorName = user.UserName
		}
	}

	// 获取分类信息
	var cat *response.CategoryInfo
	if s.categoryRepo != nil {
		category, err := s.categoryRepo.FindByID(article.TypeID)
		if err == nil && category != nil {
			cat = &response.CategoryInfo{
				ID:   category.ID,
				Name: category.CategoryName,
			}
		}
	}

	// 获取标签信息
	var tags []*response.TagInfo
	if s.tagRepo != nil {
		tagList, err := s.articleRepo.GetTagsByArticleID(article.ID)
		if err == nil {
			for _, t := range tagList {
				tags = append(tags, &response.TagInfo{
					ID:   t.ID,
					Name: t.TagName,
				})
			}
		}
	}

	// 获取点赞状态
	isLiked := false
	if userID > 0 {
		liked, err := s.articleRepo.FindLiked(article.ID, userID)
		if err == nil {
			isLiked = liked
		}
	}

	return &response.ArticleItem{
		ID:           article.ID,
		Title:        article.Title,
		Summary:      article.Summary,
		CoverURL:     article.CoverURL,
		Status:       article.Status,
		Views:        article.Views,
		LikeCount:    article.LikeCount,
		CommentCount: article.CommentCount,
		IsLiked:      isLiked,
		AuthorName:   authorName,
		Category:     cat,
		Tags:         tags,
		CreatedAt:    article.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// ListPublic 前台获取文章列表
func (s *articleService) ListPublic(page, pageSize int, sort string, categoryID uint, tagIDs []uint, keyword string, userID uint) (*response.ArticleListResponse, error) {
	list, total, err := s.articleRepo.ListPublic(page, pageSize, sort, categoryID, tagIDs, keyword)
	if err != nil {
		return nil, fmt.Errorf("获取文章列表失败: %w", err)
	}

	var items []*response.ArticleItem
	for _, article := range list {
		item, err := s.buildArticleResponse(article, userID)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return &response.ArticleListResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetDetail 获取文章详情
func (s *articleService) GetDetail(id uint, userID uint) (*response.ArticleDetailResponse, error) {
	article, err := s.articleRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("查找文章失败: %w", err)
	}
	if article == nil {
		return nil, errors.New(errors.CodeNotFound, "文章不存在")
	}

	// 增加浏览量
	_ = s.articleRepo.IncrementViews(id)

	item, err := s.buildArticleResponse(article, userID)
	if err != nil {
		return nil, err
	}

	return &response.ArticleDetailResponse{
		ArticleItem: *item,
		Content:     article.Content,
	}, nil
}

// LikeArticle 点赞文章
func (s *articleService) LikeArticle(articleID, userID uint) error {
	article, err := s.articleRepo.FindByID(articleID)
	if err != nil {
		return fmt.Errorf("查找文章失败: %w", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}

	// 检查是否已点赞
	liked, err := s.articleRepo.FindLiked(articleID, userID)
	if err != nil {
		return fmt.Errorf("检查点赞状态失败: %w", err)
	}
	if liked {
		return errors.New(errors.CodeBadRequest, "您已经点赞过该文章")
	}

	// 创建点赞记录
	if err := s.articleRepo.CreateLike(articleID, userID); err != nil {
		return fmt.Errorf("点赞失败: %w", err)
	}
	// 增加点赞计数
	return s.articleRepo.IncrementLikeCount(articleID)
}

// UnlikeArticle 取消点赞
func (s *articleService) UnlikeArticle(articleID, userID uint) error {
	article, err := s.articleRepo.FindByID(articleID)
	if err != nil {
		return fmt.Errorf("查找文章失败: %w", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}

	// 检查是否已点赞
	liked, err := s.articleRepo.FindLiked(articleID, userID)
	if err != nil {
		return fmt.Errorf("检查点赞状态失败: %w", err)
	}
	if !liked {
		return errors.New(errors.CodeBadRequest, "您尚未点赞该文章")
	}

	// 删除点赞记录
	if err := s.articleRepo.DeleteLike(articleID, userID); err != nil {
		return fmt.Errorf("取消点赞失败: %w", err)
	}
	// 减少点赞计数
	return s.articleRepo.DecrementLikeCount(articleID)
}

// AdminList 后台获取文章列表
func (s *articleService) AdminList(page, pageSize int, status *int, categoryID uint, tagIDs []uint, keyword string) (*response.ArticleListResponse, error) {
	list, total, err := s.articleRepo.ListAdmin(page, pageSize, status, categoryID, tagIDs, keyword)
	if err != nil {
		return nil, fmt.Errorf("获取文章列表失败: %w", err)
	}

	var items []*response.ArticleItem
	for _, article := range list {
		item, err := s.buildArticleResponse(article, 0)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return &response.ArticleListResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// AdminCreate 后台创建文章
func (s *articleService) AdminCreate(req *request.CreateArticleRequest, userID uint) (*response.CreateArticleResponse, error) {
	// 验证分类是否存在
	if s.categoryRepo != nil {
		category, err := s.categoryRepo.FindByID(req.TypeID)
		if err != nil {
			return nil, fmt.Errorf("查找分类失败: %w", err)
		}
		if category == nil {
			return nil, errors.New(errors.CodeBadRequest, "分类不存在")
		}
	}

	article := &entity.Article{
		UserID:   userID,
		Title:    req.Title,
		Content:  req.Content,
		CoverURL: req.CoverURL,
		Summary:  req.Summary,
		TypeID:   req.TypeID,
		Status:   req.Status,
	}

	if err := s.articleRepo.Create(article); err != nil {
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}

	// 关联标签
	if len(req.TagIDs) > 0 {
		if err := s.articleRepo.SetArticleTags(article.ID, req.TagIDs); err != nil {
			return nil, fmt.Errorf("关联标签失败: %w", err)
		}
	}

	return &response.CreateArticleResponse{ID: article.ID}, nil
}

// AdminUpdate 后台更新文章
func (s *articleService) AdminUpdate(id uint, req *request.UpdateArticleRequest) error {
	article, err := s.articleRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("查找文章失败: %w", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}

	// 验证分类是否存在
	if s.categoryRepo != nil {
		category, err := s.categoryRepo.FindByID(req.TypeID)
		if err != nil {
			return fmt.Errorf("查找分类失败: %w", err)
		}
		if category == nil {
			return errors.New(errors.CodeBadRequest, "分类不存在")
		}
	}

	article.Title = req.Title
	article.Content = req.Content
	article.CoverURL = req.CoverURL
	article.Summary = req.Summary
	article.TypeID = req.TypeID
	article.Status = req.Status

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("更新文章失败: %w", err)
	}

	// 更新标签关联
	if req.TagIDs != nil {
		if err := s.articleRepo.SetArticleTags(id, req.TagIDs); err != nil {
			return fmt.Errorf("关联标签失败: %w", err)
		}
	}

	return nil
}

// AdminDelete 后台删除文章
func (s *articleService) AdminDelete(id uint) error {
	article, err := s.articleRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("查找文章失败: %w", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}
	return s.articleRepo.Delete(id)
}

// UploadImage 上传图片
func (s *articleService) UploadImage(fileHeader *multipart.FileHeader) (string, error) {
	if s.minio == nil {
		return "", errors.New(errors.CodeInternalError, "图片服务未配置")
	}

	if fileHeader.Size > constant.ImageMaxLength {
		str := fmt.Sprintf("图片大小不能超过 %dMB", constant.ImageMaxLength>>20)
		return "", errors.New(errors.CodeBadRequest, str)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if _, ok := constant.AllowedImageExt[ext]; !ok {
		return "", errors.New(errors.CodeBadRequest, "不支持的文件格式")
	}

	objectName := fmt.Sprintf("articles/%s/%s%s",
		time.Now().Format("20060102"),
		fmt.Sprintf("%d", time.Now().UnixNano()),
		ext)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fileKey, err := s.minio.Upload(ctx, objectName, src, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return "", fmt.Errorf("上传图片失败: %w", err)
	}
	return s.minio.GetFileURL(fileKey), nil
}
