package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

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
	accessLogMenuListPath   = "/observability/access-logs"
	accessLogMenuCodeList   = "access-log.list"
	accessLogModuleOwner    = "core.httpx"
	accessLogRouteGroup     = "/access-log"
	accessLogRouteItemParam = "id"
	accessLogMenuListOrder  = 211
	accessLogSortPartCount  = 2
)

type accessLogReadGuard struct {
	read gin.HandlerFunc
}

type accessLogExplorerRouteDependencies struct {
	localizer   *i18n.Service
	logger      *zap.Logger
	repo        AccessLogRepository
	authService moduleapi.AuthService
	authorizer  moduleapi.Authorizer
	bus         eventbus.Bus
	savedViews  moduleapi.SavedViewService
}

// AccessLogExplorerRegistration 收口 access-log explorer 所需的 core 注册依赖。
type AccessLogExplorerRegistration struct {
	I18n               *i18n.Service
	Logger             *zap.Logger
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
		Module:         accessLogModuleOwner,
	})
}

// registerAccessLogExplorerMenu registers the access-log explorer entry under the observability menu.
func registerAccessLogExplorerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       accessLogMenuCodeList,
		ParentCode: "domain.observability",
		Kind:       menu.NodeKindEntry,
		TitleKey:   "menu.accessLog.title",
		Path:       accessLogMenuListPath,
		Icon:       "search",
		Order:      accessLogMenuListOrder,
		Permission: AccessLogReadPermission,
		Module:     accessLogModuleOwner,
	})
}

// registerAccessLogExplorerRoutes registers the access-log explorer routes and returns an error when a required dependency is missing.
func registerAccessLogExplorerRoutes(router gin.IRouter, dependencies accessLogExplorerRouteDependencies) error {
	if router == nil || dependencies.repo == nil || dependencies.authService == nil || dependencies.authorizer == nil || dependencies.savedViews == nil {
		return errors.New("access-log explorer dependencies are required")
	}

	publisher := NewSecurityAuditPublisher(dependencies.bus, dependencies.logger, accessLogModuleOwner)

	guard := accessLogReadGuard{
		read: RequirePermissionWithLogger(dependencies.logger, dependencies.localizer, dependencies.authService, dependencies.authorizer, AccessLogReadPermission, publisher),
	}
	group := router.Group(accessLogRouteGroup)
	group.GET("", guard.read, handleListAccessLogs(dependencies.logger, dependencies.localizer, dependencies.repo))
	registerAccessLogSavedViewRoutes(group, dependencies.localizer, guard.read, dependencies.savedViews)
	group.GET("/:"+accessLogRouteItemParam, guard.read, handleGetAccessLogDetail(dependencies.logger, dependencies.localizer, dependencies.repo))
	return nil
}

// RegisterAccessLogExplorer 将访问日志浏览器的权限、菜单和 HTTP 路由注册到核心运行时，并返回路由注册过程中发生的错误。
func RegisterAccessLogExplorer(
	ctx AccessLogExplorerRegistration,
	router gin.IRouter,
	repo AccessLogRepository,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	savedViews moduleapi.SavedViewService,
) error {
	registerAccessLogExplorerPermissions(ctx.PermissionRegistry)
	registerAccessLogExplorerMenu(ctx.MenuRegistry)
	return registerAccessLogExplorerRoutes(router, accessLogExplorerRouteDependencies{
		localizer:   ctx.I18n,
		logger:      ctx.Logger,
		repo:        repo,
		authService: authService,
		authorizer:  authorizer,
		bus:         ctx.EventBus,
		savedViews:  savedViews,
	})
}

// handleListAccessLogs 创建访问日志列表请求处理器。
// 请求参数无效时返回 400，查询失败时返回 500，查询成功时返回访问日志列表。
func handleListAccessLogs(logger *zap.Logger, localizer *i18n.Service, repo AccessLogRepository) gin.HandlerFunc {
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
			AbortAppError(ctx, localizer, logger, err)
			return
		}

		WriteSuccess(ctx, http.StatusOK, toAccessLogListResponse(result))
	}
}

// handleGetAccessLogDetail 返回一个按 ID 查询访问日志详情的处理器，并将查询结果写回响应。
// 当路径参数 `id` 无效、记录不存在或查询失败时，返回相应的本地化错误响应。
func handleGetAccessLogDetail(logger *zap.Logger, localizer *i18n.Service, repo AccessLogRepository) gin.HandlerFunc {
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
			AbortAppError(ctx, localizer, logger, err)
			return
		}

		WriteSuccess(ctx, http.StatusOK, toAccessLogDetailResponse(record))
	}
}
