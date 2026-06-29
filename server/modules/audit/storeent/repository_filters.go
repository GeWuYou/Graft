package storeent

import (
	"fmt"
	"strings"
	"time"

	auditstore "graft/server/modules/audit/store"
)

// buildAuditLogFilters 根据查询条件构建审计日志过滤条件的 WHERE 子句及参数列表。
// 当没有任何过滤条件时，返回空字符串和参数列表；否则返回以 `WHERE` 开头的拼接结果。
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

// validateListAuditLogsQuery 校验“列出审计日志”查询参数是否合法。
// 它检查分页参数、时间预设和排序字段是否符合允许范围。
//
// @return 若所有参数都有效则返回 nil，否则返回描述具体非法字段和值的错误。
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

// isSupportedAuditTimePreset 判断审计时间预设是否受支持。
// 支持 Last24Hours、Last7Days 和 Last30Days。
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

// addAuditPresetRange 在未指定创建时间区间时，按时间预设添加创建时间下限过滤条件。
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

// addAuditVisibilityFilter 按可见性范围追加审计日志过滤条件。
// 其中，All 不添加条件，HiddenOnly 仅匹配隐藏审计日志，其余情况仅匹配可见审计日志。
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
// auditPresetStart 返回指定时间预设对应的起始时间。
// 对于最近 24 小时、7 天和 30 天，返回相对于 now 的起点；其他预设返回零时间。
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
// highRiskOperationsWhereClause 返回匹配高风险操作的 SQL 条件。
//
// 该条件检查审计风险等级表达式是否为 `HIGH` 或 `CRITICAL`。
func highRiskOperationsWhereClause() string {
	return `(` + auditRiskLevelExpression() + ` IN ('HIGH', 'CRITICAL'))`
}

// failedOperationsWhereClause 返回用于筛选失败、拒绝或错误审计结果的 SQL WHERE 子句。
func failedOperationsWhereClause() string {
	return `(` + auditResultExpression() + ` IN ('FAILED', 'DENIED', 'ERROR'))`
}

// sensitiveOperationsWhereClause 返回匹配敏感操作权限关键词的 SQL 条件。
// 该条件对 action 进行大小写不敏感匹配，并将多个关键词用 OR 组合。
func sensitiveOperationsWhereClause() string {
	keywords := sensitiveOperationAuthorityKeywords()
	orClauses := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		orClauses = append(orClauses, fmt.Sprintf("LOWER(action) LIKE '%%%s%%'", strings.ToLower(keyword)))
	}
	return "(" + strings.Join(orClauses, "\n\t\tOR ") + ")"
}

// authFailuresWhereClause 返回用于鉴权失败和认证失败事件的 SQL 条件。
// 条件要求 success = false，并匹配 action、resource_type 或请求路径中的鉴权相关特征。
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

// permissionDenialsWhereClause 返回用于标识权限拒绝事件的 SQL 条件。
// 条件要求 `success = false`，并匹配状态码为 `403`、消息为 `common.forbidden`，或消息包含 `forbidden`、`permission` 的记录。
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

// rbacChangesWhereClause 返回匹配 RBAC 相关变更操作前缀的 SQL 条件。
func rbacChangesWhereClause() string {
	return `(
		LOWER(action) LIKE 'rbac.%'
		OR LOWER(action) LIKE 'role.%'
		OR LOWER(action) LIKE 'permission.%'
	)`
}

// criticalSecurityWhereClause 返回用于识别关键安全事件的 SQL 条件。
// 该条件要求成功状态为 false，并匹配 403 状态码、状态码大于等于 500 的元数据值、system 错误类型或非空错误信息。
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

// addBusinessCategoryFilter 向条件列表追加指定业务分类对应的审计过滤条件。
// 不支持的分类不会追加任何条件。
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

// addScalarFilter 在值非空时追加一个标量筛选条件。
func addScalarFilter(add func(string, any), format string, value string) {
	if value == "" {
		return
	}
	add(format, value)
}

// addPrefixFilter 将前缀匹配条件追加到参数列表中。
// 当 value 为空时不添加任何条件；否则将其作为 SQL LIKE 前缀模式写入。
func addPrefixFilter(add func(string, any), format string, value string) {
	if value == "" {
		return
	}

	add(format, escapeLikePattern(value)+"%")
}

// addPrefixAnyFilter 为给定列追加前缀匹配的 OR 条件。
// 每个值都会被转义后作为 `LIKE` 参数，所有条件会合并为一个括号包裹的表达式。
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

// addKeywordAnyFilter 为指定列添加多个关键字的大小写不敏感匹配条件。
//
// values 为空时不追加任何条件。每个值都会被转换为包含前后通配符的 LIKE 模式，并与 column 进行 OR 组合。
//
// @param clauses SQL 条件片段集合。
// @param args 参数值集合。
// @param column 参与匹配的列名。
// @param values 关键字列表。
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

// addKeywordFilter 将关键字按大小写不敏感方式匹配到多个审计日志字段。
// 它会为动作、请求 ID、消息、资源信息、操作者信息以及请求路径元数据生成一个 OR 条件，并为每个字段追加相同的 LIKE 参数。
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

// addActorFilter 为操作者名称和显示名添加大小写不敏感的模糊匹配条件。
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

// addAnyScalarFilter 为多个标量值追加按等号匹配的 OR 条件。
//
// 每个值都会作为一个独立参数加入 args，并将对应的列比较条件合并为一组括号内的 OR 表达式。
//
// 当 values 为空时不做任何修改。
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

// addPrefixAnyJSONMetadataFilter 为指定的 JSON 元数据字段追加前缀匹配条件。
// 每个值都会生成一个 `OR` 条件，并以不区分大小写的 `LIKE` 方式匹配 `metadata ->> key`。
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

// addScalarJSONMetadataFilter 为指定的 JSON 元数据字段追加等值过滤条件。
// 仅在 value 经过去除首尾空白后非空时追加条件。
func addScalarJSONMetadataFilter(clauses *[]string, args *[]any, key string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	*args = append(*args, strings.TrimSpace(value))
	*clauses = append(*clauses, fmt.Sprintf("COALESCE(metadata ->> '%s', '') = $%d", key, len(*args)))
}

// addAnyExpressionFilter 将一组值按给定表达式组合为 `OR` 条件并追加到筛选子句中。
// 每个值都会作为参数加入 `args`，表达式中的参数占位符按当前参数位置依次填充。
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

// auditResultValues 将审计结果枚举值转换为字符串切片，并跳过空值。
// @returns 转换后的字符串切片；当输入为空时返回 nil。
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

// auditRiskLevelValues 将审计风险级别枚举转换为字符串切片。
//
// 返回空输入对应的 nil，并跳过空值元素。
//
// @param values 审计风险级别列表。
// @returns 转换后的字符串切片。
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

// escapeLikePattern 转义 SQL `LIKE` 模式中的特殊字符。
// 它会转义反斜杠、百分号和下划线。
func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}

// addUint64Filter 在值存在时追加一个 uint64 参数化过滤条件，并将对应占位符加入子句列表。
func addUint64Filter(clauses *[]string, args *[]any, format string, value *uint64) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

// addBoolFilter 在值存在时追加布尔条件和对应参数。
func addBoolFilter(clauses *[]string, args *[]any, format string, value *bool) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}

// addTimeFilter 将时间条件追加到子句和参数列表中。
// 当 value 非 nil 时，写入其 UTC 时间值，并按当前参数位置格式化生成条件。
func addTimeFilter(clauses *[]string, args *[]any, format string, value *time.Time) {
	if value == nil {
		return
	}
	*args = append(*args, value.UTC())
	*clauses = append(*clauses, fmt.Sprintf(format, len(*args)))
}
