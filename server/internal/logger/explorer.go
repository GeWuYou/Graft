package logger

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	applogopenapi "graft/server/internal/contract/openapi/applog"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
)

var _ applogopenapi.ServerInterface = appLogGeneratedHandler{}

type appLogGeneratedHandler struct{}

func (h appLogGeneratedHandler) GetAppLogs(params applogopenapi.GetAppLogsParams) {
	_ = h
	_ = params
}

func (appLogGeneratedHandler) GetAppLogSavedViews(applogopenapi.GetAppLogSavedViewsParams) {}

func (appLogGeneratedHandler) PostAppLogSavedView(applogopenapi.PostAppLogSavedViewParams, applogopenapi.PostAppLogSavedViewJSONRequestBody) {
}

func (appLogGeneratedHandler) PutAppLogSavedView(int64, applogopenapi.PutAppLogSavedViewParams, applogopenapi.PutAppLogSavedViewJSONRequestBody) {
}

func (appLogGeneratedHandler) DeleteAppLogSavedView(int64, applogopenapi.DeleteAppLogSavedViewParams) {
}

func (h appLogGeneratedHandler) GetAppLogDetail(id int64, params applogopenapi.GetAppLogDetailParams) {
	_ = h
	_ = id
	_ = params
}

func (h appLogGeneratedHandler) PostAppLogDeletion(
	params applogopenapi.PostAppLogDeletionParams,
	body applogopenapi.PostAppLogDeletionJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

const (
	// AppLogReadPermission constrains read-only App Log Explorer access.
	AppLogReadPermission = "app_log.read"
	// AppLogDeletePermission constrains explicit manual deletion of retained App Log rows.
	AppLogDeletePermission   = "app_log.delete"
	appLogMenuListPath       = "/observability/application-logs"
	appLogMenuCodeList       = "app-log.list"
	appLogModuleOwner        = "core.logger"
	appLogRouteGroup         = "/app-log"
	appLogRouteItemParam     = "id"
	appLogBatchDeleteRoute   = "/batch-delete"
	appLogMenuListOrder      = 212
	appLogSortPartCount      = 2
	appLogManualDeleteAction = "app_log.manual_delete"
	appLogResourceType       = "app_log"
)

// AppLogExplorerRegistration carries the core registries required by the logger-owned read surface.
type AppLogExplorerRegistration struct {
	I18n               *i18n.Service
	MenuRegistry       *menu.Registry
	PermissionRegistry *permission.Registry
	EventBus           eventbus.Bus
}

type appLogReadGuard struct {
	read   gin.HandlerFunc
	delete gin.HandlerFunc
}

type appLogExplorerRouteDependencies struct {
	localizer   *i18n.Service
	repo        AppLogRepository
	authService moduleapi.AuthService
	authorizer  moduleapi.Authorizer
	bus         eventbus.Bus
	savedViews  moduleapi.SavedViewService
}

// registerAppLogExplorerPermissions 注册应用日志读取和删除权限。
func registerAppLogExplorerPermissions(registry *permission.Registry) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:           AppLogReadPermission,
		DisplayKey:     "rbac.permissionCatalog.appLogRead.display",
		DescriptionKey: "rbac.permissionCatalog.appLogRead.description",
		Module:         appLogModuleOwner,
		Resource:       "app_log",
		Action:         "read",
		RiskLevel:      permission.RiskLevelLow,
		RiskCategory:   permission.RiskCategoryRead,
	})
	registry.Register(permission.Item{
		Code:           AppLogDeletePermission,
		DisplayKey:     "rbac.permissionCatalog.appLogDelete.display",
		DescriptionKey: "rbac.permissionCatalog.appLogDelete.description",
		Module:         appLogModuleOwner,
		Resource:       "app_log",
		Action:         "delete",
		RiskLevel:      permission.RiskLevelHigh,
		RiskCategory:   permission.RiskCategoryDestructive,
	})
}

// registerAppLogExplorerMenu 注册应用日志浏览器列表菜单项。
func registerAppLogExplorerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       appLogMenuCodeList,
		ParentCode: "domain.observability",
		Kind:       menu.NodeKindEntry,
		TitleKey:   "menu.appLog.title",
		Path:       appLogMenuListPath,
		Icon:       "application-log",
		Order:      appLogMenuListOrder,
		Permission: AppLogReadPermission,
		Module:     appLogModuleOwner,
	})
}

// registerAppLogExplorerRoutes 注册应用日志浏览器的权限守卫、列表、已保存视图、详情及删除路由。
// 当必需依赖缺失时返回错误；路由注册成功时返回 nil。
func registerAppLogExplorerRoutes(router gin.IRouter, dependencies appLogExplorerRouteDependencies) error {
	if router == nil {
		return errors.New("app log explorer router is required")
	}
	if dependencies.repo == nil {
		return errors.New("app log explorer repository is required")
	}
	if dependencies.authService == nil {
		return errors.New("app log explorer auth service is required")
	}
	if dependencies.authorizer == nil {
		return errors.New("app log explorer authorizer is required")
	}
	if dependencies.savedViews == nil {
		return errors.New("app log explorer saved-view service is required")
	}

	publisher := httpx.NewSecurityAuditPublisher(dependencies.bus, nil, appLogModuleOwner)
	guard := appLogReadGuard{
		read:   httpx.RequirePermission(dependencies.localizer, dependencies.authService, dependencies.authorizer, AppLogReadPermission, publisher),
		delete: httpx.RequirePermission(dependencies.localizer, dependencies.authService, dependencies.authorizer, AppLogDeletePermission, publisher),
	}
	group := router.Group(appLogRouteGroup)
	group.GET("", guard.read, handleListAppLogs(dependencies.localizer, dependencies.repo))
	registerAppLogSavedViewRoutes(group, dependencies.localizer, guard.read, dependencies.savedViews)
	group.POST("/deletions", guard.delete, handleBatchDeleteAppLogs(dependencies.localizer, dependencies.repo, dependencies.bus))
	group.GET("/:"+appLogRouteItemParam, guard.read, handleGetAppLogDetail(dependencies.localizer, dependencies.repo))
	return nil
}

// RegisterAppLogExplorer 将应用日志浏览器的权限、菜单和 HTTP 路由注册到核心运行时，并返回路由注册过程中的错误。
func RegisterAppLogExplorer(
	ctx AppLogExplorerRegistration,
	router gin.IRouter,
	repo AppLogRepository,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	savedViews moduleapi.SavedViewService,
) error {
	registerAppLogExplorerPermissions(ctx.PermissionRegistry)
	registerAppLogExplorerMenu(ctx.MenuRegistry)
	if err := registerAppLogExplorerRoutes(router, appLogExplorerRouteDependencies{
		localizer:   ctx.I18n,
		repo:        repo,
		authService: authService,
		authorizer:  authorizer,
		bus:         ctx.EventBus,
		savedViews:  savedViews,
	}); err != nil {
		return fmt.Errorf("register app log explorer routes: %w", err)
	}
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
			httpx.AbortAppError(ctx, localizer, zap.L(), err)
			return
		}

		httpx.WriteSuccess(ctx, http.StatusOK, toAppLogListResponse(result))
	}
}

func handleGetAppLogDetail(localizer *i18n.Service, repo AppLogRepository) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, ok := bindAppLogID(ctx, localizer)
		if !ok {
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
			httpx.AbortAppError(ctx, localizer, zap.L(), err)
			return
		}

		httpx.WriteSuccess(ctx, http.StatusOK, toAppLogDetailResponse(record))
	}
}

type appLogBatchDeleteRequest struct {
	IDs []uint64 `json:"ids"`
}

func handleBatchDeleteAppLogs(localizer *i18n.Service, repo AppLogRepository, bus eventbus.Bus) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var request appLogBatchDeleteRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			httpx.AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": "ids",
			})
			return
		}

		ids, err := normalizeAppLogDeleteIDs(request.IDs)
		if err != nil {
			httpx.AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": "ids",
			})
			return
		}

		receipted, ok := repo.(interface {
			DeleteAppLogsByIDsWithReceipt(context.Context, []uint64, string) (int64, error)
		})
		var deleted int64
		if ok {
			deleted, err = receipted.DeleteAppLogsByIDsWithReceipt(ctx.Request.Context(), ids, ctx.GetHeader("Idempotency-Key"))
		} else {
			deleted, err = deleteNormalizedAppLogsByIDs(ctx.Request.Context(), repo, ids)
		}
		if err != nil {
			httpx.AbortAppError(ctx, localizer, zap.L(), err)
			return
		}
		if deleted != int64(len(ids)) {
			httpx.AbortLocalizedError(ctx, localizer, http.StatusNotFound, "common.not_found", map[string]any{
				"field": "ids",
			})
			return
		}

		if err := publishAppLogDeleteAudit(ctx, bus, ids, deleted); err != nil {
			httpx.AbortAppError(ctx, localizer, zap.L(), err)
			return
		}
		results := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			results = append(results, map[string]any{"id": strconv.FormatUint(id, 10), "status": "deleted"})
		}
		httpx.WriteSuccess(ctx, http.StatusOK, map[string]any{
			"operation_id": ctx.GetHeader("Idempotency-Key"),
			"summary":      map[string]int{"requested": len(ids), "succeeded": len(ids), "failed": 0},
			"results":      results,
		})
	}
}

func deleteNormalizedAppLogsByIDs(ctx context.Context, repo AppLogRepository, ids []uint64) (int64, error) {
	if storageRepo, ok := repo.(*appLogRepository); ok {
		return storageRepo.deleteAppLogsByNormalizedIDs(ctx, ids)
	}
	return repo.DeleteAppLogsByIDs(ctx, ids)
}

func bindAppLogID(ctx *gin.Context, localizer *i18n.Service) (uint64, bool) {
	rawID := strings.TrimSpace(ctx.Param(appLogRouteItemParam))
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		httpx.AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": appLogRouteItemParam,
		})
		return 0, false
	}

	return id, true
}

func publishAppLogDeleteAudit(ctx *gin.Context, bus eventbus.Bus, ids []uint64, deletedCount int64) error {
	if bus == nil || ctx == nil || ctx.Request == nil {
		return nil
	}

	event := moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindDomain,
		Action:        appLogManualDeleteAction,
		ResourceType:  appLogResourceType,
		RequestMethod: strings.TrimSpace(ctx.Request.Method),
		RequestPath:   strings.TrimSpace(ctx.FullPath()),
		StatusCode:    http.StatusOK,
		RequestID:     httpx.EnsureRequestID(ctx),
		IP:            strings.TrimSpace(ctx.ClientIP()),
		UserAgent:     strings.TrimSpace(ctx.Request.UserAgent()),
		Success:       true,
		Message:       "manual app log delete",
		Metadata: map[string]any{
			"deletedCount":   deletedCount,
			"ids":            ids,
			"retentionOwner": string(AppLogRetentionOwnerLogger),
		},
	}
	if len(ids) == 1 {
		event.ResourceID = strconv.FormatUint(ids[0], 10)
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
		user := *requestAuth.User
		event.Operator = &user
	}

	return bus.Publish(ctx.Request.Context(), eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  appLogModuleOwner,
		Payload: event,
	})
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
	Category   string            `json:"category"`
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
		Category:   string(record.Category),
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
	"category":      {},
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

	if invalidField := bindAppLogCategory(ctx, &query); invalidField != "" {
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

func bindAppLogCategory(ctx *gin.Context, query *AppLogListQuery) string {
	category := LogCategory(strings.TrimSpace(ctx.Query("category")))
	if category == "" {
		return ""
	}
	if !isRegisteredCategory(category) {
		return "category"
	}
	query.Category = category
	return ""
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
	if parsed <= 0 {
		return 0, false, fmt.Errorf("must be positive")
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
