package storeent

import (
	"fmt"
	"strings"
	"time"

	auditstore "graft/server/modules/audit/store"
)

// buildAuditLogFilters 根据查询条件构建审计日志过滤条件的 WHERE 子句及参数列表。
// 当未生成任何过滤条件时，返回空字符串和参数列表；否则返回以 `WHERE` 开头的拼接结果。
func buildAuditLogFilters(query auditstore.ListAuditLogsQuery) (string, []any) {
	clauses := make([]string, 0, defaultFilterCapacity)
	args := make([]any, 0, defaultFilterCapacity)

	add := func(format string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(format, len(args)))
	}

	addAuditVisibilityFilter(&clauses, &args, query.VisibilityScope)
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

// addAuditPresetRange 在未指定显式时间范围时，按时间预设添加创建时间下限过滤条件。
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

// 其他情况仅匹配可见审计日志。
func addAuditVisibilityFilter(
	clauses *[]string,
	args *[]any,
	scope auditstore.AuditVisibilityScope,
) {
	switch scope {
	case auditstore.AuditVisibilityScopeAll:
		return
	case auditstore.AuditVisibilityScopeHiddenOnly:
		*args = append(*args, string(auditstore.AuditVisibilityStrategyHidden))
		*clauses = append(*clauses, fmt.Sprintf("visibility = $%d", len(*args)))
	default:
		*args = append(*args, string(auditstore.AuditVisibilityStrategyVisible))
		*clauses = append(*clauses, fmt.Sprintf("visibility = $%d", len(*args)))
	}
}

// auditPresetStart 根据时间预设计算查询起始时间。
// auditPresetStart 返回指定时间预设对应的起始时间。
// 对于最近 24 小时、7 天、30 天分别返回相对于 now 的起点；其他预设返回零时间。
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

// highRiskOperationsWhereClause 返回高风险操作分类对应的 SQL WHERE 子句。
// 该条件直接复用标准化风险等级表达式，避免与运行时分类规则漂移。
func highRiskOperationsWhereClause() string {
	return `(` + auditRiskLevelExpression() + ` IN ('HIGH', 'CRITICAL'))`
}

// failedOperationsWhereClause 返回用于筛选失败、拒绝或错误审计结果的 SQL WHERE 子句。
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
