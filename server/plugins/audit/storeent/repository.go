// Package storeent provides the audit plugin's Ent-backed repository implementation.
package storeent

import (
	"context"
	"errors"
	"fmt"

	"graft/server/internal/ent"
	auditstore "graft/server/plugins/audit/store"
)

type repository struct {
	client *ent.Client
}

// NewRepository builds the audit plugin's Ent-backed repository.
func NewRepository(client *ent.Client) (auditstore.AuditRepository, error) {
	if client == nil {
		return nil, errors.New("audit storeent repository requires a non-nil ent client")
	}

	return &repository{client: client}, nil
}

// CreateAuditLog persists one minimal audit record.
func (r *repository) CreateAuditLog(ctx context.Context, input auditstore.CreateAuditLogInput) (auditstore.AuditLog, error) {
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
		return auditstore.AuditLog{}, fmt.Errorf("create audit log: %w", err)
	}

	return auditstore.AuditLog{
		ID:            toStoreID(record.ID),
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

func toStoreID(id int) uint64 {
	//nolint:gosec // Ent IDs come from the controlled schema and remain positive.
	return uint64(id)
}
