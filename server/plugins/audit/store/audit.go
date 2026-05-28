// Package store defines audit-plugin-owned persistence contracts.
package store

import (
	"context"
	"encoding/json"
	"time"
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
