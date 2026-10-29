package llmconfig

import (
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/pkg/llmclient"
)

// SchemaEnsurer 向量表初始化器。
//
// 定义在 llmconfig 里而不是直接依赖 vectorschema：本服务只需要「按给定维度
// 确保表就绪」这一个能力，不需要知道 DDL、不需要 *gorm.DB。用最小接口注入，
// 依赖方向保持 llmconfig → 抽象 → vectorschema。
//
// 为什么不设计成无参的 Ensure()（内部读库取维度）：新建向量模型的流程里
// 新记录还没落库，此时读库只能读到旧模型或读不到，拿到的维度必然是错的。
// 维度必须由调用方显式传入，这也是它叫 EnsureFor 而不是 Ensure 的原因。
type SchemaEnsurer interface {
	EnsureFor(dim int) error
}

// Service 模型配置服务接口
type Service interface {
	// List 配置列表（密钥脱敏）
	List() ([]*response.LLMConfigResponse, error)
	// ListByType 按类型筛选（1=对话 2=向量，0=全部）
	ListByType(t int8) ([]*response.LLMConfigResponse, error)
	// Get 配置详情（密钥脱敏）
	Get(id uint) (*response.LLMConfigResponse, error)
	// Test 测试连接（不落库）。向量模型测 /embeddings 并回传实际维度
	Test(req *request.TestLLMConfigRequest) (*response.TestConnectionResponse, error)
	// Create 新建配置（校验通过才入库）
	Create(req *request.CreateLLMConfigRequest) (*response.LLMConfigResponse, error)
	// Update 更新配置（仅修改运行参数与状态，不测试连接）
	Update(id uint, req *request.UpdateLLMConfigRequest) error
	// Delete 删除配置
	Delete(id uint) error
	// Resolve 解析指定配置（必须是启用的；未找到或已禁用则报错）
	Resolve(id uint) (llmclient.Config, error)
	// ResolveEmbedding 解析当前启用的向量模型，并返回其维度
	ResolveEmbedding() (llmclient.Config, int, error)
}
