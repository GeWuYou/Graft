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

func toAuditOverviewResponse(result auditOverviewResult) (map[string]any, error) {
	failedAuth, err := toAuditOverviewItems(result.FailedAuth)
	if err != nil {
		return nil, err
	}
	permissionDenied, err := toAuditOverviewItems(result.PermissionDenied)
	if err != nil {
		return nil, err
	}
	sensitiveOps, err := toAuditOverviewItems(result.SensitiveOps)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"window": string(result.Window),
		"summary": map[string]any{
			"total_logs":           result.Summary.TotalLogs,
			"failed_operations":    result.Summary.FailedOperations,
			"high_risk_events":     result.Summary.HighRiskEvents,
			"sensitive_operations": result.Summary.SensitiveOperations,
		},
		"failed_auth":          failedAuth,
		"permission_denied":    permissionDenied,
		"sensitive_operations": sensitiveOps,
	}, nil
}

func toAuditLogListItem(item auditstore.AuditLog) (generated.AuditLogListItem, error) {
	id, err := mustConvertAuditGeneratedID(item.ID, "audit log id")
	if err != nil {
		return generated.AuditLogListItem{}, err
	}

	converted := generated.AuditLogListItem{
		Id:           id,
		Action:       item.Action,
		ResourceType: item.ResourceType,
		Success:      item.Success,
		RequestId:    item.RequestID,
		Ip:           item.IP,
		UserAgent:    item.UserAgent,
		Message:      item.Message,
		CreatedAt:    item.CreatedAt.UTC(),
	}

	appendAuditLogOptionalStrings(&converted, item)
	appendAuditLogOptionalEnums(&converted, item)
	appendAuditLogOptionalRequest(&converted, item)

	if err := appendAuditLogActorUserID(&converted, item.ActorUserID); err != nil {
		return generated.AuditLogListItem{}, err
	}
	if err := appendAuditLogMetadata(&converted, item.Metadata); err != nil {
		return generated.AuditLogListItem{}, err
	}

	return converted, nil
}

func appendAuditLogOptionalStrings(converted *generated.AuditLogListItem, item auditstore.AuditLog) {
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
	if item.TargetType != "" {
		targetType := item.TargetType
		converted.TargetType = &targetType
	}
	if item.TargetLabel != "" {
		targetLabel := item.TargetLabel
		converted.TargetLabel = &targetLabel
	}
	if item.TraceID != "" {
		traceID := item.TraceID
		converted.TraceId = &traceID
	}
	if item.SessionID != "" {
		sessionID := item.SessionID
		converted.SessionId = &sessionID
	}
}

func appendAuditLogOptionalEnums(converted *generated.AuditLogListItem, item auditstore.AuditLog) {
	if item.Result != "" {
		result := generated.AuditLogListItemResult(item.Result)
		converted.Result = &result
	}
	if item.RiskLevel != "" {
		riskLevel := generated.AuditLogListItemRiskLevel(item.RiskLevel)
		converted.RiskLevel = &riskLevel
	}
}

func appendAuditLogOptionalRequest(converted *generated.AuditLogListItem, item auditstore.AuditLog) {
	if item.RequestMethod != "" {
		requestMethod := item.RequestMethod
		converted.RequestMethod = &requestMethod
	}
	if item.RequestPath != "" {
		requestPath := item.RequestPath
		converted.RequestPath = &requestPath
	}
	if item.StatusCode > 0 {
		statusCode := item.StatusCode
		converted.StatusCode = &statusCode
	}
}

func appendAuditLogActorUserID(converted *generated.AuditLogListItem, actorUserID *uint64) error {
	if actorUserID == nil {
		return nil
	}

	convertedActorUserID, err := mustConvertAuditGeneratedID(*actorUserID, "audit actor user id")
	if err != nil {
		return err
	}
	converted.ActorUserId = &convertedActorUserID
	return nil
}

func appendAuditLogMetadata(converted *generated.AuditLogListItem, rawMetadata json.RawMessage) error {
	if len(rawMetadata) == 0 {
		return nil
	}

	var metadata map[string]any
	if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
		return fmt.Errorf("decode audit metadata: %w", err)
	}
	converted.Metadata = metadata
	return nil
}

func toAuditOverviewItems(items []auditstore.OverviewItem) ([]map[string]any, error) {
	converted := make([]map[string]any, 0, len(items))
	for _, item := range items {
		mapped, err := toAuditOverviewItem(item)
		if err != nil {
			return nil, err
		}
		converted = append(converted, mapped)
	}
	return converted, nil
}

func toAuditOverviewItem(item auditstore.OverviewItem) (map[string]any, error) {
	id, err := mustConvertAuditGeneratedID(item.ID, "audit overview item id")
	if err != nil {
		return nil, err
	}

	converted := map[string]any{
		"id":         id,
		"action":     item.Action,
		"success":    item.Success,
		"request_id": item.RequestID,
		"message":    item.Message,
		"created_at": item.CreatedAt.UTC(),
	}
	if err := appendAuditOverviewActor(converted, item); err != nil {
		return nil, err
	}
	appendAuditOverviewResource(converted, item)

	metadata, err := decodeAuditOverviewMetadata(item.Metadata)
	if err != nil {
		return nil, err
	}
	converted["metadata"] = metadata

	return converted, nil
}

func mustConvertAuditGeneratedID(id uint64, label string) (int64, error) {
	if id > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds int64: %d", label, id)
	}

	return int64(id), nil
}

func appendAuditOverviewActor(converted map[string]any, item auditstore.OverviewItem) error {
	if item.ActorUserID != nil {
		actorUserID, err := mustConvertAuditGeneratedID(*item.ActorUserID, "audit overview actor user id")
		if err != nil {
			return err
		}
		converted["actor_user_id"] = actorUserID
	}
	if item.ActorUsername != "" {
		converted["actor_username"] = item.ActorUsername
	}
	if item.ActorDisplayName != "" {
		converted["actor_display_name"] = item.ActorDisplayName
	}

	return nil
}

func appendAuditOverviewResource(converted map[string]any, item auditstore.OverviewItem) {
	if item.ResourceType != "" {
		converted["resource_type"] = item.ResourceType
	}
	if item.ResourceID != "" {
		converted["resource_id"] = item.ResourceID
	}
	if item.ResourceName != "" {
		converted["resource_name"] = item.ResourceName
	}
}

func decodeAuditOverviewMetadata(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("decode audit overview metadata: %w", err)
	}

	return metadata, nil
}
