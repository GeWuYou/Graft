package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/pluginapi"
)

const (
	// AccessLogReadPermission 约束 access-log explorer 的只读访问权限码。
	AccessLogReadPermission = "access_log.read"
	accessLogMenuRootPath   = "/logs"
	accessLogMenuListPath   = "/logs/access"
	accessLogMenuCodeRoot   = "log-center.root"
	accessLogMenuCodeList   = "access-log.list"
	accessLogPluginOwner    = "core.httpx"
	accessLogRouteGroup     = "/access-log"
	accessLogRouteItemParam = "id"
	accessLogMenuRootOrder  = 210
	accessLogMenuListOrder  = 211
)

type accessLogReadGuard struct {
	read gin.HandlerFunc
}

// AccessLogExplorerRegistration 收口 access-log explorer 所需的 core 注册依赖。
type AccessLogExplorerRegistration struct {
	I18n               *i18n.Service
	MenuRegistry       *menu.Registry
	PermissionRegistry *permission.Registry
	EventBus           eventbus.Bus
}

func registerAccessLogExplorerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "access-log",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: "menu.logCenter.title", Text: "日志中心"},
				{Key: "menu.accessLog.title", Text: "访问日志"},
			},
		},
		{
			Namespace: "access-log",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: "menu.logCenter.title", Text: "Log Center"},
				{Key: "menu.accessLog.title", Text: "Access Logs"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register access log messages: %w", err)
		}
	}

	return nil
}

func registerAccessLogExplorerPermissions(registry *permission.Registry) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:        AccessLogReadPermission,
		Name:        "Read Access Logs",
		Description: "Allows reading canonical access-log explorer data.",
		Category:    "api",
		Plugin:      accessLogPluginOwner,
	})
}

func registerAccessLogExplorerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       accessLogMenuCodeRoot,
		Title:      "日志中心",
		TitleKey:   "menu.logCenter.title",
		Path:       accessLogMenuRootPath,
		Icon:       "bulletpoint",
		Order:      accessLogMenuRootOrder,
		Permission: "",
		Plugin:     accessLogPluginOwner,
	})
	registry.Register(menu.Item{
		Code:       accessLogMenuCodeList,
		Title:      "访问日志",
		TitleKey:   "menu.accessLog.title",
		Path:       accessLogMenuListPath,
		Icon:       "search",
		Order:      accessLogMenuListOrder,
		Permission: AccessLogReadPermission,
		Plugin:     accessLogPluginOwner,
	})
}

func registerAccessLogExplorerRoutes(
	router gin.IRouter,
	localizer *i18n.Service,
	repo AccessLogRepository,
	authService pluginapi.AuthService,
	authorizer pluginapi.Authorizer,
	bus eventbus.Bus,
) {
	if router == nil || repo == nil || authService == nil {
		return
	}

	publisher := NewSecurityAuditPublisher(bus, nil, accessLogPluginOwner)

	guard := accessLogReadGuard{
		read: RequirePermission(localizer, authService, authorizer, AccessLogReadPermission, publisher),
	}
	group := router.Group(accessLogRouteGroup)
	group.GET("", guard.read, handleListAccessLogs(localizer, repo))
	group.GET("/:"+accessLogRouteItemParam, guard.read, handleGetAccessLogDetail(localizer, repo))
}

// RegisterAccessLogExplorer 把 access-log explorer 的消息、权限、菜单和路由注册到 core runtime。
func RegisterAccessLogExplorer(
	ctx AccessLogExplorerRegistration,
	router gin.IRouter,
	repo AccessLogRepository,
	authService pluginapi.AuthService,
	authorizer pluginapi.Authorizer,
) error {
	if err := registerAccessLogExplorerMessages(ctx.I18n); err != nil {
		return err
	}
	registerAccessLogExplorerPermissions(ctx.PermissionRegistry)
	registerAccessLogExplorerMenu(ctx.MenuRegistry)
	registerAccessLogExplorerRoutes(router, ctx.I18n, repo, authService, authorizer, ctx.EventBus)
	return nil
}

func handleListAccessLogs(localizer *i18n.Service, repo AccessLogRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query, invalidField := bindAccessLogListQuery(ctx)
		if invalidField != "" {
			AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": invalidField,
			})
			return
		}

		result, err := repo.ListAccessLogs(ctx.Request.Context(), query)
		if err != nil {
			AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		WriteSuccess(ctx, http.StatusOK, toAccessLogListResponse(result))
	}
}

func handleGetAccessLogDetail(localizer *i18n.Service, repo AccessLogRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rawID := strings.TrimSpace(ctx.Param(accessLogRouteItemParam))
		id, err := strconv.ParseUint(rawID, 10, 64)
		if err != nil || id == 0 {
			AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": accessLogRouteItemParam,
			})
			return
		}

		record, err := repo.GetAccessLogByID(ctx.Request.Context(), id)
		if err != nil {
			if errors.Is(err, ErrAccessLogNotFound) {
				AbortLocalizedError(ctx, localizer, http.StatusNotFound, "common.not_found", map[string]any{
					"field": accessLogRouteItemParam,
				})
				return
			}
			AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		WriteSuccess(ctx, http.StatusOK, toAccessLogDetailResponse(record))
	}
}

type accessLogListResponse struct {
	Items    []accessLogDetailResponse `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

type accessLogDetailResponse struct {
	ID           uint64  `json:"id"`
	RequestID    string  `json:"request_id"`
	TraceID      string  `json:"trace_id"`
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	Route        string  `json:"route"`
	StatusCode   int     `json:"status_code"`
	DurationMS   int64   `json:"duration_ms"`
	ClientIP     string  `json:"client_ip"`
	UserAgent    string  `json:"user_agent"`
	UserID       *uint64 `json:"user_id,omitempty"`
	Username     string  `json:"username"`
	RequestSize  *int64  `json:"request_size,omitempty"`
	ResponseSize *int64  `json:"response_size,omitempty"`
	OccurredAt   string  `json:"occurred_at"`
}

func toAccessLogListResponse(result AccessLogListResult) accessLogListResponse {
	items := make([]accessLogDetailResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAccessLogDetailResponse(item))
	}

	return accessLogListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}
}

func toAccessLogDetailResponse(record AccessLog) accessLogDetailResponse {
	return accessLogDetailResponse{
		ID:           record.ID,
		RequestID:    record.RequestID,
		TraceID:      record.TraceID,
		Method:       record.Method,
		Path:         record.Path,
		Route:        record.Route,
		StatusCode:   record.StatusCode,
		DurationMS:   record.DurationMS,
		ClientIP:     record.ClientIP,
		UserAgent:    record.UserAgent,
		UserID:       cloneUint64Pointer(record.UserID),
		Username:     record.Username,
		RequestSize:  cloneInt64Pointer(record.RequestSize),
		ResponseSize: cloneInt64Pointer(record.ResponseSize),
		OccurredAt:   record.OccurredAt.UTC().Format(time.RFC3339),
	}
}

var accessLogAllowedListQueryKeys = map[string]struct{}{
	"page":            {},
	"page_size":       {},
	"request_id":      {},
	"trace_id":        {},
	"user_id":         {},
	"username":        {},
	"method":          {},
	"path":            {},
	"path_match":      {},
	"route":           {},
	"status_code":     {},
	"duration_min_ms": {},
	"duration_max_ms": {},
	"occurred_from":   {},
	"occurred_to":     {},
	"sort_by":         {},
	"sort_order":      {},
}

func bindAccessLogListQuery(ctx *gin.Context) (AccessLogListQuery, string) {
	query := AccessLogListQuery{}

	if invalidField := bindAccessLogPagination(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	bindAccessLogIdentityFilters(ctx, &query)
	if invalidField := bindAccessLogNumericFilters(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := bindAccessLogTimeFilters(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := bindAccessLogOrdering(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := rejectUnknownAccessLogListQueryKeys(ctx); invalidField != "" {
		return query, invalidField
	}

	return query, ""
}

func parseOptionalIntQueryValue(raw string) (int, bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, err
	}

	return parsed, true, nil
}

func bindAccessLogPagination(ctx *gin.Context, query *AccessLogListQuery) string {
	page, ok, err := parseOptionalIntQueryValue(ctx.Query("page"))
	if err != nil {
		return "page"
	}
	if ok {
		query.Page = page
	}

	pageSize, ok, err := parseOptionalIntQueryValue(ctx.Query("page_size"))
	if err != nil {
		return "page_size"
	}
	if ok {
		query.PageSize = pageSize
	}

	return ""
}

func bindAccessLogIdentityFilters(ctx *gin.Context, query *AccessLogListQuery) {
	query.RequestID = strings.TrimSpace(ctx.Query("request_id"))
	query.TraceID = strings.TrimSpace(ctx.Query("trace_id"))
	query.Username = strings.TrimSpace(ctx.Query("username"))
	query.Method = strings.TrimSpace(ctx.Query("method"))
	query.Path = strings.TrimSpace(ctx.Query("path"))
	query.Route = strings.TrimSpace(ctx.Query("route"))

	userID, ok, err := parseOptionalUint64QueryValue(ctx.Query("user_id"))
	if err == nil && ok {
		query.UserID = &userID
	}
}

func bindAccessLogNumericFilters(ctx *gin.Context, query *AccessLogListQuery) string {
	userID, ok, err := parseOptionalUint64QueryValue(ctx.Query("user_id"))
	if err != nil {
		return "user_id"
	}
	if ok {
		query.UserID = &userID
	}

	if queryValue := strings.TrimSpace(ctx.Query("status_code")); queryValue != "" {
		value, convErr := strconv.Atoi(queryValue)
		if convErr != nil {
			return "status_code"
		}
		query.StatusCode = &value
	}

	if queryValue := strings.TrimSpace(ctx.Query("duration_min_ms")); queryValue != "" {
		value, convErr := strconv.ParseInt(queryValue, 10, 64)
		if convErr != nil {
			return "duration_min_ms"
		}
		query.DurationMinMS = &value
	}

	if queryValue := strings.TrimSpace(ctx.Query("duration_max_ms")); queryValue != "" {
		value, convErr := strconv.ParseInt(queryValue, 10, 64)
		if convErr != nil {
			return "duration_max_ms"
		}
		query.DurationMaxMS = &value
	}

	return ""
}

func bindAccessLogTimeFilters(ctx *gin.Context, query *AccessLogListQuery) string {
	if queryValue := strings.TrimSpace(ctx.Query("occurred_from")); queryValue != "" {
		value, convErr := time.Parse(time.RFC3339, queryValue)
		if convErr != nil {
			return "occurred_from"
		}
		query.OccurredFrom = &value
	}

	if queryValue := strings.TrimSpace(ctx.Query("occurred_to")); queryValue != "" {
		value, convErr := time.Parse(time.RFC3339, queryValue)
		if convErr != nil {
			return "occurred_to"
		}
		query.OccurredTo = &value
	}

	return ""
}

func bindAccessLogOrdering(ctx *gin.Context, query *AccessLogListQuery) string {
	pathMatch := strings.TrimSpace(ctx.Query("path_match"))
	switch pathMatch {
	case "", string(AccessLogPathMatchExact):
		query.PathMatchMode = AccessLogPathMatchExact
	case string(AccessLogPathMatchPrefix):
		query.PathMatchMode = AccessLogPathMatchPrefix
	default:
		return "path_match"
	}

	sortBy := strings.TrimSpace(ctx.Query("sort_by"))
	switch AccessLogSortField(sortBy) {
	case "", AccessLogSortOccurredAt:
		query.SortBy = AccessLogSortOccurredAt
	case AccessLogSortDurationMS, AccessLogSortStatusCode:
		query.SortBy = AccessLogSortField(sortBy)
	default:
		return "sort_by"
	}

	sortOrder := strings.TrimSpace(ctx.Query("sort_order"))
	switch AccessLogSortOrder(sortOrder) {
	case "", AccessLogSortOrderDesc:
		query.SortOrder = AccessLogSortOrderDesc
	case AccessLogSortOrderAsc:
		query.SortOrder = AccessLogSortOrderAsc
	default:
		return "sort_order"
	}

	return ""
}

func rejectUnknownAccessLogListQueryKeys(ctx *gin.Context) string {
	for key := range ctx.Request.URL.Query() {
		if _, ok := accessLogAllowedListQueryKeys[key]; !ok {
			return key
		}
	}

	return ""
}

func parseOptionalUint64QueryValue(raw string) (uint64, bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false, err
	}

	return parsed, true, nil
}
