package audit

import (
	"encoding/json"
	"fmt"
	"math"

	generated "graft/server/internal/contract/openapi/generated"
	auditstore "graft/server/plugins/audit/store"
)

func toAuditLogListResponse(result auditListResult) (generated.AuditLogListResponse, error) {
	items := make([]generated.AuditLogListItem, 0, len(result.Items))
	for _, item := range result.Items {
		converted, err := toAuditLogListItem(item)
		if err != nil {
			return generated.AuditLogListResponse{}, err
		}
		items = append(items, converted)
	}

	return generated.AuditLogListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func toAuditLogListItem(item auditstore.AuditLog) (generated.AuditLogListItem, error) {
	id, err := mustConvertAuditGeneratedID(item.ID, "audit log id")
	if err != nil {
		return generated.AuditLogListItem{}, err
	}

	converted := generated.AuditLogListItem{
		Id:               id,
		Action:           item.Action,
		ResourceType:     item.ResourceType,
		Success:          item.Success,
		RequestId:        item.RequestID,
		Ip:               item.IP,
		UserAgent:        item.UserAgent,
		Message:          item.Message,
		CreatedAt:        item.CreatedAt.UTC(),
	}

	if item.ActorUsername != "" {
		actorUsername := item.ActorUsername
		converted.ActorUsername = &actorUsername
	}
	if item.ActorDisplayName != "" {
		actorDisplayName := item.ActorDisplayName
		converted.ActorDisplayName = &actorDisplayName
	}
	if item.ResourceID != "" {
		resourceID := item.ResourceID
		converted.ResourceId = &resourceID
	}
	if item.ResourceName != "" {
		resourceName := item.ResourceName
		converted.ResourceName = &resourceName
	}

	if item.ActorUserID != nil {
		actorUserID, actorErr := mustConvertAuditGeneratedID(*item.ActorUserID, "audit actor user id")
		if actorErr != nil {
			return generated.AuditLogListItem{}, actorErr
		}
		converted.ActorUserId = &actorUserID
	}

	if len(item.Metadata) > 0 {
		var metadata map[string]any
		if err := json.Unmarshal(item.Metadata, &metadata); err != nil {
			return generated.AuditLogListItem{}, fmt.Errorf("decode audit metadata: %w", err)
		}
		converted.Metadata = metadata
	}

	return converted, nil
}

func mustConvertAuditGeneratedID(id uint64, label string) (int64, error) {
	if id > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds int64: %d", label, id)
	}

	return int64(id), nil
}
