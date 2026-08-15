package module

import "reflect"

// CapabilityKey 标识组合声明中的 typed capability contract。
//
// key 由 Go contract 类型派生，不使用字符串 service name；它只用于元数据，
// 实际解析仍沿用现有显式 container 边界。
type CapabilityKey struct {
	// Type 是 capability contract 的 Go 类型，作为显式容器 key 的 authority。
	Type reflect.Type
}

// TypedCapability 根据 typed capability contract 返回声明 key。
func TypedCapability[T any]() CapabilityKey {
	return CapabilityKey{Type: reflect.TypeOf((*T)(nil)).Elem()}
}

// CapabilityDeclaration 记录一个 required 或 exposed typed capability。
type CapabilityDeclaration struct {
	// Key 保存 required/exposed capability 的 typed contract 标识。
	Key CapabilityKey
}

// ResourceDeclaration 记录组合单元拥有的长期资源；Cleanup 只指向 owner 已有的
// disposer，不创建新的生命周期 API。
type ResourceDeclaration struct {
	// Name 是 owner 可追踪的资源名称。
	Name string
	// Owner 是负责资源存活、失败观测和最终清理的 composition unit。
	Owner string
	// Cleanup 指向 owner 已有的幂等 disposer 名称。
	Cleanup string
}

// CapabilityHealth 表示外部或可选组合单元的 capability-local 观察状态，
// 不代表应用整体可用性。
type CapabilityHealth string

const (
	// CapabilityHealthReady 表示 capability 可按其 contract 提供服务。
	CapabilityHealthReady CapabilityHealth = "ready"
	// CapabilityHealthDegraded 表示 capability 可用但存在局部退化。
	CapabilityHealthDegraded CapabilityHealth = "degraded"
	// CapabilityHealthUnavailable 表示 capability 当前不可用。
	CapabilityHealthUnavailable CapabilityHealth = "unavailable"
)
