package request

// AdminCreateUserRequest 后台创建用户请求
type AdminCreateUserRequest struct {
	UserName string `json:"user_name" binding:"required,min=1,max=15"`
	Account  string `json:"account" binding:"required,len=11"`
	Password string `json:"password" binding:"required,min=11,max=20"`
	Status   int8   `json:"status" binding:"oneof=0 1"`
}

// AdminUpdateUserRequest 后台更新用户状态请求
type AdminUpdateUserRequest struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name   string `json:"name" binding:"required,max=50"`
	Status int8   `json:"status" binding:"oneof=0 1"`
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name   string `json:"name" binding:"required,max=50"`
	Status int8   `json:"status" binding:"oneof=0 1"`
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name   string `json:"name" binding:"required,max=100"`
	Status int8   `json:"status" binding:"oneof=0 1"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name   string `json:"name" binding:"required,max=100"`
	Status int8   `json:"status" binding:"oneof=0 1"`
}

// ListQuery 后台列表查询参数
type ListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   int    `form:"status"`
	Keyword  string `form:"keyword"`
}
