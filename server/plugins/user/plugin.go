// Package user provides the first sample business plugin wired into the MVP shell.
package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"graft/server/internal/container"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
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

// Register declares user menus, permissions, routes, and public services.
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

	if err := ctx.Services.RegisterSingleton((*pluginapi.UserService)(nil), func(resolver container.Resolver) (any, error) {
		return userService{users: ctx.Stores.Users()}, nil
	}); err != nil {
		return err
	}

	group := ctx.Router.Group("/users")
	group.GET("/:id", func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			ginCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svcAny, err := ctx.Services.Resolve((*pluginapi.UserService)(nil))
		if err != nil {
			ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "resolve user service"})
			return
		}

		svc := svcAny.(pluginapi.UserService)
		summary, err := svc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, store.ErrUserNotFound) {
				status = http.StatusNotFound
			}
			ginCtx.JSON(status, gin.H{"error": err.Error()})
			return
		}

		ginCtx.JSON(http.StatusOK, summary)
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

type userService struct {
	users store.UserRepository
}

func (s userService) GetUserByID(ctx context.Context, id uint64) (pluginapi.UserSummary, error) {
	record, err := s.users.GetByID(ctx, id)
	if err != nil {
		return pluginapi.UserSummary{}, err
	}

	return pluginapi.UserSummary{
		ID:       record.ID,
		Username: record.Username,
		Display:  record.Display,
	}, nil
}

func parseUserID(input string) (uint64, error) {
	id, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse user id %q: %w", input, err)
	}
	if id == 0 {
		return 0, errors.New("id must be greater than zero")
	}
	return id, nil
}
