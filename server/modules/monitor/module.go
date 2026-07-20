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
	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger"
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

// defaultDiskUsagePath 返回当前操作系统使用的默认磁盘统计路径。
func defaultDiskUsagePath() string {
	return config.DefaultDiskUsagePath(runtime.GOOS)
}

// Module 实现 monitor 模块的服务器状态、趋势采样和请求性能能力。
// 趋势采样器由模块生命周期管理，Shutdown 必须在共享运行时资源释放前完成收敛。
type Module struct {
	startedAtUnixNs          atomic.Int64
	db                       *sql.DB
	logger                   *zap.Logger
	authService              moduleapi.AuthService
	routeAuthorizer          moduleapi.Authorizer
	trendStore               statex.TimeSeriesStore
	redisHealth              redisx.HealthReporter
	requestPerformanceReader moduleapi.RequestPerformanceReader

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

// NewModule 创建 monitor 模块实例。
func NewModule() *Module {
	return &Module{}
}

// Register 声明 monitor 模块的菜单、权限、路由、本地化消息和公开能力。
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

// Boot 记录首次稳定启动时间，并在趋势存储可用时启动由生命周期上下文约束的趋势采样器。
func (p *Module) Boot(ctx *module.Context) error {
	p.startedAtUnixNs.CompareAndSwap(0, time.Now().UTC().UnixNano())
	if ctx != nil {
		p.logger = ctx.Logger
	}

	p.startTrendSampler(ctx)
	return nil
}

// Shutdown 在共享运行时资源释放前停止 monitor 自有的趋势采样器。
func (p *Module) Shutdown(ctx *module.Context) error {
	return p.stopTrendSampler(ctx)
}

// 当本地化服务不可用或任一必需消息资源缺失时返回错误。
func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range []monitorcontract.MessageKey{
			monitorcontract.ServerStatusOverviewMenuTitle,
			monitorcontract.ServerStatusServiceStatusMenuTitle,
			monitorcontract.ServerStatusDependenciesMenuTitle,
			monitorcontract.RequestPerformanceMenuTitle,
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

	requestPerformanceReader, err := module.ResolveService[moduleapi.RequestPerformanceReader](ctx.Services, (*moduleapi.RequestPerformanceReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve request performance reader: %w", err)
	}
	p.requestPerformanceReader = requestPerformanceReader
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

// resolveOptionalTrendStore 从依赖容器解析可选的时间序列存储服务。
// 上下文无效或服务未注册时返回 nil；其它解析失败返回错误。
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

// resolveOptionalRedisHealthReporter 从依赖容器解析可选的 Redis 健康报告器。
// 上下文无效、服务容器不存在或服务未注册时返回 nil；仅其它解析失败返回错误。
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

// registerMonitorPermissions 向权限注册表注册服务器状态读取权限；注册表为 nil 时不执行任何操作。
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
		Module:         moduleName,
	})
}

const (
	monitorMenuOrderOverview           = 101
	monitorMenuOrderRuntime            = 102
	monitorMenuOrderDependencies       = 103
	monitorMenuOrderRequestPerformance = 104
)

// registerMonitorMenu 注册监控模块的服务器状态和请求性能菜单项；registry 为 nil 时不执行任何操作。
func registerMonitorMenu(registry *menu.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       "monitor.server-status.overview",
		ParentCode: "domain.observability",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   monitorcontract.ServerStatusOverviewMenuTitle.String(),
		Path:       monitorcontract.ServerStatusOverviewMenuPath,
		Icon:       "observability-overview",
		Order:      monitorMenuOrderOverview,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.runtime",
		ParentCode: "domain.observability",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   monitorcontract.ServerStatusServiceStatusMenuTitle.String(),
		Path:       monitorcontract.ServerStatusServiceStatusMenuPath,
		Icon:       "service-health",
		Order:      monitorMenuOrderRuntime,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.dependencies",
		ParentCode: "domain.observability",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   monitorcontract.ServerStatusDependenciesMenuTitle.String(),
		Path:       monitorcontract.ServerStatusDependenciesMenuPath,
		Icon:       "dependency-health",
		Order:      monitorMenuOrderDependencies,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.request-performance",
		ParentCode: "domain.observability",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   monitorcontract.RequestPerformanceMenuTitle.String(),
		Path:       monitorcontract.RequestPerformanceMenuPath,
		Icon:       "request-performance",
		Order:      monitorMenuOrderRequestPerformance,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})
}

// registerMonitorRoutes 注册服务器状态和请求性能 HTTP 路由，并为两者安装请求 ID 中间件和权限校验。
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
	group.GET(
		monitorcontract.RequestPerformanceRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, monitorcontract.ServerStatusReadPermission.String()),
		newRequestPerformanceHandler(&monitorServerHandler{
			ctx:        ctx,
			instance:   instance,
			moduleName: moduleName,
		}),
	)
}

// newServerStatusHandler 创建处理服务器状态请求的 Gin 处理函数，并在成功时返回状态数据，发生错误时返回本地化的内部错误。
func newServerStatusHandler(handler *monitorServerHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		params := bindGeneratedMonitorParams(ginCtx)
		if err := handler.GetMonitorServerStatus(ginCtx.Request.Context(), params); err != nil {
			reported := err
			if handler.ctx != nil && handler.ctx.AppLogger != nil {
				reported = logger.ReportError(ginCtx.Request.Context(), handler.ctx.AppLogger.Named("modules.monitor.server_status"), "validate monitor server status params failed", err,
					logger.StringField("module", handler.moduleName),
					logger.StringField(logger.FieldOperation, "validate_server_status_request"),
				)
			}
			httpx.AbortAppError(ginCtx, handler.localizer(), handler.runtimeLogger(), reported)
			return
		}
		trendRange := parseGeneratedTrendRange(params.TrendRange)
		payload, buildErr := buildServerStatusResponse(ginCtx.Request.Context(), handler.ctx, handler.instance, trendRange)
		if buildErr != nil {
			reported := buildErr
			if handler.ctx != nil && handler.ctx.AppLogger != nil {
				reported = logger.ReportError(ginCtx.Request.Context(), handler.ctx.AppLogger.Named("modules.monitor.server_status"), "build monitor server status failed", buildErr,
					logger.StringField("module", handler.moduleName),
					logger.StringField(logger.FieldOperation, "read_server_status"),
				)
			}
			httpx.AbortAppError(ginCtx, handler.localizer(), handler.runtimeLogger(), reported)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func (h *monitorServerHandler) runtimeLogger() *zap.Logger {
	if h == nil || h.ctx == nil {
		return nil
	}
	return h.ctx.Logger
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
