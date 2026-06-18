package article

import (
	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	articleRepo "blog/internal/repository/article"
	articleSvc "blog/internal/service/article"
	"blog/pkg/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Controller 文章控制器
type Controller struct {
	articleService articleSvc.ArticleService
}

// NewController 创建文章控制器
func NewController(articleService articleSvc.ArticleService) *Controller {
	return &Controller{
		articleService: articleService,
	}
}

// ListArticles 文章列表
func (c *Controller) ListArticles(ctx *gin.Context) {
	var req request.ListArticlesReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	// 默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 20 {
		req.PageSize = 20
	}
	if req.Sort == "" {
		req.Sort = "latest"
	}

	// 获取登录用户ID（可选）
	userID := middleware.GetUserID(ctx)

	// 解析标签ID
	var tagIDs []uint
	if req.TagIDs != "" {
		for _, s := range strings.Split(req.TagIDs, ",") {
			s = strings.TrimSpace(s)
			if id, err := strconv.ParseUint(s, 10, 64); err == nil && id > 0 {
				tagIDs = append(tagIDs, uint(id))
			}
		}
	}

	params := &articleRepo.ArticleListParams{
		Page:       req.Page,
		PageSize:   req.PageSize,
		Sort:       req.Sort,
		CategoryID: req.CategoryID,
		TagIDs:     tagIDs,
		Keyword:    req.Keyword,
		UserID:     userID,
	}

	result, err := c.articleService.ListArticles(params)
	if err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetArticleDetail 文章详情
func (c *Controller) GetArticleDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(ctx, "文章ID无效")
		return
	}

	// 获取登录用户ID（可选）
	userID := middleware.GetUserID(ctx)

	detail, err := c.articleService.GetArticleDetail(uint(id), userID)
	if err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, detail)
}

// LikeArticle 点赞文章
func (c *Controller) LikeArticle(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	idStr := ctx.Param("id")
	articleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || articleID == 0 {
		response.BadRequest(ctx, "文章ID无效")
		return
	}

	if err := c.articleService.LikeArticle(uint(articleID), userID); err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, "点赞成功")
}

// UnlikeArticle 取消点赞
func (c *Controller) UnlikeArticle(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	idStr := ctx.Param("id")
	articleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || articleID == 0 {
		response.BadRequest(ctx, "文章ID无效")
		return
	}

	if err := c.articleService.UnlikeArticle(uint(articleID), userID); err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, "取消点赞成功")
}

// BatchLikeStatus 批量查询点赞状态
func (c *Controller) BatchLikeStatus(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	idsStr := ctx.Query("ids")
	if idsStr == "" {
		response.Success(ctx, map[string]interface{}{"liked_map": map[uint]bool{}})
		return
	}

	// 解析ID列表
	var articleIDs []uint
	for _, s := range strings.Split(idsStr, ",") {
		s = strings.TrimSpace(s)
		if id, err := strconv.ParseUint(s, 10, 64); err == nil && id > 0 {
			articleIDs = append(articleIDs, uint(id))
		}
	}

	result, err := c.articleService.BatchLikeStatus(userID, articleIDs)
	if err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// ListCategories 获取分类列表
func (c *Controller) ListCategories(ctx *gin.Context) {
	categories, err := c.articleService.ListCategories()
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, categories)
}

// ListTags 获取标签列表
func (c *Controller) ListTags(ctx *gin.Context) {
	tags, err := c.articleService.ListTags()
	if err != nil {
		response.BizError(ctx, err)
		return
	}
	response.Success(ctx, tags)
}
