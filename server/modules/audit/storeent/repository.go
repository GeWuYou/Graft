// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package storeent 提供 audit 模块基于 SQL 的 repository 实现。
package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
)

type repository struct {
	db              *sql.DB
	localizer       *i18n.Service
	monitorEvidence moduleapi.MonitorIncidentEvidenceService
}

type auditLocaleContextKey struct{}

type actorKey struct {
	id       uint64
	username string
	display  string
}

const defaultFilterCapacity = 8
const paginationParamCount = 2
const overviewRecentLimit = 3
const overviewRiskGroupLimit = 4
const overviewTrendPointLimit = 12
const overviewSecurityTimelineLimit = 6
const incidentRelatedEventLimit = 20
const incidentActorLimit = 5
const incidentResourceLimit = 5
const incidentRequestLimit = 5
const httpStatusForbidden = 403
const httpStatusServerErrorMin = 500
const overviewTrendDayStep = "1 day"
const overviewTrendThreeDayStep = "3 day"
const overviewTrendTwoHourStep = "2 hour"
const overviewTrendDayBucketSize = 1
const overviewTrendThreeDayBucketSize = 3
const overviewTrendTwoHourBucketSize = 2
const overviewTrendOneDayDuration = 24 * time.Hour
const overviewTrendThreeDayDuration = 72 * time.Hour
const overviewTrendTwoHourDuration = 2 * time.Hour
const incidentCorrelationWindow = 30 * time.Minute
const incidentCandidateScanLimit = 200
const sqlLikeEscapeClause = " ESCAPE '\\'"

// NewRepository 基于共享连接池构建 audit 模块的 SQL repository。
func NewRepository(
	db *sql.DB,
	localizer *i18n.Service,
	monitorEvidence moduleapi.MonitorIncidentEvidenceService,
) (auditstore.AuditRepository, error) {
	if db == nil {
		return nil, errors.New("audit repository requires a non-nil sql db")
	}
	if localizer == nil {
		return nil, errors.New("audit repository requires a non-nil i18n service")
	}

	return &repository{db: db, localizer: localizer, monitorEvidence: monitorEvidence}, nil
}

func (r *repository) BindMonitorEvidence(service moduleapi.MonitorIncidentEvidenceService) {
	if r == nil {
		return
	}
	r.monitorEvidence = service
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
	actorUserID, err := nullableUint64(input.ActorUserID)
	if err != nil {
		return auditstore.AuditLog{}, fmt.Errorf("create audit log: %w", err)
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
		actorUserID,
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
	if err := validateListAuditLogsQuery(query); err != nil {
		return auditstore.ListAuditLogsResult{}, err
	}
	if r == nil || r.db == nil {
		return auditstore.ListAuditLogsResult{}, errors.New("audit repository is unavailable")
	}

	whereSQL, args := buildAuditLogFilters(query)

	countSQL := `SELECT COUNT(*) FROM audit_logs` + whereSQL
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return auditstore.ListAuditLogsResult{}, fmt.Errorf("count audit logs: %w", err)
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, query.Limit, query.Offset)
	orderBySQL := buildAuditLogOrderBy(query)

	//nolint:gosec // Query text is assembled from fixed SQL fragments; all dynamic values stay parameterized.
	selectSQL := `SELECT
		id,
		COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') AS source,
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
		" %s LIMIT $%d OFFSET $%d",
		orderBySQL,
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
		record, err := scanAuditLog(ctx, r.localizer, rows)
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

// ReadAuditLog returns one immutable audit evidence record by id.
func (r *repository) ReadAuditLog(ctx context.Context, id uint64) (auditstore.AuditLog, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditLog{}, errors.New("audit repository is unavailable")
	}
	if id == 0 {
		return auditstore.AuditLog{}, auditstore.ErrAuditLogNotFound
	}

	record, err := r.readAuditLogByID(ctx, id)
	if err != nil {
		return auditstore.AuditLog{}, err
	}
	return record, nil
}

// DeleteAuditLogsBefore deletes audit records older than the caller-owned retention cutoff.
func (r *repository) DeleteAuditLogsBefore(ctx context.Context, createdBefore time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("audit repository is unavailable")
	}
	if createdBefore.IsZero() {
		return 0, errors.New("audit log cleanup cutoff is required")
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM audit_logs WHERE created_at < $1`, createdBefore.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete audit logs before cutoff: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted audit log count: %w", err)
	}

	return deleted, nil
}

func buildAuditLogOrderBy(query auditstore.ListAuditLogsQuery) string {
	for _, raw := range query.Sorts {
		switch strings.TrimSpace(raw) {
		case "created_at:asc":
			return "ORDER BY created_at ASC, id ASC"
		case "created_at:desc":
			return "ORDER BY created_at DESC, id DESC"
		}
	}

	return "ORDER BY created_at DESC, id DESC"
}

// ReadAuditOverview aggregates real overview data from the settled audit log table.
func (r *repository) ReadAuditOverview(ctx context.Context, preset auditstore.AuditTimePreset) (auditstore.AuditOverview, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditOverview{}, errors.New("audit repository is unavailable")
	}

	now := time.Now().UTC()
	startedAt := auditPresetStart(now, preset)
	args := []any{startedAt}

	summary, err := r.readAuditOverviewSummary(ctx, args)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}
	riskGroups, err := r.readOverviewRiskGroups(ctx, args)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}
	trend, err := r.readOverviewTrend(ctx, preset, startedAt, now)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}
	securityTimeline, err := r.readOverviewSecurityTimeline(ctx, args)
	if err != nil {
		return auditstore.AuditOverview{}, err
	}

	failedAuth, err := r.readAuditOverviewItems(ctx, args, authFailuresWhereClause())
	if err != nil {
		return auditstore.AuditOverview{}, err
	}
	permissionDenied, err := r.readAuditOverviewItems(ctx, args, permissionDenialsWhereClause())
	if err != nil {
		return auditstore.AuditOverview{}, err
	}
	sensitiveOps, err := r.readAuditOverviewItems(ctx, args, sensitiveOperationsWhereClause())
	if err != nil {
		return auditstore.AuditOverview{}, err
	}

	return auditstore.AuditOverview{
		TimePreset:       preset,
		Summary:          summary,
		RiskGroups:       riskGroups,
		Trend:            trend,
		SecurityTimeline: securityTimeline,
		FailedAuth:       failedAuth,
		PermissionDenied: permissionDenied,
		SensitiveOps:     sensitiveOps,
	}, nil
}

// ReadIncident returns the audit-owned incident drilldown derived from one seed event.
func (r *repository) ReadIncident(ctx context.Context, eventID uint64) (auditstore.AuditIncident, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditIncident{}, errors.New("audit repository is unavailable")
	}

	seed, err := r.readAuditLogByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, auditstore.ErrAuditLogNotFound) {
			return auditstore.AuditIncident{}, auditstore.ErrIncidentNotFound
		}
		return auditstore.AuditIncident{}, err
	}

	windowStart := seed.CreatedAt.Add(-incidentCorrelationWindow)
	windowEnd := seed.CreatedAt.Add(incidentCorrelationWindow)

	candidates, err := r.readIncidentCandidateLogs(ctx, windowStart, windowEnd)
	if err != nil {
		return auditstore.AuditIncident{}, err
	}

	relatedEvents := correlateIncidentEvents(seed, candidates)
	relatedActors := summarizeIncidentActors(relatedEvents)
	relatedResources := summarizeIncidentResources(relatedEvents)
	relatedRequests := summarizeIncidentRequests(relatedEvents)

	return auditstore.AuditIncident{
		SeedEvent: seed,
		Incident: auditstore.AuditIncidentSummary{
			IncidentKey:       buildIncidentKey(seed),
			Title:             buildIncidentTitle(seed),
			Summary:           buildIncidentSummary(seed, relatedEvents),
			RiskLevel:         incidentRiskLevel(relatedEvents),
			StartedAt:         incidentStartedAt(relatedEvents),
			EndedAt:           incidentEndedAt(relatedEvents),
			CorrelationReason: correlationReason(seed),
		},
		RelatedEvents:    relatedEvents,
		RelatedActors:    relatedActors,
		RelatedResources: relatedResources,
		RelatedRequests:  relatedRequests,
		MonitorContext:   r.resolveIncidentMonitorContext(ctx, seed, relatedEvents),
	}, nil
}

func (r *repository) resolveIncidentMonitorContext(
	ctx context.Context,
	seed auditstore.AuditLog,
	relatedEvents []auditstore.AuditLog,
) auditstore.AuditIncidentMonitorContext {
	if r == nil || r.monitorEvidence == nil {
		return auditstore.AuditIncidentMonitorContext{
			State:         auditstore.MonitorContextStateUnavailable,
			Summary:       "Monitor capability is unavailable for this audit incident.",
			Reason:        "Monitor module capability is unavailable.",
			EvidenceLinks: buildIncidentMonitorEvidenceLinks(seed, relatedEvents),
		}
	}

	resolved, err := r.monitorEvidence.ResolveAuditIncidentMonitorEvidence(ctx, moduleapi.ResolveAuditIncidentMonitorEvidenceInput{
		IncidentSeedEventID: seed.ID,
		IncidentStartedAt:   incidentStartedAt(relatedEvents),
		IncidentEndedAt:     incidentEndedAt(relatedEvents),
		RequestID:           seed.RequestID,
		ResourceType:        seed.ResourceType,
		ResourceID:          seed.ResourceID,
		ResourceName:        seed.ResourceName,
		AuditSource:         string(seed.Source),
		AuditResult:         string(seed.Result),
		AuditRiskLevel:      string(seed.RiskLevel),
	})
	if err != nil {
		return auditstore.AuditIncidentMonitorContext{
			State:         auditstore.MonitorContextStateUnavailable,
			Summary:       "Monitor capability could not resolve incident evidence.",
			Reason:        "Monitor capability is unavailable.",
			EvidenceLinks: buildIncidentMonitorEvidenceLinks(seed, relatedEvents),
		}
	}

	return auditstore.AuditIncidentMonitorContext{
		State:         monitorContextStateFromAvailability(resolved.Availability),
		Summary:       resolved.Summary,
		Reason:        resolved.Reason,
		AnomalyKey:    resolved.AnomalyKey,
		ScopeKind:     resolved.ScopeKind,
		ScopeRef:      resolved.ScopeRef,
		ObservedAt:    resolved.ObservedAt,
		EvidenceLinks: toAuditEvidenceLinksFromMonitor(resolved.EvidenceLinks, seed, relatedEvents),
	}
}

func monitorContextStateFromAvailability(availability moduleapi.MonitorEvidenceAvailability) auditstore.MonitorContextState {
	if availability == moduleapi.MonitorEvidenceAvailable {
		return auditstore.MonitorContextStateAvailable
	}
	return auditstore.MonitorContextStateUnavailable
}

func toAuditEvidenceLinksFromMonitor(
	links []moduleapi.MonitorEvidenceLink,
	seed auditstore.AuditLog,
	relatedEvents []auditstore.AuditLog,
) []auditstore.EvidenceLink {
	if len(links) == 0 {
		return buildIncidentMonitorEvidenceLinks(seed, relatedEvents)
	}

	converted := make([]auditstore.EvidenceLink, 0, len(links))
	for _, link := range links {
		entry := auditstore.EvidenceLink{
			TargetKind: link.TargetKind,
			LinkState:  link.LinkState,
			Title:      link.Title,
			Reason:     link.Reason,
		}
		if link.TimeWindow != nil {
			entry.TimeWindow = &auditstore.EvidenceLinkTimeWindow{
				CreatedFrom: link.TimeWindow.CreatedFrom,
				CreatedTo:   link.TimeWindow.CreatedTo,
			}
		}
		if link.AuditContext != nil {
			entry.AuditContext = &auditstore.AuditEvidenceContext{
				Action:       link.AuditContext.Action,
				ActionPrefix: link.AuditContext.ActionPrefix,
				Source:       auditstore.AuditSource(link.AuditContext.Source),
				ResourceType: link.AuditContext.ResourceType,
				ResourceID:   link.AuditContext.ResourceID,
				ResourceName: link.AuditContext.ResourceName,
				RequestID:    link.AuditContext.RequestID,
				Result:       auditstore.AuditResult(link.AuditContext.Result),
				RiskLevel:    auditstore.AuditRiskLevel(link.AuditContext.RiskLevel),
				CreatedFrom:  link.AuditContext.CreatedFrom,
				CreatedTo:    link.AuditContext.CreatedTo,
			}
		}
		if link.IncidentSeed != nil {
			entry.IncidentSeed = &auditstore.IncidentSeedLink{EventID: link.IncidentSeed.EventID}
		}
		converted = append(converted, entry)
	}

	return converted
}

func buildIncidentMonitorEvidenceLinks(seed auditstore.AuditLog, relatedEvents []auditstore.AuditLog) []auditstore.EvidenceLink {
	window := incidentEvidenceWindow(relatedEvents)
	link := auditstore.EvidenceLink{
		TargetKind: "audit_incident",
		LinkState:  "available",
		Title:      "Audit incident evidence",
		IncidentSeed: &auditstore.IncidentSeedLink{
			EventID: seed.ID,
		},
	}
	if window != nil {
		link.TimeWindow = window
	}

	context := auditstore.AuditEvidenceContext{
		RequestID:    seed.RequestID,
		ResourceType: seed.ResourceType,
		ResourceID:   seed.ResourceID,
		ResourceName: seed.ResourceName,
		Result:       seed.Result,
		RiskLevel:    seed.RiskLevel,
	}
	if seed.Source != "" {
		context.Source = seed.Source
	}
	if window != nil {
		context.CreatedFrom = &window.CreatedFrom
		context.CreatedTo = &window.CreatedTo
	}
	link.AuditContext = &context

	return []auditstore.EvidenceLink{link}
}

func incidentEvidenceWindow(events []auditstore.AuditLog) *auditstore.EvidenceLinkTimeWindow {
	if len(events) == 0 {
		return nil
	}
	startedAt := incidentStartedAt(events)
	endedAt := incidentEndedAt(events)
	if startedAt.IsZero() || endedAt.IsZero() {
		return nil
	}
	return &auditstore.EvidenceLinkTimeWindow{
		CreatedFrom: startedAt,
		CreatedTo:   endedAt,
	}
}

func buildAuditLogFilters(query auditstore.ListAuditLogsQuery) (string, []any) {
	clauses := make([]string, 0, defaultFilterCapacity)
	args := make([]any, 0, defaultFilterCapacity)

	add := func(format string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	addAuditPresetRange(&clauses, &args, query)
	addUint64Filter(&clauses, &args, "actor_user_id = $%d", query.ActorUserID)
	addKeywordFilter(&clauses, &args, query.Keyword)
	addActorFilter(&clauses, &args, query.Actor)
	addScalarFilter(add, "action = $%d", query.Action)
	addPrefixFilter(add, "action LIKE $%d"+sqlLikeEscapeClause, query.ActionPrefix)
	addPrefixAnyFilter(&clauses, &args, "action", query.ActionPrefixes)
	addKeywordAnyFilter(&clauses, &args, "action", query.ActionKeywords)
	addScalarFilter(add, sourceWhereClause(), string(query.Source))
	addBusinessCategoryFilter(&clauses, query.BusinessCategory)
	addScalarFilter(add, "resource_type = $%d", query.ResourceType)
	addAnyScalarFilter(&clauses, &args, "resource_type", query.ResourceTypes)
	addScalarFilter(add, "resource_id = $%d", query.ResourceID)
	addScalarFilter(add, "resource_name = $%d", query.ResourceName)
	addPrefixAnyJSONMetadataFilter(&clauses, &args, "request_path", query.RequestPathPrefixes)
	addBoolFilter(&clauses, &args, "success = $%d", query.Success)
	addScalarJSONMetadataFilter(&clauses, &args, "session_id", query.SessionID)
	addScalarFilter(add, "request_id = $%d", query.RequestID)
	addScalarFilter(add, auditResultWhereClause(), string(query.Result))
	addAnyExpressionFilter(&clauses, &args, auditResultWhereClause(), auditResultValues(query.Results))
	addScalarFilter(add, riskLevelWhereClause(), string(query.RiskLevel))
	addAnyExpressionFilter(&clauses, &args, riskLevelWhereClause(), auditRiskLevelValues(query.RiskLevels))
	addTimeFilter(&clauses, &args, "created_at >= $%d", query.CreatedFrom)
	addTimeFilter(&clauses, &args, "created_at <= $%d", query.CreatedTo)
	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func validateListAuditLogsQuery(query auditstore.ListAuditLogsQuery) error {
	if query.Limit <= 0 {
		return fmt.Errorf("list audit logs: invalid limit %d", query.Limit)
	}
	if query.Offset < 0 {
		return fmt.Errorf("list audit logs: invalid offset %d", query.Offset)
	}
	if query.TimePreset != "" && !isSupportedAuditTimePreset(query.TimePreset) {
		return fmt.Errorf("list audit logs: invalid time preset %q", query.TimePreset)
	}
	for _, raw := range query.Sorts {
		switch strings.TrimSpace(raw) {
		case "created_at:asc", "created_at:desc":
		default:
			return fmt.Errorf("list audit logs: invalid sort %q", raw)
		}
	}

	return nil
}

func isSupportedAuditTimePreset(preset auditstore.AuditTimePreset) bool {
	switch preset {
	case auditstore.AuditTimePresetLast24Hours,
		auditstore.AuditTimePresetLast7Days,
		auditstore.AuditTimePresetLast30Days:
		return true
	default:
		return false
	}
}

func addAuditPresetRange(clauses *[]string, args *[]any, query auditstore.ListAuditLogsQuery) {
	if query.CreatedFrom != nil || query.CreatedTo != nil {
		return
	}
	if query.TimePreset == "" {
		return
	}

	now := time.Now().UTC()
	startedAt := auditPresetStart(now, query.TimePreset)
	addTimeFilter(clauses, args, "created_at >= $%d", &startedAt)
}

func auditPresetStart(now time.Time, preset auditstore.AuditTimePreset) time.Time {
	switch preset {
	case auditstore.AuditTimePresetLast24Hours:
		return now.Add(-24 * time.Hour)
	case auditstore.AuditTimePresetLast7Days:
		return now.Add(-7 * 24 * time.Hour)
	case auditstore.AuditTimePresetLast30Days:
		return now.Add(-30 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}

func highRiskOperationsWhereClause() string {
	return `(` + auditRiskLevelExpression() + ` IN ('HIGH', 'CRITICAL'))`
}

func failedOperationsWhereClause() string {
	return `(` + auditResultExpression() + ` IN ('FAILED', 'DENIED', 'ERROR'))`
}

func sensitiveOperationsWhereClause() string {
	keywords := sensitiveOperationAuthorityKeywords()
	orClauses := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		orClauses = append(orClauses, fmt.Sprintf("LOWER(action) LIKE '%%%s%%'", strings.ToLower(keyword)))
	}
	return "(" + strings.Join(orClauses, "\n\t\tOR ") + ")"
}

func authFailuresWhereClause() string {
	return `
	success = false AND (
		LOWER(action) LIKE '%auth%'
		OR resource_type = 'auth'
		OR resource_type = 'session'
		OR LOWER(` + overviewMetadataRequestPathSQL + `) LIKE '/api/auth%'
	)
`
}

func permissionDenialsWhereClause() string {
	return `
	success = false AND (
		` + overviewMetadataStatusCodeSQL + ` = '403'
		OR message = 'common.forbidden'
		OR LOWER(message) LIKE '%forbidden%'
		OR LOWER(message) LIKE '%permission%'
	)
`
}

func rbacChangesWhereClause() string {
	return `(
		LOWER(action) LIKE 'rbac.%'
		OR LOWER(action) LIKE 'role.%'
		OR LOWER(action) LIKE 'permission.%'
	)`
}

func criticalSecurityWhereClause() string {
	return `
	success = false AND (
		` + overviewMetadataStatusCodeSQL + ` = '403'
		OR (
			COALESCE(NULLIF(metadata ->> 'status_code', ''), '') <> ''
			AND metadata ->> 'status_code' ~ '^[0-9]{1,5}$'
			AND CAST(metadata ->> 'status_code' AS INTEGER) >= 500
		)
		OR COALESCE(metadata ->> 'error_kind', '') = 'system'
		OR COALESCE(metadata ->> 'error', '') <> ''
	)
`
}

func addBusinessCategoryFilter(clauses *[]string, category auditstore.AuditBusinessCategory) {
	switch category {
	case auditstore.AuditBusinessCategoryFailedOperations:
		*clauses = append(*clauses, "("+failedOperationsWhereClause()+")")
	case auditstore.AuditBusinessCategoryHighRiskOperations:
		*clauses = append(*clauses, highRiskOperationsWhereClause())
	case auditstore.AuditBusinessCategorySensitiveOperations:
		*clauses = append(*clauses, sensitiveOperationsWhereClause())
	case auditstore.AuditBusinessCategoryAuthFailures:
		*clauses = append(*clauses, "("+authFailuresWhereClause()+")")
	case auditstore.AuditBusinessCategoryPermissionDenials:
		*clauses = append(*clauses, "("+permissionDenialsWhereClause()+")")
	case auditstore.AuditBusinessCategoryRBACChanges:
		*clauses = append(*clauses, "("+rbacChangesWhereClause()+")")
	case auditstore.AuditBusinessCategoryCriticalSecurity:
		*clauses = append(*clauses, "("+criticalSecurityWhereClause()+")")
	default:
	}
}

func addScalarFilter(add func(string, any), format string, value string) {
	if value == "" {
		return
	}
	add(format, value)
}

func addPrefixFilter(add func(string, any), format string, value string) {
	if value == "" {
		return
	}

	add(format, escapeLikePattern(value)+"%")
}

func addPrefixAnyFilter(clauses *[]string, args *[]any, column string, values []string) {
	if len(values) == 0 {
		return
	}

	orClauses := make([]string, 0, len(values))
	for _, value := range values {
		*args = append(*args, escapeLikePattern(value)+"%")
		orClauses = append(orClauses, fmt.Sprintf("%s LIKE $%d%s", column, len(*args), sqlLikeEscapeClause))
	}
	*clauses = append(*clauses, "("+strings.Join(orClauses, " OR ")+")")
}

func addKeywordAnyFilter(clauses *[]string, args *[]any, column string, values []string) {
	if len(values) == 0 {
		return
	}

	orClauses := make([]string, 0, len(values))
	for _, value := range values {
		*args = append(*args, "%"+escapeLikePattern(strings.ToLower(value))+"%")
		orClauses = append(orClauses, fmt.Sprintf("LOWER(%s) LIKE $%d%s", column, len(*args), sqlLikeEscapeClause))
	}
	*clauses = append(*clauses, "("+strings.Join(orClauses, " OR ")+")")
}

func addKeywordFilter(clauses *[]string, args *[]any, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	pattern := "%" + escapeLikePattern(strings.ToLower(strings.TrimSpace(value))) + "%"
	fields := []string{
		"LOWER(action)",
		"LOWER(request_id)",
		"LOWER(message)",
		"LOWER(resource_type)",
		"LOWER(resource_id)",
		"LOWER(resource_name)",
		"LOWER(actor_username)",
		"LOWER(actor_display_name)",
		fmt.Sprintf("LOWER(COALESCE(metadata ->> '%s', ''))", "request_path"),
	}
	orClauses := make([]string, 0, len(fields))
	for _, field := range fields {
		*args = append(*args, pattern)
		orClauses = append(orClauses, fmt.Sprintf("%s LIKE $%d%s", field, len(*args), sqlLikeEscapeClause))
	}
	*clauses = append(*clauses, "("+strings.Join(orClauses, " OR ")+")")
}

func addActorFilter(clauses *[]string, args *[]any, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	pattern := "%" + escapeLikePattern(strings.ToLower(strings.TrimSpace(value))) + "%"
	fields := []string{
		"LOWER(actor_username)",
		"LOWER(actor_display_name)",
	}
	orClauses := make([]string, 0, len(fields))
	for _, field := range fields {
		*args = append(*args, pattern)
		orClauses = append(orClauses, fmt.Sprintf("%s LIKE $%d%s", field, len(*args), sqlLikeEscapeClause))
	}
	*clauses = append(*clauses, "("+strings.Join(orClauses, " OR ")+")")
}

func addAnyScalarFilter(clauses *[]string, args *[]any, column string, values []string) {
	if len(values) == 0 {
		return
	}

	orClauses := make([]string, 0, len(values))
	for _, value := range values {
		*args = append(*args, value)
		orClauses = append(orClauses, fmt.Sprintf("%s = $%d", column, len(*args)))
	}
	*clauses = append(*clauses, "("+strings.Join(orClauses, " OR ")+")")
}

func addPrefixAnyJSONMetadataFilter(clauses *[]string, args *[]any, key string, values []string) {
	if len(values) == 0 {
		return
	}

	orClauses := make([]string, 0, len(values))
	for _, value := range values {
		*args = append(*args, escapeLikePattern(strings.ToLower(value))+"%")
		orClauses = append(
			orClauses,
			fmt.Sprintf("LOWER(COALESCE(metadata ->> '%s', '')) LIKE $%d%s", key, len(*args), sqlLikeEscapeClause),
		)
	}
	*clauses = append(*clauses, "("+strings.Join(orClauses, " OR ")+")")
}

func addScalarJSONMetadataFilter(clauses *[]string, args *[]any, key string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	*args = append(*args, strings.TrimSpace(value))
	*clauses = append(*clauses, fmt.Sprintf("COALESCE(metadata ->> '%s', '') = $%d", key, len(*args)))
}

func addAnyExpressionFilter(clauses *[]string, args *[]any, expression string, values []string) {
	if len(values) == 0 {
		return
	}

	orClauses := make([]string, 0, len(values))
	for _, value := range values {
		*args = append(*args, value)
		orClauses = append(orClauses, fmt.Sprintf(expression, len(*args)))
	}
	*clauses = append(*clauses, "("+strings.Join(orClauses, " OR ")+")")
}

func auditResultValues(values []auditstore.AuditResult) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		result = append(result, string(value))
	}
	return result
}

func auditRiskLevelValues(values []auditstore.AuditRiskLevel) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		result = append(result, string(value))
	}
	return result
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
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

func scanAuditLog(
	ctx context.Context,
	localizer *i18n.Service,
	scanner interface {
	Scan(dest ...any) error
},
) (auditstore.AuditLog, error) {
	var (
		record      auditstore.AuditLog
		actorUserID sql.NullInt64
		metadata    []byte
	)
	if err := scanner.Scan(
		&record.ID,
		&record.Source,
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
	enrichAuditLog(ctx, &record, localizer)

	return record, nil
}

func enrichAuditLog(ctx context.Context, record *auditstore.AuditLog, localizer *i18n.Service) {
	if record == nil {
		return
	}

	metadata := decodeAuditMetadata(record.Metadata)
	record.Source = normalizeAuditSource(metadataTextFirst(metadata, "auditSource", "audit_source"))
	record.TraceID = stringMetadataValue(metadata, "trace_id")
	if record.TraceID == "" {
		record.TraceID = record.RequestID
	}
	record.SessionID = stringMetadataValue(metadata, "session_id")
	record.RequestMethod = stringMetadataValue(metadata, "request_method")
	record.RequestPath = stringMetadataValue(metadata, "request_path")
	record.StatusCode = intMetadataValue(metadata, "status_code")
	record.Result = classifyAuditResult(*record, metadata)
	record.RiskLevel = classifyAuditRiskLevel(*record)
	record.TargetType = normalizeAuditTargetType(record.ResourceType)
	record.TargetLabel = firstNonEmpty(record.ResourceName, displayTargetLabel(ctx, localizer, record.TargetType), record.ResourceID)
	record.Target = buildAuditTarget(*record)
}

func buildAuditTarget(record auditstore.AuditLog) auditstore.AuditTarget {
	targetType := firstNonEmpty(record.TargetType, record.ResourceType)
	label := firstNonEmpty(record.TargetLabel, record.ResourceName, record.ResourceID, record.Action)
	target := auditstore.AuditTarget{
		Kind:  "resource",
		Type:  targetType,
		ID:    record.ResourceID,
		Label: label,
	}

	switch {
	case record.RequestID != "":
		target.Kind = "request"
		target.Type = firstNonEmpty(target.Type, "request")
		target.ID = record.RequestID
		target.Label = firstNonEmpty(label, record.RequestID)
	case record.SessionID != "":
		target.Kind = "session"
		target.Type = firstNonEmpty(target.Type, "session")
		target.ID = record.SessionID
		target.Label = firstNonEmpty(label, record.SessionID)
	case record.ActorUserID != nil || record.ActorUsername != "" || record.ActorDisplayName != "":
		target.Kind = "actor"
		target.Type = firstNonEmpty(target.Type, "user")
		if target.ID == "" && record.ActorUserID != nil {
			target.ID = strconv.FormatUint(*record.ActorUserID, 10)
		}
		target.Label = firstNonEmpty(record.ActorDisplayName, record.ActorUsername, target.Label)
	}

	if shouldLinkAuditIncident(record) {
		target.Kind = "incident"
		target.Type = firstNonEmpty(target.Type, "incident")
		target.ID = strconv.FormatUint(record.ID, 10)
		target.Label = firstNonEmpty(target.Label, label, record.Action, target.ID)
		target.RouteRef = strings.Replace(auditcontract.AuditIncidentItem, ":"+auditcontract.AuditIncidentParam, target.ID, 1)
	}

	if target.Label == "" {
		target.Label = firstNonEmpty(target.Type, target.Kind, record.Action)
	}

	return target
}

func shouldLinkAuditIncident(record auditstore.AuditLog) bool {
	switch record.Result {
	case auditstore.AuditResultDenied, auditstore.AuditResultError:
		return true
	}

	switch record.Source {
	case auditstore.AuditSourceSecurityEvent:
		return true
	}

	switch record.RiskLevel {
	case auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
		return true
	}

	return false
}

func (r *repository) readAuditLogByID(ctx context.Context, eventID uint64) (auditstore.AuditLog, error) {
	row := r.db.QueryRowContext(ctx, `SELECT
		id,
		COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') AS source,
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
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC, id DESC
		LIMIT $3`, windowStart, windowEnd, incidentCandidateScanLimit)
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

func decodeAuditMetadata(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return map[string]any{}
	}

	return metadata
}

func stringMetadataValue(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	default:
		return ""
	}
}

func metadataTextFirst(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringMetadataValue(metadata, key); value != "" {
			return value
		}
	}
	return ""
}

func intMetadataValue(metadata map[string]any, key string) int {
	value, ok := metadata[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func classifyAuditResult(record auditstore.AuditLog, metadata map[string]any) auditstore.AuditResult {
	if record.Success {
		return auditstore.AuditResultSuccess
	}

	statusCode := record.StatusCode
	if statusCode == 0 {
		statusCode = intMetadataValue(metadata, "status_code")
	}
	if statusCode == httpStatusForbidden {
		return auditstore.AuditResultDenied
	}
	if statusCode >= 500 || stringMetadataValue(metadata, "error_kind") == "system" || stringMetadataValue(metadata, "error") != "" {
		return auditstore.AuditResultError
	}

	return auditstore.AuditResultFailed
}

func classifyAuditRiskLevel(record auditstore.AuditLog) auditstore.AuditRiskLevel {
	action := strings.ToLower(strings.TrimSpace(record.Action))

	if record.Result == auditstore.AuditResultError || record.Result == auditstore.AuditResultDenied {
		return auditstore.AuditRiskLevelCritical
	}
	if containsAny(action, []string{"reset_password", "update_permission", "update_role", "assign_role", "token_revoke"}) {
		return auditstore.AuditRiskLevelCritical
	}
	if record.Result == auditstore.AuditResultFailed || sensitiveOperationMatch(action) {
		return auditstore.AuditRiskLevelHigh
	}
	if containsAny(action, []string{"login_failed", "login", "permission", "role", "auth"}) {
		return auditstore.AuditRiskLevelMedium
	}
	return auditstore.AuditRiskLevelLow
}

func containsAny(source string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(source, keyword) {
			return true
		}
	}
	return false
}

func sensitiveOperationAuthorityKeywords() []string {
	return []string{"delete", "reset", "grant", "assign", "revoke", "remove", "replace"}
}

func sensitiveOperationMatch(action string) bool {
	return containsAny(action, sensitiveOperationAuthorityKeywords())
}

func normalizeAuditTargetType(resourceType string) string {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "user", "users":
		return "USER"
	case "role", "roles":
		return "ROLE"
	case "permission", "permissions":
		return "PERMISSION"
	case "audit":
		return "AUDIT"
	case "monitor", "server-status", "server_status":
		return "SERVER_STATUS"
	case "auth", "session", "sessions", "login":
		return "AUTH"
	default:
		if resourceType == "" {
			return "AUDIT"
		}
		return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(resourceType), "-", "_"))
	}
}

func displayTargetLabel(ctx context.Context, localizer *i18n.Service, targetType string) string {
	key := targetLabelMessageKey(targetType)
	if key == "" || localizer == nil {
		return ""
	}

	return localizer.Lookup(i18n.LookupRequest{
		Namespace: "audit",
		Locale:    i18n.LocaleTag(auditLocaleFromContext(ctx)),
		Key:       i18n.MessageKey(key),
	})
}

func targetLabelMessageKey(targetType string) string {
	switch targetType {
	case "USER":
		return auditcontract.AuditTargetLabelUser.String()
	case "ROLE":
		return auditcontract.AuditTargetLabelRole.String()
	case "PERMISSION":
		return auditcontract.AuditTargetLabelPermission.String()
	case "AUDIT":
		return auditcontract.AuditTargetLabelAudit.String()
	case "SERVER_STATUS":
		return auditcontract.AuditTargetLabelServerStatus.String()
	case "AUTH":
		return auditcontract.AuditTargetLabelAuth.String()
	default:
		return ""
	}
}

// WithAuditLocale attaches the resolved request locale to one audit read path.
func WithAuditLocale(ctx context.Context, locale string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, auditLocaleContextKey{}, strings.TrimSpace(locale))
}

func auditLocaleFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	locale, _ := ctx.Value(auditLocaleContextKey{}).(string)
	return strings.TrimSpace(locale)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeAuditSource(value string) auditstore.AuditSource {
	switch auditstore.AuditSource(strings.ToUpper(strings.TrimSpace(value))) {
	case auditstore.AuditSourceRequest:
		return auditstore.AuditSourceRequest
	case auditstore.AuditSourceSecurityEvent:
		return auditstore.AuditSourceSecurityEvent
	case auditstore.AuditSourceDomainEvent:
		return auditstore.AuditSourceDomainEvent
	default:
		return ""
	}
}

func auditResultWhereClause() string {
	return auditResultPostgresExpression() + ` = $%d`
}

func auditResultExpression() string {
	return auditResultExpressionFor("success", "metadata")
}

func auditResultExpressionFor(successColumn string, metadataColumn string) string {
	return auditResultExpressionWith(successColumn, metadataColumn, auditPortableMetadataExpressions)
}

func auditResultExpressionWith(
	successColumn string,
	metadataColumn string,
	metadata auditMetadataExpressionBuilder,
) string {
	return `CASE
		WHEN ` + successColumn + ` THEN 'SUCCESS'
		ELSE CASE
			WHEN ` + metadata.textValue(metadataColumn, "status_code") + ` = '403' THEN 'DENIED'
			WHEN ` + metadata.numericAtLeast(metadataColumn, "status_code", httpStatusServerErrorMin) + `
			  OR ` + metadata.textValue(metadataColumn, "error_kind") + ` = 'system'
			  OR ` + metadata.textValue(metadataColumn, "error") + ` <> '' THEN 'ERROR'
			ELSE 'FAILED'
		END
	END`
}

func auditResultPostgresExpression() string {
	return auditResultExpressionWith("success", "metadata", auditPostgresMetadataExpressions)
}

func riskLevelWhereClause() string {
	return auditRiskLevelPostgresExpression() + ` = $%d`
}

func auditRiskLevelExpression() string {
	return auditRiskLevelExpressionFor("success", "action", "metadata")
}

func auditRiskLevelExpressionFor(successColumn string, actionColumn string, metadataColumn string) string {
	return auditRiskLevelExpressionWith(successColumn, actionColumn, metadataColumn, auditPortableMetadataExpressions)
}

func auditRiskLevelExpressionWith(
	successColumn string,
	actionColumn string,
	metadataColumn string,
	metadata auditMetadataExpressionBuilder,
) string {
	return `CASE
		WHEN ` + successColumn + ` = false AND (
			` + metadata.textValue(metadataColumn, "status_code") + ` = '403'
			OR ` + metadata.numericAtLeast(metadataColumn, "status_code", httpStatusServerErrorMin) + `
			OR ` + metadata.textValue(metadataColumn, "error_kind") + ` = 'system'
			OR ` + metadata.textValue(metadataColumn, "error") + ` <> ''
		) THEN 'CRITICAL'
		WHEN LOWER(` + actionColumn + `) LIKE '%%reset_password%%' OR LOWER(` + actionColumn + `) LIKE '%%update_permission%%' OR LOWER(` + actionColumn + `) LIKE '%%update_role%%' OR LOWER(` + actionColumn + `) LIKE '%%assign_role%%' OR LOWER(` + actionColumn + `) LIKE '%%token_revoke%%' THEN 'CRITICAL'
		WHEN ` + successColumn + ` = false OR LOWER(` + actionColumn + `) LIKE '%%delete%%' OR LOWER(` + actionColumn + `) LIKE '%%reset%%' OR LOWER(` + actionColumn + `) LIKE '%%grant%%' OR LOWER(` + actionColumn + `) LIKE '%%assign%%' OR LOWER(` + actionColumn + `) LIKE '%%revoke%%' OR LOWER(` + actionColumn + `) LIKE '%%remove%%' OR LOWER(` + actionColumn + `) LIKE '%%replace%%' THEN 'HIGH'
		WHEN LOWER(` + actionColumn + `) LIKE '%%login_failed%%' OR LOWER(` + actionColumn + `) LIKE '%%login%%' OR LOWER(` + actionColumn + `) LIKE '%%permission%%' OR LOWER(` + actionColumn + `) LIKE '%%role%%' OR LOWER(` + actionColumn + `) LIKE '%%auth%%' THEN 'MEDIUM'
		ELSE 'LOW'
	END`
}

func auditRiskLevelPostgresExpression() string {
	return auditRiskLevelExpressionWith("success", "action", "metadata", auditPostgresMetadataExpressions)
}

func auditOverviewTrendResultExpression() string {
	return auditResultExpressionFor("logs.success", "logs.metadata")
}

func auditOverviewTrendRiskLevelExpression() string {
	return auditRiskLevelExpressionFor("logs.success", "logs.action", "logs.metadata")
}

func sourceWhereClause() string {
	return `COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') = $%d`
}

var overviewSummarySQL = `
SELECT
	COUNT(*) AS total_logs,
	COUNT(*) FILTER (
		WHERE ` + auditResultExpression() + ` IN ('FAILED', 'DENIED', 'ERROR')
	) AS failed_operations,
	COUNT(*) FILTER (
		WHERE ` + auditRiskLevelExpression() + ` IN ('HIGH', 'CRITICAL')
	) AS high_risk_events,
	COUNT(*) FILTER (
		WHERE ` + sensitiveOperationsWhereClause() + `
	) AS sensitive_operations
FROM audit_logs
WHERE created_at >= $1
`

const overviewRecentBaseSQL = `
SELECT
	id,
	COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') AS source,
	actor_user_id,
	actor_username,
	actor_display_name,
	action,
	resource_type,
	resource_id,
	resource_name,
	success,
	request_id,
	message,
	metadata,
	created_at
FROM audit_logs
WHERE created_at >= $1 AND %s
ORDER BY created_at DESC, id DESC
LIMIT 3
`

func metadataTextValueSQL(column string, key string) string {
	return fmt.Sprintf("COALESCE(%s ->> '%s', '')", column, key)
}

type auditMetadataExpressionBuilder struct {
	textValue      func(column string, key string) string
	numericAtLeast func(column string, key string, threshold int) string
}

var (
	auditPortableMetadataExpressions = auditMetadataExpressionBuilder{
		textValue:      metadataTextValueSQL,
		numericAtLeast: metadataNumericAtLeastSQL,
	}
	auditPostgresMetadataExpressions = auditMetadataExpressionBuilder{
		textValue:      metadataTextValueSQL,
		numericAtLeast: metadataPostgresNumericAtLeastSQL,
	}
)

func metadataNumericAtLeastSQL(column string, key string, threshold int) string {
	return fmt.Sprintf("%s >= %d", metadataNumericValueSQL(column, key), threshold)
}

func metadataPostgresNumericAtLeastSQL(column string, key string, threshold int) string {
	return fmt.Sprintf(`(
				COALESCE(%[1]s ->> '%[2]s', '') ~ '^[0-9]+$'
				AND (%[1]s ->> '%[2]s')::int >= %[3]d
			)`, column, key, threshold)
}

func metadataNumericValueSQL(column string, key string) string {
	return fmt.Sprintf(`CASE
		WHEN COALESCE(NULLIF(%[1]s ->> '%[2]s', ''), '') <> ''
			AND REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(
				%[1]s ->> '%[2]s',
				'0', ''
			), '1', ''), '2', ''), '3', ''), '4', ''), '5', ''), '6', ''), '7', ''), '8', ''), '9', '') = ''
		THEN CAST(%[1]s ->> '%[2]s' AS INTEGER)
		ELSE 0
	END`, column, key)
}

var (
	overviewMetadataRequestPathSQL = metadataTextValueSQL("metadata", "request_path")
	overviewMetadataStatusCodeSQL  = metadataTextValueSQL("metadata", "status_code")
)

//nolint:gosec // Query text is assembled from fixed SQL fragments; all dynamic values stay parameterized.
var overviewRiskGroupsSQL = `
SELECT key, label_key, risk_level, count
FROM (
	SELECT
		'critical_security' AS key,
		'audit.overview.riskGroups.criticalSecurity' AS label_key,
		'CRITICAL' AS risk_level,
		COUNT(*) FILTER (
			WHERE success = false
			  AND (
				(metadata ->> 'status_code') = '403'
				OR (
					COALESCE(NULLIF(metadata ->> 'status_code', ''), '') <> ''
					AND REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(
						metadata ->> 'status_code',
						'0', ''
					), '1', ''), '2', ''), '3', ''), '4', ''), '5', ''), '6', ''), '7', ''), '8', ''), '9', '') = ''
					AND CAST(metadata ->> 'status_code' AS INTEGER) >= 500
				)
				OR COALESCE(metadata ->> 'error_kind', '') = 'system'
				OR COALESCE(metadata ->> 'error', '') <> ''
			  )
		) AS count
	FROM audit_logs
	WHERE created_at >= $1
	UNION ALL
	SELECT
		'high_risk_operations',
		'audit.overview.riskGroups.highRiskOperations',
		'HIGH',
		COUNT(*) FILTER (
			WHERE ` + auditRiskLevelExpression() + ` IN ('HIGH', 'CRITICAL')
		)
	FROM audit_logs
	WHERE created_at >= $1
	UNION ALL
	SELECT
		'auth_failures',
		'audit.overview.riskGroups.authFailures',
		'HIGH',
		COUNT(*) FILTER (WHERE ` + authFailuresWhereClause() + `)
	FROM audit_logs
	WHERE created_at >= $1
	UNION ALL
	SELECT
		'permission_denials',
		'audit.overview.riskGroups.permissionDenials',
		'CRITICAL',
		COUNT(*) FILTER (WHERE ` + permissionDenialsWhereClause() + `)
	FROM audit_logs
	WHERE created_at >= $1
) groups
WHERE count > 0
ORDER BY count DESC, key ASC
LIMIT 4
`

var overviewSecurityTimelineSQL = `
SELECT
	id,
	created_at,
	COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') AS source,
	action,
	request_id,
	actor_display_name,
	actor_username,
	resource_name,
	resource_type,
	success,
	message,
	metadata
FROM audit_logs
WHERE created_at >= $1
  AND (
	COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') = 'SECURITY_EVENT'
	OR NOT success
	OR LOWER(action) LIKE '%delete%'
	OR LOWER(action) LIKE '%reset%'
	OR LOWER(action) LIKE '%grant%'
	OR LOWER(action) LIKE '%assign%'
	OR LOWER(action) LIKE '%revoke%'
	OR LOWER(action) LIKE '%remove%'
	OR LOWER(action) LIKE '%replace%'
  )
ORDER BY created_at DESC, id DESC
LIMIT 6
`

func (r *repository) readAuditOverviewSummary(ctx context.Context, args []any) (auditstore.OverviewSummary, error) {
	var summary auditstore.OverviewSummary
	if err := r.db.QueryRowContext(ctx, overviewSummarySQL, args...).Scan(
		&summary.TotalLogs,
		&summary.FailedOperations,
		&summary.HighRiskEvents,
		&summary.SensitiveOperations,
	); err != nil {
		return auditstore.OverviewSummary{}, fmt.Errorf("read audit overview summary: %w", err)
	}
	return summary, nil
}

func (r *repository) readOverviewRiskGroups(ctx context.Context, args []any) ([]auditstore.OverviewRiskGroup, error) {
	rows, err := r.db.QueryContext(ctx, overviewRiskGroupsSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("read audit overview risk groups: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	groups := make([]auditstore.OverviewRiskGroup, 0, overviewRiskGroupLimit)
	for rows.Next() {
		var group auditstore.OverviewRiskGroup
		if err := rows.Scan(&group.Key, &group.LabelKey, &group.RiskLevel, &group.Count); err != nil {
			return nil, fmt.Errorf("scan audit overview risk group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit overview risk groups: %w", err)
	}

	return groups, nil
}

func (r *repository) readOverviewTrend(
	ctx context.Context,
	preset auditstore.AuditTimePreset,
	startedAt time.Time,
	now time.Time,
) (auditstore.OverviewTrend, error) {
	bucketUnit, bucketSize, step := overviewTrendConfig(preset)
	seriesSQL := overviewTrendSeriesSQL(step)

	rows, err := r.db.QueryContext(ctx, seriesSQL, startedAt, now)
	if err != nil {
		return r.readOverviewTrendFallback(ctx, startedAt, now, bucketUnit, bucketSize, step)
	}
	defer func() {
		_ = rows.Close()
	}()

	points := make([]auditstore.OverviewTrendPoint, 0, overviewTrendPointLimit)
	for rows.Next() {
		var point auditstore.OverviewTrendPoint
		if err := rows.Scan(&point.BucketStart, &point.BucketEnd, &point.Total, &point.Failed, &point.HighRisk, &point.SecurityEvents); err != nil {
			return auditstore.OverviewTrend{}, fmt.Errorf("scan audit overview trend: %w", err)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return auditstore.OverviewTrend{}, fmt.Errorf("iterate audit overview trend: %w", err)
	}

	return auditstore.OverviewTrend{
		BucketUnit: bucketUnit,
		BucketSize: bucketSize,
		Points:     points,
	}, nil
}

func overviewTrendSeriesSQL(step string) string {
	//nolint:gosec // step comes from overviewTrendConfig and is limited to fixed internal interval literals.
	return fmt.Sprintf(`
SELECT
	bucket_start,
	bucket_start + INTERVAL '%[1]s' AS bucket_end,
	COUNT(logs.id) AS total,
	COUNT(logs.id) FILTER (
		WHERE logs.id IS NOT NULL
		  AND `+auditOverviewTrendResultExpression()+` IN ('FAILED', 'DENIED', 'ERROR')
	) AS failed,
	COUNT(logs.id) FILTER (
		WHERE logs.id IS NOT NULL
		  AND `+auditOverviewTrendRiskLevelExpression()+` IN ('HIGH', 'CRITICAL')
	) AS high_risk,
	COUNT(*) FILTER (
		WHERE COALESCE(logs.metadata ->> 'auditSource', logs.metadata ->> 'audit_source', '') = 'SECURITY_EVENT'
	) AS security_events
FROM generate_series($1::timestamptz, $2::timestamptz - INTERVAL '%[1]s', INTERVAL '%[1]s') AS bucket_start
LEFT JOIN audit_logs logs
	ON logs.created_at >= bucket_start
	AND logs.created_at < bucket_start + INTERVAL '%[1]s'
GROUP BY bucket_start
ORDER BY bucket_start ASC
`, step)
}

func buildOverviewTrendPoints(startedAt time.Time, now time.Time, stepDuration time.Duration) []auditstore.OverviewTrendPoint {
	points := make([]auditstore.OverviewTrendPoint, 0, overviewTrendPointLimit)
	for bucketStart := startedAt; bucketStart.Before(now); bucketStart = bucketStart.Add(stepDuration) {
		bucketEnd := bucketStart.Add(stepDuration)
		if bucketEnd.After(now) {
			bucketEnd = now
		}
		points = append(points, auditstore.OverviewTrendPoint{
			BucketStart: bucketStart,
			BucketEnd:   bucketEnd,
		})
	}

	return points
}

func applyOverviewTrendRecord(points []auditstore.OverviewTrendPoint, record auditstore.AuditLog, startedAt time.Time, stepDuration time.Duration) {
	index := int(record.CreatedAt.Sub(startedAt) / stepDuration)
	if index < 0 || index >= len(points) {
		return
	}

	points[index].Total++
	if !record.Success {
		points[index].Failed++
	}
	if record.RiskLevel == auditstore.AuditRiskLevelHigh || record.RiskLevel == auditstore.AuditRiskLevelCritical {
		points[index].HighRisk++
	}
	if record.Source == auditstore.AuditSourceSecurityEvent {
		points[index].SecurityEvents++
	}
}

func (r *repository) readOverviewTrendFallback(
	ctx context.Context,
	startedAt time.Time,
	now time.Time,
	bucketUnit string,
	bucketSize int,
	step string,
) (auditstore.OverviewTrend, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT
	id,
	action,
	success,
	request_id,
	resource_type,
	resource_id,
	resource_name,
	actor_username,
	actor_display_name,
	message,
	metadata,
	created_at
FROM audit_logs
WHERE created_at >= $1 AND created_at < $2
ORDER BY created_at ASC, id ASC
`, startedAt, now)
	if err != nil {
		return auditstore.OverviewTrend{}, fmt.Errorf("read audit overview trend: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	stepDuration := parseOverviewTrendStep(step)
	points := buildOverviewTrendPoints(startedAt, now, stepDuration)

	for rows.Next() {
		record, scanErr := scanAuditTrendRecord(rows)
		if scanErr != nil {
			return auditstore.OverviewTrend{}, scanErr
		}
		enrichAuditLog(ctx, &record, r.localizer)
		applyOverviewTrendRecord(points, record, startedAt, stepDuration)
	}
	if err := rows.Err(); err != nil {
		return auditstore.OverviewTrend{}, fmt.Errorf("iterate audit overview trend: %w", err)
	}

	return auditstore.OverviewTrend{
		BucketUnit: bucketUnit,
		BucketSize: bucketSize,
		Points:     points,
	}, nil
}

func parseOverviewTrendStep(step string) time.Duration {
	switch step {
	case overviewTrendDayStep:
		return overviewTrendOneDayDuration
	case overviewTrendThreeDayStep:
		return overviewTrendThreeDayDuration
	default:
		return overviewTrendTwoHourDuration
	}
}

func scanAuditTrendRecord(scanner interface {
	Scan(dest ...any) error
}) (auditstore.AuditLog, error) {
	var (
		record   auditstore.AuditLog
		metadata []byte
	)
	if err := scanner.Scan(
		&record.ID,
		&record.Action,
		&record.Success,
		&record.RequestID,
		&record.ResourceType,
		&record.ResourceID,
		&record.ResourceName,
		&record.ActorUsername,
		&record.ActorDisplayName,
		&record.Message,
		&metadata,
		&record.CreatedAt,
	); err != nil {
		return auditstore.AuditLog{}, fmt.Errorf("scan audit overview trend record: %w", err)
	}
	record.Metadata = cloneRawMessage(metadata)
	return record, nil
}

func overviewTrendConfig(preset auditstore.AuditTimePreset) (string, int, string) {
	switch preset {
	case auditstore.AuditTimePresetLast7Days:
		return "day", overviewTrendDayBucketSize, overviewTrendDayStep
	case auditstore.AuditTimePresetLast30Days:
		return "day", overviewTrendThreeDayBucketSize, overviewTrendThreeDayStep
	default:
		return "hour", overviewTrendTwoHourBucketSize, overviewTrendTwoHourStep
	}
}

func (r *repository) readOverviewSecurityTimeline(ctx context.Context, args []any) ([]auditstore.OverviewSecurityTimelineItem, error) {
	rows, err := r.db.QueryContext(ctx, overviewSecurityTimelineSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("read audit overview security timeline: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]auditstore.OverviewSecurityTimelineItem, 0, overviewSecurityTimelineLimit)
	for rows.Next() {
		var (
			item     auditstore.OverviewSecurityTimelineItem
			success  bool
			message  string
			metadata []byte
		)
		if err := rows.Scan(
			&item.ID,
			&item.CreatedAt,
			&item.Source,
			&item.Action,
			&item.RequestID,
			&item.ActorDisplayName,
			&item.ActorUsername,
			&item.ResourceName,
			&item.ResourceType,
			&success,
			&message,
			&metadata,
		); err != nil {
			return nil, fmt.Errorf("scan audit overview security timeline: %w", err)
		}

		record := auditstore.AuditLog{
			ID:               item.ID,
			Source:           item.Source,
			Action:           item.Action,
			ResourceName:     item.ResourceName,
			ResourceType:     item.ResourceType,
			Success:          success,
			RequestID:        item.RequestID,
			ActorDisplayName: item.ActorDisplayName,
			ActorUsername:    item.ActorUsername,
			Message:          message,
			Metadata:         cloneRawMessage(metadata),
			CreatedAt:        item.CreatedAt,
		}
		enrichAuditLog(ctx, &record, r.localizer)
		item.Source = record.Source
		item.RiskLevel = record.RiskLevel
		item.Result = record.Result
		if item.ResourceName == "" {
			item.ResourceName = firstNonEmpty(record.TargetLabel, record.ResourceType)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit overview security timeline: %w", err)
	}

	return items, nil
}

func (r *repository) readAuditOverviewItems(ctx context.Context, args []any, where string) ([]auditstore.OverviewItem, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(overviewRecentBaseSQL, where), args...)
	if err != nil {
		return nil, fmt.Errorf("read audit overview items: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]auditstore.OverviewItem, 0, overviewRecentLimit)
	for rows.Next() {
		item, scanErr := scanAuditOverviewItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit overview items: %w", err)
	}

	return items, nil
}

func scanAuditOverviewItem(scanner interface {
	Scan(dest ...any) error
}) (auditstore.OverviewItem, error) {
	var (
		item        auditstore.OverviewItem
		actorUserID sql.NullInt64
		metadata    []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Source,
		&actorUserID,
		&item.ActorUsername,
		&item.ActorDisplayName,
		&item.Action,
		&item.ResourceType,
		&item.ResourceID,
		&item.ResourceName,
		&item.Success,
		&item.RequestID,
		&item.Message,
		&metadata,
		&item.CreatedAt,
	); err != nil {
		return auditstore.OverviewItem{}, fmt.Errorf("scan audit overview item: %w", err)
	}

	if actorUserID.Valid {
		value := toStoreID(actorUserID.Int64)
		item.ActorUserID = &value
	}
	item.Metadata = cloneRawMessage(metadata)
	return item, nil
}

func nullableUint64(value *uint64) (any, error) {
	if value == nil {
		return nil, nil
	}
	if *value > math.MaxInt64 {
		return nil, fmt.Errorf("actor user id %d exceeds bigint range", *value)
	}

	return *value, nil
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
