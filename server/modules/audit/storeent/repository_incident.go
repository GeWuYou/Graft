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

// correlateIncidentEvents 汇总与 seed 相关的审计事件并按时间排序。
// 如果相关事件中未包含 seed，则将其补入结果；最终按 CreatedAt 降序、ID 降序稳定排序。
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

// collectRelatedIncidentEvents 从候选事件中收集与种子事件相关的审计事件，并标记结果中是否已包含种子事件。
// 结果最多保留 `incidentRelatedEventLimit` 条。
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

// appendRelatedIncidentCandidate 将与种子事件相关的候选事件加入结果集，并在需要时标记种子事件已包含。
// 当候选事件本身就是种子事件时，函数会确保其只加入一次；对其他候选事件，则仅在满足事故关联条件且未超过数量限制时加入结果集。
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

// incidentMatches 判断两个审计事件是否属于同一事故。
// 当事件 ID、请求 ID、会话 ID、执行人身份或资源标识任一匹配时返回 true。
func incidentMatches(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.ID == candidate.ID ||
		matchIncidentRequest(seed, candidate) ||
		matchIncidentSession(seed, candidate) ||
		matchIncidentActor(seed, candidate) ||
		matchIncidentResource(seed, candidate)
}

// matchIncidentRequest 在种子事件存在请求 ID 时，按请求 ID 判断两个事件是否属于同一请求。
// 
// @return 如果种子事件的 RequestID 非空且与候选事件相同，则返回 true，否则返回 false。
func matchIncidentRequest(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.RequestID != "" && seed.RequestID == candidate.RequestID
}

// matchIncidentSession 在种子事件包含会话 ID 时，判断候选事件是否具有相同的会话 ID。
// 当种子事件的 SessionID 为空时，始终返回 false。
func matchIncidentSession(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.SessionID != "" && seed.SessionID == candidate.SessionID
}

// matchIncidentActor 当两个事件的 ActorUserID 都存在且相等时返回匹配结果。
// @returns `true` 如果两个事件的 ActorUserID 都存在且相等，`false` 否则。
func matchIncidentActor(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.ActorUserID != nil && candidate.ActorUserID != nil && *seed.ActorUserID == *candidate.ActorUserID
}

// matchIncidentResource 在资源类型和资源 ID 都一致时匹配事件。
//
// 当种子事件同时包含资源类型和资源 ID 时，返回该候选事件是否具有相同的资源标识。
func matchIncidentResource(seed auditstore.AuditLog, candidate auditstore.AuditLog) bool {
	return seed.ResourceType != "" &&
		seed.ResourceType == candidate.ResourceType &&
		seed.ResourceID != "" &&
		seed.ResourceID == candidate.ResourceID
}

// summarizeIncidentActors 汇总事件中的关联操作者信息并按出现次数排序。
// 结果按事件数降序排列，事件数相同时按用户名与显示名的字典序排序；超过上限时只保留前 `incidentActorLimit` 条。
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

// hasIncidentActorIdentity 报告审计事件是否包含用于标识操作者的身份信息。
// 当事件包含用户 ID、用户名或显示名中的任意一项时，返回 true。
func hasIncidentActorIdentity(event auditstore.AuditLog) bool {
	return event.ActorUserID != nil || event.ActorUsername != "" || event.ActorDisplayName != ""
}

// incidentActorKeyFromLog 从审计日志提取用于聚合的 actorKey。
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

// summarizeIncidentResources 按资源身份汇总事件并按相关性排序。
// 以资源类型、资源 ID 和资源名称作为分组键，统计每组事件数，并按事件数降序、资源字段字典序升序排序。
// 结果最多返回 incidentResourceLimit 条。
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

// summarizeIncidentRequests 按请求 ID 汇总审计事件并限制结果数量。
//
// 以 RequestID 为键合并事件，统计每个请求的事件数量，并按事件数、结束时间和请求 ID 排序。
// 返回按优先级排列的请求汇总列表，最多包含 incidentRequestLimit 条。
//
// @param events 审计事件列表。
// @returns 按请求汇总后的事件列表。
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

// mergeIncidentRequest 更新审计请求汇总的时间范围和事件计数。
// 它使用事件的请求 ID 覆盖当前值，并将开始时间更新为最早的创建时间、结束时间更新为最晚的创建时间。
//
// @returns 更新后的审计请求汇总。
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

// 当事件包含请求 ID 时，使用请求维度生成键；否则使用事件 ID 生成键。
func buildIncidentKey(seed auditstore.AuditLog) string {
	if seed.RequestID != "" {
		return "incident:req:" + seed.RequestID
	}
	return "incident:event:" + strconv.FormatUint(seed.ID, 10)
}

// buildIncidentTitle 根据审计事件的结果和来源生成事故标题。
// 当事件被拒绝、来源为安全事件或结果为错误时，分别返回对应的专用标题；否则返回通用标题。
// @returns 生成的事故标题。
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

// buildIncidentSummary 生成事故摘要文本，包含标题、关联事件数量和种子事件 ID。
// 返回格式为“<标题> correlated <事件数> audit events around seed event <种子事件ID>.”。
func buildIncidentSummary(seed auditstore.AuditLog, events []auditstore.AuditLog) string {
	return fmt.Sprintf("%s correlated %d audit events around seed event %d.", buildIncidentTitle(seed), len(events), seed.ID)
}

// incidentRiskLevel 返回事件集中最高的风险等级。
// @returns 事件中观察到的最高风险等级。
func incidentRiskLevel(events []auditstore.AuditLog) auditstore.AuditRiskLevel {
	level := auditstore.AuditRiskLevelLow
	for _, event := range events {
		if riskRank(event.RiskLevel) > riskRank(level) {
			level = event.RiskLevel
		}
	}
	return level
}

// riskRank 返回审计风险等级的排序权重。 
// Critical 为 4，High 为 3，Medium 为 2，Low 及其他值为 1。
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

// incidentStartedAt 返回事件集合中最早的创建时间。
func incidentStartedAt(events []auditstore.AuditLog) time.Time {
	var startedAt time.Time
	for _, event := range events {
		if startedAt.IsZero() || event.CreatedAt.Before(startedAt) {
			startedAt = event.CreatedAt
		}
	}
	return startedAt
}

// incidentEndedAt 返回事件中最晚的创建时间。
//
// @return 最晚的 CreatedAt；如果 events 为空，则返回零值时间。
func incidentEndedAt(events []auditstore.AuditLog) time.Time {
	var endedAt time.Time
	for _, event := range events {
		if endedAt.IsZero() || event.CreatedAt.After(endedAt) {
			endedAt = event.CreatedAt
		}
	}
	return endedAt
}

// correlationReason 返回种子事件的关联说明。
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

// normalizeStoredAuditVisibility 将存储的审计可见性策略归一化为已知值。
// 它会去除首尾空白，并将未知值映射为 `Visible`。
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

// cloneRawMessage 返回 value 的独立副本；当输入为空时返回 `{}`。
func cloneRawMessage(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage([]byte("{}"))
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return json.RawMessage(cloned)
}
