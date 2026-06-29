// Package storeent 提供 audit 模块基于 SQL 的 repository 实现。
package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
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
		Visibility:       normalizeStoredAuditVisibility(input.Visibility),
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
			visibility,
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id`,
		actorUserID,
		input.ActorUsername,
		input.ActorDisplayName,
		input.Action,
		record.Visibility,
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
		record, scanErr := scanAuditLog(ctx, r.localizer, rows)
		if scanErr != nil {
			return auditstore.ListAuditLogsResult{}, scanErr
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

	return r.readAuditLogByID(ctx, id)
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

// incidentEvidenceWindow 返回事件集合对应的证据时间窗口。
// 当事件为空，或无法确定开始/结束时间时，返回 nil。
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

// auditResultWhereClause 返回用于按审计结果筛选的 SQL 条件片段。
// 它将审计结果表达式与参数占位符组合为可直接拼接到 WHERE 子句中的比较表达式。
func auditResultWhereClause() string {
	return auditResultPostgresExpression() + ` = $%d`
}

// auditResultExpression 返回用于从 success 和 metadata 推导审计结果分类的 SQL 表达式。
func auditResultExpression() string {
	return auditResultExpressionFor("success", "metadata")
}

// auditResultExpressionFor 生成基于成功列和元数据列的审计结果分类 SQL 表达式。
// 它使用可移植的元数据解析规则，将记录归类为 SUCCESS、DENIED、ERROR 或 FAILED。
func auditResultExpressionFor(successColumn string, metadataColumn string) string {
	return auditResultExpressionWith(successColumn, metadataColumn, auditPortableMetadataExpressions)
}

// 当成功列为真时返回 'SUCCESS'；否则根据 metadata 中的状态码、错误类型和错误内容分别归类为 'DENIED'、'ERROR' 或 'FAILED'。
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

// auditResultPostgresExpression 生成用于 PostgreSQL 审计记录结果分类的 SQL 表达式。
// @returns 基于 `success` 和 `metadata` 的结果分类表达式。
func auditResultPostgresExpression() string {
	return auditResultExpressionWith("success", "metadata", auditPostgresMetadataExpressions)
}

// riskLevelWhereClause 返回用于按风险等级筛选的 SQL 条件表达式。
func riskLevelWhereClause() string {
	return auditRiskLevelPostgresExpression() + ` = $%d`
}

// auditRiskLevelExpression 返回用于推导审计风险等级的 SQL 表达式。
//
// @returns 基于 success、action、resource_type 和 metadata 生成的风险等级分类 SQL。
func auditRiskLevelExpression() string {
	return auditRiskLevelExpressionFor("success", "action", "resource_type", "metadata")
}

// 该表达式使用可移植的元数据解析方式，适用于不同数据库方言。
func auditRiskLevelExpressionFor(successColumn string, actionColumn string, resourceTypeColumn string, metadataColumn string) string {
	return auditRiskLevelExpressionWith(successColumn, actionColumn, resourceTypeColumn, metadataColumn, auditPortableMetadataExpressions)
}

// auditRiskLevelExpressionWith 生成用于推导审计风险等级的 CASE SQL 表达式。
// 该表达式会结合成功标记、操作类型、资源类型和元数据，将记录分类为 CRITICAL、HIGH、MEDIUM 或 LOW。
func auditRiskLevelExpressionWith(
	successColumn string,
	actionColumn string,
	resourceTypeColumn string,
	metadataColumn string,
	metadata auditMetadataExpressionBuilder,
) string {
	normalizedAction := normalizedAuditClassifierColumn(actionColumn)
	return `CASE
		WHEN ` + successColumn + ` = false AND (
			` + metadata.textValue(metadataColumn, "status_code") + ` = '403'
			OR ` + metadata.numericAtLeast(metadataColumn, "status_code", httpStatusServerErrorMin) + `
			OR ` + metadata.textValue(metadataColumn, "error_kind") + ` = 'system'
			OR ` + metadata.textValue(metadataColumn, "error") + ` <> ''
		) THEN 'CRITICAL'
		WHEN ` + containerDangerousActionExpression(actionColumn, resourceTypeColumn) + ` THEN 'HIGH'
		WHEN ` + normalizedAction + ` LIKE '%%reset_password%%' OR ` + normalizedAction + ` LIKE '%%update_permission%%' OR ` + normalizedAction + ` LIKE '%%update_role%%' OR ` + normalizedAction + ` LIKE '%%assign_role%%' OR ` + normalizedAction + ` LIKE '%%token_revoke%%' THEN 'CRITICAL'
		WHEN ` + successColumn + ` = false OR ` + normalizedAction + ` LIKE '%%delete%%' OR ` + normalizedAction + ` LIKE '%%reset%%' OR ` + normalizedAction + ` LIKE '%%grant%%' OR ` + normalizedAction + ` LIKE '%%assign%%' OR ` + normalizedAction + ` LIKE '%%revoke%%' OR ` + normalizedAction + ` LIKE '%%remove%%' OR ` + normalizedAction + ` LIKE '%%replace%%' THEN 'HIGH'
		WHEN ` + normalizedAction + ` LIKE '%%login_failed%%' OR ` + normalizedAction + ` LIKE '%%login%%' OR ` + normalizedAction + ` LIKE '%%permission%%' OR ` + normalizedAction + ` LIKE '%%role%%' OR ` + normalizedAction + ` LIKE '%%auth%%' THEN 'MEDIUM'
		ELSE 'LOW'
	END`
}

// auditRiskLevelPostgresExpression 返回用于 PostgreSQL 审计日志风险等级分类的 SQL 表达式。
func auditRiskLevelPostgresExpression() string {
	return auditRiskLevelExpressionWith("success", "action", "resource_type", "metadata", auditPostgresMetadataExpressions)
}

// auditOverviewTrendResultExpression 生成审计概览趋势使用的结果分类 SQL 表达式。
// 它根据日志记录的 success 和 metadata 字段推导审计结果。
func auditOverviewTrendResultExpression() string {
	return auditResultExpressionFor("logs.success", "logs.metadata")
}

// auditOverviewTrendRiskLevelExpression 返回用于审计概览趋势统计的风险等级分类 SQL 表达式。
func auditOverviewTrendRiskLevelExpression() string {
	return auditRiskLevelExpressionFor("logs.success", "logs.action", "logs.resource_type", "logs.metadata")
}

// sourceWhereClause 返回用于按审计来源筛选的 SQL 条件表达式。
func sourceWhereClause() string {
	return `COALESCE(metadata ->> 'auditSource', metadata ->> 'audit_source', '') = $%d`
}

// metadataTextValueSQL 返回从 JSON 列中提取指定键文本值并以空字符串兜底的 SQL 片段。
// @param column JSON 列名。
// @param key JSON 键名。
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

// metadataNumericAtLeastSQL 生成用于判断元数据数值是否达到阈值的 SQL 表达式。
// @returns 与阈值进行大于等于比较的 SQL 片段。
func metadataNumericAtLeastSQL(column string, key string, threshold int) string {
	return fmt.Sprintf("%s >= %d", metadataNumericValueSQL(column, key), threshold)
}

// metadataPostgresNumericAtLeastSQL 返回用于判断 JSON 元数据中指定字段是否大于等于阈值的 PostgreSQL 表达式。
// 该表达式会先确认字段值仅包含数字，再将其转换为整数进行比较。
func metadataPostgresNumericAtLeastSQL(column string, key string, threshold int) string {
	return fmt.Sprintf(`(
				COALESCE(%[1]s ->> '%[2]s', '') ~ '^[0-9]+$'
				AND (%[1]s ->> '%[2]s')::int >= %[3]d
			)`, column, key, threshold)
}

// metadataNumericValueSQL 返回将 JSON 元数据字段解析为整数的 SQL 表达式。
// 当字段值为空或包含非数字字符时，表达式结果为 0。
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

// nullableUint64 将可选的 uint64 转换为可用于数据库参数绑定的值。
// nullableUint64 将 uint64 指针转换为数据库可绑定值。
// 当 value 为 nil 时返回 nil；当值大于 bigint 可表示范围时返回错误。
func nullableUint64(value *uint64) (any, error) {
	if value == nil {
		return nil, nil
	}
	if *value > math.MaxInt64 {
		return nil, fmt.Errorf("actor user id %d exceeds bigint range", *value)
	}

	return *value, nil
}

// toStoreID 将数据库中的 ID 转为 uint64。
func toStoreID(id int64) uint64 {
	//nolint:gosec // 数据库 ID 来自受控 schema，并保持为正数。
	return uint64(id)
}
