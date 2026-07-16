package moduleapi

import (
	"context"
	"time"
)

// AuditOverviewPreset 定义安全概览读取使用的有界审计时间窗口。
type AuditOverviewPreset string

const (
	// AuditOverviewPresetLast24Hours 表示最近 24 小时窗口。
	AuditOverviewPresetLast24Hours AuditOverviewPreset = "last_24h"
	// AuditOverviewPresetLast7Days 表示最近 7 天窗口。
	AuditOverviewPresetLast7Days AuditOverviewPreset = "last_7d"
	// AuditOverviewPresetLast30Days 表示最近 30 天窗口。
	AuditOverviewPresetLast30Days AuditOverviewPreset = "last_30d"
)

// AuditSecurityRiskGroup 是暴露给安全模块的有界风险分组。
type AuditSecurityRiskGroup struct {
	Key       string `json:"key"`
	LabelKey  string `json:"label_key"`
	Count     int    `json:"count"`
	RiskLevel string `json:"risk_level"`
}

// AuditSecurityEvent 是安全概览使用的最小近期事件投影。
type AuditSecurityEvent struct {
	ID        uint64    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	RiskLevel string    `json:"risk_level"`
	Result    string    `json:"result"`
	RequestID string    `json:"request_id"`
}

// AuditSecuritySnapshot 包含审计模块拥有的计数器和有界近期安全信号。
type AuditSecuritySnapshot struct {
	TimePreset          AuditOverviewPreset      `json:"time_preset"`
	TotalLogs           int                      `json:"total_logs"`
	FailedOperations    int                      `json:"failed_operations"`
	HighRiskEvents      int                      `json:"high_risk_events"`
	SensitiveOperations int                      `json:"sensitive_operations"`
	RiskGroups          []AuditSecurityRiskGroup `json:"risk_groups"`
	RecentEvents        []AuditSecurityEvent     `json:"recent_events"`
}

// AuditSecurityReader 暴露由审计模块拥有的有界安全只读模型。
type AuditSecurityReader interface {
	ReadSecuritySnapshot(ctx context.Context, preset AuditOverviewPreset) (AuditSecuritySnapshot, error)
}

// EventName 标识跨模块稳定事件名契约。
type EventName string

// AuditRecordEventName 是业务模块主动发布审计事件时使用的稳定事件名。
const AuditRecordEventName EventName = "audit.record"

// AuditEventKind 标识通过事件总线发布的审计候选来源类别。
type AuditEventKind string

const (
	// AuditEventKindDomain 表示模块发布的业务域审计事件。
	AuditEventKindDomain AuditEventKind = "DOMAIN_EVENT"
	// AuditEventKindSecurity 表示请求守卫发出的认证或授权安全事件。
	AuditEventKindSecurity AuditEventKind = "SECURITY_EVENT"
)

// AuditEvent 描述跨模块可发布的最小审计事件载荷。
//
// 该 DTO 服务于“主动审计”路径：发布方提供明确业务语义，audit 模块负责
// 把事件收敛为稳定持久化记录。调用方可依赖以下稳定语义：
// - Action 必填；其余字符串字段允许为空，audit 模块会按需 trim 后落库。
// - Operator 可为空，表示当前事件不绑定明确操作者。
// - CreatedAt 为零值时由接收方补齐当前 UTC 时间；非零值会原样保留。
// - Message 允许为空，通常只在 Success 为 false 时携带稳定失败语义。
type AuditEvent struct {
	Kind          AuditEventKind
	Operator      *CurrentUser
	Action        string
	ResourceType  string
	ResourceID    string
	ResourceName  string
	RequestMethod string
	RequestPath   string
	StatusCode    int
	RequestID     string
	IP            string
	UserAgent     string
	Success       bool
	MessageKey    string
	Message       string
	Metadata      map[string]any
	CreatedAt     time.Time
}
