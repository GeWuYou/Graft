package store

import (
	"context"
	"time"
)

// AuditLog 表示审计插件对上层暴露的稳定审计记录 DTO。
//
// 该 DTO 只表达当前 MVP 阶段确认需要保留的最小审计字段，不泄漏底层 ORM
// 结构，也不提前扩展检索 DSL、归档策略或风险评分语义。
type AuditLog struct {
	ID            uint64
	OperatorID    *uint64
	OperatorName  string
	Action        string
	ResourceType  string
	ResourceID    string
	RequestMethod string
	RequestPath   string
	IP            string
	UserAgent     string
	Success       bool
	ErrorMessage  string
	CreatedAt     time.Time
}

// CreateAuditLogInput 描述一次审计记录落盘所需的最小输入。
type CreateAuditLogInput struct {
	OperatorID    *uint64
	OperatorName  string
	Action        string
	ResourceType  string
	ResourceID    string
	RequestMethod string
	RequestPath   string
	IP            string
	UserAgent     string
	Success       bool
	ErrorMessage  string
	CreatedAt     time.Time
}

// AuditRepository 暴露审计插件所需的最小持久化写入能力。
//
// 当前阶段只提供写入入口，避免在没有真实检索需求前把查询、过滤和分页
// 语义过早固化到跨插件仓储边界。
type AuditRepository interface {
	// CreateAuditLog 持久化一条审计记录。
	CreateAuditLog(ctx context.Context, input CreateAuditLogInput) (AuditLog, error)
}
