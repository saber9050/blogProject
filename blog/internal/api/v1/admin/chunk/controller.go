package chunk

import (
	"blog/internal/service/chunk"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Controller 后台切分管理控制器
type Controller struct {
	chunkService chunk.Service
}

// NewController 创建后台切分管理控制器
func NewController(chunkService chunk.Service) *Controller {
	return &Controller{chunkService: chunkService}
}

// List 切分块明细分页
//
// template_id 与 article_id 都不传时返回全部块（按文章倒序），
// 用于「最近切分情况」总览。
func (ctrl *Controller) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var templateID, articleID uint
	if v := c.Query("template_id"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			response.BadRequest(c, "template_id 无效")
			return
		}
		templateID = uint(n)
	}
	if v := c.Query("article_id"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			response.BadRequest(c, "article_id 无效")
			return
		}
		articleID = uint(n)
	}

	result, err := ctrl.chunkService.List(templateID, articleID, c.Query("keyword"), page, pageSize)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Stat 聚合统计。?group_by=template|article
func (ctrl *Controller) Stat(c *gin.Context) {
	result, err := ctrl.chunkService.Stat(c.Query("group_by"))
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Get 单块详情（完整原文）
func (ctrl *Controller) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的块ID")
		return
	}

	result, err := ctrl.chunkService.Get(uint(id))
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}
