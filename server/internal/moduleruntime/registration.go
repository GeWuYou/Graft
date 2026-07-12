package moduleruntime

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
)

const (
	moduleOwner         = "core.module-runtime"
	routeGroup          = "/modules/runtime"
	routeModuleKeyParam = "module_key"

	menuCodeRuntime  = "module-runtime.list"
	menuRuntimePath  = "/system/modules"
	menuRuntimeOrder = 104

	menuModulesRuntimeTitleKey = "menu.modulesRuntime.title"
)

// MenuRuntimePath identifies the canonical module runtime menu path.
func MenuRuntimePath() string {
	return menuRuntimePath
}

// MenuRuntimeTitleKey returns the stable module runtime menu title message key.
func MenuRuntimeTitleKey() string {
	return menuModulesRuntimeTitleKey
}

// Registration contains the core dependencies needed to expose module runtime routes and metadata.
type Registration struct {
	I18n               *i18n.Service
	MenuRegistry       *menu.Registry
	PermissionRegistry *permission.Registry
	EventBus           eventbus.Bus
	Config             *config.Config
	Specs              []module.Spec
}

// Register declares module runtime messages, permissions, menu metadata, and read-only HTTP routes.
func Register(
	registration Registration,
	router gin.IRouter,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if err := registerMessages(registration.I18n); err != nil {
		return err
	}
	registerPermissions(registration.PermissionRegistry)
	registerMenu(registration.MenuRegistry)
	if err := registerRoutes(registration, router, authService, authorizer); err != nil {
		return err
	}
	return nil
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(menuModulesRuntimeTitleKey))
		if len(matches) == 0 {
			return fmt.Errorf("register module runtime messages: locale resource %s missing key %s", locale, menuModulesRuntimeTitleKey)
		}
	}

	return nil
}

func registerPermissions(registry *permission.Registry) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:           PermissionRead,
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.moduleRuntimeRead.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.moduleRuntimeRead.description",
		Module:         moduleOwner,
	})
}

// registerMenu 将模块运行时菜单项注册到可用的菜单注册表中。
func registerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       menuCodeRuntime,
		ParentCode: "domain.observability",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   menuModulesRuntimeTitleKey,
		Path:       menuRuntimePath,
		Icon:       "module-runtime",
		Order:      menuRuntimeOrder,
		Permission: PermissionRead,
		Module:     moduleOwner,
	})
}

// registerRoutes 注册模块运行时的只读 HTTP 路由，支持获取运行时快照列表和按模块键获取单项详情。
// 路由要求有效的路由器、认证服务和授权器，并对请求执行读取权限校验。
// 当指定模块不存在时返回本地化的 404 错误；依赖不可用时返回错误。
func registerRoutes(
	registration Registration,
	router gin.IRouter,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if router == nil {
		return errors.New("module runtime router is unavailable")
	}
	if authService == nil {
		return errors.New("module runtime auth service is unavailable")
	}
	if authorizer == nil {
		return errors.New("module runtime authorizer is unavailable")
	}

	group := router.Group(routeGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET("", httpx.RequirePermission(registration.I18n, authService, authorizer, PermissionRead), func(ctx *gin.Context) {
		httpx.WriteSuccess(ctx, http.StatusOK, BuildSnapshot(registration.Config, registration.Specs))
	})
	group.GET("/:"+routeModuleKeyParam, httpx.RequirePermission(registration.I18n, authService, authorizer, PermissionRead), func(ctx *gin.Context) {
		moduleKey := strings.TrimSpace(ctx.Param(routeModuleKeyParam))
		snapshot := BuildSnapshot(registration.Config, registration.Specs)
		for _, item := range snapshot.Items {
			if item.ModuleKey == moduleKey {
				httpx.WriteSuccess(ctx, http.StatusOK, item)
				return
			}
		}

		httpx.AbortLocalizedError(ctx, registration.I18n, http.StatusNotFound, "common.not_found", map[string]any{
			"field": routeModuleKeyParam,
		})
	})

	return nil
}
