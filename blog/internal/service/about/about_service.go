package about

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repoAbout "blog/internal/repository/about"
	repoUser "blog/internal/repository/user"
	"blog/pkg/errors"
	"blog/pkg/logger"
	minioPkg "blog/pkg/minio"

	"go.uber.org/zap"
)

// aboutService 关于页面服务实现
type aboutService struct {
	aboutRepo repoAbout.AboutRepository
	minio     *minioPkg.Client
	userRepo  repoUser.UserRepository
}

// NewAboutService 创建关于页面服务实例
func NewAboutService(
	aboutRepo repoAbout.AboutRepository,
	minio *minioPkg.Client,
	userRepo repoUser.UserRepository,
) AboutService {
	return &aboutService{
		aboutRepo: aboutRepo,
		minio:     minio,
		userRepo:  userRepo,
	}
}

// GetAboutInfo 获取关于页面完整信息
func (s *aboutService) GetAboutInfo() (*response.AboutInfoResponse, error) {
	// 获取管理员信息
	list, _, err := s.userRepo.ListByRole(1, 1, 1, "", nil, "", "")
	if err != nil {
		logger.Error("获取管理员信息失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取管理员信息失败", err)
	}
	if len(list) == 0 {
		return nil, errors.ErrNotFound
	}
	admin := list[0]

	// 获取关于页面配置
	about, err := s.aboutRepo.GetAbout()
	if err != nil {
		logger.Error("获取关于页面失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "获取关于页面失败", err)
	}

	resp := &response.AboutInfoResponse{
		AdminName:    admin.UserName,
		AvatarURL:    s.minio.GetFileURL(admin.AvatarURL),
		Email:        admin.Email,
		Introduction: admin.Introduction,
	}

	// 如果 about 记录存在，填充字段
	if about != nil {
		resp.TechStack = about.TechStack
		resp.MyStory = about.MyStory
		resp.Why = about.Why
		resp.Interest = about.Interest
		resp.GitHub = about.GitHub
		resp.CSDN = about.CSDN
	}

	return resp, nil
}

// UpdateAbout 更新关于页面内容（部分更新）
func (s *aboutService) UpdateAbout(req *request.AboutUpdateRequest) error {
	// 获取现有记录
	about, err := s.aboutRepo.GetAbout()
	if err != nil {
		logger.Error("获取关于页面失败", zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "获取关于页面失败", err)
	}

	// 如果记录不存在，创建一条默认记录
	if about == nil {
		about = &entity.About{}
		err = s.aboutRepo.CreateAbout(about)
		if err != nil {
			logger.Error("创建关于页面失败", zap.Error(err))
			return errors.NewWithErr(errors.CodeInternalError, "创建关于页面失败", err)
		}
		// 重新获取以得到完整的实体（包含 ID 等）
		about, err = s.aboutRepo.GetAbout()
		if err != nil {
			logger.Error("获取关于页面失败", zap.Error(err))
			return errors.NewWithErr(errors.CodeInternalError, "获取关于页面失败", err)
		}
	}

	// 部分更新：只更新请求中提供的字段
	if req.TechStack != nil {
		about.TechStack = *req.TechStack
	}
	if req.MyStory != nil {
		about.MyStory = *req.MyStory
	}
	if req.Why != nil {
		about.Why = *req.Why
	}
	if req.Interest != nil {
		about.Interest = *req.Interest
	}
	if req.GitHub != nil {
		about.GitHub = *req.GitHub
	}
	if req.CSDN != nil {
		about.CSDN = *req.CSDN
	}

	// 保存更新
	return s.aboutRepo.UpdateAbout(about)
}
