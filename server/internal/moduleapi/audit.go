// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package moduleapi

import "time"

// EventName 标识跨模块稳定事件名契约。
type EventName string

// AuditRecordEventName 是业务模块主动发布审计事件时使用的稳定事件名。
const AuditRecordEventName EventName = "audit.record"

// AuditEventKind identifies the audit candidate source class published on the bus.
type AuditEventKind string

const (
	// AuditEventKindDomain marks business-domain audit events published by modules.
	AuditEventKindDomain AuditEventKind = "DOMAIN_EVENT"
	// AuditEventKindSecurity marks auth/authz security events emitted from request guards.
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
