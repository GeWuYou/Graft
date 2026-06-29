package audit

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	auditstore "graft/server/modules/audit/store"
	"graft/server/modules/audit/storeent"
)

// handleReadAuditOverview 生成用于读取审计概览的 Gin 处理器。
// 处理请求中的概览参数，校验并归一化时间预设，按请求语言环境读取审计概览结果，
// 再将结果映射为接口响应并写回成功响应；参数非法或处理失败时返回相应错误。
func handleReadAuditOverview(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := zap.NewNop()
	if ctx != nil && ctx.Logger != nil {
		logger = ctx.Logger
	}

	return func(ginCtx *gin.Context) {
		params, invalidField := bindGeneratedAuditOverviewParams(ginCtx)
		if invalidField != "" {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": invalidField,
			})
			return
		}
		preset := normalizeAuditOverviewPreset(params.Preset)

		result, err := reader.Overview(withAuditRequestLocale(ginCtx, ctx), preset)
		if err != nil {
			logger.Error("read audit overview failed",
				zap.String("module", moduleName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := toAuditOverviewResponse(result)
		if mapErr != nil {
			logger.Error("map audit overview response failed",
				zap.String("module", moduleName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

type auditReadGeneratedHandler struct{}
type auditListBinder func(*gin.Context, *auditopenapi.GetAuditLogsParams, *ListQuery) string

var auditAllowedListQueryKeys = map[string]struct{}{
	"page":                    {},
	"page_size":               {},
	"actor_user_id":           {},
	"keyword":                 {},
	"actor":                   {},
	"action":                  {},
	"preset":                  {},
	"scope":                   {},
	"business_category":       {},
	"action_prefix":           {},
	"action_prefixes":         {},
	"action_prefixes[]":       {},
	"action_keywords":         {},
	"action_keywords[]":       {},
	"source":                  {},
	"resource_type":           {},
	"resource_types":          {},
	"resource_types[]":        {},
	"resource_id":             {},
	"resource_name":           {},
	"request_path_prefixes":   {},
	"request_path_prefixes[]": {},
	"request_id":              {},
	"session_id":              {},
	"result":                  {},
	"results":                 {},
	"results[]":               {},
	"risk_level":              {},
	"risk_levels":             {},
	"risk_levels[]":           {},
	"visibility_scope":        {},
	"success":                 {},
	"created_from":            {},
	"created_to":              {},
	"sort":                    {},
	"sort[]":                  {},
}

// 如果请求或国际化上下文不可用，则返回请求上下文。
func withAuditRequestLocale(ginCtx *gin.Context, ctx *module.Context) context.Context {
	requestCtx := context.Background()
	if ginCtx != nil && ginCtx.Request != nil {
		requestCtx = ginCtx.Request.Context()
	}
	if ctx == nil || ctx.I18n == nil || ginCtx == nil {
		return requestCtx
	}

	locale := ctx.I18n.ResolveRequestLocale(ginCtx.Request, "")
	return storeent.WithAuditLocale(requestCtx, locale)
}

func (auditReadGeneratedHandler) GetAuditLogs(auditopenapi.GetAuditLogsParams) {}

func (auditReadGeneratedHandler) GetAuditLogDetail(int64, auditopenapi.GetAuditLogDetailParams) {}

func (auditReadGeneratedHandler) GetAuditOverview(auditopenapi.GetAuditOverviewParams) {}

func (auditReadGeneratedHandler) GetAuditIncident(auditopenapi.GetAuditIncidentParams) {}

// bindGeneratedAuditListParams 绑定审计日志列表请求参数并校验查询字段。
//
// 返回生成的 OpenAPI 参数、内部查询对象，以及首个无效字段名；当所有参数都有效时，字段名为空字符串。
func bindGeneratedAuditListParams(
	ginCtx *gin.Context,
) (auditopenapi.GetAuditLogsParams, ListQuery, string) {
	params := newAuditListParams(ginCtx)
	query := ListQuery{}

	if field := bindAuditPrimaryFilters(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	bindAuditStringFilters(ginCtx, &params, &query)
	bindAuditStringSliceFilters(ginCtx, &params, &query)
	if field := bindAuditSecondaryFilters(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := rejectUnknownAuditListQueryKeys(ginCtx); field != "" {
		return params, query, field
	}

	return params, query, ""
}

// bindAuditPrimaryFilters 按优先级绑定审计日志的主过滤条件，并返回首个无效字段名。
func bindAuditPrimaryFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	return runAuditListBinders(ginCtx, params, query,
		bindAuditPagination,
		bindAuditActorUserID,
		bindAuditPreset,
		bindAuditScope,
		func(ginCtx *gin.Context, _ *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
			return bindAuditVisibilityScope(ginCtx, query)
		},
	)
}

// bindAuditSecondaryFilters 依次绑定审计列表的次级筛选条件。
// 当任一筛选项解析失败时，返回对应的非法字段名；全部成功时返回空字符串。
func bindAuditSecondaryFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	return runAuditListBinders(ginCtx, params, query,
		bindAuditEnumFilters,
		bindAuditSuccessFilter,
		bindAuditCreatedRange,
		bindAuditSort,
	)
}

// runAuditListBinders 依次执行一组审计列表绑定器。
// @return 第一个返回的非法字段名；如果全部绑定成功则返回空字符串。
func runAuditListBinders(
	ginCtx *gin.Context,
	params *auditopenapi.GetAuditLogsParams,
	query *ListQuery,
	binders ...auditListBinder,
) string {
	for _, binder := range binders {
		if field := binder(ginCtx, params, query); field != "" {
			return field
		}
	}
	return ""
}

// 其中包含语言区域和请求 ID。
func newAuditListParams(ginCtx *gin.Context) auditopenapi.GetAuditLogsParams {
	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	return auditopenapi.GetAuditLogsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

// bindAuditPagination 为审计日志列表绑定分页参数。
// 解析 page 和 page_size；当任一参数格式无效时返回对应字段名。
func bindAuditPagination(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	page, ok, err := parseOptionalIntQuery(ginCtx, "page")
	if err != nil {
		return "page"
	}
	if ok {
		params.Page = &page
		query.Page = page
	}

	pageSize, ok, err := parseOptionalIntQuery(ginCtx, "page_size")
	if err != nil {
		return "page_size"
	}
	if ok {
		params.PageSize = &pageSize
		query.PageSize = pageSize
	}

	return ""
}

// bindAuditActorUserID 绑定审计日志查询中的 actor_user_id 参数。
// 成功时将其写入 OpenAPI 参数和内部查询结构；解析、转换或校验失败时返回 "actor_user_id"。
func bindAuditActorUserID(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	value, ok, err := parseOptionalUint64Query(ginCtx, "actor_user_id")
	if err != nil {
		return "actor_user_id"
	}
	if !ok {
		return ""
	}

	converted, convErr := mustConvertAuditGeneratedID(value, "audit actor user id query")
	if convErr != nil {
		return "actor_user_id"
	}
	params.ActorUserId = &converted
	query.ActorUserID = &value
	return ""
}

// bindAuditPreset 绑定审计列表的时间预设参数。
//
// @returns 字段名；当 preset 无效时返回 `"preset"`，否则返回空字符串。
func bindAuditPreset(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	value, invalidField := bindAuditPresetValue[auditopenapi.GetAuditLogsParamsPreset](ginCtx)
	if invalidField != "" {
		return invalidField
	}
	if value != nil {
		params.Preset = value
		query.TimePreset = auditstore.AuditTimePreset(*value)
	}

	return ""
}

// bindAuditScope 绑定审计日志的作用域筛选条件。
//
// @returns 若 `scope` 参数有效则返回空字符串；否则返回 `"scope"`。
func bindAuditScope(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.TrimSpace(ginCtx.Query("scope")); raw != "" {
		value := auditopenapi.GetAuditLogsParamsScope(raw)
		if !value.Valid() {
			return "scope"
		}
		params.Scope = &value
		query.Scope = raw
	}
	return ""
}

// bindAuditVisibilityScope 绑定审计列表的可见性范围筛选条件。
// 当 visibility_scope 为空时不做处理；当其值为 Default、All 或 HiddenOnly 时写入查询条件，
// 否则返回 "visibility_scope"。
func bindAuditVisibilityScope(ginCtx *gin.Context, query *ListQuery) string {
	raw := strings.TrimSpace(ginCtx.Query("visibility_scope"))
	if raw == "" {
		return ""
	}
	switch auditstore.AuditVisibilityScope(raw) {
	case auditstore.AuditVisibilityScopeDefault,
		auditstore.AuditVisibilityScopeAll,
		auditstore.AuditVisibilityScopeHiddenOnly:
		query.VisibilityScope = auditstore.AuditVisibilityScope(raw)
		return ""
	default:
		return "visibility_scope"
	}
}

// bindAuditStringFilters 绑定审计日志查询中的字符串过滤条件。
// 处理 keyword、actor、action、action_prefix、resource_type、resource_id、resource_name、session_id 和 request_id。
func bindAuditStringFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) {
	bindAuditStringFilter(ginCtx, "keyword", &params.Keyword, &query.Keyword)
	bindAuditStringFilter(ginCtx, "actor", &params.Actor, &query.Actor)
	bindAuditStringFilter(ginCtx, "action", &params.Action, &query.Action)
	bindAuditStringFilter(ginCtx, "action_prefix", &params.ActionPrefix, &query.ActionPrefix)
	bindAuditStringFilter(ginCtx, "resource_type", &params.ResourceType, &query.ResourceType)
	bindAuditStringFilter(ginCtx, "resource_id", &params.ResourceId, &query.ResourceID)
	bindAuditStringFilter(ginCtx, "resource_name", &params.ResourceName, &query.ResourceName)
	bindAuditStringFilter(ginCtx, "session_id", &params.SessionId, &query.SessionID)
	bindAuditStringFilter(ginCtx, "request_id", &params.RequestId, &query.RequestID)
}

// bindAuditStringSliceFilters 绑定审计日志列表中的字符串数组查询条件。
func bindAuditStringSliceFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) {
	bindAuditStringSliceFilter(ginCtx, "action_prefixes", &params.ActionPrefixes, &query.ActionPrefixes)
	bindAuditStringSliceFilter(ginCtx, "action_keywords", &params.ActionKeywords, &query.ActionKeywords)
	bindAuditStringSliceFilter(ginCtx, "resource_types", &params.ResourceTypes, &query.ResourceTypes)
	bindAuditStringSliceFilter(ginCtx, "request_path_prefixes", &params.RequestPathPrefixes, &query.RequestPathPrefixes)
}

// bindAuditEnumFilters 绑定审计日志查询中的枚举筛选条件。
// 出现无效值时返回对应的字段名。
func bindAuditEnumFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if errField := bindAuditBusinessCategoryFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if errField := bindAuditUpperEnumFilter(
		ginCtx,
		"source",
		isAllowedAuditSource,
		func(value auditopenapi.GetAuditLogsParamsSource, storeValue auditstore.AuditSource) {
			params.Source = &value
			query.Source = storeValue
		},
	); errField != "" {
		return errField
	}
	if errField := bindAuditUpperEnumFilter(
		ginCtx,
		"result",
		isAllowedAuditResult,
		func(value auditopenapi.GetAuditLogsParamsResult, storeValue auditstore.AuditResult) {
			params.Result = &value
			query.Result = storeValue
		},
	); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditUpperEnumSlice[auditopenapi.GetAuditLogsParamsResults, auditstore.AuditResult](
		queryArrayCompat(ginCtx, "results"),
		isAllowedAuditResult,
	); !ok {
		return "results"
	} else if len(values) > 0 {
		params.Results = &values
		query.Results = normalized
	}
	if errField := bindAuditUpperEnumFilter(
		ginCtx,
		"risk_level",
		isAllowedAuditRiskLevel,
		func(value auditopenapi.GetAuditLogsParamsRiskLevel, storeValue auditstore.AuditRiskLevel) {
			params.RiskLevel = &value
			query.RiskLevel = storeValue
		},
	); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditUpperEnumSlice[auditopenapi.GetAuditLogsParamsRiskLevels, auditstore.AuditRiskLevel](
		queryArrayCompat(ginCtx, "risk_levels"),
		isAllowedAuditRiskLevel,
	); !ok {
		return "risk_levels"
	} else if len(values) > 0 {
		params.RiskLevels = &values
		query.RiskLevels = normalized
	}
	return ""
}

// bindAuditBusinessCategoryFilter 绑定审计日志的业务分类过滤条件。
// 当 `business_category` 存在且值在允许范围内时，写入 OpenAPI 参数和内部查询条件。
// 返回非法字段名；未提供该参数时返回空字符串。
func bindAuditBusinessCategoryFilter(
	ginCtx *gin.Context,
	params *auditopenapi.GetAuditLogsParams,
	query *ListQuery,
) string {
	raw := strings.TrimSpace(ginCtx.Query("business_category"))
	if raw == "" {
		return ""
	}

	switch auditstore.AuditBusinessCategory(raw) {
	case auditstore.AuditBusinessCategoryFailedOperations,
		auditstore.AuditBusinessCategoryHighRiskOperations,
		auditstore.AuditBusinessCategorySensitiveOperations,
		auditstore.AuditBusinessCategoryAuthFailures,
		auditstore.AuditBusinessCategoryPermissionDenials,
		auditstore.AuditBusinessCategoryRBACChanges,
		auditstore.AuditBusinessCategoryCriticalSecurity:
		value := auditopenapi.GetAuditLogsParamsBusinessCategory(raw)
		params.BusinessCategory = &value
		query.BusinessCategory = auditstore.AuditBusinessCategory(raw)
		return ""
	default:
		return "business_category"
	}
}

// 返回转换为大写且去除首尾空白后的值；如果任一值不被允许，则返回 false。
func normalizeAuditEnumQuerySlice(rawValues []string, isAllowed func(string) bool) ([]string, bool) {
	if len(rawValues) == 0 {
		return nil, true
	}

	normalized := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		value := strings.ToUpper(strings.TrimSpace(raw))
		if !isAllowed(value) {
			return nil, false
		}
		normalized = append(normalized, value)
	}

	return normalized, true
}

// bindAuditUpperEnumFilter 读取并校验指定的枚举查询参数。
// 当参数存在且属于允许值时，将其同时写入 API 值和存储层值；否则返回字段名。
func bindAuditUpperEnumFilter[API ~string, Store ~string](
	ginCtx *gin.Context,
	key string,
	isAllowed func(Store) bool,
	assign func(API, Store),
) string {
	raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query(key)))
	if raw == "" {
		return ""
	}

	storeValue := Store(raw)
	if !isAllowed(storeValue) {
		return key
	}

	assign(API(raw), storeValue)
	return ""
}

// bindAuditUpperEnumSlice 规范化并校验枚举字符串数组，并转换为 API 和存储层类型。
// 如果任一值不被允许，返回空结果和 false。
// @param rawValues 原始查询值。
// @param isAllowed 判断值是否允许的函数。
// @returns 规范化后的 API 值切片、存储层值切片，以及表示是否全部有效的布尔值。
func bindAuditUpperEnumSlice[API ~string, Store ~string](
	rawValues []string,
	isAllowed func(Store) bool,
) ([]API, []Store, bool) {
	normalized, ok := normalizeAuditEnumQuerySlice(rawValues, func(value string) bool {
		return isAllowed(Store(value))
	})
	if !ok {
		return nil, nil, false
	}

	return collectAuditEnumSlice[API](normalized), collectAuditEnumSlice[Store](normalized), true
}

// collectAuditEnumSlice 将字符串切片转换为目标字符串类型切片。
func collectAuditEnumSlice[T ~string](values []string) []T {
	collected := make([]T, 0, len(values))
	for _, value := range values {
		collected = append(collected, T(value))
	}
	return collected
}

// isAllowedAuditSource 判断审计来源是否属于允许的来源集合。
// 允许的值包括 Request、SecurityEvent 和 DomainEvent。
func isAllowedAuditSource(value auditstore.AuditSource) bool {
	switch value {
	case auditstore.AuditSourceRequest, auditstore.AuditSourceSecurityEvent, auditstore.AuditSourceDomainEvent:
		return true
	default:
		return false
	}
}

// isAllowedAuditResult 判断审计结果是否属于允许的取值。
// 允许的取值包括 Success、Failed、Denied 和 Error。
func isAllowedAuditResult(value auditstore.AuditResult) bool {
	switch value {
	case auditstore.AuditResultSuccess, auditstore.AuditResultFailed, auditstore.AuditResultDenied, auditstore.AuditResultError:
		return true
	default:
		return false
	}
}

// isAllowedAuditRiskLevel 报告给定的审计风险级别是否受支持。
// 它返回 `true` 表示级别为 `Low`、`Medium`、`High` 或 `Critical`，否则返回 `false`。
func isAllowedAuditRiskLevel(value auditstore.AuditRiskLevel) bool {
	switch value {
	case auditstore.AuditRiskLevelLow, auditstore.AuditRiskLevelMedium, auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
		return true
	default:
		return false
	}
}

// normalizeAuditStringQuerySlice 规范化字符串查询值切片，去除首尾空白并丢弃空项。
func normalizeAuditStringQuerySlice(values []string) []string {
	return normalizeAuditStringFilters(values)
}

// queryArrayCompat 读取指定键的数组查询参数，并兼容带 `[]` 后缀的写法。
//
// 当 `key[]` 形式存在时，会同时返回 `key` 和 `key[]` 的值；否则只返回 `key` 的值。
//
// queryArrayCompat 返回指定查询参数及其方括号形式的所有值。
//
// 当同时存在 `key` 和 `key[]` 时，结果按 `key` 的值在前、`key[]` 的值在后合并。
//
// @param key 查询参数名。
// @returns 合并后的查询参数值列表。
func queryArrayCompat(ginCtx *gin.Context, key string) []string {
	values := ginCtx.QueryArray(key)
	bracketValues := ginCtx.QueryArray(key + "[]")
	if len(bracketValues) == 0 {
		return values
	}

	combined := make([]string, 0, len(values)+len(bracketValues))
	combined = append(combined, values...)
	combined = append(combined, bracketValues...)
	return combined
}

// bindAuditStringFilter 绑定指定查询键的字符串过滤条件。
// 当查询值经修剪后非空时，将其同时写入 OpenAPI 参数和内部查询值。
func bindAuditStringFilter(ginCtx *gin.Context, key string, targetParam **string, targetQuery *string) {
	if raw := strings.TrimSpace(ginCtx.Query(key)); raw != "" {
		*targetParam = &raw
		*targetQuery = raw
	}
}

// bindAuditStringSliceFilter 将归一化后的字符串数组查询值写入参数和内部查询结构。
// 仅在查询中存在非空值时赋值，并同时兼容 `key` 与 `key[]` 两种写法。
func bindAuditStringSliceFilter(ginCtx *gin.Context, key string, targetParam **[]string, targetQuery *[]string) {
	values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, key))
	if len(values) == 0 {
		return
	}
	*targetParam = &values
	*targetQuery = append((*targetQuery)[:0], values...)
}

// bindAuditSuccessFilter 绑定审计日志的 success 查询条件。
// 当 success 值解析失败时返回 "success"；解析成功且存在时，会同时写入 API 参数和内部查询对象。
func bindAuditSuccessFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	value, ok, err := parseOptionalBoolQuery(ginCtx, "success")
	if err != nil {
		return "success"
	}
	if ok {
		params.Success = &value
		query.Success = &value
	}
	return ""
}

// bindAuditCreatedRange 绑定审计日志的创建时间范围查询参数。
// 解析 `created_from` 和 `created_to`，并在成功时同时写入 API 参数与内部查询结构；API 参数使用 UTC 时间。
// @returns 返回空字符串表示成功；若任一时间参数解析失败，则返回对应字段名。
func bindAuditCreatedRange(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	createdFrom, ok, err := parseOptionalTimeQuery(ginCtx, "created_from")
	if err != nil {
		return "created_from"
	}
	if ok {
		converted := createdFrom.UTC()
		params.CreatedFrom = &converted
		query.CreatedFrom = &createdFrom
	}

	createdTo, ok, err := parseOptionalTimeQuery(ginCtx, "created_to")
	if err != nil {
		return "created_to"
	}
	if ok {
		converted := createdTo.UTC()
		params.CreatedTo = &converted
		query.CreatedTo = &createdTo
	}

	return ""
}

// bindAuditSort 绑定审计日志列表的排序条件。
// @param ginCtx 请求上下文。
// @param params OpenAPI 请求参数。
// @param query 内部查询条件。
// @returns 解析失败时返回 "sort"，否则返回空字符串。
func bindAuditSort(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	rawValues := queryArrayCompat(ginCtx, "sort")
	if len(rawValues) == 0 {
		return ""
	}

	sorts := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		field, order, ok := ParseAuditSortExpressionForBinding(raw)
		if !ok {
			return "sort"
		}
		sorts = append(sorts, field+":"+order)
	}
	params.Sort = &sorts
	query.Sorts = sorts
	return ""
}

// rejectUnknownAuditListQueryKeys 检查审计列表请求中的查询键，并返回第一个未允许的键名。
func rejectUnknownAuditListQueryKeys(ginCtx *gin.Context) string {
	for key := range ginCtx.Request.URL.Query() {
		if _, ok := auditAllowedListQueryKeys[key]; !ok {
			return key
		}
	}
	return ""
}

// parseOptionalQuery 读取并解析可选的查询参数。
// 参数为空时返回零值、false 和 nil；参数存在且解析成功时返回解析结果、true 和 nil；解析失败时返回零值、false 和错误。
func parseOptionalQuery[T any](
	ginCtx *gin.Context,
	key string,
	parse func(string) (T, error),
) (T, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		var zero T
		return zero, false, nil
	}

	value, err := parse(raw)
	if err != nil {
		var zero T
		return zero, false, err
	}
	return value, true, nil
}

// parseOptionalIntQuery 解析指定查询参数为整数。
//
// @param ginCtx 当前请求上下文。
// @param key 查询参数名。
// @returns 解析后的整数值、是否存在以及解析错误。值缺失时返回零值、false 和 nil。
func parseOptionalIntQuery(ginCtx *gin.Context, key string) (int, bool, error) {
	return parseOptionalQuery(ginCtx, key, strconv.Atoi)
}

// parseOptionalUint64Query 解析指定查询参数为 uint64。
// 当参数存在时返回解析结果；当参数缺失时返回零值和 false；当参数格式无效时返回解析错误。
func parseOptionalUint64Query(ginCtx *gin.Context, key string) (uint64, bool, error) {
	return parseOptionalQuery(ginCtx, key, func(raw string) (uint64, error) {
		return strconv.ParseUint(raw, 10, 64)
	})
}

// parseOptionalBoolQuery 解析可选的布尔查询参数。
// 返回解析后的布尔值、是否存在该参数以及解析错误。
func parseOptionalBoolQuery(ginCtx *gin.Context, key string) (bool, bool, error) {
	return parseOptionalQuery(ginCtx, key, strconv.ParseBool)
}

// parseOptionalTimeQuery 解析可选的 RFC3339 时间查询参数。
// 返回解析后的时间值、参数是否存在以及解析错误；参数缺失时返回零时间、false 和 nil。
func parseOptionalTimeQuery(ginCtx *gin.Context, key string) (time.Time, bool, error) {
	return parseOptionalQuery(ginCtx, key, func(raw string) (time.Time, error) {
		return time.Parse(time.RFC3339, raw)
	})
}

// bindGeneratedAuditReadHeaders 提取请求中的语言环境和请求 ID 头。
// bindGeneratedAuditReadHeaders 从请求头提取区域设置和请求 ID 指针。
func bindGeneratedAuditReadHeaders(ginCtx *gin.Context) (locale *string, requestID *string) {
	return auditHeaderPointer(ginCtx.GetHeader(string(httpheader.Locale))),
		auditHeaderPointer(ginCtx.GetHeader(httpx.RequestIDHeader))
}

// bindAuditPresetValue 解析并校验 `preset` 查询参数。
// 当参数为空时返回 `nil`；当参数值无效时返回字段名 `"preset"`。
//
// @return 符合类型约束且通过 `Valid()` 校验的值指针，以及表示无效字段名的字符串。
func bindAuditPresetValue[T interface {
	~string
	Valid() bool
}](ginCtx *gin.Context) (*T, string) {
	raw := strings.TrimSpace(ginCtx.Query("preset"))
	if raw == "" {
		return nil, ""
	}

	value := T(raw)
	if !value.Valid() {
		return nil, "preset"
	}
	return &value, ""
}

// bindGeneratedAuditOverviewParams 绑定审计概览的请求参数。
// 它会读取语言和请求 ID 头，并校验可选的 preset 参数。
// @returns 解析后的审计概览参数；如果 preset 无效，则返回对应的字段名。
func bindGeneratedAuditOverviewParams(ginCtx *gin.Context) (auditopenapi.GetAuditOverviewParams, string) {
	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	params := auditopenapi.GetAuditOverviewParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}

	value, invalidField := bindAuditPresetValue[auditopenapi.GetAuditOverviewParamsPreset](ginCtx)
	if invalidField != "" {
		return params, invalidField
	}
	if value != nil {
		params.Preset = value
	}

	return params, ""
}

// normalizeAuditOverviewPreset 将概览预设规范化为存储层时间预设。
// 当值为空、为空白或不受支持时，返回默认的最近 24 小时预设。
//
// normalizeAuditOverviewPreset 将概览参数中的预设转换为审计时间预设。
// 当未提供预设时，返回默认的最近 24 小时预设。
//
// 返回对应的审计时间预设。
func normalizeAuditOverviewPreset(value *auditopenapi.GetAuditOverviewParamsPreset) auditstore.AuditTimePreset {
	if value == nil {
		return auditstore.AuditTimePresetLast24Hours
	}
	return normalizeAuditOverviewTimePreset(auditstore.AuditTimePreset(strings.TrimSpace(string(*value))))
}

// auditHeaderPointer 将空白字符串转换为 nil，否则返回其指针。
func auditHeaderPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
