package storeent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	auditstore "graft/server/modules/audit/store"
)

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
WHERE visibility = 'visible' AND created_at >= $1
`

const overviewRecentBaseSQL = `
SELECT
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
	message,
	metadata,
	created_at
FROM audit_logs
WHERE visibility = 'visible' AND created_at >= $1 AND %s
ORDER BY created_at DESC, id DESC
LIMIT 3
`

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
	WHERE visibility = 'visible' AND created_at >= $1
	UNION ALL
	SELECT
		'high_risk_operations',
		'audit.overview.riskGroups.highRiskOperations',
		'HIGH',
		COUNT(*) FILTER (
			WHERE ` + auditRiskLevelExpression() + ` IN ('HIGH', 'CRITICAL')
		)
	FROM audit_logs
	WHERE visibility = 'visible' AND created_at >= $1
	UNION ALL
	SELECT
		'auth_failures',
		'audit.overview.riskGroups.authFailures',
		'HIGH',
		COUNT(*) FILTER (WHERE ` + authFailuresWhereClause() + `)
	FROM audit_logs
	WHERE visibility = 'visible' AND created_at >= $1
	UNION ALL
	SELECT
		'permission_denials',
		'audit.overview.riskGroups.permissionDenials',
		'CRITICAL',
		COUNT(*) FILTER (WHERE ` + permissionDenialsWhereClause() + `)
	FROM audit_logs
	WHERE visibility = 'visible' AND created_at >= $1
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
WHERE visibility = 'visible'
  AND created_at >= $1
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

// overviewTrendSeriesSQL 生成按固定时间步长聚合审计日志概览趋势的 SQL。
// 查询结果按时间桶返回每个区间的起始和结束时间，以及该区间内的总数、失败数、高风险数和安全事件数。
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
	AND logs.visibility = 'visible'
GROUP BY bucket_start
ORDER BY bucket_start ASC
`, step)
}

// buildOverviewTrendPoints 生成从 startedAt 到 now 之前的连续趋势桶。
// 每个桶的结束时间不会超过 now。
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

// applyOverviewTrendRecord 将审计日志累加到对应的趋势桶中。
// 统计项包括总数、失败数、高风险数和安全事件数。
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
WHERE visibility = 'visible'
  AND created_at >= $1 AND created_at < $2
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

// parseOverviewTrendStep 将趋势步长标识映射为对应的时间间隔。
// @return 对应的时间间隔；`overviewTrendDayStep` 返回 `overviewTrendOneDayDuration`，`overviewTrendThreeDayStep` 返回 `overviewTrendThreeDayDuration`，其他值返回 `overviewTrendTwoHourDuration`。
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

// scanAuditTrendRecord 从行数据中解析一条审计趋势记录，并保留原始元数据。
// @returns auditstore.AuditLog 解析得到的审计日志记录；error 扫描失败时返回的错误。
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

// overviewTrendConfig 根据时间预设选择趋势图的桶单位、桶大小和步长。
//
// @returns 桶单位、桶大小和步长。
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

// scanAuditOverviewItem 解析一条审计概览记录并补齐元数据。
// 它会将 actor 用户 ID 转换为可选值，并复制 metadata 以避免共享底层字节。
// scanAuditOverviewItem 解析一条概览条目记录，并补充可选的演员用户 ID 和元数据副本。
// 当 actor_user_id 有值时，将其转换后写入 ActorUserID；metadata 会被克隆后保存到结果中。
// 解析失败时返回包装后的错误。
func scanAuditOverviewItem(scanner interface {
	Scan(dest ...any) error
}) (auditstore.OverviewItem, error) {
	var (
		item        auditstore.OverviewItem
		actorUserID sql.NullInt64
		visibility  string
		metadata    []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Source,
		&visibility,
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
	_ = visibility
	return item, nil
}
