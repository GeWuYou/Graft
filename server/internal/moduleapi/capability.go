package moduleapi

import (
	"context"
	"time"
)

// CapabilityImpact describes the blast radius of an unavailable capability.
type CapabilityImpact string

const (
	// CapabilityImpactPlatform 表示平台不可用会影响整体控制面。
	CapabilityImpactPlatform CapabilityImpact = "platform"
	// CapabilityImpactFeature 表示单项功能不可用但平台仍可使用。
	CapabilityImpactFeature CapabilityImpact = "feature"
	// CapabilityImpactAdvisory 表示仅供提示，不改变可用性。
	CapabilityImpactAdvisory CapabilityImpact = "advisory"
)

// CapabilityCategory groups capabilities for discovery and presentation.
type CapabilityCategory string

const (
	CapabilityCategoryInfrastructure CapabilityCategory = "infrastructure"
	CapabilityCategoryRuntime        CapabilityCategory = "runtime"
	CapabilityCategoryStorage        CapabilityCategory = "storage"
	CapabilityCategoryIntegration    CapabilityCategory = "integration"
	CapabilityCategorySecurity       CapabilityCategory = "security"
	CapabilityCategoryObservability  CapabilityCategory = "observability"
	CapabilityCategoryPlatform       CapabilityCategory = "platform"
	CapabilityCategoryAI             CapabilityCategory = "ai"
	CapabilityCategoryExtension      CapabilityCategory = "extension"
)

// CapabilityDescriptor is the static declaration of a capability.
type CapabilityDescriptor struct {
	Key        string             `json:"key"`
	Category   CapabilityCategory `json:"category"`
	Impact     CapabilityImpact   `json:"impact"`
	StaleAfter time.Duration      `json:"-"`
}

// CapabilityStatus is the coordinator's normalized observation state.
type CapabilityStatus string

const (
	// CapabilityStatusUnknown 表示尚无可信观测，或观测已过期。
	CapabilityStatusUnknown CapabilityStatus = "unknown"
	// CapabilityStatusChecking 表示正在执行一次短暂的健康检查。
	CapabilityStatusChecking CapabilityStatus = "checking"
	// CapabilityStatusHealthy 表示最近一次观测满足能力要求。
	CapabilityStatusHealthy CapabilityStatus = "healthy"
	// CapabilityStatusDegraded 表示能力可用但存在受限条件。
	CapabilityStatusDegraded CapabilityStatus = "degraded"
	// CapabilityStatusUnavailable 表示 provider 明确报告能力不可用。
	CapabilityStatusUnavailable CapabilityStatus = "unavailable"
	// CapabilityStatusDisabled 表示能力被配置或模块显式关闭。
	CapabilityStatusDisabled CapabilityStatus = "disabled"
	// CapabilityStatusUnsupported 表示当前构建没有该能力的实现。
	CapabilityStatusUnsupported CapabilityStatus = "unsupported"
)

// CapabilityObservation is a provider result. The coordinator fills freshness fields.
type CapabilityObservation struct {
	Status     CapabilityStatus `json:"status"`
	Summary    string           `json:"summary,omitempty"`
	ObservedAt time.Time        `json:"observed_at"`
	ExpiresAt  time.Time        `json:"expires_at"`
	Stale      bool             `json:"stale"`
}

// CapabilityProvider supplies observations for one declared capability.
type CapabilityProvider interface {
	Observe(context.Context) (CapabilityObservation, error)
}

// CapabilityObservationSource 允许模块把已有健康事实投影为能力观测；不暴露模块内部存储。
type CapabilityObservationSource interface {
	ObserveCapability(context.Context) (CapabilityObservation, error)
}

// Category 是能力分类的稳定契约别名。
type Category = CapabilityCategory

// Descriptor 是能力静态描述的稳定契约别名。
type Descriptor = CapabilityDescriptor

// Status 是能力状态的稳定契约别名。
type Status = CapabilityStatus

// Observation 是能力观测结果的稳定契约别名。
type Observation = CapabilityObservation

// Provider 是能力观测 provider 的稳定契约别名。
type Provider = CapabilityProvider
