package request

// DeleteChunksRequest 批量删除文章向量
//
// 一次最多 200 篇：前端是多选，正常不会超过这个量；设上限是为了避免
// 一次请求把几万篇文章塞进 IN 条件，既拖慢数据库也让「部分失败」难解释。
type DeleteChunksRequest struct {
	ArticleIDs []uint `json:"article_ids" binding:"required,min=1,max=200,dive,gt=0"`
}
