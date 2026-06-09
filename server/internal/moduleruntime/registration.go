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

	menuCodeRoot     = "module-runtime.root"
	menuCodeRuntime  = "module-runtime.list"
	menuRootPath     = "/server"
	menuRuntimePath  = "/server/modules"
	menuRootOrder    = 100
	menuRuntimeOrder = 104

	menuServerTitleKey         = "menu.server.title"
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

	for _, registration := range []i18n.Registration{
		{
			Namespace: "module-runtime",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: menuModulesRuntimeTitleKey, Text: "模块运行时"},
			},
		},
		{
			Namespace: "module-runtime",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: menuModulesRuntimeTitleKey, Text: "Module Runtime"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register module runtime messages: %w", err)
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
		Name:           "Read Module Runtime",
		DisplayKey:     "rbac.permissionCatalog.moduleRuntimeRead.display",
		Description:    "Allows reading the core module runtime snapshot.",
		DescriptionKey: "rbac.permissionCatalog.moduleRuntimeRead.description",
		Category:       "api",
		Module:         moduleOwner,
	})
}

func registerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}

	if !hasMenuPath(registry.Items(), menuRootPath) {
		registry.Register(menu.Item{
			Code:       menuCodeRoot,
			Title:      "服务管理",
			TitleKey:   menuServerTitleKey,
			Path:       menuRootPath,
			Icon:       "server",
			Order:      menuRootOrder,
			Permission: "",
			Module:     moduleOwner,
		})
	}

	registry.Register(menu.Item{
		Code:       menuCodeRuntime,
		Title:      "模块运行时",
		TitleKey:   menuModulesRuntimeTitleKey,
		Path:       menuRuntimePath,
		Icon:       "module",
		Order:      menuRuntimeOrder,
		Permission: PermissionRead,
		Module:     moduleOwner,
	})
}

func hasMenuPath(items []menu.Item, path string) bool {
	for _, item := range items {
		if strings.TrimSpace(item.Path) == path {
			return true
		}
	}

	return false
}

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
