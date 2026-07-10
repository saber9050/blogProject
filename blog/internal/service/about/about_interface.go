package about

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
)

// AboutService 关于页面服务接口
type AboutService interface {
	// GetAdminInfo 获取管理员公开信息
	GetAdminInfo() (*response.AdminInfoResponse, error)
	// GetAboutInfo 获取关于页面完整信息
	GetAboutInfo() (*response.AboutInfoResponse, error)
	// UpdateAbout 更新关于页面内容（部分更新）
	UpdateAbout(req *request.AboutUpdateRequest) error
}
