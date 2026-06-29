package monitor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/internal/redisx"
	"graft/server/internal/statex"
	monitorcontract "graft/server/modules/monitor/contract"
)

const (
	healthCheckTimeout             = 2 * time.Second
	trendSampleInterval            = 5 * time.Second
	maxTrendRetentionWindow        = time.Hour
	trendStorageTTL                = 2 * time.Hour
	samplerShutdownTimeout         = 3 * time.Second
	millisecondsPerSecond          = 1000
	latencyPrecisionScale          = 100
	trendStorageKeyPrefix          = "graft:monitor:server-status:trend"
	statusHealthy                  = "healthy"
	statusDegraded                 = "degraded"
	statusDisabled                 = "disabled"
	statusUnknown                  = "unknown"
	anomalyStatusActive            = "active"
	scopeKindDependency            = "dependency"
	scopeKindModule                = "module"
	scopeKindRuntime               = "runtime"
	scopeKindResource              = "resource"
	evidenceTargetAudit            = "audit_context"
	evidenceStateAvailable         = "available"
	evidenceStateUnavailable       = "unavailable"
	cpuPressureWarningPercent      = 70
	cpuPressureCriticalPercent     = 90
	memoryPressureWarningPercent   = 85
	memoryPressureCriticalPercent  = 95
	diskPressureWarningPercent     = 85
	diskPressureCriticalPercent    = 95
	loadPressureWarningPercent     = 100
	loadPressureCriticalPercent    = 150
	percentageScale                = 100
	goroutinePressureWarningCount  = 200
	goroutinePressureCriticalCount = 500
	runtimeHeapWarningBytes        = 512 * 1024 * 1024
	runtimeHeapCriticalBytes       = 1024 * 1024 * 1024
	serverDependencyCount          = 2
)

// defaultDiskUsagePath returns the default disk usage path for the current operating system.
func defaultDiskUsagePath() string {
	return config.DefaultDiskUsagePath(runtime.GOOS)
}

// Module implements the monitor/server-status slice.
type Module struct {
	startedAtUnixNs atomic.Int64
	db              *sql.DB
	logger          *zap.Logger
	authService     moduleapi.AuthService
	routeAuthorizer moduleapi.Authorizer
	trendStore      statex.TimeSeriesStore
	redisHealth     redisx.HealthReporter

	samplerMu     sync.Mutex
	samplerCancel context.CancelFunc
	samplerDone   chan struct{}
}

var _ monitoropenapi.ServerInterface = (*monitorServerHandler)(nil)

type monitorServerHandler struct {
	ctx        *module.Context
	instance   *Module
	moduleName string
}

type serverStatusAnomalyInputs struct {
	runtimeSnapshot generated.ServerStatusRuntime
	dependencies    generated.ServerStatusDependencies
	modules         []generated.ServerStatusModule
	trend           generated.ServerStatusTrend
}

type metricAnomalySpec struct {
	key       monitorcontract.AnomalyKey
	scopeKind string
	scopeRef  string
	severity  monitorcontract.Severity
	summary   string
}

// NewModule creates the monitor module.
func NewModule() *Module {
	return &Module{}
}

// Register declares menu, permission, routes, and i18n messages.
func (p *Module) Register(ctx *module.Context) error {
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := p.bindDependencies(ctx); err != nil {
		return err
	}

	registerMonitorPermissions(ctx.PermissionRegistry, moduleID)
	registerMonitorMenu(ctx.MenuRegistry, moduleID)
	if err := registerIncidentEvidenceCapability(ctx, p); err != nil {
		return fmt.Errorf("register monitor incident evidence capability: %w", err)
	}
	if err := registerMonitorDashboardWidget(ctx, p); err != nil {
		return err
	}
	registerMonitorRoutes(ctx, p, moduleID, p.authService, p.routeAuthorizer)
	return nil
}

// Boot records the first stable startup timestamp and starts the Redis-backed trend sampler.
func (p *Module) Boot(ctx *module.Context) error {
	p.startedAtUnixNs.CompareAndSwap(0, time.Now().UTC().UnixNano())
	if ctx != nil {
		p.logger = ctx.Logger
	}

	p.startTrendSampler(ctx)
	return nil
}

// Shutdown stops the owned trend sampler before shared runtime resources are released.
func (p *Module) Shutdown(ctx *module.Context) error {
	return p.stopTrendSampler(ctx)
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range []monitorcontract.MessageKey{
			monitorcontract.ServerStatusMenuTitle,
			monitorcontract.ServerStatusOverviewMenuTitle,
			monitorcontract.ServerStatusRuntimeMenuTitle,
			monitorcontract.ServerStatusDependenciesMenuTitle,
			monitorcontract.AuditEvidenceUnavailableTitle,
		} {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key.String()))
			if len(matches) == 0 {
				return fmt.Errorf("register monitor module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}

	return nil
}

func (p *Module) bindDependencies(ctx *module.Context) error {
	db, err := resolveDatabaseDependency(ctx)
	if err != nil {
		return err
	}
	p.db = db
	p.logger = ctx.Logger

	trendStore, err := resolveOptionalTrendStore(ctx)
	if err != nil {
		return err
	}
	p.trendStore = trendStore

	redisHealthReporter, err := resolveOptionalRedisHealthReporter(ctx)
	if err != nil {
		return err
	}
	p.redisHealth = redisHealthReporter

	authResolved, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}

	authService, ok := authResolved.(moduleapi.AuthService)
	if !ok {
		return fmt.Errorf("resolve auth service: unexpected type %T", authResolved)
	}

	authorizerResolved, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return fmt.Errorf("resolve route authorizer: %w", err)
	}

	authorizer, ok := authorizerResolved.(moduleapi.Authorizer)
	if !ok {
		return fmt.Errorf("resolve route authorizer: unexpected type %T", authorizerResolved)
	}

	p.authService = authService
	p.routeAuthorizer = authorizer
	return nil
}

// resolveDatabaseDependency 从模块依赖容器解析可选的 SQL 数据库服务。
// 若上下文不可用或服务未注册，返回 nil；若解析失败或已解析的类型不正确，返回错误。
func resolveDatabaseDependency(ctx *module.Context) (*sql.DB, error) {
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

// resolveOptionalTrendStore resolves an optional time-series store service from the dependency container.
// It returns nil if the context is invalid or the service is not registered and
// returns an error for other resolution failures.
func resolveOptionalTrendStore(ctx *module.Context) (statex.TimeSeriesStore, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, nil
	}

	store, err := module.ResolveService[statex.TimeSeriesStore](ctx.Services, (*statex.TimeSeriesStore)(nil))
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve monitor trend store: %w", err)
	}

	return store, nil
}

// resolveOptionalRedisHealthReporter resolves an optional Redis health reporter service. It returns nil if the context is nil, has no services, or the service is not registered. It returns an error only if service resolution fails for reasons other than the service not being registered.
func resolveOptionalRedisHealthReporter(ctx *module.Context) (redisx.HealthReporter, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, nil
	}

	reporter, err := module.ResolveService[redisx.HealthReporter](ctx.Services, (*redisx.HealthReporter)(nil))
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve redis health reporter: %w", err)
	}

	return reporter, nil
}

// registerMonitorPermissions 注册服务器状态读权限。若注册表为 nil，则直接返回。
func registerMonitorPermissions(registry *permission.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:           monitorcontract.ServerStatusReadPermission.String(),
		Name:           "",
		DisplayKey:     "rbac.permissionCatalog.monitorServerStatusRead.display",
		Description:    "",
		DescriptionKey: "rbac.permissionCatalog.monitorServerStatusRead.description",
		Category:       "api",
		Module:         moduleName,
	})
}

const (
	monitorMenuOrderRoot         = 100
	monitorMenuOrderOverview     = 101
	monitorMenuOrderRuntime      = 102
	monitorMenuOrderDependencies = 103
)

func registerMonitorMenu(registry *menu.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       "monitor.section",
		Title:      "",
		TitleKey:   monitorcontract.ServerStatusMenuTitle.String(),
		Path:       monitorcontract.ServerStatusMenuPath,
		Icon:       "server",
		Order:      monitorMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.overview",
		Title:      "",
		TitleKey:   monitorcontract.ServerStatusOverviewMenuTitle.String(),
		Path:       monitorcontract.ServerStatusOverviewMenuPath,
		Icon:       "dashboard",
		Order:      monitorMenuOrderOverview,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.runtime",
		Title:      "",
		TitleKey:   monitorcontract.ServerStatusRuntimeMenuTitle.String(),
		Path:       monitorcontract.ServerStatusRuntimeMenuPath,
		Icon:       "time",
		Order:      monitorMenuOrderRuntime,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.dependencies",
		Title:      "",
		TitleKey:   monitorcontract.ServerStatusDependenciesMenuTitle.String(),
		Path:       monitorcontract.ServerStatusDependenciesMenuPath,
		Icon:       "data-base",
		Order:      monitorMenuOrderDependencies,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})
}

func registerMonitorRoutes(
	ctx *module.Context,
	instance *Module,
	moduleName string,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) {
	group := ctx.Router.Group(monitorcontract.MonitorGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(
		monitorcontract.ServerStatusRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, monitorcontract.ServerStatusReadPermission.String()),
		newServerStatusHandler(&monitorServerHandler{
			ctx:        ctx,
			instance:   instance,
			moduleName: moduleName,
		}),
	)
}

func newServerStatusHandler(handler *monitorServerHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		params := bindGeneratedMonitorParams(ginCtx)
		if err := handler.GetMonitorServerStatus(ginCtx.Request.Context(), params); err != nil {
			var localizer *i18n.Service
			if handler.ctx != nil {
				localizer = handler.ctx.I18n
				if handler.ctx.Logger != nil {
					logsafe.Error(handler.ctx.Logger, "validate monitor server status params failed",
						zap.String("module", handler.moduleName),
						zap.String("requestId", httpx.EnsureRequestID(ginCtx)),
						zap.Error(err),
					)
				}
			}
			httpx.AbortLocalizedError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}
		trendRange := parseGeneratedTrendRange(params.TrendRange)
		payload, buildErr := buildServerStatusResponse(ginCtx.Request.Context(), handler.ctx, handler.instance, trendRange)
		if buildErr != nil {
			var localizer *i18n.Service
			if handler.ctx != nil {
				localizer = handler.ctx.I18n
				if handler.ctx.Logger != nil {
					logsafe.Error(handler.ctx.Logger, "build monitor server status failed",
						zap.String("module", handler.moduleName),
						zap.String("requestId", httpx.EnsureRequestID(ginCtx)),
						zap.Error(buildErr),
					)
				}
			}
			httpx.AbortLocalizedError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func (h *monitorServerHandler) GetMonitorServerStatus(ctx context.Context, params monitoropenapi.GetMonitorServerStatusParams) error {
	_ = ctx
	_ = params
	return nil
}

func bindGeneratedMonitorParams(ginCtx *gin.Context) monitoropenapi.GetMonitorServerStatusParams {
	params := monitoropenapi.GetMonitorServerStatusParams{}

	if raw := strings.TrimSpace(ginCtx.Query(monitorcontract.TrendRangeQueryKey)); raw != "" {
		value := monitoropenapi.GetMonitorServerStatusParamsTrendRange(raw)
		if value.Valid() {
			params.TrendRange = &value
		}
	}

	if raw := strings.TrimSpace(ginCtx.GetHeader(httpx.RequestIDHeader)); raw != "" {
		params.XRequestId = &raw
	}

	if raw := strings.TrimSpace(ginCtx.GetHeader(string(httpheader.Locale))); raw != "" {
		params.XGraftLocale = &raw
	}

	return params
}

var _ module.Module = (*Module)(nil)
