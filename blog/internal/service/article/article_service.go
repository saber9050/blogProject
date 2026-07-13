package article

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/article"
	"blog/internal/repository/category"
	commentRepo "blog/internal/repository/comment"
	"blog/internal/repository/tag"
	"blog/internal/repository/user"
	user2 "blog/internal/service/user"
	"blog/pkg/errors"
	"blog/pkg/logger"
	minioPkg "blog/pkg/minio"
	"context"
	"mime/multipart"
	"regexp"

	"go.uber.org/zap"
)

// articleService 文章服务实现
type articleService struct {
	articleRepo  repo.ArticleRepository
	userRepo     user.UserRepository
	categoryRepo category.CategoryRepository
	tagRepo      tag.TagRepository
	commentRepo  commentRepo.CommentRepository
	userSvc      user2.UserService
	minio        *minioPkg.Client
}

// NewArticleService 创建文章服务实例
func NewArticleService(
	articleRepo repo.ArticleRepository,
	userRepo user.UserRepository,
	categoryRepo category.CategoryRepository,
	tagRepo tag.TagRepository,
	commentRepo commentRepo.CommentRepository,
	userSvc user2.UserService,
	minio *minioPkg.Client,
) ArticleService {
	return &articleService{
		articleRepo:  articleRepo,
		userRepo:     userRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		commentRepo:  commentRepo,
		userSvc:      userSvc,
		minio:        minio,
	}
}

// buildArticleResponse 构建文章响应对象
func (s *articleService) buildArticleResponse(article *entity.Article, userID uint) (*response.ArticleItem, error) {
	// 获取作者信息
	authorName := ""
	if s.userRepo != nil {
		curuser, err := s.userRepo.FindByID(article.UserID)
		if err == nil && curuser != nil {
			authorName = curuser.UserName
		}
	}

	// 获取分类信息
	var cat *response.CategoryInfo
	if s.categoryRepo != nil {
		curcategory, err := s.categoryRepo.FindByID(article.TypeID)
		if err == nil && curcategory != nil {
			cat = &response.CategoryInfo{
				ID:   curcategory.ID,
				Name: curcategory.CategoryName,
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

	url := ""
	if article.CoverURL != "" {
		url = s.minio.GetFileURL(article.CoverURL)
	}

	// 获取实时点赞数和评论数（从 likes/comments 表 COUNT）
	likeCount, _ := s.articleRepo.CountLikes(article.ID)
	commentCount, _ := s.articleRepo.CountComments(article.ID)

	return &response.ArticleItem{
		ID:           article.ID,
		Title:        article.Title,
		Summary:      article.Summary,
		CoverURL:     url,
		Status:       article.Status,
		Views:        article.Views,
		LikeCount:    uint(likeCount),
		CommentCount: uint(commentCount),
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
		logger.Error("获取文章列表失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取文章列表失败", err)
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
		logger.Error("查找文章失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "查找文章失败", err)
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
		logger.Error("查找文章失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "查找文章失败", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}

	// 检查是否已点赞
	liked, err := s.articleRepo.FindLiked(articleID, userID)
	if err != nil {
		logger.Error("检查点赞状态失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "检查点赞状态失败", err)
	}
	if liked {
		return errors.New(errors.CodeBadRequest, "您已经点赞过该文章")
	}

	// 创建点赞记录
	return s.articleRepo.CreateLike(articleID, userID)
}

// UnlikeArticle 取消点赞
func (s *articleService) UnlikeArticle(articleID, userID uint) error {
	article, err := s.articleRepo.FindByID(articleID)
	if err != nil {
		logger.Error("查找文章失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "查找文章失败", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}

	// 检查是否已点赞
	liked, err := s.articleRepo.FindLiked(articleID, userID)
	if err != nil {
		logger.Error("检查点赞状态失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "检查点赞状态失败", err)
	}
	if !liked {
		return errors.New(errors.CodeBadRequest, "您尚未点赞该文章")
	}

	// 删除点赞记录
	return s.articleRepo.DeleteLike(articleID, userID)
}

// AdminList 后台获取文章列表
func (s *articleService) AdminList(page, pageSize int, status *int, categoryID uint, tagIDs []uint, keyword string) (*response.ArticleListResponse, error) {
	list, total, err := s.articleRepo.ListAdmin(page, pageSize, status, categoryID, tagIDs, keyword)
	if err != nil {
		logger.Error("获取文章列表失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取文章列表失败", err)
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
		curcategory, err := s.categoryRepo.FindByID(req.TypeID)
		if err != nil {
			logger.Error("查找分类失败", zap.Error(err))
			return nil, errors.NewWithErr(errors.CodeInternalError, "查找分类失败", err)
		}
		if curcategory == nil {
			return nil, errors.New(errors.CodeBadRequest, "分类不存在")
		}
	}
	url := ""
	var err error
	if req.CoverURL != "" {
		url, err = s.minio.ParseFileKey(req.CoverURL)
		if err != nil {
			return nil, errors.New(errors.CodeInternalError, "解析url失败")
		}
	}
	article := &entity.Article{
		UserID:   userID,
		Title:    req.Title,
		Content:  req.Content,
		CoverURL: url,
		Summary:  req.Summary,
		TypeID:   req.TypeID,
		Status:   req.Status,
	}

	if err := s.articleRepo.Create(article); err != nil {
		logger.Error("创建文章失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "创建文章失败", err)
	}

	// 关联标签
	if len(req.TagIDs) > 0 {
		if err := s.articleRepo.SetArticleTags(article.ID, req.TagIDs); err != nil {
			logger.Error("关联标签失败", zap.Error(err))
			return nil, errors.NewWithErr(errors.CodeInternalError, "关联标签失败", err)
		}
	}

	// 同步图片引用（自动解析 content 中的 <img> 并写入 article_images 表）
	s.syncArticleImages(article.ID, req.Content)

	return &response.CreateArticleResponse{ID: article.ID}, nil
}

// AdminUpdate 后台更新文章
func (s *articleService) AdminUpdate(id uint, req *request.UpdateArticleRequest) error {
	article, err := s.articleRepo.FindByID(id)
	if err != nil {
		logger.Error("查找文章失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "查找文章失败", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}

	// 验证分类是否存在
	if s.categoryRepo != nil && req.TypeID != 0 {
		curcategory, err := s.categoryRepo.FindByID(req.TypeID)
		if err != nil {
			logger.Error("查找分类失败", zap.Error(err))
			return errors.NewWithErr(errors.CodeInternalError, "查找分类失败", err)
		}
		if curcategory == nil {
			return errors.New(errors.CodeBadRequest, "分类不存在")
		}
	}

	// 构建需要更新的字段
	fields := make(map[string]interface{})

	lastURL := ""
	if req.CoverURL != "" && s.minio.GetFileURL(article.CoverURL) != req.CoverURL {
		url, err := s.minio.ParseFileKey(req.CoverURL)
		if err != nil {
			return errors.New(errors.CodeInternalError, "解析url失败")
		}
		fields["cover_url"] = url
		lastURL = article.CoverURL
	}

	if req.Title != "" {
		fields["title"] = req.Title
	}

	if req.Content != "" {
		fields["content"] = req.Content

	}

	if req.Summary != "" {
		fields["summary"] = req.Summary

	}

	if req.TypeID != 0 {
		fields["type_id"] = req.TypeID
	}

	fields["status"] = req.Status

	if err := s.articleRepo.UpdateFields(id, fields); err != nil {
		logger.Error("更新文章失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "更新文章失败", err)
	}

	// 更新标签关联
	if req.TagIDs != nil {
		if err := s.articleRepo.SetArticleTags(id, req.TagIDs); err != nil {
			logger.Error("关联标签失败", zap.Error(err))
			return errors.NewWithErr(errors.CodeInternalError, "关联标签失败", err)
		}
	}

	// 同步图片引用（当 content 变更时）
	if req.Content != "" {
		s.syncArticleImages(id, req.Content)
	}

	if lastURL != "" {
		ctx := context.Background()
		err = s.minio.Delete(ctx, lastURL)
		if err != nil {
			logger.Error("删除头像失败", zap.Error(err))
		}
	}

	return nil
}

// AdminDelete 后台删除文章
func (s *articleService) AdminDelete(id uint) error {
	article, err := s.articleRepo.FindByID(id)
	if err != nil {
		logger.Error("查找文章失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "查找文章失败", err)
	}
	if article == nil {
		return errors.New(errors.CodeNotFound, "文章不存在")
	}

	ctx := context.Background()

	// 1. 查询文章关联的图片
	images, err := s.articleRepo.FindArticleImages(id)
	if err != nil {
		logger.Error("查询文章图片记录失败", zap.Uint("article_id", id), zap.Error(err))
	} else {
		// 2. 删除 MinIO 中的图片文件（失败不中断，记录日志）
		for _, img := range images {
			if err := s.minio.Delete(ctx, img.URL); err != nil {
				logger.Error("删除MinIO图片失败", zap.String("key", img.URL), zap.Error(err))
			}
		}
		// 3. 删除 article_images 表记录
		if err := s.articleRepo.DeleteArticleImagesByArticleID(id); err != nil {
			logger.Error("删除文章图片DB记录失败", zap.Uint("article_id", id), zap.Error(err))
		}
	}

	// 4. 删除封面图
	if article.CoverURL != "" {
		if err := s.minio.Delete(ctx, article.CoverURL); err != nil {
			logger.Error("删除封面图失败", zap.Error(err))
		}
	}

	// 5. 软删除该文章关联的所有评论
	if err := s.commentRepo.DeleteByArticleID(id); err != nil {
		logger.Error("删除文章评论失败", zap.Uint("article_id", id), zap.Error(err))
	}

	// 6. 软删除文章
	return s.articleRepo.Delete(id)
}

// UploadImage 上传图片,返回完整路径
func (s *articleService) UploadImage(fileHeader *multipart.FileHeader) (string, error) {
	str, err := s.userSvc.UpLoadImage(fileHeader)
	if err != nil {
		return "", err
	}
	return s.minio.GetFileURL(str), nil
}

// extractImageURLs 从 HTML 内容中提取所有 <img> 标签的 src 属性值
func extractImageURLs(html string) []string {
	re := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
	matches := re.FindAllStringSubmatch(html, -1)
	var urls []string
	seen := make(map[string]struct{})
	for _, m := range matches {
		if len(m) >= 2 {
			u := m[1]
			if _, ok := seen[u]; !ok && u != "" {
				seen[u] = struct{}{}
				urls = append(urls, u)
			}
		}
	}
	return urls
}

// syncArticleImages 同步文章内容中引用的图片记录
// 1. 解析 content 提取所有 img src URL
// 2. 转为相对路径（file key）
// 3. 与 DB 中已有记录做 diff，删除被移除的图片（从 MinIO + DB），新增不存在的图片
func (s *articleService) syncArticleImages(articleID uint, content string) {
	// 1. 提取图片 URL
	fullURLs := extractImageURLs(content)
	if len(fullURLs) == 0 {
		// 内容中没有图片时，无需同步（保留原有记录，避免清理已上传的图片）
		return
	}

	// 2. 转为相对路径并去重
	newKeys := make(map[string]struct{})
	for _, u := range fullURLs {
		key, err := s.minio.ParseFileKey(u)
		if err != nil {
			logger.Warn("解析图片URL失败", zap.String("url", u), zap.Error(err))
			continue
		}
		newKeys[key] = struct{}{}
	}

	if len(newKeys) == 0 {
		return
	}

	// 3. 查询 DB 中已有记录
	oldImages, err := s.articleRepo.FindArticleImages(articleID)
	if err != nil {
		logger.Error("查询文章图片记录失败", zap.Uint("article_id", articleID), zap.Error(err))
		return
	}

	oldKeyMap := make(map[string]uint) // key -> id
	for _, img := range oldImages {
		oldKeyMap[img.URL] = img.ID
	}

	// 4. 计算差集
	var toDelete []uint                // 需要从 DB + MinIO 删除的图片记录 ID
	var toCreate []entity.ArticleImage // 需要新增的图片

	for key, id := range oldKeyMap {
		if _, exists := newKeys[key]; !exists {
			toDelete = append(toDelete, id)
		}
	}

	for key := range newKeys {
		if _, exists := oldKeyMap[key]; !exists {
			toCreate = append(toCreate, entity.ArticleImage{
				ArticleID: articleID,
				URL:       key,
			})
		}
	}

	// 5. 执行删除：先从 MinIO 删文件，再从 DB 删记录
	if len(toDelete) > 0 {
		ctx := context.Background()
		for _, id := range toDelete {
			// 找到对应的 key
			for key, imgID := range oldKeyMap {
				if imgID == id {
					if err := s.minio.Delete(ctx, key); err != nil {
						logger.Error("删除MinIO图片失败", zap.String("key", key), zap.Error(err))
					}
					break
				}
			}
		}
		if err := s.articleRepo.DeleteArticleImages(toDelete); err != nil {
			logger.Error("删除文章图片DB记录失败", zap.Uint("article_id", articleID), zap.Error(err))
		}
	}

	// 6. 执行新增
	if len(toCreate) > 0 {
		if err := s.articleRepo.CreateArticleImages(toCreate); err != nil {
			logger.Error("创建文章图片DB记录失败", zap.Uint("article_id", articleID), zap.Error(err))
		}
	}
}

// TransferCategory 将一个分类下的所有文章转移到另一个分类
func (s *articleService) TransferCategory(fromTypeID, toTypeID uint) (int64, error) {
	// 1. 验证源分类是否存在
	srcCategory, err := s.categoryRepo.FindByID(fromTypeID)
	if err != nil {
		logger.Error("查找源分类失败", zap.Error(err))
		return 0, errors.NewWithErr(errors.CodeInternalError, "查找源分类失败", err)
	}
	if srcCategory == nil {
		return 0, errors.New(errors.CodeBadRequest, "源分类不存在")
	}

	// 2. 验证目标分类是否存在
	dstCategory, err := s.categoryRepo.FindByID(toTypeID)
	if err != nil {
		logger.Error("查找目标分类失败", zap.Error(err))
		return 0, errors.NewWithErr(errors.CodeInternalError, "查找目标分类失败", err)
	}
	if dstCategory == nil {
		return 0, errors.New(errors.CodeBadRequest, "目标分类不存在")
	}

	// 3. 验证源分类和目标分类不能相同
	if fromTypeID == toTypeID {
		return 0, errors.New(errors.CodeBadRequest, "源分类和目标分类不能相同")
	}

	// 4. 执行转移
	affected, err := s.articleRepo.TransferCategory(fromTypeID, toTypeID)
	if err != nil {
		logger.Error("转移分类失败", zap.Error(err))
		return 0, errors.NewWithErr(errors.CodeInternalError, "转移分类失败", err)
	}

	return affected, nil
}

// GetStats 获取已发布文章的统计数据
func (s *articleService) GetStats() (*response.ArticleStatsResponse, error) {
	count, views, likes, err := s.articleRepo.GetStats()
	if err != nil {
		logger.Error("获取文章统计数据失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取文章统计数据失败", err)
	}
	return &response.ArticleStatsResponse{
		ArticleCount: count,
		TotalViews:   views,
		TotalLikes:   likes,
	}, nil
}

// GetRandomArticleID 随机获取一篇已发布文章的 ID
func (s *articleService) GetRandomArticleID() (uint, error) {
	id, err := s.articleRepo.GetRandomID()
	if err != nil {
		logger.Error("获取随机文章失败", zap.Error(err))
		return 0, errors.NewWithErr(errors.CodeInternalError, "获取随机文章失败", err)
	}
	if id == 0 {
		return 0, errors.New(errors.CodeNotFound, "暂无已发布的文章")
	}
	return id, nil
}
