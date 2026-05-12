// Package user provides the first sample business plugin wired into the MVP shell.
package user

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"graft/server/internal/container"
	"graft/server/internal/menu"
	"graft/server/internal/migration"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
)

// Plugin is the sample user capability plugin used to prove the extension path.
type Plugin struct{}

// NewPlugin creates the sample user plugin.
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name returns the stable plugin identifier.
func (p *Plugin) Name() string {
	return "user"
}

// Version returns the current sample plugin version.
func (p *Plugin) Version() string {
	return "0.1.0"
}

// DependsOn declares plugin dependencies for startup ordering.
func (p *Plugin) DependsOn() []string {
	return nil
}

// Register declares user menus, permissions, migrations, routes, and public services.
func (p *Plugin) Register(ctx *plugin.Context) error {
	ctx.PermissionRegistry.Register(permission.Item{
		Code:        "user.read",
		Name:        "Read Users",
		Description: "Allows reading user management data.",
		Plugin:      p.Name(),
	})

	ctx.MenuRegistry.Register(menu.Item{
		Code:       "user.list",
		Title:      "用户管理",
		Path:       "/users",
		Icon:       "usergroup",
		Permission: "user.read",
		Plugin:     p.Name(),
	})

	ctx.MigrationRegistry.Register(migration.Item{
		Name:   "202605120001_init_user_tables",
		Plugin: p.Name(),
	})

	if err := ctx.Services.RegisterSingleton((*pluginapi.UserService)(nil), func(resolver container.Resolver) (any, error) {
		return userService{}, nil
	}); err != nil {
		return err
	}

	group := ctx.Router.Group("/users")
	group.GET("/:id", func(ginCtx *gin.Context) {
		ginCtx.JSON(http.StatusOK, gin.H{
			"id":      ginCtx.Param("id"),
			"message": "user plugin shell endpoint",
		})
	})

	return nil
}

// Boot starts user runtime behavior after registration completes.
func (p *Plugin) Boot(ctx *plugin.Context) error {
	return nil
}

// Shutdown releases user runtime resources during application stop.
func (p *Plugin) Shutdown(ctx *plugin.Context) error {
	return nil
}

type userService struct{}

func (s userService) GetUserByID(ctx context.Context, id uint64) (pluginapi.UserSummary, error) {
	if id == 0 {
		return pluginapi.UserSummary{}, errors.New("id must be greater than zero")
	}

	return pluginapi.UserSummary{
		ID:       id,
		Username: "shell-admin",
		Display:  "Shell Admin",
	}, nil
}
