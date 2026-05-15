package entstore

import (
	"context"
	"fmt"

	"graft/server/internal/ent"
	"graft/server/internal/store"
)

type auditRepository struct {
	client *ent.Client
}

// CreateAuditLog 持久化一条最小审计记录。
func (r *auditRepository) CreateAuditLog(ctx context.Context, input store.CreateAuditLogInput) (store.AuditLog, error) {
	if r.client == nil {
		return store.AuditLog{}, fmt.Errorf("create audit log: nil ent client")
	}

	builder := r.client.AuditLog.Create().
		SetOperatorName(input.OperatorName).
		SetAction(input.Action).
		SetResourceType(input.ResourceType).
		SetResourceID(input.ResourceID).
		SetRequestMethod(input.RequestMethod).
		SetRequestPath(input.RequestPath).
		SetIP(input.IP).
		SetUserAgent(input.UserAgent).
		SetSuccess(input.Success).
		SetErrorMessage(input.ErrorMessage).
		SetCreatedAt(input.CreatedAt)
	if input.OperatorID != nil {
		builder = builder.SetOperatorID(*input.OperatorID)
	}

	record, err := builder.Save(ctx)
	if err != nil {
		return store.AuditLog{}, fmt.Errorf("create audit log: %w", err)
	}

	return store.AuditLog{
		ID:            uint64(record.ID),
		OperatorID:    record.OperatorID,
		OperatorName:  record.OperatorName,
		Action:        record.Action,
		ResourceType:  record.ResourceType,
		ResourceID:    record.ResourceID,
		RequestMethod: record.RequestMethod,
		RequestPath:   record.RequestPath,
		IP:            record.IP,
		UserAgent:     record.UserAgent,
		Success:       record.Success,
		ErrorMessage:  record.ErrorMessage,
		CreatedAt:     record.CreatedAt,
	}, nil
}
