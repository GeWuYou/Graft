// Package store defines audit-module-owned persistence contracts.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrAuditLogNotFound indicates that the requested audit evidence record does not exist.
	ErrAuditLogNotFound = errors.New("audit log not found")
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
	// AuditSourceDomainEvent marks module-published business events.
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

// AuditVisibilityStrategy describes how one audit event should be persisted and exposed.
type AuditVisibilityStrategy string

const (
	// AuditVisibilityStrategyVisible persists the event and includes it in default read surfaces.
	AuditVisibilityStrategyVisible AuditVisibilityStrategy = "visible"
	// AuditVisibilityStrategyHidden persists the event but excludes it from default read surfaces.
	AuditVisibilityStrategyHidden AuditVisibilityStrategy = "hidden"
	// AuditVisibilityStrategyIgnore drops the event before persistence.
	AuditVisibilityStrategyIgnore AuditVisibilityStrategy = "ignore"
)

// AuditVisibilityScope describes which visibility slice a list query wants to read.
type AuditVisibilityScope string

const (
	// AuditVisibilityScopeDefault returns only default-visible audit records.
	AuditVisibilityScopeDefault AuditVisibilityScope = "default"
	// AuditVisibilityScopeAll returns both visible and hidden records.
	AuditVisibilityScopeAll AuditVisibilityScope = "all"
	// AuditVisibilityScopeHiddenOnly returns only hidden records.
	AuditVisibilityScopeHiddenOnly AuditVisibilityScope = "hidden_only"
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

// AuditBusinessCategory identifies backend-owned editable business semantics for audit list filtering.
type AuditBusinessCategory string

const (
	// AuditBusinessCategoryFailedOperations represents failed operations in the active window.
	AuditBusinessCategoryFailedOperations AuditBusinessCategory = "failed_operations"
	// AuditBusinessCategoryHighRiskOperations represents high-risk operations in the active window.
	AuditBusinessCategoryHighRiskOperations AuditBusinessCategory = "high_risk_operations"
	// AuditBusinessCategorySensitiveOperations represents sensitive operations in the active window.
	AuditBusinessCategorySensitiveOperations AuditBusinessCategory = "sensitive_operations"
	// AuditBusinessCategoryAuthFailures represents authentication failures in the active window.
	AuditBusinessCategoryAuthFailures AuditBusinessCategory = "auth_failures"
	// AuditBusinessCategoryPermissionDenials represents permission denial activity in the active window.
	AuditBusinessCategoryPermissionDenials AuditBusinessCategory = "permission_denials"
	// AuditBusinessCategoryRBACChanges represents RBAC and permission-configuration changes in the active window.
	AuditBusinessCategoryRBACChanges AuditBusinessCategory = "rbac_changes"
	// AuditBusinessCategoryCriticalSecurity represents critical security activity in the active window.
	AuditBusinessCategoryCriticalSecurity AuditBusinessCategory = "critical_security"
)

// AuditLog is the audit module's stable DTO for a persisted audit record.
type AuditLog struct {
	ID               uint64
	Source           AuditSource
	Visibility       AuditVisibilityStrategy
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
	Visibility       AuditVisibilityStrategy
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
	Visibility       AuditVisibilityStrategy
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

// AuditPolicyRule is the module-owned persistence DTO for one policy rule.
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

// AuditVisibilityDefault stores the module-owned global default strategy for audit events.
type AuditVisibilityDefault struct {
	Key           string
	Strategy      AuditVisibilityStrategy
	UpdatedAt     time.Time
	UpdatedBy     *uint64
	UpdatedByName string
}

// AuditVisibilityActor records the operator metadata attached to a visibility policy write.
type AuditVisibilityActor struct {
	UserID   *uint64
	Username string
}

// AuditVisibilityOverride stores one source+action visibility override owned by the audit module.
type AuditVisibilityOverride struct {
	ID            uint64
	Source        AuditSource
	ActionKey     string
	Strategy      AuditVisibilityStrategy
	Description   string
	CreatedAt     time.Time
	CreatedBy     *uint64
	CreatedByName string
	UpdatedAt     time.Time
	UpdatedBy     *uint64
	UpdatedByName string
}

// UpsertAuditVisibilityOverrideInput describes one audit-owned source+action visibility override mutation.
type UpsertAuditVisibilityOverrideInput struct {
	Source      AuditSource
	ActionKey   string
	Strategy    AuditVisibilityStrategy
	Description string
	Actor       AuditVisibilityActor
}

// AuditEventCatalogItem describes one selectable event/action in the audit strategy editor.
type AuditEventCatalogItem struct {
	Source            AuditSource
	ActionKey         string
	DisplayName       string
	Description       string
	Category          string
	DefaultStrategy   AuditVisibilityStrategy
	EffectiveStrategy AuditVisibilityStrategy
	Overridden        bool
}

// AuditVisibilityPolicySnapshot is the audit-owned read model for strategy management.
type AuditVisibilityPolicySnapshot struct {
	Default   AuditVisibilityDefault
	Overrides []AuditVisibilityOverride
	Catalog   []AuditEventCatalogItem
}

// ListAuditLogsQuery describes the audit module's stable repository-side query contract.
type ListAuditLogsQuery struct {
	ActorUserID         *uint64
	Keyword             string
	Actor               string
	Action              string
	ActionPrefix        string
	ActionPrefixes      []string
	ActionKeywords      []string
	TimePreset          AuditTimePreset
	Source              AuditSource
	BusinessCategory    AuditBusinessCategory
	ResourceType        string
	ResourceTypes       []string
	ResourceID          string
	ResourceName        string
	RequestPathPrefixes []string
	Success             *bool
	SessionID           string
	RequestID           string
	VisibilityScope     AuditVisibilityScope
	Result              AuditResult
	Results             []AuditResult
	RiskLevel           AuditRiskLevel
	RiskLevels          []AuditRiskLevel
	CreatedFrom         *time.Time
	CreatedTo           *time.Time
	Sorts               []string
	Limit               int
	Offset              int
}

// ListAuditLogsResult returns a bounded page plus total count for future API pagination.
type ListAuditLogsResult struct {
	Items []AuditLog
	Total int
}

// AuditTimePreset identifies the supported relative time preset.
type AuditTimePreset string

const (
	// AuditTimePresetLast24Hours selects the trailing 24-hour window.
	AuditTimePresetLast24Hours AuditTimePreset = "last_24h"
	// AuditTimePresetLast7Days selects the trailing 7-day window.
	AuditTimePresetLast7Days AuditTimePreset = "last_7d"
	// AuditTimePresetLast30Days selects the trailing 30-day window.
	AuditTimePresetLast30Days AuditTimePreset = "last_30d"
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
	TimePreset       AuditTimePreset
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

// AuditIncident is the canonical incident drilldown payload owned by the audit module.
type AuditIncident struct {
	SeedEvent        AuditLog
	Incident         AuditIncidentSummary
	RelatedEvents    []AuditLog
	RelatedActors    []AuditIncidentActor
	RelatedResources []AuditIncidentResource
	RelatedRequests  []AuditIncidentRequest
	MonitorContext   AuditIncidentMonitorContext
}

// AuditRepository exposes the audit module's persistence contract.
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, input CreateAuditLogInput) (AuditLog, error)
	ListAuditLogs(ctx context.Context, query ListAuditLogsQuery) (ListAuditLogsResult, error)
	ReadAuditLog(ctx context.Context, id uint64) (AuditLog, error)
	ReadAuditOverview(ctx context.Context, preset AuditTimePreset) (AuditOverview, error)
	ReadIncident(ctx context.Context, eventID uint64) (AuditIncident, error)
	ListAuditPolicyRules(ctx context.Context) ([]AuditPolicyRule, error)
	GetAuditVisibilityDefault(ctx context.Context, key string) (AuditVisibilityDefault, error)
	UpsertAuditVisibilityDefault(
		ctx context.Context,
		key string,
		strategy AuditVisibilityStrategy,
		userID *uint64,
		username string,
	) (AuditVisibilityDefault, error)
	ListAuditVisibilityOverrides(ctx context.Context) ([]AuditVisibilityOverride, error)
	UpsertAuditVisibilityOverride(ctx context.Context, input UpsertAuditVisibilityOverrideInput) (AuditVisibilityOverride, error)
	DeleteAuditVisibilityOverride(ctx context.Context, source AuditSource, actionKey string) error
	DeleteAuditLogsBefore(ctx context.Context, createdBefore time.Time) (int64, error)
}
