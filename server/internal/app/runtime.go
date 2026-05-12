// Package app assembles the explicit runtime shell for Graft.
package app

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/httpx"
	"graft/server/internal/menu"
	"graft/server/internal/migration"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
)

// Runtime owns core assembly and plugin lifecycle execution for the MVP shell.
type Runtime struct {
	server             *httpx.Server
	services           *container.Container
	menuRegistry       *menu.Registry
	permissionRegistry *permission.Registry
	migrationRegistry  *migration.Registry
	cronRegistry       *cronx.Registry
	pluginManager      *plugin.Manager
}

// NewRuntime constructs the explicit MVP runtime shell with the provided plugins.
func NewRuntime(plugins ...plugin.Plugin) (*Runtime, error) {
	server := httpx.NewServer()
	services := container.New()
	menuRegistry := menu.NewRegistry()
	permissionRegistry := permission.NewRegistry()
	migrationRegistry := migration.NewRegistry()
	cronRegistry := cronx.NewRegistry()
	pluginManager := plugin.NewManager()

	runtime := &Runtime{
		server:             server,
		services:           services,
		menuRegistry:       menuRegistry,
		permissionRegistry: permissionRegistry,
		migrationRegistry:  migrationRegistry,
		cronRegistry:       cronRegistry,
		pluginManager:      pluginManager,
	}

	runtime.registerCoreRoutes(server.Engine())

	for _, current := range plugins {
		if err := runtime.pluginManager.RegisterPlugin(current); err != nil {
			return nil, err
		}
	}

	return runtime, nil
}

// Run executes Register and Boot before starting the HTTP server.
func (r *Runtime) Run() error {
	ctx := &plugin.Context{
		Router:             r.server.Engine().Group("/api"),
		Services:           r.services,
		MenuRegistry:       r.menuRegistry,
		PermissionRegistry: r.permissionRegistry,
		MigrationRegistry:  r.migrationRegistry,
		CronRegistry:       r.cronRegistry,
	}

	ordered, err := r.pluginManager.Ordered()
	if err != nil {
		return err
	}

	for _, p := range ordered {
		if err := p.Register(ctx); err != nil {
			return err
		}
	}

	for _, p := range ordered {
		if err := p.Boot(ctx); err != nil {
			return err
		}
	}

	return r.server.Run(":8080")
}

func (r *Runtime) registerCoreRoutes(engine *gin.Engine) {
	engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"menus":       len(r.menuRegistry.Items()),
			"permissions": len(r.permissionRegistry.Items()),
			"migrations":  len(r.migrationRegistry.Items()),
			"jobs":        len(r.cronRegistry.Items()),
		})
	})
}
