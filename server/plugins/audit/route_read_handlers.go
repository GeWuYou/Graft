package audit

import (
	"context"
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
	auditstore "graft/server/plugins/audit/store"
)

type auditReader interface {
	List(ctx context.Context, query auditcore.ListQuery) (auditcore.ListResult, error)
	Overview(ctx context.Context, window auditstore.OverviewWindow) (auditcore.OverviewResult, error)
}

type auditListResult = auditcore.ListResult
type auditOverviewResult = auditcore.OverviewResult

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
		window := normalizeOverviewWindow(params.Window)

		result, err := reader.Overview(ginCtx, window)
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

type auditReadGeneratedHandler struct{}

func (h auditReadGeneratedHandler) GetAuditLogs(params auditopenapi.GetAuditLogsParams) {
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
	bindAuditStringFilters(ginCtx, &params, &query)
	if field := bindAuditEnumFilters(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := bindAuditSuccessFilter(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := bindAuditCreatedRange(ginCtx, &params, &query); field != "" {
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

func bindAuditStringFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) {
	bindAuditStringFilter(ginCtx, "action", &params.Action, &query.Action)
	bindAuditStringFilter(ginCtx, "action_prefix", &params.ActionPrefix, &query.ActionPrefix)
	bindAuditStringFilter(ginCtx, "resource_type", &params.ResourceType, &query.ResourceType)
	bindAuditStringFilter(ginCtx, "resource_id", &params.ResourceId, &query.ResourceID)
	bindAuditStringFilter(ginCtx, "resource_name", &params.ResourceName, &query.ResourceName)
	bindAuditStringFilter(ginCtx, "request_id", &params.RequestId, &query.RequestID)
}

func bindAuditEnumFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *auditcore.ListQuery) string {
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("source"))); raw != "" {
		switch auditstore.AuditSource(raw) {
		case auditstore.AuditSourceRequest, auditstore.AuditSourceSecurityEvent, auditstore.AuditSourceDomainEvent:
		default:
			return "source"
		}
		value := auditopenapi.GetAuditLogsParamsSource(raw)
		params.Source = &value
		query.Source = auditstore.AuditSource(raw)
	}
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("result"))); raw != "" {
		switch auditstore.AuditResult(raw) {
		case auditstore.AuditResultSuccess, auditstore.AuditResultFailed, auditstore.AuditResultDenied, auditstore.AuditResultError:
		default:
			return "result"
		}
		value := auditopenapi.GetAuditLogsParamsResult(raw)
		params.Result = &value
		query.Result = auditstore.AuditResult(raw)
	}
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("risk_level"))); raw != "" {
		switch auditstore.AuditRiskLevel(raw) {
		case auditstore.AuditRiskLevelLow, auditstore.AuditRiskLevelMedium, auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
		default:
			return "risk_level"
		}
		value := auditopenapi.GetAuditLogsParamsRiskLevel(raw)
		params.RiskLevel = &value
		query.RiskLevel = auditstore.AuditRiskLevel(raw)
	}
	return ""
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

	if raw := strings.TrimSpace(ginCtx.Query("window")); raw != "" {
		value := auditopenapi.GetAuditOverviewParamsWindow(raw)
		if value.Valid() {
			params.Window = &value
		}
	}

	return params
}

func normalizeOverviewWindow(value *auditopenapi.GetAuditOverviewParamsWindow) auditstore.OverviewWindow {
	if value == nil {
		return auditstore.OverviewWindow24Hours
	}
	switch strings.TrimSpace(string(*value)) {
	case string(auditstore.OverviewWindow7Days):
		return auditstore.OverviewWindow7Days
	case string(auditstore.OverviewWindow30Days):
		return auditstore.OverviewWindow30Days
	default:
		return auditstore.OverviewWindow24Hours
	}
}
