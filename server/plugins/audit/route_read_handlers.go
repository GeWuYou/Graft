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

	auditcore "graft/server/internal/audit"
	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	auditcontract "graft/server/plugins/audit/contract"
	auditstore "graft/server/plugins/audit/store"
)

type auditReader interface {
	List(ctx context.Context, query auditcore.ListQuery) (auditcore.ListResult, error)
	Overview(ctx context.Context, preset auditstore.AuditTimePreset) (auditcore.OverviewResult, error)
	Incident(ctx context.Context, eventID uint64) (auditcore.IncidentResult, error)
}

type auditListResult = auditcore.ListResult
type auditOverviewResult = auditcore.OverviewResult
type auditIncidentResult = auditcore.IncidentResult

type auditGuard struct {
	read gin.HandlerFunc
}

func handleListAuditLogs(
	ctx *plugin.Context,
	pluginName string,
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

		result, err := reader.List(ginCtx, query)
		if err != nil {
			logger.Error("list audit logs failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := toAuditLogListResponse(result)
		if mapErr != nil {
			logger.Error("map audit logs response failed",
				zap.String("plugin", pluginName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func handleReadAuditOverview(
	ctx *plugin.Context,
	pluginName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := zap.NewNop()
	if ctx != nil && ctx.Logger != nil {
		logger = ctx.Logger
	}

	return func(ginCtx *gin.Context) {
		params := bindGeneratedAuditOverviewParams(ginCtx)
		preset := normalizeAuditOverviewPreset(params.Preset)

		result, err := reader.Overview(ginCtx, preset)
		if err != nil {
			logger.Error("read audit overview failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := toAuditOverviewResponse(result)
		if mapErr != nil {
			logger.Error("map audit overview response failed",
				zap.String("plugin", pluginName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func handleReadAuditIncident(
	ctx *plugin.Context,
	pluginName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := zap.NewNop()
	if ctx != nil && ctx.Logger != nil {
		logger = ctx.Logger
	}

	return func(ginCtx *gin.Context) {
		eventID, ok, err := parseOptionalUint64Param(ginCtx, auditcontract.AuditIncidentParam)
		if err != nil || !ok {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": "event_id",
			})
			return
		}

		result, readErr := reader.Incident(ginCtx, eventID)
		if readErr != nil {
			if errors.Is(readErr, auditstore.ErrIncidentNotFound) {
				httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusNotFound, "common.not_found", map[string]any{
					"field": "event_id",
				})
				return
			}
			logger.Error("read audit incident failed",
				zap.String("plugin", pluginName),
				zap.Uint64("event_id", eventID),
				zap.Error(readErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := toAuditIncidentResponse(result)
		if mapErr != nil {
			logger.Error("map audit incident response failed",
				zap.String("plugin", pluginName),
				zap.Uint64("event_id", eventID),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

type auditReadGeneratedHandler struct{}

func (h auditReadGeneratedHandler) GetAuditLogs(params auditopenapi.GetAuditLogsParams) {
	_ = h
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
) (auditopenapi.GetAuditLogsParams, auditcore.ListQuery, string) {
	params := newAuditListParams(ginCtx)
	query := auditcore.ListQuery{}

	if field := bindAuditPagination(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := bindAuditActorUserID(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := bindAuditPreset(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := rejectLegacyAuditScope(ginCtx); field != "" {
		return params, query, field
	}
	bindAuditStringFilters(ginCtx, &params, &query)
	bindAuditStringSliceFilters(ginCtx, &params, &query)
	if field := bindAuditEnumFilters(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := bindAuditSuccessFilter(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := bindAuditCreatedRange(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := bindAuditSort(ginCtx, &params, &query); field != "" {
		return params, query, field
	}

	return params, query, ""
}

func newAuditListParams(ginCtx *gin.Context) auditopenapi.GetAuditLogsParams {
	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	return auditopenapi.GetAuditLogsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindAuditPagination(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func bindAuditActorUserID(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func bindAuditPreset(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func rejectLegacyAuditScope(ginCtx *gin.Context) string {
	if strings.TrimSpace(ginCtx.Query("scope")) != "" {
		return "scope"
	}
	return ""
}

func bindAuditStringFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) {
	bindAuditStringFilter(ginCtx, "action", &params.Action, &query.Action)
	bindAuditStringFilter(ginCtx, "action_prefix", &params.ActionPrefix, &query.ActionPrefix)
	bindAuditStringFilter(ginCtx, "resource_type", &params.ResourceType, &query.ResourceType)
	bindAuditStringFilter(ginCtx, "resource_id", &params.ResourceId, &query.ResourceID)
	bindAuditStringFilter(ginCtx, "resource_name", &params.ResourceName, &query.ResourceName)
	bindAuditStringFilter(ginCtx, "request_id", &params.RequestId, &query.RequestID)
}

func bindAuditStringSliceFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) {
	if values := normalizeAuditStringQuerySlice(ginCtx.QueryArray("action_prefixes")); len(values) > 0 {
		params.ActionPrefixes = &values
		query.ActionPrefixes = values
	}
	if values := normalizeAuditStringQuerySlice(ginCtx.QueryArray("action_keywords")); len(values) > 0 {
		params.ActionKeywords = &values
		query.ActionKeywords = values
	}
	if values := normalizeAuditStringQuerySlice(ginCtx.QueryArray("resource_types")); len(values) > 0 {
		params.ResourceTypes = &values
		query.ResourceTypes = values
	}
	if values := normalizeAuditStringQuerySlice(ginCtx.QueryArray("request_path_prefixes")); len(values) > 0 {
		params.RequestPathPrefixes = &values
		query.RequestPathPrefixes = values
	}
}

func bindAuditEnumFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
	if errField := bindAuditSourceFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if errField := bindAuditResultFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditResultSliceFilter(ginCtx.QueryArray("results")); !ok {
		return "results"
	} else if len(values) > 0 {
		params.Results = &values
		query.Results = normalized
	}
	if errField := bindAuditRiskLevelFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditRiskLevelSliceFilter(ginCtx.QueryArray("risk_levels")); !ok {
		return "risk_levels"
	} else if len(values) > 0 {
		params.RiskLevels = &values
		query.RiskLevels = normalized
	}
	return ""
}

func bindAuditSourceFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func bindAuditResultFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func bindAuditRiskLevelFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func bindAuditStringFilter(ginCtx *gin.Context, key string, targetParam **string, targetQuery *string) {
	if raw := strings.TrimSpace(ginCtx.Query(key)); raw != "" {
		*targetParam = &raw
		*targetQuery = raw
	}
}

func bindAuditSuccessFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func bindAuditCreatedRange(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
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

func bindAuditSort(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
	sortBy := strings.TrimSpace(ginCtx.Query("sort_by"))
	if sortBy != "" {
		if sortBy != "created_at" {
			return "sort_by"
		}
		sortByValue := auditopenapi.GetAuditLogsParamsSortBy(sortBy)
		params.SortBy = &sortByValue
		query.SortBy = sortBy
	}

	sortOrder := strings.TrimSpace(ginCtx.Query("sort_order"))
	if sortOrder != "" {
		if sortBy == "" {
			return "sort_order"
		}
		if sortOrder != "asc" && sortOrder != "desc" {
			return "sort_order"
		}
		sortOrderValue := auditopenapi.GetAuditLogsParamsSortOrder(sortOrder)
		params.SortOrder = &sortOrderValue
		query.SortOrder = sortOrder
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

func parseOptionalUint64Param(ginCtx *gin.Context, key string) (uint64, bool, error) {
	raw := strings.TrimSpace(ginCtx.Param(key))
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
