package chunk

import (
	"blog/internal/model/dto/request"
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
// article_id 不传时返回全部块（按文章倒序），用于「最近切分情况」总览。
func (ctrl *Controller) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var articleID uint
	if v := c.Query("article_id"); v != "" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			response.BadRequest(c, "article_id 无效")
			return
		}
		articleID = uint(n)
	}

	result, err := ctrl.chunkService.List(articleID, c.Query("keyword"), page, pageSize)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.Success(c, result)
}

// Stat 聚合统计（按文章分组）
func (ctrl *Controller) Stat(c *gin.Context) {
	result, err := ctrl.chunkService.Stat()
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

// DeleteArticles 批量删除文章向量
//
// 只有整篇删除，没有单块删除：块是「整篇重新切分、整体替换」的产物，
// 删掉其中一块之后这篇文章的块集合既不是这次切分的结果、也不是上次的。
//
// 用 POST 而不是 DELETE：DELETE 带 body 在部分代理/客户端上会被丢掉，
// 而这里必须传一个 id 数组。
func (ctrl *Controller) DeleteArticles(c *gin.Context) {
	var req request.DeleteChunksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误：article_ids 必填，单次最多 200 篇")
		return
	}

	result, err := ctrl.chunkService.DeleteArticles(req.ArticleIDs)
	if err != nil {
		response.BizError(c, err)
		return
	}
	response.SuccessWithMessage(c, "已处理选中文章的向量删除", result)
}
