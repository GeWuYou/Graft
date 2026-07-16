// Package store 定义审计模块拥有的持久化契约、审计事实 DTO 和证据读取模型。
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrAuditLogNotFound 表示请求的审计证据记录不存在。
	ErrAuditLogNotFound = errors.New("audit log not found")
	// ErrIncidentNotFound 表示审计模块拥有的事件种子不存在。
	ErrIncidentNotFound = errors.New("audit incident not found")
	// ErrAuditValidation 表示服务边界收到审计模块判定为无效的输入。
	ErrAuditValidation = errors.New("audit validation failed")
)

// AuditSource 标识审计候选记录的来源边界。
type AuditSource string

const (
	// AuditSourceRequest 表示由 HTTP 请求派生的候选记录。
	AuditSourceRequest AuditSource = "REQUEST"
	// AuditSourceSecurityEvent 表示由认证或授权安全事件派生的候选记录。
	AuditSourceSecurityEvent AuditSource = "SECURITY_EVENT"
	// AuditSourceDomainEvent 表示由业务模块发布的领域事件派生的候选记录。
	AuditSourceDomainEvent AuditSource = "DOMAIN_EVENT"
)

// AuditPolicyEffect 描述策略规则对候选记录产生的最终效果。
type AuditPolicyEffect string

const (
	// AuditPolicyEffectInclude 表示候选记录通过策略并落入审计日志。
	AuditPolicyEffectInclude AuditPolicyEffect = "include"
	// AuditPolicyEffectExclude 表示候选记录在持久化前被策略丢弃。
	AuditPolicyEffectExclude AuditPolicyEffect = "exclude"
)

// AuditVisibilityStrategy 描述审计事件的持久化方式和默认读面可见性。
type AuditVisibilityStrategy string

const (
	// AuditVisibilityStrategyVisible 持久化事件，并纳入默认审计读取面。
	AuditVisibilityStrategyVisible AuditVisibilityStrategy = "visible"
	// AuditVisibilityStrategyHidden 持久化事件，但从默认审计读取面排除，保留供授权调查使用。
	AuditVisibilityStrategyHidden AuditVisibilityStrategy = "hidden"
	// AuditVisibilityStrategyIgnore 在持久化前丢弃事件，不形成审计证据记录。
	AuditVisibilityStrategyIgnore AuditVisibilityStrategy = "ignore"
)

// AuditVisibilityScope 描述列表查询要读取的审计可见性范围。
type AuditVisibilityScope string

const (
	// AuditVisibilityScopeDefault 只返回默认可见的审计记录。
	AuditVisibilityScopeDefault AuditVisibilityScope = "default"
	// AuditVisibilityScopeAll 返回可见和隐藏的审计记录，通常需要管理权限。
	AuditVisibilityScopeAll AuditVisibilityScope = "all"
	// AuditVisibilityScopeHiddenOnly 只返回隐藏的审计记录，通常需要管理权限。
	AuditVisibilityScopeHiddenOnly AuditVisibilityScope = "hidden_only"
)

// AuditPolicyMatchType 描述 MVP 支持的路由或事件匹配方式。
type AuditPolicyMatchType string

const (
	// AuditPolicyMatchTypeExact 要求来源和动作完全匹配。
	AuditPolicyMatchTypeExact AuditPolicyMatchType = "exact"
	// AuditPolicyMatchTypePrefix 要求动作以前缀匹配。
	AuditPolicyMatchTypePrefix AuditPolicyMatchType = "prefix"
)

// AuditRiskLevel 描述审计事件的相对风险等级。
type AuditRiskLevel string

const (
	// AuditRiskLevelLow 表示例行低风险审计活动。
	AuditRiskLevelLow AuditRiskLevel = "LOW"
	// AuditRiskLevelMedium 表示需要操作员复核的升高风险审计活动。
	AuditRiskLevelMedium AuditRiskLevel = "MEDIUM"
	// AuditRiskLevelHigh 表示高风险审计活动。
	AuditRiskLevelHigh AuditRiskLevel = "HIGH"
	// AuditRiskLevelCritical 表示需要紧急复核的关键审计活动。
	AuditRiskLevelCritical AuditRiskLevel = "CRITICAL"
)

// AuditResult 归一化审计事件的结果，供列表、概览和通知共用。
type AuditResult string

const (
	// AuditResultSuccess 表示审计活动成功。
	AuditResultSuccess AuditResult = "SUCCESS"
	// AuditResultFailed 表示操作失败，但不是明确拒绝或系统错误。
	AuditResultFailed AuditResult = "FAILED"
	// AuditResultDenied 表示操作被授权检查拒绝。
	AuditResultDenied AuditResult = "DENIED"
	// AuditResultError 表示操作因系统级错误失败。
	AuditResultError AuditResult = "ERROR"
)

// AuditBusinessCategory 标识由后端拥有、可编辑的审计列表业务分类。
type AuditBusinessCategory string

const (
	// AuditBusinessCategoryFailedOperations 表示当前时间窗口内的失败操作。
	AuditBusinessCategoryFailedOperations AuditBusinessCategory = "failed_operations"
	// AuditBusinessCategoryHighRiskOperations 表示当前时间窗口内的高风险操作。
	AuditBusinessCategoryHighRiskOperations AuditBusinessCategory = "high_risk_operations"
	// AuditBusinessCategorySensitiveOperations 表示当前时间窗口内的敏感操作。
	AuditBusinessCategorySensitiveOperations AuditBusinessCategory = "sensitive_operations"
	// AuditBusinessCategoryAuthFailures 表示当前时间窗口内的认证失败。
	AuditBusinessCategoryAuthFailures AuditBusinessCategory = "auth_failures"
	// AuditBusinessCategoryPermissionDenials 表示当前时间窗口内的权限拒绝活动。
	AuditBusinessCategoryPermissionDenials AuditBusinessCategory = "permission_denials"
	// AuditBusinessCategoryRBACChanges 表示当前时间窗口内的 RBAC 与权限配置变更。
	AuditBusinessCategoryRBACChanges AuditBusinessCategory = "rbac_changes"
	// AuditBusinessCategoryCriticalSecurity 表示当前时间窗口内的关键安全活动。
	AuditBusinessCategoryCriticalSecurity AuditBusinessCategory = "critical_security"
)

// AuditLog 是审计模块对外稳定的持久化审计记录 DTO；其 Metadata 保留脱敏后的调查上下文。
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

// AuditTarget 是审计读取模型对外暴露的规范化目标对象。
type AuditTarget struct {
	Kind     string
	Type     string
	ID       string
	Label    string
	RouteRef string
}

// CreateAuditLogInput 描述持久化审计事实所需的最小输入。
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

// AuditCandidate 是写入审计记录前经过来源和字段归一化、等待策略评估的候选事实。
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

// AuditPolicyRule 是审计模块拥有的单条策略规则持久化 DTO。
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

// AuditPolicyDecision 是策略评估器返回的稳定决策结果。
type AuditPolicyDecision struct {
	Matched bool
	Allowed bool
	Rule    *AuditPolicyRule
}

// AuditVisibilityDefault 保存审计事件的模块级全局默认策略。
type AuditVisibilityDefault struct {
	Key           string
	Strategy      AuditVisibilityStrategy
	UpdatedAt     time.Time
	UpdatedBy     *uint64
	UpdatedByName string
}

// AuditVisibilityActor 保存可见性策略写入时附带的操作者快照。
type AuditVisibilityActor struct {
	UserID   *uint64
	Username string
}

// AuditVisibilityOverride 保存审计模块拥有的单个来源加动作可见性覆盖规则。
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

// UpsertAuditVisibilityOverrideInput 描述一次由审计模块拥有的来源加动作可见性覆盖变更。
type UpsertAuditVisibilityOverrideInput struct {
	Source      AuditSource
	ActionKey   string
	Strategy    AuditVisibilityStrategy
	Description string
	Actor       AuditVisibilityActor
}

// AuditEventCatalogItem 描述策略编辑器中的一个可选事件或动作目录项。
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

// AuditVisibilityPolicySnapshot 是审计模块拥有的策略管理读取模型；目录项以最终覆盖策略为准。
type AuditVisibilityPolicySnapshot struct {
	Default   AuditVisibilityDefault
	Overrides []AuditVisibilityOverride
	Catalog   []AuditEventCatalogItem
}

// ListAuditLogsQuery 描述审计模块稳定的仓储查询契约；仓储不会隐式补时间范围。
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

// ListAuditLogsResult 返回有界分页结果及总数，供 API 层构造稳定分页响应。
type ListAuditLogsResult struct {
	Items []AuditLog
	Total int
}

// AuditTimePreset 标识审计查询支持的相对时间窗口。
type AuditTimePreset string

const (
	// AuditTimePresetLast24Hours selects the trailing 24-hour window.
	AuditTimePresetLast24Hours AuditTimePreset = "last_24h"
	// AuditTimePresetLast7Days selects the trailing 7-day window.
	AuditTimePresetLast7Days AuditTimePreset = "last_7d"
	// AuditTimePresetLast30Days selects the trailing 30-day window.
	AuditTimePresetLast30Days AuditTimePreset = "last_30d"
)

// OverviewSummary 汇总选定时间窗口内的审计活动数量。
type OverviewSummary struct {
	TotalLogs           int
	FailedOperations    int
	HighRiskEvents      int
	SensitiveOperations int
}

// OverviewItem 是概览工作台展示的一条近期事件摘要。
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

// OverviewRiskGroup 是后端按固定边界计算的一组风险聚合摘要。
type OverviewRiskGroup struct {
	Key       string
	LabelKey  string
	Count     int
	RiskLevel AuditRiskLevel
}

// OverviewTrendPoint 是概览趋势序列中的一个服务端计算桶。
type OverviewTrendPoint struct {
	BucketStart    time.Time
	BucketEnd      time.Time
	Total          int
	Failed         int
	HighRisk       int
	SecurityEvents int
}

// OverviewTrend 描述选定时间窗口的固定桶趋势结构。
type OverviewTrend struct {
	BucketUnit string
	BucketSize int
	Points     []OverviewTrendPoint
}

// OverviewSecurityTimelineItem 是受数量边界限制的近期安全事件摘要。
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

// AuditOverview 将窗口计数与概览页面使用的近期事件切片组合为读取模型。
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

// IncidentSeed 标识一个由审计模块拥有的稳定事故入口事件。
type IncidentSeed struct {
	EventID uint64
}

// AuditIncidentSummary 描述基于入口事件聚合得到的事故摘要。
type AuditIncidentSummary struct {
	IncidentKey       string
	Title             string
	Summary           string
	RiskLevel         AuditRiskLevel
	StartedAt         time.Time
	EndedAt           time.Time
	CorrelationReason string
}

// AuditIncidentActor 聚合受限事故上下文中的一个关联操作者。
type AuditIncidentActor struct {
	ActorUserID      *uint64
	ActorUsername    string
	ActorDisplayName string
	EventCount       int
}

// AuditIncidentResource 聚合受限事故上下文中的一个关联资源。
type AuditIncidentResource struct {
	ResourceType string
	ResourceID   string
	ResourceName string
	EventCount   int
}

// AuditIncidentRequest 聚合受限事故上下文中的一个关联请求。
type AuditIncidentRequest struct {
	RequestID  string
	EventCount int
	StartedAt  time.Time
	EndedAt    time.Time
}

// MonitorContextState 记录事故读取模型是否获得受限的 monitor 参与证据。
type MonitorContextState string

const (
	// MonitorContextStateAvailable 表示已附加且最新的 monitor 参与信息。
	MonitorContextStateAvailable MonitorContextState = "available"
	// MonitorContextStatePartial 表示 monitor 参与信息仅部分可用。
	MonitorContextStatePartial MonitorContextState = "partial"
	// MonitorContextStateUnavailable 表示事件无法按规范附加 monitor 参与信息。
	MonitorContextStateUnavailable MonitorContextState = "unavailable"
)

// AuditIncidentMonitorContext 保存附加到事故上的受限 monitor 参与状态。
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

// EvidenceLinkTimeWindow 保存下钻链接使用的规范化、受限证据时间范围。
type EvidenceLinkTimeWindow struct {
	CreatedFrom time.Time
	CreatedTo   time.Time
}

// AuditEvidenceContext 将消费者指向规范化的审计证据筛选条件。
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

// IncidentSeedLink 指向一个稳定的审计事故入口事件。
type IncidentSeedLink struct {
	EventID uint64
}

// EvidenceLink 是 audit 与 monitor 复用的规范化跨页面证据链接 DTO。
type EvidenceLink struct {
	TargetKind   string
	LinkState    string
	Title        string
	Reason       string
	TimeWindow   *EvidenceLinkTimeWindow
	AuditContext *AuditEvidenceContext
	IncidentSeed *IncidentSeedLink
}

// AuditIncident 是审计模块拥有的事故下钻规范载荷；monitor 只能作为受限证据参与其中。
type AuditIncident struct {
	SeedEvent        AuditLog
	Incident         AuditIncidentSummary
	RelatedEvents    []AuditLog
	RelatedActors    []AuditIncidentActor
	RelatedResources []AuditIncidentResource
	RelatedRequests  []AuditIncidentRequest
	MonitorContext   AuditIncidentMonitorContext
}

// AuditRepository 暴露审计模块的持久化契约，调用方不得绕过它直接依赖审计表结构。
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, input CreateAuditLogInput) (AuditLog, error)
	ListAuditLogs(ctx context.Context, query ListAuditLogsQuery) (ListAuditLogsResult, error)
	ReadAuditLog(ctx context.Context, id uint64) (AuditLog, error)
	ReadAuditOverview(ctx context.Context, preset AuditTimePreset) (AuditOverview, error)
	ReadIncident(ctx context.Context, eventID uint64) (AuditIncident, error)
	// ListAuditPolicyRules 按运行时评估顺序返回全部审计策略规则。
	ListAuditPolicyRules(ctx context.Context) ([]AuditPolicyRule, error)
	// GetAuditVisibilityDefault 返回指定名称的默认审计可见性策略。
	GetAuditVisibilityDefault(ctx context.Context, key string) (AuditVisibilityDefault, error)
	// UpsertAuditVisibilityDefault 创建或更新一条指定名称的默认审计可见性策略。
	UpsertAuditVisibilityDefault(
		ctx context.Context,
		key string,
		strategy AuditVisibilityStrategy,
		userID *uint64,
		username string,
	) (AuditVisibilityDefault, error)
	// ListAuditVisibilityOverrides 返回全部来源加动作可见性覆盖规则。
	ListAuditVisibilityOverrides(ctx context.Context) ([]AuditVisibilityOverride, error)
	// FindAuditVisibilityOverride 在规则存在时返回精确匹配的来源加动作覆盖规则。
	FindAuditVisibilityOverride(ctx context.Context, source AuditSource, actionKey string) (AuditVisibilityOverride, bool, error)
	// UpsertAuditVisibilityOverride 创建或更新一条来源加动作可见性覆盖规则。
	UpsertAuditVisibilityOverride(ctx context.Context, input UpsertAuditVisibilityOverrideInput) (AuditVisibilityOverride, error)
	// DeleteAuditVisibilityOverride 删除一条来源加动作可见性覆盖规则。
	DeleteAuditVisibilityOverride(ctx context.Context, source AuditSource, actionKey string) error
	DeleteAuditLogsBefore(ctx context.Context, createdBefore time.Time) (int64, error)
}
