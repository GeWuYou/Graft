// Package store defines audit-plugin-owned persistence contracts.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrIncidentNotFound indicates that the requested audit-owned incident seed does not exist.
	ErrIncidentNotFound = errors.New("audit incident not found")
)

// AuditSource identifies where one audit candidate originated.
type AuditSource string

const (
	// AuditSourceRequest marks request-derived candidates.
	AuditSourceRequest AuditSource = "REQUEST"
	// AuditSourceSecurityEvent marks auth/authz security-event candidates.
	AuditSourceSecurityEvent AuditSource = "SECURITY_EVENT"
	// AuditSourceDomainEvent marks plugin-published business events.
	AuditSourceDomainEvent AuditSource = "DOMAIN_EVENT"
)

// AuditPolicyEffect describes the final effect of one policy rule.
type AuditPolicyEffect string

const (
	// AuditPolicyEffectInclude writes a candidate into the audit log.
	AuditPolicyEffectInclude AuditPolicyEffect = "include"
	// AuditPolicyEffectExclude drops a candidate before audit persistence.
	AuditPolicyEffectExclude AuditPolicyEffect = "exclude"
)

// AuditPolicyMatchType describes the route/event match mode supported in MVP.
type AuditPolicyMatchType string

const (
	// AuditPolicyMatchTypeExact requires an exact match.
	AuditPolicyMatchTypeExact AuditPolicyMatchType = "exact"
	// AuditPolicyMatchTypePrefix requires a prefix match.
	AuditPolicyMatchTypePrefix AuditPolicyMatchType = "prefix"
)

// AuditRiskLevel classifies the relative severity of one audit event.
type AuditRiskLevel string

const (
	// AuditRiskLevelLow marks routine low-risk audit activity.
	AuditRiskLevelLow AuditRiskLevel = "LOW"
	// AuditRiskLevelMedium marks elevated audit activity that still needs operator review.
	AuditRiskLevelMedium AuditRiskLevel = "MEDIUM"
	// AuditRiskLevelHigh marks high-risk audit activity.
	AuditRiskLevelHigh AuditRiskLevel = "HIGH"
	// AuditRiskLevelCritical marks critical audit activity that needs urgent review.
	AuditRiskLevelCritical AuditRiskLevel = "CRITICAL"
)

// AuditResult normalizes the outcome of one audit event.
type AuditResult string

const (
	// AuditResultSuccess marks successful audit activity.
	AuditResultSuccess AuditResult = "SUCCESS"
	// AuditResultFailed marks a failed operation without an explicit deny or system error.
	AuditResultFailed AuditResult = "FAILED"
	// AuditResultDenied marks operations rejected by authorization.
	AuditResultDenied AuditResult = "DENIED"
	// AuditResultError marks operations that failed because of system-level errors.
	AuditResultError AuditResult = "ERROR"
)

// AuditLog is the audit plugin's stable DTO for a persisted audit record.
type AuditLog struct {
	ID               uint64
	Source           AuditSource
	ActorUserID      *uint64
	ActorUsername    string
	ActorDisplayName string
	Action           string
	ResourceType     string
	ResourceID       string
	ResourceName     string
	Success          bool
	RequestID        string
	IP               string
	UserAgent        string
	Message          string
	Metadata         json.RawMessage
	Result           AuditResult
	RiskLevel        AuditRiskLevel
	Target           AuditTarget
	TargetType       string
	TargetLabel      string
	TraceID          string
	SessionID        string
	RequestMethod    string
	RequestPath      string
	StatusCode       int
	CreatedAt        time.Time
}

// AuditTarget is the canonical typed target exposed by audit read models.
type AuditTarget struct {
	Kind     string
	Type     string
	ID       string
	Label    string
	RouteRef string
}

// CreateAuditLogInput describes the minimum fields required to persist an audit record.
type CreateAuditLogInput struct {
	ActorUserID      *uint64
	ActorUsername    string
	ActorDisplayName string
	Action           string
	ResourceType     string
	ResourceID       string
	ResourceName     string
	Success          bool
	RequestID        string
	IP               string
	UserAgent        string
	Message          string
	Metadata         json.RawMessage
	CreatedAt        time.Time
}

// AuditCandidate is the normalized input evaluated before one audit record is written.
type AuditCandidate struct {
	Source           AuditSource
	ActorUserID      *uint64
	ActorUsername    string
	ActorDisplayName string
	Action           string
	ResourceType     string
	ResourceID       string
	ResourceName     string
	TargetType       string
	EventType        string
	RequestMethod    string
	RequestPath      string
	StatusCode       int
	RequestID        string
	TraceID          string
	SessionID        string
	IP               string
	UserAgent        string
	Success          bool
	Message          string
	Metadata         json.RawMessage
	CreatedAt        time.Time
}

// AuditPolicyRule is the plugin-owned persistence DTO for one policy rule.
type AuditPolicyRule struct {
	ID            uint64
	Name          string
	Description   string
	Source        AuditSource
	Enabled       bool
	Priority      int
	Effect        AuditPolicyEffect
	MatchType     AuditPolicyMatchType
	Method        string
	PathPattern   string
	EventType     string
	RiskLevel     AuditRiskLevel
	TargetType    string
	ConditionExpr string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AuditPolicyDecision is the stable result returned by the evaluator.
type AuditPolicyDecision struct {
	Matched bool
	Allowed bool
	Rule    *AuditPolicyRule
}

// ListAuditLogsQuery describes the audit plugin's stable repository-side query contract.
type ListAuditLogsQuery struct {
	ActorUserID  *uint64
	Action       string
	ActionPrefix string
	Source       AuditSource
	ResourceType string
	ResourceID   string
	ResourceName string
	Success      *bool
	RequestID    string
	Result       AuditResult
	RiskLevel    AuditRiskLevel
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	Limit        int
	Offset       int
}

// ListAuditLogsResult returns a bounded page plus total count for future API pagination.
type ListAuditLogsResult struct {
	Items []AuditLog
	Total int
}

// OverviewWindow identifies the supported overview aggregation window.
type OverviewWindow string

const (
	// OverviewWindow24Hours selects the trailing 24-hour overview window.
	OverviewWindow24Hours OverviewWindow = "24h"
	// OverviewWindow7Days selects the trailing 7-day overview window.
	OverviewWindow7Days OverviewWindow = "7d"
	// OverviewWindow30Days selects the trailing 30-day overview window.
	OverviewWindow30Days OverviewWindow = "30d"
)

// OverviewSummary aggregates audit activity counts for the selected window.
type OverviewSummary struct {
	TotalLogs           int
	FailedOperations    int
	HighRiskEvents      int
	SensitiveOperations int
}

// OverviewItem is one recent event preview shown in the overview workbench.
type OverviewItem struct {
	ID               uint64
	Source           AuditSource
	ActorUserID      *uint64
	ActorUsername    string
	ActorDisplayName string
	Action           string
	ResourceType     string
	ResourceID       string
	ResourceName     string
	Success          bool
	RequestID        string
	Message          string
	Metadata         json.RawMessage
	CreatedAt        time.Time
}

// OverviewRiskGroup is one bounded backend-owned risk grouping summary.
type OverviewRiskGroup struct {
	Key       string
	LabelKey  string
	Count     int
	RiskLevel AuditRiskLevel
}

// OverviewTrendPoint is one server-computed bucket in the overview trend series.
type OverviewTrendPoint struct {
	BucketStart    time.Time
	BucketEnd      time.Time
	Total          int
	Failed         int
	HighRisk       int
	SecurityEvents int
}

// OverviewTrend describes the fixed-bucket trend shape for the selected window.
type OverviewTrend struct {
	BucketUnit string
	BucketSize int
	Points     []OverviewTrendPoint
}

// OverviewSecurityTimelineItem is one bounded recent security event preview.
type OverviewSecurityTimelineItem struct {
	ID               uint64
	CreatedAt        time.Time
	Source           AuditSource
	RiskLevel        AuditRiskLevel
	Action           string
	Result           AuditResult
	RequestID        string
	ActorDisplayName string
	ActorUsername    string
	ResourceName     string
	ResourceType     string
}

// AuditOverview groups window-level counters with the recent slices used by the overview page.
type AuditOverview struct {
	Window           OverviewWindow
	Summary          OverviewSummary
	RiskGroups       []OverviewRiskGroup
	Trend            OverviewTrend
	SecurityTimeline []OverviewSecurityTimelineItem
	FailedAuth       []OverviewItem
	PermissionDenied []OverviewItem
	SensitiveOps     []OverviewItem
}

// IncidentSeed identifies one stable audit-owned incident entrypoint.
type IncidentSeed struct {
	EventID uint64
}

// AuditIncidentSummary describes the aggregate incident computed from one seed event.
type AuditIncidentSummary struct {
	IncidentKey       string
	Title             string
	Summary           string
	RiskLevel         AuditRiskLevel
	StartedAt         time.Time
	EndedAt           time.Time
	CorrelationReason string
}

// AuditIncidentActor aggregates one related actor inside the bounded incident context.
type AuditIncidentActor struct {
	ActorUserID      *uint64
	ActorUsername    string
	ActorDisplayName string
	EventCount       int
}

// AuditIncidentResource aggregates one related resource inside the bounded incident context.
type AuditIncidentResource struct {
	ResourceType string
	ResourceID   string
	ResourceName string
	EventCount   int
}

// AuditIncidentRequest aggregates one related request inside the bounded incident context.
type AuditIncidentRequest struct {
	RequestID  string
	EventCount int
	StartedAt  time.Time
	EndedAt    time.Time
}

// MonitorContextState records whether bounded monitor participation is available to the incident read model.
type MonitorContextState string

const (
	// MonitorContextStateAvailable indicates monitor participation is attached and current.
	MonitorContextStateAvailable MonitorContextState = "available"
	// MonitorContextStatePartial indicates monitor participation is only partially available.
	MonitorContextStatePartial MonitorContextState = "partial"
	// MonitorContextStateUnavailable indicates the incident cannot attach monitor participation canonically.
	MonitorContextStateUnavailable MonitorContextState = "unavailable"
)

// AuditIncidentMonitorContext returns the bounded monitor participation state attached to one incident.
type AuditIncidentMonitorContext struct {
	State         MonitorContextState
	Summary       string
	Reason        string
	AnomalyKey    string
	ScopeKind     string
	ScopeRef      string
	ObservedAt    *time.Time
	EvidenceLinks []EvidenceLink
}

// EvidenceLinkTimeWindow keeps canonical bounded evidence timing for drilldown links.
type EvidenceLinkTimeWindow struct {
	CreatedFrom time.Time
	CreatedTo   time.Time
}

// AuditEvidenceContext points consumers at canonical audit evidence filters.
type AuditEvidenceContext struct {
	Action       string
	ActionPrefix string
	Source       AuditSource
	ResourceType string
	ResourceID   string
	ResourceName string
	RequestID    string
	Result       AuditResult
	RiskLevel    AuditRiskLevel
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}

// IncidentSeedLink points at one stable audit incident seed event.
type IncidentSeedLink struct {
	EventID uint64
}

// EvidenceLink is the canonical cross-surface evidence link DTO reused by audit and monitor.
type EvidenceLink struct {
	TargetKind   string
	LinkState    string
	Title        string
	Reason       string
	TimeWindow   *EvidenceLinkTimeWindow
	AuditContext *AuditEvidenceContext
	IncidentSeed *IncidentSeedLink
}

// AuditIncident is the canonical incident drilldown payload owned by the audit plugin.
type AuditIncident struct {
	SeedEvent        AuditLog
	Incident         AuditIncidentSummary
	RelatedEvents    []AuditLog
	RelatedActors    []AuditIncidentActor
	RelatedResources []AuditIncidentResource
	RelatedRequests  []AuditIncidentRequest
	MonitorContext   AuditIncidentMonitorContext
}

// AuditRepository exposes the audit plugin's persistence contract.
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, input CreateAuditLogInput) (AuditLog, error)
	ListAuditLogs(ctx context.Context, query ListAuditLogsQuery) (ListAuditLogsResult, error)
	ReadAuditOverview(ctx context.Context, window OverviewWindow) (AuditOverview, error)
	ReadIncident(ctx context.Context, eventID uint64) (AuditIncident, error)
	ListAuditPolicyRules(ctx context.Context) ([]AuditPolicyRule, error)
}
