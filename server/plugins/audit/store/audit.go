// Package store defines audit-plugin-owned persistence contracts.
package store

import (
	"context"
	"time"
)

// AuditLog is the audit plugin's stable DTO for a persisted audit record.
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

// CreateAuditLogInput describes the minimum fields required to persist an audit record.
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

// AuditRepository exposes the audit plugin's write-only persistence contract.
type AuditRepository interface {
	CreateAuditLog(ctx context.Context, input CreateAuditLogInput) (AuditLog, error)
}
