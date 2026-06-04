package logger

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	applogopenapi "graft/server/internal/contract/openapi/applog"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
)

var _ applogopenapi.ReadServerInterface = appLogGeneratedHandler{}

type appLogGeneratedHandler struct{}

func (h appLogGeneratedHandler) GetAppLogs(params applogopenapi.GetAppLogsParams) {
	_ = h
	_ = params
}

func (h appLogGeneratedHandler) GetAppLogDetail(id int64, params applogopenapi.GetAppLogDetailParams) {
	_ = h
	_ = id
	_ = params
}

const (
	// AppLogReadPermission constrains read-only App Log Explorer access.
	AppLogReadPermission = "app_log.read"
	appLogMenuRootPath   = "/logs"
	appLogMenuListPath   = "/logs/app"
	appLogMenuCodeRoot   = "log-center.root"
	appLogMenuCodeList   = "app-log.list"
	appLogModuleOwner    = "core.logger"
	appLogRouteGroup     = "/app-log"
	appLogRouteItemParam = "id"
	appLogMenuRootOrder  = 210
	appLogMenuListOrder  = 212
	appLogSortPartCount  = 2
)

// AppLogExplorerRegistration carries the core registries required by the logger-owned read surface.
type AppLogExplorerRegistration struct {
	I18n               *i18n.Service
	MenuRegistry       *menu.Registry
	PermissionRegistry *permission.Registry
	EventBus           eventbus.Bus
}

type appLogReadGuard struct {
	read gin.HandlerFunc
}

func registerAppLogExplorerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "app-log",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: "menu.logCenter.title", Text: "日志中心"},
				{Key: "menu.appLog.title", Text: "应用日志"},
			},
		},
		{
			Namespace: "app-log",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: "menu.logCenter.title", Text: "Log Center"},
				{Key: "menu.appLog.title", Text: "App Logs"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register app log messages: %w", err)
		}
	}

	return nil
}

func registerAppLogExplorerPermissions(registry *permission.Registry) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:        AppLogReadPermission,
		Name:        "Read App Logs",
		Description: "Allows reading logger-owned app-log explorer data.",
		Category:    "api",
		Module:      appLogModuleOwner,
	})
}

func registerAppLogExplorerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       appLogMenuCodeRoot,
		Title:      "日志中心",
		TitleKey:   "menu.logCenter.title",
		Path:       appLogMenuRootPath,
		Icon:       "bulletpoint",
		Order:      appLogMenuRootOrder,
		Permission: "",
		Module:     appLogModuleOwner,
	})
	registry.Register(menu.Item{
		Code:       appLogMenuCodeList,
		Title:      "应用日志",
		TitleKey:   "menu.appLog.title",
		Path:       appLogMenuListPath,
		Icon:       "file-search",
		Order:      appLogMenuListOrder,
		Permission: AppLogReadPermission,
		Module:     appLogModuleOwner,
	})
}

func registerAppLogExplorerRoutes(
	router gin.IRouter,
	localizer *i18n.Service,
	repo AppLogRepository,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	bus eventbus.Bus,
) {
	if router == nil || repo == nil || authService == nil {
		return
	}

	publisher := httpx.NewSecurityAuditPublisher(bus, nil, appLogModuleOwner)
	guard := appLogReadGuard{
		read: httpx.RequirePermission(localizer, authService, authorizer, AppLogReadPermission, publisher),
	}
	group := router.Group(appLogRouteGroup)
	group.GET("", guard.read, handleListAppLogs(localizer, repo))
	group.GET("/:"+appLogRouteItemParam, guard.read, handleGetAppLogDetail(localizer, repo))
}

// RegisterAppLogExplorer registers the logger-owned App Log Explorer messages, permission, menu, and routes.
func RegisterAppLogExplorer(
	ctx AppLogExplorerRegistration,
	router gin.IRouter,
	repo AppLogRepository,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if err := registerAppLogExplorerMessages(ctx.I18n); err != nil {
		return err
	}
	registerAppLogExplorerPermissions(ctx.PermissionRegistry)
	registerAppLogExplorerMenu(ctx.MenuRegistry)
	registerAppLogExplorerRoutes(router, ctx.I18n, repo, authService, authorizer, ctx.EventBus)
	return nil
}

func handleListAppLogs(localizer *i18n.Service, repo AppLogRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query, invalidField := bindAppLogListQuery(ctx)
		if invalidField != "" {
			httpx.AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": invalidField,
			})
			return
		}

		result, err := repo.ListAppLogs(ctx.Request.Context(), query)
		if err != nil {
			httpx.AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ctx, http.StatusOK, toAppLogListResponse(result))
	}
}

func handleGetAppLogDetail(localizer *i18n.Service, repo AppLogRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rawID := strings.TrimSpace(ctx.Param(appLogRouteItemParam))
		id, err := strconv.ParseUint(rawID, 10, 64)
		if err != nil || id == 0 {
			httpx.AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": appLogRouteItemParam,
			})
			return
		}

		record, err := repo.GetAppLogByID(ctx.Request.Context(), id)
		if err != nil {
			if errors.Is(err, ErrAppLogNotFound) {
				httpx.AbortLocalizedError(ctx, localizer, http.StatusNotFound, "common.not_found", map[string]any{
					"field": appLogRouteItemParam,
				})
				return
			}
			httpx.AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ctx, http.StatusOK, toAppLogDetailResponse(record))
	}
}

type appLogListResponse struct {
	Items    []appLogDetailResponse `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type appLogDetailResponse struct {
	ID         uint64            `json:"id"`
	OccurredAt string            `json:"occurred_at"`
	Severity   string            `json:"severity"`
	Component  string            `json:"component"`
	Message    string            `json:"message"`
	Operation  string            `json:"operation"`
	RequestID  string            `json:"request_id"`
	TraceID    string            `json:"trace_id"`
	Route      string            `json:"route"`
	Method     string            `json:"method"`
	Error      string            `json:"error"`
	Fields     map[string]string `json:"fields"`
}

func toAppLogListResponse(result AppLogListResult) appLogListResponse {
	items := make([]appLogDetailResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAppLogDetailResponse(item))
	}

	return appLogListResponse{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}
}

func toAppLogDetailResponse(record AppLogRecord) appLogDetailResponse {
	fields := record.Fields
	if fields == nil {
		fields = map[string]string{}
	}

	return appLogDetailResponse{
		ID:         record.ID,
		OccurredAt: record.OccurredAt.UTC().Format(time.RFC3339),
		Severity:   string(record.Severity),
		Component:  record.Component,
		Message:    record.Message,
		Operation:  record.Operation,
		RequestID:  record.RequestID,
		TraceID:    record.TraceID,
		Route:      record.Route,
		Method:     record.Method,
		Error:      record.Error,
		Fields:     fields,
	}
}

var appLogAllowedListQueryKeys = map[string]struct{}{
	"page":          {},
	"page_size":     {},
	"occurred_from": {},
	"occurred_to":   {},
	"severity":      {},
	"component":     {},
	"operation":     {},
	"request_id":    {},
	"trace_id":      {},
	"keyword":       {},
	"message":       {},
	"error":         {},
	"sort":          {},
}

func bindAppLogListQuery(ctx *gin.Context) (AppLogListQuery, string) {
	query := AppLogListQuery{}

	if invalidField := rejectUnknownAppLogListQueryKeys(ctx); invalidField != "" {
		return query, invalidField
	}
	if invalidField := bindAppLogPagination(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := bindAppLogSeverity(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := bindAppLogTimeFilters(ctx, &query); invalidField != "" {
		return query, invalidField
	}

	query.Component = strings.TrimSpace(ctx.Query("component"))
	query.Operation = strings.TrimSpace(ctx.Query("operation"))
	query.RequestID = strings.TrimSpace(ctx.Query("request_id"))
	query.TraceID = strings.TrimSpace(ctx.Query("trace_id"))
	query.Keyword = strings.TrimSpace(ctx.Query("keyword"))
	query.Message = strings.TrimSpace(ctx.Query("message"))
	query.Error = strings.TrimSpace(ctx.Query("error"))
	if invalidField := bindAppLogSort(ctx, &query); invalidField != "" {
		return query, invalidField
	}

	return query, ""
}

func bindAppLogSort(ctx *gin.Context, query *AppLogListQuery) string {
	values := ctx.QueryArray("sort")
	if len(values) == 0 {
		return ""
	}

	sorters := make([]AppLogSorter, 0, len(values))
	seen := make(map[AppLogSortField]struct{}, len(values))
	for _, rawValue := range values {
		sorter, ok := parseAppLogSorter(rawValue)
		if !ok {
			return "sort"
		}
		if _, exists := seen[sorter.Field]; exists {
			continue
		}
		seen[sorter.Field] = struct{}{}
		sorters = append(sorters, sorter)
	}

	query.Sorters = sorters
	return ""
}

func parseAppLogSorter(rawValue string) (AppLogSorter, bool) {
	parts := strings.Split(strings.TrimSpace(rawValue), ":")
	if len(parts) == 0 || len(parts) > appLogSortPartCount {
		return AppLogSorter{}, false
	}

	field := AppLogSortField(strings.TrimSpace(parts[0]))
	if !isAllowedAppLogSortField(field) {
		return AppLogSorter{}, false
	}

	order := AppLogSortOrderDesc
	if len(parts) == appLogSortPartCount {
		order = AppLogSortOrder(strings.TrimSpace(parts[1]))
	}
	if order != AppLogSortOrderAsc && order != AppLogSortOrderDesc {
		return AppLogSorter{}, false
	}

	return AppLogSorter{Field: field, Order: order}, true
}

func isAllowedAppLogSortField(field AppLogSortField) bool {
	switch field {
	case AppLogSortFieldOccurredAt, AppLogSortFieldSeverity, AppLogSortFieldComponent:
		return true
	default:
		return false
	}
}

func bindAppLogPagination(ctx *gin.Context, query *AppLogListQuery) string {
	page, ok, err := parseOptionalAppLogIntQueryValue(ctx.Query("page"))
	if err != nil {
		return "page"
	}
	if ok {
		query.Page = page
	}

	pageSize, ok, err := parseOptionalAppLogIntQueryValue(ctx.Query("page_size"))
	if err != nil {
		return "page_size"
	}
	if ok {
		query.PageSize = pageSize
	}

	return ""
}

func bindAppLogSeverity(ctx *gin.Context, query *AppLogListQuery) string {
	rawSeverity := strings.TrimSpace(ctx.Query("severity"))
	if rawSeverity == "" {
		return ""
	}

	severity := AppLogSeverity(rawSeverity)
	if err := severity.Validate(); err != nil {
		return "severity"
	}
	query.Severity = severity
	return ""
}

func bindAppLogTimeFilters(ctx *gin.Context, query *AppLogListQuery) string {
	occurredFrom, invalidKey := parseOptionalAppLogRFC3339QueryValue(ctx, "occurred_from")
	if invalidKey != "" {
		return invalidKey
	}
	query.OccurredFrom = occurredFrom

	occurredTo, invalidKey := parseOptionalAppLogRFC3339QueryValue(ctx, "occurred_to")
	if invalidKey != "" {
		return invalidKey
	}
	query.OccurredTo = occurredTo
	return ""
}

func parseOptionalAppLogIntQueryValue(raw string) (int, bool, error) {
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

func parseOptionalAppLogRFC3339QueryValue(ctx *gin.Context, key string) (*time.Time, string) {
	queryValue := strings.TrimSpace(ctx.Query(key))
	if queryValue == "" {
		return nil, ""
	}

	value, err := time.Parse(time.RFC3339, queryValue)
	if err != nil {
		return nil, key
	}

	return &value, ""
}

func rejectUnknownAppLogListQueryKeys(ctx *gin.Context) string {
	for key := range ctx.Request.URL.Query() {
		if _, ok := appLogAllowedListQueryKeys[key]; !ok {
			return key
		}
	}

	return ""
}
