package about

import (
	"blog/internal/model/entity"
)

// AboutRepository 关于页面数据访问接口
type AboutRepository interface {
	// GetAbout 获取关于页面配置（单例）
	GetAbout() (*entity.About, error)
	// UpdateAbout 更新关于页面配置（部分更新）
	UpdateAbout(about *entity.About) error
	// CreateAbout 创建关于页面配置（单例）
	CreateAbout(about *entity.About) error
}
