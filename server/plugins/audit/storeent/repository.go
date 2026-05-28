// Package storeent 提供 audit 插件基于 SQL 的 repository 实现。
package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	auditstore "graft/server/plugins/audit/store"
)

type repository struct {
	db *sql.DB
}

const defaultFilterCapacity = 8
const paginationParamCount = 2

// NewRepository 基于共享连接池构建 audit 插件的 SQL repository。
func NewRepository(db *sql.DB) (auditstore.AuditRepository, error) {
	if db == nil {
		return nil, errors.New("audit repository requires a non-nil sql db")
	}

	return &repository{db: db}, nil
}

// CreateAuditLog 持久化一条审计日志记录。
func (r *repository) CreateAuditLog(ctx context.Context, input auditstore.CreateAuditLogInput) (auditstore.AuditLog, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditLog{}, errors.New("audit repository is unavailable")
	}

	metadata := cloneRawMessage(input.Metadata)
	record := auditstore.AuditLog{
		ActorUserID:      input.ActorUserID,
		ActorUsername:    input.ActorUsername,
		ActorDisplayName: input.ActorDisplayName,
		Action:           input.Action,
		ResourceType:     input.ResourceType,
		ResourceID:       input.ResourceID,
		ResourceName:     input.ResourceName,
		Success:          input.Success,
		RequestID:        input.RequestID,
		IP:               input.IP,
		UserAgent:        input.UserAgent,
		Message:          input.Message,
		Metadata:         metadata,
		CreatedAt:        input.CreatedAt,
	}

	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO audit_logs (
			actor_user_id,
			actor_username,
			actor_display_name,
			action,
			resource_type,
			resource_id,
			resource_name,
			success,
			request_id,
			ip,
			user_agent,
			message,
			metadata,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`,
		nullableUint64(input.ActorUserID),
		input.ActorUsername,
		input.ActorDisplayName,
		input.Action,
		input.ResourceType,
		input.ResourceID,
		input.ResourceName,
		input.Success,
		input.RequestID,
		input.IP,
		input.UserAgent,
		input.Message,
		metadata,
		input.CreatedAt,
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		return auditstore.AuditLog{}, fmt.Errorf("create audit log: %w", err)
	}
	record.ID = toStoreID(id)

	return record, nil
}

// ListAuditLogs returns a stable page of audit records plus total count.
func (r *repository) ListAuditLogs(ctx context.Context, query auditstore.ListAuditLogsQuery) (auditstore.ListAuditLogsResult, error) {
	if r == nil || r.db == nil {
		return auditstore.ListAuditLogsResult{}, errors.New("audit repository is unavailable")
	}
	if query.Limit <= 0 {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("list audit logs: invalid limit %d", query.Limit)
	}
	if query.Offset < 0 {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("list audit logs: invalid offset %d", query.Offset)
	}

	whereSQL, args := buildAuditLogFilters(query)

	countSQL := `SELECT COUNT(*) FROM audit_logs` + whereSQL
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("count audit logs: %w", err)
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, query.Limit, query.Offset)

	//nolint:gosec // Query text is assembled from fixed SQL fragments; all dynamic values stay parameterized.
	selectSQL := `SELECT
		id,
		actor_user_id,
		actor_username,
		actor_display_name,
		action,
		resource_type,
		resource_id,
		resource_name,
		success,
		request_id,
		ip,
		user_agent,
		message,
		metadata,
		created_at
	FROM audit_logs` + whereSQL + fmt.Sprintf(
		" ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d",
		len(args)+1,
		len(args)+paginationParamCount,
	)

	rows, err := r.db.QueryContext(ctx, selectSQL, queryArgs...)
	if err != nil {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("list audit logs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]auditstore.AuditLog, 0, query.Limit)
	for rows.Next() {
		record, err := scanAuditLog(rows)
		if err != nil {
			return auditstore.ListAuditLogsResult{}, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("iterate audit logs: %w", err)
	}

	return auditstore.ListAuditLogsResult{Items: items, Total: total}, nil
}

func buildAuditLogFilters(query auditstore.ListAuditLogsQuery) (string, []any) {
	clauses := make([]string, 0, defaultFilterCapacity)
	args := make([]any, 0, defaultFilterCapacity)

	add := func(format string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	addUint64Filter(&clauses, &args, "actor_user_id = $%d", query.ActorUserID)
	addScalarFilter(add, "action = $%d", query.Action)
	addScalarFilter(add, "resource_type = $%d", query.ResourceType)
	addScalarFilter(add, "resource_id = $%d", query.ResourceID)
	addScalarFilter(add, "resource_name = $%d", query.ResourceName)
	addBoolFilter(&clauses, &args, "success = $%d", query.Success)
	addScalarFilter(add, "request_id = $%d", query.RequestID)
	addTimeFilter(&clauses, &args, "created_at >= $%d", query.CreatedFrom)
	addTimeFilter(&clauses, &args, "created_at <= $%d", query.CreatedTo)
	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func addScalarFilter(add func(string, any), format string, value string) {
	if value == "" {
		return
	}
	add(format, value)
}

func addUint64Filter(clauses *[]string, args *[]any, format string, value *uint64) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

func addBoolFilter(clauses *[]string, args *[]any, format string, value *bool) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

func addTimeFilter(clauses *[]string, args *[]any, format string, value *time.Time) {
	if value == nil {
		return
	}
	*args = append(*args, value.UTC())
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

func scanAuditLog(scanner interface {
	Scan(dest ...any) error
}) (auditstore.AuditLog, error) {
	var (
		record      auditstore.AuditLog
		actorUserID sql.NullInt64
		metadata    []byte
	)
	if err := scanner.Scan(
		&record.ID,
		&actorUserID,
		&record.ActorUsername,
		&record.ActorDisplayName,
		&record.Action,
		&record.ResourceType,
		&record.ResourceID,
		&record.ResourceName,
		&record.Success,
		&record.RequestID,
		&record.IP,
		&record.UserAgent,
		&record.Message,
		&metadata,
		&record.CreatedAt,
	); err != nil {
		return auditstore.AuditLog{}, fmt.Errorf("scan audit log: %w", err)
	}

	if actorUserID.Valid {
		value := toStoreID(actorUserID.Int64)
		record.ActorUserID = &value
	}
	record.Metadata = cloneRawMessage(metadata)

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

func cloneRawMessage(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage([]byte("{}"))
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return json.RawMessage(cloned)
}
