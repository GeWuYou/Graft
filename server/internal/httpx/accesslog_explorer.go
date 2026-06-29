package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
)

const (
	// AccessLogReadPermission 约束 access-log explorer 的只读访问权限码。
	AccessLogReadPermission = "access_log.read"
	accessLogMenuRootPath   = "/logs"
	accessLogMenuListPath   = "/logs/access"
	accessLogMenuCodeRoot   = "log-center.root"
	accessLogMenuCodeList   = "access-log.list"
	accessLogModuleOwner    = "core.httpx"
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

func registerAccessLogExplorerPermissions(registry *permission.Registry) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:           AccessLogReadPermission,
		DisplayKey:     "rbac.permissionCatalog.accessLogRead.display",
		DescriptionKey: "rbac.permissionCatalog.accessLogRead.description",
		Category:       "api",
		Module:         accessLogModuleOwner,
	})
}

func registerAccessLogExplorerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       accessLogMenuCodeRoot,
		TitleKey:   "menu.logCenter.title",
		Path:       accessLogMenuRootPath,
		Icon:       "bulletpoint",
		Order:      accessLogMenuRootOrder,
		Permission: "",
		Module:     accessLogModuleOwner,
	})
	registry.Register(menu.Item{
		Code:       accessLogMenuCodeList,
		TitleKey:   "menu.accessLog.title",
		Path:       accessLogMenuListPath,
		Icon:       "search",
		Order:      accessLogMenuListOrder,
		Permission: AccessLogReadPermission,
		Module:     accessLogModuleOwner,
	})
}

func registerAccessLogExplorerRoutes(
	router gin.IRouter,
	localizer *i18n.Service,
	repo AccessLogRepository,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	bus eventbus.Bus,
) {
	if router == nil || repo == nil || authService == nil {
		return
	}

	publisher := NewSecurityAuditPublisher(bus, nil, accessLogModuleOwner)

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
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
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
