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

func bindAuditSecondaryFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	return runAuditListBinders(ginCtx, params, query,
		bindAuditEnumFilters,
		bindAuditSuccessFilter,
		bindAuditCreatedRange,
		bindAuditSort,
	)
}

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

func newAuditListParams(ginCtx *gin.Context) auditopenapi.GetAuditLogsParams {
	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	return auditopenapi.GetAuditLogsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

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

func bindAuditStringSliceFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) {
	bindAuditStringSliceFilter(ginCtx, "action_prefixes", &params.ActionPrefixes, &query.ActionPrefixes)
	bindAuditStringSliceFilter(ginCtx, "action_keywords", &params.ActionKeywords, &query.ActionKeywords)
	bindAuditStringSliceFilter(ginCtx, "resource_types", &params.ResourceTypes, &query.ResourceTypes)
	bindAuditStringSliceFilter(ginCtx, "request_path_prefixes", &params.RequestPathPrefixes, &query.RequestPathPrefixes)
}

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

func collectAuditEnumSlice[T ~string](values []string) []T {
	collected := make([]T, 0, len(values))
	for _, value := range values {
		collected = append(collected, T(value))
	}
	return collected
}

func isAllowedAuditSource(value auditstore.AuditSource) bool {
	switch value {
	case auditstore.AuditSourceRequest, auditstore.AuditSourceSecurityEvent, auditstore.AuditSourceDomainEvent:
		return true
	default:
		return false
	}
}

func isAllowedAuditResult(value auditstore.AuditResult) bool {
	switch value {
	case auditstore.AuditResultSuccess, auditstore.AuditResultFailed, auditstore.AuditResultDenied, auditstore.AuditResultError:
		return true
	default:
		return false
	}
}

func isAllowedAuditRiskLevel(value auditstore.AuditRiskLevel) bool {
	switch value {
	case auditstore.AuditRiskLevelLow, auditstore.AuditRiskLevelMedium, auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
		return true
	default:
		return false
	}
}

func normalizeAuditStringQuerySlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

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

func bindAuditStringFilter(ginCtx *gin.Context, key string, targetParam **string, targetQuery *string) {
	if raw := strings.TrimSpace(ginCtx.Query(key)); raw != "" {
		*targetParam = &raw
		*targetQuery = raw
	}
}

func bindAuditStringSliceFilter(ginCtx *gin.Context, key string, targetParam **[]string, targetQuery *[]string) {
	values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, key))
	if len(values) == 0 {
		return
	}
	*targetParam = &values
	*targetQuery = append((*targetQuery)[:0], values...)
}

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

func rejectUnknownAuditListQueryKeys(ginCtx *gin.Context) string {
	for key := range ginCtx.Request.URL.Query() {
		if _, ok := auditAllowedListQueryKeys[key]; !ok {
			return key
		}
	}
	return ""
}

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

func parseOptionalIntQuery(ginCtx *gin.Context, key string) (int, bool, error) {
	return parseOptionalQuery(ginCtx, key, strconv.Atoi)
}

func parseOptionalUint64Query(ginCtx *gin.Context, key string) (uint64, bool, error) {
	return parseOptionalQuery(ginCtx, key, func(raw string) (uint64, error) {
		return strconv.ParseUint(raw, 10, 64)
	})
}

func parseOptionalBoolQuery(ginCtx *gin.Context, key string) (bool, bool, error) {
	return parseOptionalQuery(ginCtx, key, strconv.ParseBool)
}

func parseOptionalTimeQuery(ginCtx *gin.Context, key string) (time.Time, bool, error) {
	return parseOptionalQuery(ginCtx, key, func(raw string) (time.Time, error) {
		return time.Parse(time.RFC3339, raw)
	})
}

func bindGeneratedAuditReadHeaders(ginCtx *gin.Context) (locale *string, requestID *string) {
	if raw := strings.TrimSpace(ginCtx.GetHeader(httpx.RequestIDHeader)); raw != "" {
		requestID = &raw
	}
	if raw := strings.TrimSpace(ginCtx.GetHeader(string(httpheader.Locale))); raw != "" {
		locale = &raw
	}

	return locale, requestID
}

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

func normalizeAuditOverviewPreset(value *auditopenapi.GetAuditOverviewParamsPreset) auditstore.AuditTimePreset {
	if value == nil {
		return auditstore.AuditTimePresetLast24Hours
	}
	switch strings.TrimSpace(string(*value)) {
	case string(auditstore.AuditTimePresetLast7Days):
		return auditstore.AuditTimePresetLast7Days
	case string(auditstore.AuditTimePresetLast30Days):
		return auditstore.AuditTimePresetLast30Days
	default:
		return auditstore.AuditTimePresetLast24Hours
	}
}
