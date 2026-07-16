package moduleapi

import (
	"context"
	"time"
)

// MonitorEvidenceAvailability 描述审计事件是否可以附加由 monitor 模块拥有的证据。
type MonitorEvidenceAvailability string

const (
	// MonitorEvidenceAvailable 表示已为事件窗口解析出 monitor 证据。
	MonitorEvidenceAvailable MonitorEvidenceAvailability = "available"
	// MonitorEvidenceModuleDisabled 表示 monitor 模块已禁用。
	MonitorEvidenceModuleDisabled MonitorEvidenceAvailability = "module_disabled"
	// MonitorEvidenceNoMatch 表示没有 monitor 异常匹配有界事件上下文。
	MonitorEvidenceNoMatch MonitorEvidenceAvailability = "no_match"
	// MonitorEvidenceExpired 表示事件窗口早于 monitor 证据保留期限。
	MonitorEvidenceExpired MonitorEvidenceAvailability = "expired"
	// MonitorEvidenceCapabilityUnavailable 表示 monitor capability 当前无法提供证据。
	MonitorEvidenceCapabilityUnavailable MonitorEvidenceAvailability = "capability_unavailable"
)

// MonitorEvidenceLinkTimeWindow 描述证据链接覆盖的时间范围。
type MonitorEvidenceLinkTimeWindow struct {
	CreatedFrom time.Time
	CreatedTo   time.Time
}

// MonitorAuditEvidenceContext 将证据链接限定到相关审计检索维度。
type MonitorAuditEvidenceContext struct {
	Action       string
	ActionPrefix string
	Source       string
	ResourceType string
	ResourceID   string
	ResourceName string
	RequestID    string
	Result       string
	RiskLevel    string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}

// MonitorIncidentSeedLink 指向该事件对应的初始审计事件。
type MonitorIncidentSeedLink struct {
	EventID uint64
}

// MonitorEvidenceLink 描述审计事件对应的一个 monitor-owned 下钻目标。
type MonitorEvidenceLink struct {
	TargetKind   string
	LinkState    string
	TitleKey     string
	Title        string
	Reason       string
	TimeWindow   *MonitorEvidenceLinkTimeWindow
	AuditContext *MonitorAuditEvidenceContext
	IncidentSeed *MonitorIncidentSeedLink
}

// ResolveAuditIncidentMonitorEvidenceInput 携带向 monitor 暴露的有界事件上下文。
type ResolveAuditIncidentMonitorEvidenceInput struct {
	IncidentSeedEventID uint64
	IncidentStartedAt   time.Time
	IncidentEndedAt     time.Time
	RequestID           string
	ResourceType        string
	ResourceID          string
	ResourceName        string
	AuditSource         string
	AuditResult         string
	AuditRiskLevel      string
}

// ResolvedAuditIncidentMonitorEvidence 是 monitor capability 针对审计事件返回的结果。
type ResolvedAuditIncidentMonitorEvidence struct {
	Availability  MonitorEvidenceAvailability
	Summary       string
	Reason        string
	AnomalyKey    string
	ScopeKind     string
	ScopeRef      string
	ObservedAt    *time.Time
	EvidenceLinks []MonitorEvidenceLink
}

// MonitorIncidentEvidenceService 为审计事件解析由 monitor 模块拥有的异常证据。
type MonitorIncidentEvidenceService interface {
	ResolveAuditIncidentMonitorEvidence(
		ctx context.Context,
		input ResolveAuditIncidentMonitorEvidenceInput,
	) (ResolvedAuditIncidentMonitorEvidence, error)
}
