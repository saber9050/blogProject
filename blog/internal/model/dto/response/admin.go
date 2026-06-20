package response

// AdminUserResponse 后台用户管理响应
type AdminUserResponse struct {
	ID        uint   `json:"id"`
	UserName  string `json:"user_name"`
	AvatarURL string `json:"avatar_url"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
}

// CategoryPublicResponse 前台分类响应
type CategoryPublicResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// CategoryAdminResponse 后台分类响应
type CategoryAdminResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
}

// TagPublicResponse 前台标签响应
type TagPublicResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// TagAdminResponse 后台标签响应
type TagAdminResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
}

// PaginatedResponse 分页响应
type PaginatedResponse struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
