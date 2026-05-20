package monitor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/container"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	monitorcontract "graft/server/plugins/monitor/contract"
)

const (
	fallbackServerVersion = "dev"
	healthCheckTimeout    = 2 * time.Second
)

// Plugin implements the minimal monitor/server-status slice.
type Plugin struct {
	startedAtUnixNs atomic.Int64
	db              *sql.DB
	authService     pluginapi.AuthService
	routeAuthorizer pluginapi.Authorizer
}

type serverStatusResponse struct {
	Status       string                   `json:"status"`
	ObservedAt   string                   `json:"observed_at"`
	Server       serverStatusServer       `json:"server"`
	Dependencies serverStatusDependencies `json:"dependencies"`
	Plugins      []serverStatusPlugin     `json:"plugins"`
}

type serverStatusServer struct {
	Version       string `json:"version"`
	StartedAt     string `json:"started_at"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	GoVersion     string `json:"go_version"`
	AppName       string `json:"app_name"`
	AppEnv        string `json:"app_env"`
}

type serverStatusDependencies struct {
	Database dependencyStatus `json:"database"`
	Redis    dependencyStatus `json:"redis"`
}

type dependencyStatus struct {
	Status string `json:"status"`
}

type serverStatusPlugin struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

// NewPlugin creates the minimal monitor plugin.
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name returns the stable plugin identifier.
func (p *Plugin) Name() string {
	return pluginID
}

// Version returns the current plugin version.
func (p *Plugin) Version() string {
	return pluginVersion
}

// DependsOn returns the plugin dependencies.
func (p *Plugin) DependsOn() []string {
	return append([]string(nil), pluginDependencies...)
}

// Register declares menu, permission, routes, and i18n messages.
func (p *Plugin) Register(ctx *plugin.Context) error {
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := p.bindDependencies(ctx); err != nil {
		return err
	}

	registerMonitorPermissions(ctx.PermissionRegistry, p.Name())
	registerMonitorMenu(ctx.MenuRegistry, p.Name())
	registerMonitorRoutes(ctx, p, p.Name(), p.authService, p.routeAuthorizer)
	return nil
}

// Boot records the first stable startup timestamp owned by this plugin.
func (p *Plugin) Boot(_ *plugin.Context) error {
	p.startedAtUnixNs.CompareAndSwap(0, time.Now().UTC().UnixNano())
	return nil
}

// Shutdown currently has no owned runtime resources to release.
func (p *Plugin) Shutdown(_ *plugin.Context) error {
	return nil
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "monitor",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(monitorcontract.ServerStatusMenuTitle.String()), Text: "服务器状态"},
			},
		},
		{
			Namespace: "monitor",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(monitorcontract.ServerStatusMenuTitle.String()), Text: "Server Status"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register monitor plugin messages: %w", err)
		}
	}

	return nil
}

func (p *Plugin) bindDependencies(ctx *plugin.Context) error {
	db, err := resolveDatabaseDependency(ctx)
	if err != nil {
		return err
	}
	p.db = db

	authResolved, err := ctx.Services.Resolve((*pluginapi.AuthService)(nil))
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}

	authService, ok := authResolved.(pluginapi.AuthService)
	if !ok {
		return fmt.Errorf("resolve auth service: unexpected type %T", authResolved)
	}

	authorizerResolved, err := ctx.Services.Resolve((*pluginapi.Authorizer)(nil))
	if err != nil {
		return fmt.Errorf("resolve route authorizer: %w", err)
	}

	authorizer, ok := authorizerResolved.(pluginapi.Authorizer)
	if !ok {
		return fmt.Errorf("resolve route authorizer: unexpected type %T", authorizerResolved)
	}

	p.authService = authService
	p.routeAuthorizer = authorizer
	return nil
}

func resolveDatabaseDependency(ctx *plugin.Context) (*sql.DB, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, nil
	}

	resolved, err := ctx.Services.Resolve((*sql.DB)(nil))
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve sql db: %w", err)
	}

	db, ok := resolved.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("resolve sql db: unexpected type %T", resolved)
	}

	return db, nil
}

func registerMonitorPermissions(registry *permission.Registry, pluginName string) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:        monitorcontract.ServerStatusReadPermission.String(),
		Name:        "Read Server Status",
		Description: "Allows reading the minimal server status overview.",
		Category:    "api",
		Plugin:      pluginName,
	})
}

func registerMonitorMenu(registry *menu.Registry, pluginName string) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       "monitor.server-status",
		Title:      "服务器状态",
		TitleKey:   monitorcontract.ServerStatusMenuTitle.String(),
		Path:       monitorcontract.JoinRoute(monitorcontract.MonitorGroup, monitorcontract.ServerStatusRoute),
		Icon:       "chart-bubble",
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Plugin:     pluginName,
	})
}

func registerMonitorRoutes(
	ctx *plugin.Context,
	instance *Plugin,
	pluginName string,
	authService pluginapi.AuthService,
	authorizer pluginapi.Authorizer,
) {
	group := ctx.Router.Group(monitorcontract.MonitorGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(
		monitorcontract.ServerStatusRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, monitorcontract.ServerStatusReadPermission.String()),
		newServerStatusHandler(ctx, instance, pluginName),
	)
}

func newServerStatusHandler(ctx *plugin.Context, instance *Plugin, pluginName string) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		payload, err := buildServerStatusResponse(ginCtx.Request.Context(), ctx, instance)
		if err != nil {
			ctx.Logger.Error("build monitor server status failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func buildServerStatusResponse(
	ctx context.Context,
	pluginCtx *plugin.Context,
	instance *Plugin,
) (serverStatusResponse, error) {
	observedAt := time.Now().UTC()
	startedAt := observedAt
	if instance != nil {
		if startedAtUnixNs := instance.startedAtUnixNs.Load(); startedAtUnixNs > 0 {
			startedAt = time.Unix(0, startedAtUnixNs).UTC()
		}
	}

	databaseStatus := databaseHealth(ctx, instance)
	redisStatus := redisHealth(ctx, pluginCtx)
	plugins := runtimePluginSummaries(pluginCtx)

	return serverStatusResponse{
		Status:     deriveOverallStatus(databaseStatus, redisStatus),
		ObservedAt: observedAt.Format(time.RFC3339),
		Server: serverStatusServer{
			Version:       fallbackServerVersion,
			StartedAt:     startedAt.Format(time.RFC3339),
			UptimeSeconds: int64(observedAt.Sub(startedAt).Seconds()),
			GoVersion:     runtime.Version(),
			AppName:       resolveAppName(pluginCtx),
			AppEnv:        resolveAppEnv(pluginCtx),
		},
		Dependencies: serverStatusDependencies{
			Database: dependencyStatus{Status: databaseStatus},
			Redis:    dependencyStatus{Status: redisStatus},
		},
		Plugins: plugins,
	}, nil
}

func databaseHealth(ctx context.Context, instance *Plugin) string {
	if instance == nil || instance.db == nil {
		return "unknown"
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	if err := instance.db.PingContext(pingCtx); err != nil {
		return "degraded"
	}

	return "healthy"
}

func redisHealth(ctx context.Context, pluginCtx *plugin.Context) string {
	if pluginCtx == nil || pluginCtx.Redis == nil {
		return "disabled"
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	if err := pluginCtx.Redis.Ping(pingCtx).Err(); err != nil {
		return "degraded"
	}

	return "healthy"
}

func runtimePluginSummaries(pluginCtx *plugin.Context) []serverStatusPlugin {
	if pluginCtx == nil {
		return nil
	}

	descriptors := pluginCtx.RuntimeMetadata.OrderedPluginDescriptors()
	items := make([]serverStatusPlugin, 0, len(descriptors))
	for _, descriptor := range descriptors {
		items = append(items, serverStatusPlugin{
			Name:    descriptor.Name,
			Status:  "unknown",
			Version: descriptor.Version,
		})
	}

	return items
}

func resolveAppName(pluginCtx *plugin.Context) string {
	if pluginCtx == nil || pluginCtx.Config == nil {
		return ""
	}
	return strings.TrimSpace(pluginCtx.Config.App.Name)
}

func resolveAppEnv(pluginCtx *plugin.Context) string {
	if pluginCtx == nil || pluginCtx.Config == nil {
		return ""
	}
	return strings.TrimSpace(pluginCtx.Config.App.Env)
}

func deriveOverallStatus(databaseStatus string, redisStatus string) string {
	for _, status := range []string{databaseStatus, redisStatus} {
		if status == "degraded" {
			return "degraded"
		}
	}

	if databaseStatus == "healthy" || redisStatus == "healthy" {
		return "healthy"
	}

	return "unknown"
}

var _ plugin.Plugin = (*Plugin)(nil)
