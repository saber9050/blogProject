package article

import (
	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	articleSvc "blog/internal/service/article"
	llmSvc "blog/internal/service/llm"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ArticleController 后台文章管理控制器
type ArticleController struct {
	articleService articleSvc.ArticleService
	llmService     llmSvc.LLMService
}

// NewArticleController 创建后台文章管理控制器
func NewArticleController(articleService articleSvc.ArticleService, llmService llmSvc.LLMService) *ArticleController {
	return &ArticleController{
		articleService: articleService,
		llmService:     llmService,
	}
}

// ListArticles 获取后台文章列表
func (ctrl *ArticleController) ListArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var status *int
	if s := c.Query("status"); s != "" {
		sInt, _ := strconv.Atoi(s)
		status = &sInt
	}

	var categoryID uint
	if cid := c.Query("category_id"); cid != "" {
		cidUint, _ := strconv.ParseUint(cid, 10, 32)
		categoryID = uint(cidUint)
	}

	keyword := c.Query("keyword")

	var tagIDs []uint
	if tid := c.Query("tag_id"); tid != "" {
		if id, err := strconv.ParseUint(tid, 10, 32); err == nil {
			tagIDs = []uint{uint(id)}
		}
	}

	result, err := ctrl.articleService.AdminList(page, pageSize, status, categoryID, tagIDs, keyword)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateArticle 创建文章
func (ctrl *ArticleController) CreateArticle(c *gin.Context) {
	var req request.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	result, err := ctrl.articleService.AdminCreate(&req, userID)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// UpdateArticle 更新文章
func (ctrl *ArticleController) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	var req request.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.articleService.AdminUpdate(uint(id), &req); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteArticle 删除文章
func (ctrl *ArticleController) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文章ID")
		return
	}

	if err := ctrl.articleService.AdminDelete(uint(id)); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// TransferArticleCategory 一键转移分类
func (ctrl *ArticleController) TransferArticleCategory(c *gin.Context) {
	var req request.TransferCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	affected, err := ctrl.articleService.TransferCategory(req.FromTypeID, req.ToTypeID)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.SuccessWithMessage(c, "转移成功", gin.H{
		"affected_count": affected,
	})
}

// GenerateSummary 一键生成摘要
func (ctrl *ArticleController) GenerateSummary(c *gin.Context) {
	var req request.GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	result, err := ctrl.llmService.GenerateSummary(req.Title, req.Content)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, result)
}

// UploadImage 上传文章图片,返回完整路径
func (ctrl *ArticleController) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传头像文件")
		return
	}

	url, err := ctrl.articleService.UploadImage(file)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, gin.H{
		"url": url,
	})
}
