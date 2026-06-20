package article

import (
	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	articleSvc "blog/internal/service/article"
	"blog/pkg/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ArticleController 文章控制器
type ArticleController struct {
	articleService articleSvc.ArticleService
}

// NewArticleController 创建文章控制器
func NewArticleController(articleService articleSvc.ArticleService) *ArticleController {
	return &ArticleController{articleService: articleService}
}

// ListArticles 获取文章列表
func (ctrl *ArticleController) ListArticles(c *gin.Context) {
	var query request.ArticleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	if query.Sort == "" {
		query.Sort = "latest"
	}

	// 解析 tag_ids
	var tagIDs []uint
	if query.TagIDs != "" {
		parts := strings.Split(query.TagIDs, ",")
		for _, p := range parts {
			id, err := strconv.ParseUint(strings.TrimSpace(p), 10, 32)
			if err == nil {
				tagIDs = append(tagIDs, uint(id))
			}
		}
	}

	userID := middleware.GetUserID(c)
	result, err := ctrl.articleService.ListPublic(query.Page, query.PageSize, query.Sort, query.CategoryID, tagIDs, query.Keyword, userID)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// GetArticleDetail 获取文章详情
func (ctrl *ArticleController) GetArticleDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	userID := middleware.GetUserID(c)
	result, err := ctrl.articleService.GetDetail(uint(id), userID)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// LikeArticle 点赞文章
func (ctrl *ArticleController) LikeArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	userID := middleware.GetUserID(c)
	if err := ctrl.articleService.LikeArticle(uint(id), userID); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// UnlikeArticle 取消点赞
func (ctrl *ArticleController) UnlikeArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	userID := middleware.GetUserID(c)
	if err := ctrl.articleService.UnlikeArticle(uint(id), userID); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}
