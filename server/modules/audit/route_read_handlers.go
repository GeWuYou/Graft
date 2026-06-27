package audit

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/drilldown"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
	"graft/server/modules/audit/storeent"
)

type auditReader interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
	Detail(ctx context.Context, id uint64) (DetailResult, error)
	Overview(ctx context.Context, preset auditstore.AuditTimePreset) (OverviewResult, error)
	Incident(ctx context.Context, eventID uint64) (IncidentResult, error)
	VisibilityPolicy(ctx context.Context) (VisibilityPolicyResult, error)
	UpdateVisibilityDefault(
		ctx context.Context,
		strategy auditstore.AuditVisibilityStrategy,
		userID *uint64,
		username string,
	) (auditstore.AuditVisibilityDefault, error)
	UpdateVisibilityOverride(ctx context.Context, input auditstore.UpsertAuditVisibilityOverrideInput) (auditstore.AuditVisibilityOverride, error)
	DeleteVisibilityOverride(ctx context.Context, source auditstore.AuditSource, actionKey string) error
}

type auditListResult = ListResult
type auditDetailResult = DetailResult
type auditOverviewResult = OverviewResult
type auditIncidentResult = IncidentResult

type auditGuard struct {
	read   gin.HandlerFunc
	manage gin.HandlerFunc
}

func handleListAuditLogs(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := zap.NewNop()
	if ctx != nil && ctx.Logger != nil {
		logger = ctx.Logger
	}

	return func(ginCtx *gin.Context) {
		params, query, invalidField := bindGeneratedAuditListParams(ginCtx)
		if invalidField != "" {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": invalidField,
			})
			return
		}
		_ = params

		result, err := reader.List(withAuditRequestLocale(ginCtx, ctx), query)
		if err != nil {
			if errors.Is(err, drilldown.ErrScopeNotFound) ||
				errors.Is(err, drilldown.ErrScopeDisabled) ||
				errors.Is(err, drilldown.ErrTargetMismatch) ||
				errors.Is(err, drilldown.ErrScopeConflict) {
				httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
					"field": "scope",
				})
				return
			}
			logger.Error("list audit logs failed",
				zap.String("module", moduleName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := toAuditLogListResponse(result)
		if mapErr != nil {
			logger.Error("map audit logs response failed",
				zap.String("module", moduleName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func handleReadAuditLog(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	return handleAuditReadByID(ctx, moduleName, auditLogReadConfig(reader))
}

func handleReadAuditIncident(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	return handleAuditReadByID(ctx, moduleName, auditIncidentReadConfig(reader))
}

type auditReadByIDConfig[T any] struct {
	param          string
	invalidField   string
	read           func(context.Context, uint64) (T, error)
	mapper         func(T) (any, error)
	isNotFound     func(error) bool
	notFoundField  string
	readLogMessage string
	mapLogMessage  string
}

type auditReadByIDMeta struct {
	param          string
	field          string
	readLogMessage string
	mapLogMessage  string
}

func auditLogReadConfig(reader auditReader) auditReadByIDConfig[auditDetailResult] {
	return newAuditReadConfig(
		auditReadByIDMeta{
			param:          auditcontract.AuditLogParam,
			field:          "id",
			readLogMessage: "read audit log detail failed",
			mapLogMessage:  "map audit log detail response failed",
		},
		func(requestCtx context.Context, id uint64) (auditDetailResult, error) {
			return reader.Detail(requestCtx, id)
		},
		func(result auditDetailResult) (any, error) {
			return toAuditLogDetailResponse(result)
		},
		func(err error) bool {
			return errors.Is(err, auditstore.ErrAuditLogNotFound)
		},
	)
}

func auditIncidentReadConfig(reader auditReader) auditReadByIDConfig[auditIncidentResult] {
	return newAuditReadConfig(
		auditReadByIDMeta{
			param:          auditcontract.AuditIncidentParam,
			field:          "event_id",
			readLogMessage: "read audit incident failed",
			mapLogMessage:  "map audit incident response failed",
		},
		func(requestCtx context.Context, id uint64) (auditIncidentResult, error) {
			return reader.Incident(requestCtx, id)
		},
		func(result auditIncidentResult) (any, error) {
			return toAuditIncidentResponse(result)
		},
		func(err error) bool {
			return errors.Is(err, auditstore.ErrIncidentNotFound)
		},
	)
}

func newAuditReadConfig[T any](
	meta auditReadByIDMeta,
	read func(context.Context, uint64) (T, error),
	mapper func(T) (any, error),
	isNotFound func(error) bool,
) auditReadByIDConfig[T] {
	return auditReadByIDConfig[T]{
		param:          meta.param,
		invalidField:   meta.field,
		read:           read,
		mapper:         mapper,
		isNotFound:     isNotFound,
		notFoundField:  meta.field,
		readLogMessage: meta.readLogMessage,
		mapLogMessage:  meta.mapLogMessage,
	}
}

func handleAuditReadByID[T any](
	ctx *module.Context,
	moduleName string,
	config auditReadByIDConfig[T],
) gin.HandlerFunc {
	logger := zap.NewNop()
	if ctx != nil && ctx.Logger != nil {
		logger = ctx.Logger
	}

	return func(ginCtx *gin.Context) {
		id, ok, err := parseOptionalUint64Param(ginCtx, config.param)
		if err != nil || !ok {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": config.invalidField,
			})
			return
		}

		record, readErr := config.read(withAuditRequestLocale(ginCtx, ctx), id)
		if readErr != nil {
			if config.isNotFound(readErr) {
				httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusNotFound, "common.not_found", map[string]any{
					"field": config.notFoundField,
				})
				return
			}
			logger.Error(config.readLogMessage,
				zap.String("module", moduleName),
				zap.Uint64("id", id),
				zap.Error(readErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := config.mapper(record)
		if mapErr != nil {
			logger.Error(config.mapLogMessage,
				zap.String("module", moduleName),
				zap.Uint64("id", id),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

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
		params := bindGeneratedAuditOverviewParams(ginCtx)
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

func (h auditReadGeneratedHandler) GetAuditLogs(params auditopenapi.GetAuditLogsParams) {
	_ = h
	_ = params
}

func (h auditReadGeneratedHandler) GetAuditLogDetail(id int64, params auditopenapi.GetAuditLogDetailParams) {
	_ = h
	_ = id
	_ = params
}

func (h auditReadGeneratedHandler) GetAuditOverview(params auditopenapi.GetAuditOverviewParams) {
	_ = h
	_ = params
}

func (h auditReadGeneratedHandler) GetAuditIncident(params auditopenapi.GetAuditIncidentParams) {
	_ = h
	_ = params
}

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
	if field := bindAuditPagination(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditActorUserID(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditPreset(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditScope(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditVisibilityScope(ginCtx, query); field != "" {
		return field
	}
	return ""
}

func bindAuditSecondaryFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if field := bindAuditEnumFilters(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditSuccessFilter(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditCreatedRange(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditSort(ginCtx, params, query); field != "" {
		return field
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
	if raw := strings.TrimSpace(ginCtx.Query("preset")); raw != "" {
		value := auditopenapi.GetAuditLogsParamsPreset(raw)
		if !value.Valid() {
			return "preset"
		}
		params.Preset = &value
		query.TimePreset = auditstore.AuditTimePreset(raw)
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
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "action_prefixes")); len(values) > 0 {
		params.ActionPrefixes = &values
		query.ActionPrefixes = values
	}
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "action_keywords")); len(values) > 0 {
		params.ActionKeywords = &values
		query.ActionKeywords = values
	}
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "resource_types")); len(values) > 0 {
		params.ResourceTypes = &values
		query.ResourceTypes = values
	}
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "request_path_prefixes")); len(values) > 0 {
		params.RequestPathPrefixes = &values
		query.RequestPathPrefixes = values
	}
}

func bindAuditEnumFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if errField := bindAuditBusinessCategoryFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if errField := bindAuditSourceFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if errField := bindAuditResultFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditResultSliceFilter(queryArrayCompat(ginCtx, "results")); !ok {
		return "results"
	} else if len(values) > 0 {
		params.Results = &values
		query.Results = normalized
	}
	if errField := bindAuditRiskLevelFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditRiskLevelSliceFilter(queryArrayCompat(ginCtx, "risk_levels")); !ok {
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

func bindAuditSourceFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("source"))); raw != "" {
		switch auditstore.AuditSource(raw) {
		case auditstore.AuditSourceRequest, auditstore.AuditSourceSecurityEvent, auditstore.AuditSourceDomainEvent:
			value := auditopenapi.GetAuditLogsParamsSource(raw)
			params.Source = &value
			query.Source = auditstore.AuditSource(raw)
		default:
			return "source"
		}
	}

	return ""
}

func bindAuditResultFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("result"))); raw != "" {
		switch auditstore.AuditResult(raw) {
		case auditstore.AuditResultSuccess, auditstore.AuditResultFailed, auditstore.AuditResultDenied, auditstore.AuditResultError:
			value := auditopenapi.GetAuditLogsParamsResult(raw)
			params.Result = &value
			query.Result = auditstore.AuditResult(raw)
		default:
			return "result"
		}
	}

	return ""
}

func bindAuditRiskLevelFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("risk_level"))); raw != "" {
		switch auditstore.AuditRiskLevel(raw) {
		case auditstore.AuditRiskLevelLow, auditstore.AuditRiskLevelMedium, auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
			value := auditopenapi.GetAuditLogsParamsRiskLevel(raw)
			params.RiskLevel = &value
			query.RiskLevel = auditstore.AuditRiskLevel(raw)
		default:
			return "risk_level"
		}
	}

	return ""
}

func bindAuditResultSliceFilter(rawValues []string) ([]auditopenapi.GetAuditLogsParamsResults, []auditstore.AuditResult, bool) {
	normalized, ok := normalizeAuditEnumQuerySlice(rawValues, func(value string) bool {
		switch auditstore.AuditResult(value) {
		case auditstore.AuditResultSuccess, auditstore.AuditResultFailed, auditstore.AuditResultDenied, auditstore.AuditResultError:
			return true
		default:
			return false
		}
	})
	if !ok {
		return nil, nil, false
	}

	return collectAuditResultSlice(normalized), collectAuditStoreResultSlice(normalized), true
}

func bindAuditRiskLevelSliceFilter(rawValues []string) ([]auditopenapi.GetAuditLogsParamsRiskLevels, []auditstore.AuditRiskLevel, bool) {
	normalized, ok := normalizeAuditEnumQuerySlice(rawValues, func(value string) bool {
		switch auditstore.AuditRiskLevel(value) {
		case auditstore.AuditRiskLevelLow, auditstore.AuditRiskLevelMedium, auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
			return true
		default:
			return false
		}
	})
	if !ok {
		return nil, nil, false
	}

	return collectAuditRiskLevelSlice(normalized), collectAuditStoreRiskLevelSlice(normalized), true
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

func collectAuditResultSlice(values []string) []auditopenapi.GetAuditLogsParamsResults {
	collected := make([]auditopenapi.GetAuditLogsParamsResults, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditopenapi.GetAuditLogsParamsResults(value))
	}
	return collected
}

func collectAuditStoreResultSlice(values []string) []auditstore.AuditResult {
	collected := make([]auditstore.AuditResult, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditstore.AuditResult(value))
	}
	return collected
}

func collectAuditRiskLevelSlice(values []string) []auditopenapi.GetAuditLogsParamsRiskLevels {
	collected := make([]auditopenapi.GetAuditLogsParamsRiskLevels, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditopenapi.GetAuditLogsParamsRiskLevels(value))
	}
	return collected
}

func collectAuditStoreRiskLevelSlice(values []string) []auditstore.AuditRiskLevel {
	collected := make([]auditstore.AuditRiskLevel, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditstore.AuditRiskLevel(value))
	}
	return collected
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

func parseOptionalIntQuery(ginCtx *gin.Context, key string) (int, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return 0, false, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func parseOptionalUint64Query(ginCtx *gin.Context, key string) (uint64, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return 0, false, nil
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func parseOptionalBoolQuery(ginCtx *gin.Context, key string) (bool, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return false, false, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false, err
	}
	return value, true, nil
}

func parseOptionalTimeQuery(ginCtx *gin.Context, key string) (time.Time, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return time.Time{}, false, nil
	}

	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false, err
	}
	return value, true, nil
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

func bindGeneratedAuditOverviewParams(ginCtx *gin.Context) auditopenapi.GetAuditOverviewParams {
	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	params := auditopenapi.GetAuditOverviewParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}

	if raw := strings.TrimSpace(ginCtx.Query("preset")); raw != "" {
		value := auditopenapi.GetAuditOverviewParamsPreset(raw)
		if value.Valid() {
			params.Preset = &value
		}
	}

	return params
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
