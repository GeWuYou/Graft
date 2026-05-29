package audit

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

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
	riskGroups := make([]map[string]any, 0, len(result.RiskGroups))
	for _, group := range result.RiskGroups {
		riskGroups = append(riskGroups, map[string]any{
			"key":        group.Key,
			"label_key":  group.LabelKey,
			"count":      group.Count,
			"risk_level": string(group.RiskLevel),
		})
	}
	trendPoints := make([]map[string]any, 0, len(result.Trend.Points))
	for _, point := range result.Trend.Points {
		trendPoints = append(trendPoints, map[string]any{
			"bucket_start":    point.BucketStart.UTC(),
			"bucket_end":      point.BucketEnd.UTC(),
			"total":           point.Total,
			"failed":          point.Failed,
			"high_risk":       point.HighRisk,
			"security_events": point.SecurityEvents,
		})
	}
	securityTimeline := make([]map[string]any, 0, len(result.SecurityTimeline))
	for _, item := range result.SecurityTimeline {
		id, err := mustConvertAuditGeneratedID(item.ID, "audit overview security timeline id")
		if err != nil {
			return nil, err
		}
		securityTimeline = append(securityTimeline, map[string]any{
			"id":                 id,
			"incident_seed":      map[string]any{"event_id": id},
			"created_at":         item.CreatedAt.UTC(),
			"source":             string(item.Source),
			"risk_level":         string(item.RiskLevel),
			"action":             item.Action,
			"result":             string(item.Result),
			"request_id":         item.RequestID,
			"actor_display_name": item.ActorDisplayName,
			"actor_username":     item.ActorUsername,
			"resource_name":      item.ResourceName,
			"resource_type":      item.ResourceType,
		})
	}
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
		"risk_groups":          riskGroups,
		"trend":                map[string]any{"bucket_unit": result.Trend.BucketUnit, "bucket_size": result.Trend.BucketSize, "points": trendPoints},
		"security_timeline":    securityTimeline,
		"failed_auth":          failedAuth,
		"permission_denied":    permissionDenied,
		"sensitive_operations": sensitiveOps,
	}, nil
}

func toAuditIncidentResponse(result auditIncidentResult) (map[string]any, error) {
	seedEvent, err := toAuditLogListItem(result.SeedEvent)
	if err != nil {
		return nil, err
	}

	relatedEvents, err := toAuditIncidentRelatedEvents(result.RelatedEvents)
	if err != nil {
		return nil, err
	}
	relatedActors, err := toAuditIncidentActors(result.RelatedActors)
	if err != nil {
		return nil, err
	}
	relatedResources := toAuditIncidentResources(result.RelatedResources)
	relatedRequests := toAuditIncidentRequests(result.RelatedRequests)

	return map[string]any{
		"seed_event": seedEvent,
		"incident": map[string]any{
			"incident_key":       result.Incident.IncidentKey,
			"title":              result.Incident.Title,
			"summary":            result.Incident.Summary,
			"risk_level":         string(result.Incident.RiskLevel),
			"started_at":         result.Incident.StartedAt.UTC(),
			"ended_at":           result.Incident.EndedAt.UTC(),
			"correlation_reason": result.Incident.CorrelationReason,
		},
		"related_events":    relatedEvents,
		"related_actors":    relatedActors,
		"related_resources": relatedResources,
		"related_requests":  relatedRequests,
		"monitor_context": map[string]any{
			"state":          string(result.MonitorContext.State),
			"summary":        result.MonitorContext.Summary,
			"reason":         result.MonitorContext.Reason,
			"anomaly_key":    result.MonitorContext.AnomalyKey,
			"scope_kind":     result.MonitorContext.ScopeKind,
			"scope_ref":      result.MonitorContext.ScopeRef,
			"observed_at":    optionalAuditObservedAt(result.MonitorContext.ObservedAt),
			"evidence_links": toAuditEvidenceLinks(result.MonitorContext.EvidenceLinks),
		},
	}, nil
}

func toAuditIncidentRelatedEvents(events []auditstore.AuditLog) ([]generated.AuditLogListItem, error) {
	relatedEvents := make([]generated.AuditLogListItem, 0, len(events))
	for _, item := range events {
		converted, err := toAuditLogListItem(item)
		if err != nil {
			return nil, err
		}
		relatedEvents = append(relatedEvents, converted)
	}
	return relatedEvents, nil
}

func toAuditIncidentActors(actors []auditstore.AuditIncidentActor) ([]map[string]any, error) {
	relatedActors := make([]map[string]any, 0, len(actors))
	for _, actor := range actors {
		entry, err := toAuditIncidentActor(actor)
		if err != nil {
			return nil, err
		}
		relatedActors = append(relatedActors, entry)
	}
	return relatedActors, nil
}

func toAuditIncidentActor(actor auditstore.AuditIncidentActor) (map[string]any, error) {
	entry := map[string]any{
		"event_count": actor.EventCount,
	}
	if actor.ActorUserID != nil {
		convertedID, err := mustConvertAuditGeneratedID(*actor.ActorUserID, "audit incident actor user id")
		if err != nil {
			return nil, err
		}
		entry["actor_user_id"] = convertedID
	}
	if actor.ActorUsername != "" {
		entry["actor_username"] = actor.ActorUsername
	}
	if actor.ActorDisplayName != "" {
		entry["actor_display_name"] = actor.ActorDisplayName
	}
	return entry, nil
}

func toAuditIncidentResources(resources []auditstore.AuditIncidentResource) []map[string]any {
	relatedResources := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		relatedResources = append(relatedResources, map[string]any{
			"resource_type": resource.ResourceType,
			"resource_id":   resource.ResourceID,
			"resource_name": resource.ResourceName,
			"event_count":   resource.EventCount,
		})
	}
	return relatedResources
}

func toAuditIncidentRequests(requests []auditstore.AuditIncidentRequest) []map[string]any {
	relatedRequests := make([]map[string]any, 0, len(requests))
	for _, request := range requests {
		relatedRequests = append(relatedRequests, map[string]any{
			"request_id":  request.RequestID,
			"event_count": request.EventCount,
			"started_at":  request.StartedAt.UTC(),
			"ended_at":    request.EndedAt.UTC(),
		})
	}
	return relatedRequests
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
	appendAuditLogTarget(&converted, item.Target)

	return converted, nil
}

func appendAuditLogTarget(converted *generated.AuditLogListItem, target auditstore.AuditTarget) {
	converted.Target = generated.AuditTarget{
		Kind:  generated.AuditTargetKind(target.Kind),
		Type:  target.Type,
		Label: target.Label,
	}
	if target.ID != "" {
		targetID := target.ID
		converted.Target.Id = &targetID
	}
	if target.RouteRef != "" {
		routeRef := target.RouteRef
		converted.Target.RouteRef = &routeRef
	}
}

func optionalAuditObservedAt(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC()
}

func toAuditEvidenceLinks(links []auditstore.EvidenceLink) []map[string]any {
	converted := make([]map[string]any, 0, len(links))
	for _, link := range links {
		entry := toAuditEvidenceLink(link)
		converted = append(converted, entry)
	}
	return converted
}

func toAuditEvidenceLink(link auditstore.EvidenceLink) map[string]any {
	entry := map[string]any{
		"target_kind": link.TargetKind,
		"link_state":  link.LinkState,
		"title":       link.Title,
	}
	appendAuditEvidenceReason(entry, link.Reason)
	appendAuditEvidenceTimeWindow(entry, link.TimeWindow)
	appendAuditEvidenceContext(entry, link.AuditContext)
	appendAuditEvidenceIncidentSeed(entry, link.IncidentSeed)
	return entry
}

func appendAuditEvidenceReason(entry map[string]any, reason string) {
	if reason != "" {
		entry["reason"] = reason
	}
}

func appendAuditEvidenceTimeWindow(entry map[string]any, window *auditstore.EvidenceLinkTimeWindow) {
	if window == nil {
		return
	}
	entry["time_window"] = map[string]any{
		"created_from": window.CreatedFrom.UTC(),
		"created_to":   window.CreatedTo.UTC(),
	}
}

func appendAuditEvidenceContext(entry map[string]any, context *auditstore.AuditEvidenceContext) {
	if context == nil {
		return
	}

	converted := map[string]any{}
	appendAuditEvidenceContextString(converted, "action", context.Action)
	appendAuditEvidenceContextString(converted, "action_prefix", context.ActionPrefix)
	appendAuditEvidenceContextString(converted, "source", string(context.Source))
	appendAuditEvidenceContextString(converted, "resource_type", context.ResourceType)
	appendAuditEvidenceContextString(converted, "resource_id", context.ResourceID)
	appendAuditEvidenceContextString(converted, "resource_name", context.ResourceName)
	appendAuditEvidenceContextString(converted, "request_id", context.RequestID)
	appendAuditEvidenceContextString(converted, "result", string(context.Result))
	appendAuditEvidenceContextString(converted, "risk_level", string(context.RiskLevel))
	appendAuditEvidenceContextTime(converted, "created_from", context.CreatedFrom)
	appendAuditEvidenceContextTime(converted, "created_to", context.CreatedTo)

	entry["audit_context"] = converted
}

func appendAuditEvidenceContextString(entry map[string]any, key string, value string) {
	if value != "" {
		entry[key] = value
	}
}

func appendAuditEvidenceContextTime(entry map[string]any, key string, value *time.Time) {
	if value == nil || value.IsZero() {
		return
	}
	entry[key] = value.UTC()
}

func appendAuditEvidenceIncidentSeed(entry map[string]any, seed *auditstore.IncidentSeedLink) {
	if seed == nil {
		return
	}
	id, err := mustConvertAuditGeneratedID(seed.EventID, "audit evidence incident seed id")
	if err == nil {
		entry["incident_seed"] = map[string]any{"event_id": id}
	}
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
	if item.Source != "" {
		source := generated.AuditLogListItemSource(item.Source)
		converted.Source = &source
	}
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
		"source":     string(item.Source),
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
