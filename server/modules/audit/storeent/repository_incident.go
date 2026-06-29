package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	auditstore "graft/server/modules/audit/store"
)

func (r *repository) readAuditLogByID(ctx context.Context, eventID uint64) (auditstore.AuditLog, error) {
	row := r.db.QueryRowContext(ctx, `SELECT
		id,
		COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') AS source,
		visibility,
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
	FROM audit_logs
	WHERE id = $1`, eventID)

	record, err := scanAuditLog(ctx, r.localizer, row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auditstore.AuditLog{}, auditstore.ErrAuditLogNotFound
		}
		return auditstore.AuditLog{}, fmt.Errorf("read audit log: %w", err)
	}
	return record, nil
}

func (r *repository) readIncidentCandidateLogs(ctx context.Context, windowStart time.Time, windowEnd time.Time) ([]auditstore.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT
		id,
		COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') AS source,
		visibility,
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
	FROM audit_logs
		WHERE visibility = $1
		AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC, id DESC
		LIMIT $4`,
		string(auditstore.AuditVisibilityStrategyVisible),
		windowStart,
		windowEnd,
		incidentCandidateScanLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("read audit incident candidates: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	candidates := make([]auditstore.AuditLog, 0, incidentRelatedEventLimit)
	for rows.Next() {
		record, scanErr := scanAuditLog(ctx, r.localizer, rows)
		if scanErr != nil {
			return nil, scanErr
		}
		candidates = append(candidates, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit incident candidates: %w", err)
	}
	return candidates, nil
}

func correlateIncidentEvents(seed auditstore.AuditLog, candidates []auditstore.AuditLog) []auditstore.AuditLog {
	related, seedIncluded := collectRelatedIncidentEvents(seed, candidates)
	if !seedIncluded {
		related = append(related, seed)
	}
	slices.SortStableFunc(related, func(a auditstore.AuditLog, b auditstore.AuditLog) int {
		switch {
		case a.CreatedAt.After(b.CreatedAt):
			return -1
		case a.CreatedAt.Before(b.CreatedAt):
			return 1
		case a.ID > b.ID:
			return -1
		case a.ID < b.ID:
			return 1
		default:
			return 0
		}
	})
	return related
}

func collectRelatedIncidentEvents(seed auditstore.AuditLog, candidates []auditstore.AuditLog) ([]auditstore.AuditLog, bool) {
	related := make([]auditstore.AuditLog, 0, incidentRelatedEventLimit)
	otherLimit := incidentRelatedEventLimit - 1
	seedIncluded := false
	for _, candidate := range candidates {
		related, seedIncluded = appendRelatedIncidentCandidate(seed, candidate, related, seedIncluded, otherLimit)
		if seedIncluded && len(related) == incidentRelatedEventLimit {
			break
		}
	}
	return related, seedIncluded
}

func appendRelatedIncidentCandidate(
	seed auditstore.AuditLog,
	candidate auditstore.AuditLog,
	related []auditstore.AuditLog,
	seedIncluded bool,
	otherLimit int,
) ([]auditstore.AuditLog, bool) {
	if candidate.ID == seed.ID {
		if seedIncluded {
			return related, true
		}
		return append(related, candidate), true
	}
	if !incidentMatches(seed, candidate) {
		return related, seedIncluded
	}
	if !seedIncluded && len(related) >= otherLimit {
		return related, seedIncluded
	}

	return append(related, candidate), seedIncluded
}

func incidentMatches(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.ID == candidate.ID ||
		matchIncidentRequest(seed, candidate) ||
		matchIncidentSession(seed, candidate) ||
		matchIncidentActor(seed, candidate) ||
		matchIncidentResource(seed, candidate)
}

func matchIncidentRequest(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.RequestID != "" && seed.RequestID == candidate.RequestID
}

func matchIncidentSession(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.SessionID != "" && seed.SessionID == candidate.SessionID
}

func matchIncidentActor(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.ActorUserID != nil && candidate.ActorUserID != nil && *seed.ActorUserID == *candidate.ActorUserID
}

func matchIncidentResource(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.ResourceType != "" &&
		seed.ResourceType == candidate.ResourceType &&
		seed.ResourceID != "" &&
		seed.ResourceID == candidate.ResourceID
}

func summarizeIncidentActors(events []auditstore.AuditLog) []auditstore.AuditIncidentActor {
	counts := make(map[actorKey]auditstore.AuditIncidentActor)
	for _, event := range events {
		if !hasIncidentActorIdentity(event) {
			continue
		}
		key := incidentActorKeyFromLog(event)
		entry := counts[key]
		entry.ActorUserID = event.ActorUserID
		entry.ActorUsername = event.ActorUsername
		entry.ActorDisplayName = event.ActorDisplayName
		entry.EventCount++
		counts[key] = entry
	}
	result := make([]auditstore.AuditIncidentActor, 0, len(counts))
	for _, item := range counts {
		result = append(result, item)
	}
	slices.SortStableFunc(result, func(a, b auditstore.AuditIncidentActor) int {
		switch {
		case a.EventCount > b.EventCount:
			return -1
		case a.EventCount < b.EventCount:
			return 1
		default:
			return strings.Compare(a.ActorUsername+a.ActorDisplayName, b.ActorUsername+b.ActorDisplayName)
		}
	})
	if len(result) > incidentActorLimit {
		return result[:incidentActorLimit]
	}
	return result
}

func hasIncidentActorIdentity(event auditstore.AuditLog) bool {
	return event.ActorUserID != nil || event.ActorUsername != "" || event.ActorDisplayName != ""
}

func incidentActorKeyFromLog(event auditstore.AuditLog) actorKey {
	key := actorKey{
		username: event.ActorUsername,
		display:  event.ActorDisplayName,
	}
	if event.ActorUserID != nil {
		key.id = *event.ActorUserID
	}
	return key
}

func summarizeIncidentResources(events []auditstore.AuditLog) []auditstore.AuditIncidentResource {
	type resourceKey struct {
		resourceType string
		resourceID   string
		resourceName string
	}
	counts := make(map[resourceKey]auditstore.AuditIncidentResource)
	for _, event := range events {
		if event.ResourceType == "" && event.ResourceID == "" && event.ResourceName == "" {
			continue
		}
		key := resourceKey{resourceType: event.ResourceType, resourceID: event.ResourceID, resourceName: event.ResourceName}
		entry := counts[key]
		entry.ResourceType = event.ResourceType
		entry.ResourceID = event.ResourceID
		entry.ResourceName = event.ResourceName
		entry.EventCount++
		counts[key] = entry
	}
	result := make([]auditstore.AuditIncidentResource, 0, len(counts))
	for _, item := range counts {
		result = append(result, item)
	}
	slices.SortStableFunc(result, func(a, b auditstore.AuditIncidentResource) int {
		switch {
		case a.EventCount > b.EventCount:
			return -1
		case a.EventCount < b.EventCount:
			return 1
		default:
			return strings.Compare(a.ResourceType+a.ResourceID+a.ResourceName, b.ResourceType+b.ResourceID+b.ResourceName)
		}
	})
	if len(result) > incidentResourceLimit {
		return result[:incidentResourceLimit]
	}
	return result
}

func summarizeIncidentRequests(events []auditstore.AuditLog) []auditstore.AuditIncidentRequest {
	grouped := make(map[string]auditstore.AuditIncidentRequest)
	for _, event := range events {
		if event.RequestID == "" {
			continue
		}
		grouped[event.RequestID] = mergeIncidentRequest(grouped[event.RequestID], event)
	}
	result := make([]auditstore.AuditIncidentRequest, 0, len(grouped))
	for _, item := range grouped {
		result = append(result, item)
	}
	slices.SortStableFunc(result, func(a, b auditstore.AuditIncidentRequest) int {
		switch {
		case a.EventCount > b.EventCount:
			return -1
		case a.EventCount < b.EventCount:
			return 1
		case a.EndedAt.After(b.EndedAt):
			return -1
		case a.EndedAt.Before(b.EndedAt):
			return 1
		default:
			return strings.Compare(a.RequestID, b.RequestID)
		}
	})
	if len(result) > incidentRequestLimit {
		return result[:incidentRequestLimit]
	}
	return result
}

func mergeIncidentRequest(current auditstore.AuditIncidentRequest, event auditstore.AuditLog) auditstore.AuditIncidentRequest {
	current.RequestID = event.RequestID
	current.EventCount++
	if current.StartedAt.IsZero() || event.CreatedAt.Before(current.StartedAt) {
		current.StartedAt = event.CreatedAt
	}
	if current.EndedAt.IsZero() || event.CreatedAt.After(current.EndedAt) {
		current.EndedAt = event.CreatedAt
	}
	return current
}

func buildIncidentKey(seed auditstore.AuditLog) string {
	if seed.RequestID != "" {
		return "incident:req:" + seed.RequestID
	}
	return "incident:event:" + strconv.FormatUint(seed.ID, 10)
}

func buildIncidentTitle(seed auditstore.AuditLog) string {
	if seed.Result == auditstore.AuditResultDenied {
		return "Permission denial incident"
	}
	if seed.Source == auditstore.AuditSourceSecurityEvent {
		return "Security event incident"
	}
	if seed.Result == auditstore.AuditResultError {
		return "Audit error incident"
	}
	return "Audit incident"
}

func buildIncidentSummary(seed auditstore.AuditLog, events []auditstore.AuditLog) string {
	return fmt.Sprintf("%s correlated %d audit events around seed event %d.", buildIncidentTitle(seed), len(events), seed.ID)
}

func incidentRiskLevel(events []auditstore.AuditLog) auditstore.AuditRiskLevel {
	level := auditstore.AuditRiskLevelLow
	for _, event := range events {
		if riskRank(event.RiskLevel) > riskRank(level) {
			level = event.RiskLevel
		}
	}
	return level
}

func riskRank(level auditstore.AuditRiskLevel) int {
	const (
		riskRankLow      = 1
		riskRankMedium   = 2
		riskRankHigh     = 3
		riskRankCritical = 4
	)

	switch level {
	case auditstore.AuditRiskLevelCritical:
		return riskRankCritical
	case auditstore.AuditRiskLevelHigh:
		return riskRankHigh
	case auditstore.AuditRiskLevelMedium:
		return riskRankMedium
	default:
		return riskRankLow
	}
}

func incidentStartedAt(events []auditstore.AuditLog) time.Time {
	var startedAt time.Time
	for _, event := range events {
		if startedAt.IsZero() || event.CreatedAt.Before(startedAt) {
			startedAt = event.CreatedAt
		}
	}
	return startedAt
}

func incidentEndedAt(events []auditstore.AuditLog) time.Time {
	var endedAt time.Time
	for _, event := range events {
		if endedAt.IsZero() || event.CreatedAt.After(endedAt) {
			endedAt = event.CreatedAt
		}
	}
	return endedAt
}

func correlationReason(seed auditstore.AuditLog) string {
	if seed.RequestID != "" {
		return "Correlated by stable request_id first, then expanded through bounded actor, resource, and session joins."
	}
	if seed.SessionID != "" {
		return "Correlated by stable session_id first, then expanded through bounded actor and resource joins."
	}
	if seed.ActorUserID != nil {
		return "Correlated by stable actor identity inside a bounded incident window."
	}
	if seed.ResourceType != "" && seed.ResourceID != "" {
		return "Correlated by stable resource identity inside a bounded incident window."
	}
	return "Correlated from the seed event inside a bounded incident window."
}

func normalizeStoredAuditVisibility(value auditstore.AuditVisibilityStrategy) auditstore.AuditVisibilityStrategy {
	switch auditstore.AuditVisibilityStrategy(strings.TrimSpace(string(value))) {
	case auditstore.AuditVisibilityStrategyHidden:
		return auditstore.AuditVisibilityStrategyHidden
	case auditstore.AuditVisibilityStrategyIgnore:
		return auditstore.AuditVisibilityStrategyIgnore
	default:
		return auditstore.AuditVisibilityStrategyVisible
	}
}

func cloneRawMessage(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage([]byte("{}"))
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return json.RawMessage(cloned)
}
