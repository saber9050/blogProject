package llmconfig

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/pkg/llmclient"
)

// Service 模型配置服务接口
type Service interface {
	// List 配置列表（密钥脱敏）
	List() ([]*response.LLMConfigResponse, error)
	// Get 配置详情（密钥脱敏）
	Get(id uint) (*response.LLMConfigResponse, error)
	// Test 测试连接（不落库）
	Test(req *request.TestLLMConfigRequest) (*response.TestConnectionResponse, error)
	// Create 新建配置（内部复测，未通过则拒绝入库）
	Create(req *request.CreateLLMConfigRequest) (*response.LLMConfigResponse, error)
	// Update 更新配置（内部复测）
	Update(id uint, req *request.UpdateLLMConfigRequest) error
	// Delete 删除配置
	Delete(id uint) error
	// Activate 设为当前使用
	Activate(id uint) error
	// ResolveActive 解析当前生效配置（库→兜底 config.yaml）
	ResolveActive() (llmclient.Config, uint, error)
}
