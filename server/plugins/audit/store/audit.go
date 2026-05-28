// Package store defines audit-plugin-owned persistence contracts.
package store

import (
	"context"
	"encoding/json"
	"time"
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
	TargetType       string
	TargetLabel      string
	TraceID          string
	SessionID        string
	RequestMethod    string
	RequestPath      string
	StatusCode       int
	CreatedAt        time.Time
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

// ListAuditLogsQuery describes the audit plugin's stable repository-side query contract.
type ListAuditLogsQuery struct {
	ActorUserID  *uint64
	Action       string
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

// AuditOverview groups window-level counters with the recent slices used by the overview page.
type AuditOverview struct {
	Window           OverviewWindow
	Summary          OverviewSummary
	FailedAuth       []OverviewItem
	PermissionDenied []OverviewItem
	SensitiveOps     []OverviewItem
}

// AuditRepository exposes the audit plugin's persistence contract.
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, input CreateAuditLogInput) (AuditLog, error)
	ListAuditLogs(ctx context.Context, query ListAuditLogsQuery) (ListAuditLogsResult, error)
	ReadAuditOverview(ctx context.Context, window OverviewWindow) (AuditOverview, error)
}
