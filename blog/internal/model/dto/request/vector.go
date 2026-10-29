package request

// ResetVectorRequest 清空并重建向量表请求。
//
// Confirm 必须是字面量 "RESET"。这不是形式主义——该操作会永久删除全部向量块，
// 而它通常是在改配置时顺手点的，一个误触的代价是几十篇文章要重新向量化。
//
// 前端入口已于 2026-10-29 移除，这个请求结构只服务于接口层的 curl 调用。
type ResetVectorRequest struct {
	Confirm string `json:"confirm"`
}
