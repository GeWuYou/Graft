package contract

import "fmt"

// DeploymentExecutionMode 标识适配器在兼容运行目标上选择的执行方式。
type DeploymentExecutionMode string

const (
	// DeploymentExecutionModeCompose 表示使用 Compose 兼容执行器。
	DeploymentExecutionModeCompose DeploymentExecutionMode = "compose"
	// DeploymentExecutionModeDockerStack 表示 Compose 定义经 Docker Stack 部署到 Swarm。
	DeploymentExecutionModeDockerStack DeploymentExecutionMode = "docker-stack"
)

// DeploymentAdapter 描述一种定义格式与运行目标能力之间的显式边界。
// 它不承担运行时注册或动态插件发现，未来适配器由模块装配时直接注册。
type DeploymentAdapter interface {
	Kind() DeploymentAdapterKind
	ValidateDefinition(definition []byte) error
	ResolveExecutionMode(capabilities map[string]bool) (DeploymentExecutionMode, error)
}

// DeploymentAdapterRegistry 以小型显式注册表保存当前可用的部署适配器。
type DeploymentAdapterRegistry struct {
	adapters map[DeploymentAdapterKind]DeploymentAdapter
}

// NewDeploymentAdapterRegistry 创建注册表并拒绝重复或空适配器。
func NewDeploymentAdapterRegistry(adapters ...DeploymentAdapter) (*DeploymentAdapterRegistry, error) {
	registry := &DeploymentAdapterRegistry{adapters: make(map[DeploymentAdapterKind]DeploymentAdapter, len(adapters))}
	for _, adapter := range adapters {
		if adapter == nil || adapter.Kind() == "" {
			return nil, fmt.Errorf("deployment adapter is required")
		}
		if _, exists := registry.adapters[adapter.Kind()]; exists {
			return nil, fmt.Errorf("duplicate deployment adapter %q", adapter.Kind())
		}
		registry.adapters[adapter.Kind()] = adapter
	}
	return registry, nil
}

// Get 返回指定适配器；调用方必须显式处理未注册类型。
func (r *DeploymentAdapterRegistry) Get(kind DeploymentAdapterKind) (DeploymentAdapter, bool) {
	adapter, ok := r.adapters[kind]
	return adapter, ok
}

// ComposeDeploymentAdapter 提供 Compose 定义的最小语义边界。
type ComposeDeploymentAdapter struct{}

// Kind 返回 Compose 适配器的稳定类型。
func (ComposeDeploymentAdapter) Kind() DeploymentAdapterKind { return DeploymentAdapterKindCompose }

// ValidateDefinition 拒绝空定义；Compose 的深度解析仍由 Compose 工作区服务负责。
func (ComposeDeploymentAdapter) ValidateDefinition(definition []byte) error {
	if len(definition) == 0 {
		return fmt.Errorf("compose definition is required")
	}
	return nil
}

// ResolveExecutionMode 根据 Runtime Target capabilities 选择 Compose 或 Docker Stack 执行方式。
func (ComposeDeploymentAdapter) ResolveExecutionMode(capabilities map[string]bool) (DeploymentExecutionMode, error) {
	if capabilities["docker_stack_deploy"] {
		return DeploymentExecutionModeDockerStack, nil
	}
	if capabilities["compose_execution"] {
		return DeploymentExecutionModeCompose, nil
	}
	return "", fmt.Errorf("runtime target does not support Compose execution")
}
