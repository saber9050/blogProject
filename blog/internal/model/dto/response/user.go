package response

import "time"

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	UserID    uint      `json:"user_id"`    // id
	UserName  string    `json:"user_name"`  // 昵称
	Account   string    `json:"account"`    // 账号
	Email     string    `json:"email"`      // 邮箱
	AvatarURL string    `json:"avatar_url"` // 头像
	RoleID    int8      `json:"role_id"`    // 角色ID
	Staus     int8      `json:"staus"`      // 状态
	CreateAt  time.Time `json:"create_at"`  // 创建时间
}

// UpdateUserAvatarResponse 修改头像响应
type UpdateUserAvatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

// UpdateUserEmailResponse 修改邮箱响应
type UpdateUserEmailResponse struct {
	Message string `json:"message"`
}
