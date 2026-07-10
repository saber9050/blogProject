package response

// AdminInfoResponse 管理员公开信息响应
type AdminInfoResponse struct {
	AdminName    string `json:"admin_name"`
	AvatarURL    string `json:"avatar_url"`
	Introduction string `json:"introduction"`
	Email        string `json:"email"`
}

// AboutInfoResponse 关于页面完整信息响应
type AboutInfoResponse struct {
	AdminName    string `json:"admin_name"`
	AvatarURL    string `json:"avatar_url"`
	Email        string `json:"email"`
	Introduction string `json:"introduction"`
	TechStack    string `json:"tech_stack"`
	MyStory      string `json:"my_story"`
	Why          string `json:"why"`
	Interest     string `json:"interest"`
	GitHub       string `json:"git_hub"`
	CSDN         string `json:"csdn"`
}
