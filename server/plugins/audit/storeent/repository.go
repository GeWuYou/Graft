// Package storeent 提供 audit 插件基于 SQL 的 repository 实现。
package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	auditstore "graft/server/plugins/audit/store"
)

type repository struct {
	db *sql.DB
}

// NewRepository 基于共享连接池构建 audit 插件的 SQL repository。
func NewRepository(db *sql.DB) (auditstore.AuditRepository, error) {
	if db == nil {
		return nil, errors.New("audit repository requires a non-nil sql db")
	}

	return &repository{db: db}, nil
}

// CreateAuditLog 持久化一条最小审计日志记录。
func (r *repository) CreateAuditLog(ctx context.Context, input auditstore.CreateAuditLogInput) (auditstore.AuditLog, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditLog{}, errors.New("audit repository is unavailable")
	}
	record := auditstore.AuditLog{
		OperatorID:    input.OperatorID,
		OperatorName:  input.OperatorName,
		Action:        input.Action,
		ResourceType:  input.ResourceType,
		ResourceID:    input.ResourceID,
		RequestMethod: input.RequestMethod,
		RequestPath:   input.RequestPath,
		IP:            input.IP,
		UserAgent:     input.UserAgent,
		Success:       input.Success,
		ErrorMessage:  input.ErrorMessage,
		CreatedAt:     input.CreatedAt,
	}

	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO audit_logs (
			operator_id,
			operator_name,
			action,
			resource_type,
			resource_id,
			request_method,
			request_path,
			ip,
			user_agent,
			success,
			error_message,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		nullableUint64(input.OperatorID),
		input.OperatorName,
		input.Action,
		input.ResourceType,
		input.ResourceID,
		input.RequestMethod,
		input.RequestPath,
		input.IP,
		input.UserAgent,
		input.Success,
		input.ErrorMessage,
		input.CreatedAt,
	)
	var id int64
	err := row.Scan(&id)
	if err != nil {
		return auditstore.AuditLog{}, fmt.Errorf("create audit log: %w", err)
	}
	record.ID = toStoreID(id)
	return record, nil
}

func nullableUint64(value *uint64) any {
	if value == nil {
		return nil
	}

	return *value
}

func toStoreID(id int64) uint64 {
	//nolint:gosec // 数据库 ID 来自受控 schema，并保持为正数。
	return uint64(id)
}
