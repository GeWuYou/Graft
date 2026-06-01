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
	accessLogSortPartCount  = 2
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
	StartedAt    string  `json:"started_at"`
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
		StartedAt:    record.StartedAt.UTC().Format(time.RFC3339),
		OccurredAt:   record.OccurredAt.UTC().Format(time.RFC3339),
	}
}

var accessLogAllowedListQueryKeys = map[string]struct{}{
	"page":            {},
	"page_size":       {},
	"request_id":      {},
	"trace_id":        {},
	"keyword":         {},
	"user_id":         {},
	"username":        {},
	"method":          {},
	"path":            {},
	"path_match":      {},
	"route":           {},
	"status_code":     {},
	"status_group":    {},
	"duration_min_ms": {},
	"duration_max_ms": {},
	"started_from":    {},
	"started_to":      {},
	"occurred_from":   {},
	"occurred_to":     {},
	"sort":            {},
	"sort[]":          {},
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
	query.Keyword = strings.TrimSpace(ctx.Query("keyword"))
	query.Username = strings.TrimSpace(ctx.Query("username"))
	query.Method = strings.TrimSpace(ctx.Query("method"))
	query.Path = strings.TrimSpace(ctx.Query("path"))
	query.Route = strings.TrimSpace(ctx.Query("route"))
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
	if invalidKey := bindPrimaryAccessLogTimeFilters(ctx, query); invalidKey != "" {
		return invalidKey
	}

	if invalidKey := bindLegacyAccessLogTimeFilters(ctx, query); invalidKey != "" {
		return invalidKey
	}

	return ""
}

func bindAccessLogOrdering(ctx *gin.Context, query *AccessLogListQuery) string {
	if invalidKey := bindAccessLogPathMatch(ctx, query); invalidKey != "" {
		return invalidKey
	}

	if invalidKey := bindAccessLogStatusGroups(ctx, query); invalidKey != "" {
		return invalidKey
	}
	if invalidKey := bindAccessLogSorts(ctx, query); invalidKey != "" {
		return invalidKey
	}

	return ""
}

func bindPrimaryAccessLogTimeFilters(ctx *gin.Context, query *AccessLogListQuery) string {
	startedFrom, invalidKey := parseOptionalRFC3339QueryValue(ctx, "started_from")
	if invalidKey != "" {
		return invalidKey
	}
	query.StartedFrom = startedFrom

	startedTo, invalidKey := parseOptionalRFC3339QueryValue(ctx, "started_to")
	if invalidKey != "" {
		return invalidKey
	}
	query.StartedTo = startedTo

	occurredFrom, invalidKey := parseOptionalRFC3339QueryValue(ctx, "occurred_from")
	if invalidKey != "" {
		return invalidKey
	}
	query.OccurredFrom = occurredFrom

	occurredTo, invalidKey := parseOptionalRFC3339QueryValue(ctx, "occurred_to")
	if invalidKey != "" {
		return invalidKey
	}
	query.OccurredTo = occurredTo

	return ""
}

func bindLegacyAccessLogTimeFilters(_ *gin.Context, _ *AccessLogListQuery) string {
	return ""
}

func bindAccessLogPathMatch(ctx *gin.Context, query *AccessLogListQuery) string {
	pathMatch := strings.TrimSpace(ctx.Query("path_match"))
	switch pathMatch {
	case "", string(AccessLogPathMatchExact):
		query.PathMatchMode = AccessLogPathMatchExact
	case string(AccessLogPathMatchPrefix):
		query.PathMatchMode = AccessLogPathMatchPrefix
	default:
		return "path_match"
	}

	return ""
}

func bindAccessLogStatusGroups(ctx *gin.Context, query *AccessLogListQuery) string {
	rawValues := ctx.QueryArray("status_group")
	if len(rawValues) == 0 {
		return ""
	}

	groups := make([]AccessLogStatusGroup, 0, len(rawValues))
	for _, raw := range rawValues {
		switch AccessLogStatusGroup(strings.TrimSpace(raw)) {
		case AccessLogStatusGroup4xx:
			groups = append(groups, AccessLogStatusGroup4xx)
		case AccessLogStatusGroup5xx:
			groups = append(groups, AccessLogStatusGroup5xx)
		default:
			return "status_group"
		}
	}
	query.StatusGroups = groups
	return ""
}

func bindAccessLogSorts(ctx *gin.Context, query *AccessLogListQuery) string {
	rawSorts := queryArrayCompat(ctx, "sort")
	if len(rawSorts) == 0 {
		return ""
	}

	sorts := make([]AccessLogSort, 0, len(rawSorts))
	for _, raw := range rawSorts {
		parts := strings.Split(strings.TrimSpace(raw), ":")
		if len(parts) != accessLogSortPartCount {
			return "sort"
		}
		field := normalizeAccessLogSortField(AccessLogSortField(strings.TrimSpace(parts[0])))
		if field == "" {
			return "sort"
		}
		order := strings.ToLower(strings.TrimSpace(parts[1]))
		if order != string(AccessLogSortOrderAsc) && order != string(AccessLogSortOrderDesc) {
			return "sort"
		}
		sorts = append(sorts, AccessLogSort{
			Field: field,
			Order: AccessLogSortOrder(order),
		})
	}
	query.Sorts = sorts
	return ""
}

func queryArrayCompat(ctx *gin.Context, key string) []string {
	values := ctx.QueryArray(key)
	bracketValues := ctx.QueryArray(key + "[]")
	if len(bracketValues) == 0 {
		return values
	}

	combined := make([]string, 0, len(values)+len(bracketValues))
	combined = append(combined, values...)
	combined = append(combined, bracketValues...)
	return combined
}

func parseOptionalRFC3339QueryValue(ctx *gin.Context, key string) (*time.Time, string) {
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
