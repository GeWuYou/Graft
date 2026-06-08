package monitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	monitorcontract "graft/server/modules/monitor/contract"
)

const (
	fallbackServerVersion          = "dev"
	healthCheckTimeout             = 2 * time.Second
	trendSampleInterval            = 5 * time.Second
	maxTrendRetentionWindow        = time.Hour
	trendStorageTTL                = 2 * time.Hour
	samplerShutdownTimeout         = 3 * time.Second
	millisecondsPerSecond          = 1000
	latencyPrecisionScale          = 100
	trendStorageKeyPrefix          = "graft:monitor:server-status:trend"
	maxProcessIDInt32              = int64(1<<31 - 1)
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

func defaultDiskUsagePath() string {
	return config.DefaultDiskUsagePath(runtime.GOOS)
}

// Module implements the monitor/server-status slice.
type Module struct {
	startedAtUnixNs atomic.Int64
	db              *sql.DB
	redis           *redis.Client
	logger          *zap.Logger
	authService     moduleapi.AuthService
	routeAuthorizer moduleapi.Authorizer

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
		p.redis = ctx.Redis
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

	for _, registration := range []i18n.Registration{
		{
			Namespace: "monitor",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(monitorcontract.ServerStatusMenuTitle.String()), Text: "服务器管理"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusOverviewMenuTitle.String()), Text: "概览"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusRuntimeMenuTitle.String()), Text: "运行时"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusDependenciesMenuTitle.String()), Text: "依赖服务"},
			},
		},
		{
			Namespace: "monitor",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(monitorcontract.ServerStatusMenuTitle.String()), Text: "Server Management"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusOverviewMenuTitle.String()), Text: "Overview"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusRuntimeMenuTitle.String()), Text: "Runtime"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusDependenciesMenuTitle.String()), Text: "Dependencies"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register monitor module messages: %w", err)
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
	p.redis = ctx.Redis
	p.logger = ctx.Logger

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

func registerMonitorPermissions(registry *permission.Registry, moduleName string) {
	if registry == nil {
		return
	}

	registry.Register(permission.Item{
		Code:           monitorcontract.ServerStatusReadPermission.String(),
		Name:           "Read Server Status",
		DisplayKey:     "rbac.permissionCatalog.monitorServerStatusRead.display",
		Description:    "Allows reading the server status overview.",
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
		Title:      "服务器管理",
		TitleKey:   monitorcontract.ServerStatusMenuTitle.String(),
		Path:       monitorcontract.ServerStatusMenuPath,
		Icon:       "server",
		Order:      monitorMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.overview",
		Title:      "概览",
		TitleKey:   monitorcontract.ServerStatusOverviewMenuTitle.String(),
		Path:       monitorcontract.ServerStatusOverviewMenuPath,
		Icon:       "dashboard",
		Order:      monitorMenuOrderOverview,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.runtime",
		Title:      "运行时",
		TitleKey:   monitorcontract.ServerStatusRuntimeMenuTitle.String(),
		Path:       monitorcontract.ServerStatusRuntimeMenuPath,
		Icon:       "time",
		Order:      monitorMenuOrderRuntime,
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Module:     moduleName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.dependencies",
		Title:      "依赖服务",
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
					handler.ctx.Logger.Error("validate monitor server status params failed",
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
					handler.ctx.Logger.Error("build monitor server status failed",
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

func buildServerStatusResponse(
	ctx context.Context,
	moduleCtx *module.Context,
	instance *Module,
	trendRange monitorcontract.TrendRange,
) (generated.ServerStatusResponse, error) {
	runtimeSnapshot, err := collectRuntimeSnapshot(ctx)
	if err != nil {
		return generated.ServerStatusResponse{}, err
	}
	return buildServerStatusResponseWithRuntimeSnapshot(ctx, moduleCtx, instance, trendRange, runtimeSnapshot)
}

// buildServerStatusResponseWithRuntimeSnapshot keeps the production response assembly
// logic reusable for tests that need deterministic runtime inputs instead of host-dependent metrics.
func buildServerStatusResponseWithRuntimeSnapshot(
	ctx context.Context,
	moduleCtx *module.Context,
	instance *Module,
	trendRange monitorcontract.TrendRange,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusResponse, error) {
	observedAt := time.Now().UTC()
	startedAt := observedAt
	if instance != nil {
		if startedAtUnixNs := instance.startedAtUnixNs.Load(); startedAtUnixNs > 0 {
			startedAt = time.Unix(0, startedAtUnixNs).UTC()
		}
	}

	databaseStatus, err := databaseHealth(ctx, instance)
	if err != nil {
		return generated.ServerStatusResponse{}, err
	}
	redisStatus, err := redisHealth(ctx, moduleCtx)
	if err != nil {
		return generated.ServerStatusResponse{}, err
	}
	modules := runtimeModuleSummaries(moduleCtx, databaseStatus, redisStatus)
	summary := buildServerStatusSummary(databaseStatus, redisStatus, modules)
	trend := buildServerStatusTrend(ctx, moduleCtx, instance, observedAt, trendRange)
	anomalies := buildServerStatusAnomalies(observedAt, trendRange, serverStatusAnomalyInputs{
		runtimeSnapshot: runtimeSnapshot,
		dependencies: generated.ServerStatusDependencies{
			Database: databaseStatus,
			Redis:    redisStatus,
		},
		modules: modules,
		trend:   trend,
	})

	return generated.ServerStatusResponse{
		Status:     deriveOverallStatus(databaseStatus.Status, redisStatus.Status, anomalies),
		ObservedAt: observedAt,
		Server: generated.ServerStatusServer{
			Version:       fallbackServerVersion,
			StartedAt:     startedAt,
			UptimeSeconds: int64(observedAt.Sub(startedAt).Seconds()),
			GoVersion:     runtime.Version(),
			AppName:       resolveAppName(moduleCtx),
			AppEnv:        resolveAppEnv(moduleCtx),
		},
		Runtime: runtimeSnapshot,
		Dependencies: generated.ServerStatusDependencies{
			Database: databaseStatus,
			Redis:    redisStatus,
		},
		Summary:   summary,
		Trend:     trend,
		Modules:   modules,
		Anomalies: anomalies,
	}, nil
}

func buildServerStatusAnomalies(
	observedAt time.Time,
	trendRange monitorcontract.TrendRange,
	inputs serverStatusAnomalyInputs,
) []generated.ServerStatusAnomaly {
	windowStart := observedAt.Add(-trendRange.Duration())
	anomalies := make([]generated.ServerStatusAnomaly, 0)

	anomalies = append(anomalies, buildDependencyAnomalies(observedAt, windowStart, inputs.dependencies)...)
	anomalies = append(anomalies, buildModuleDependencyAnomalies(observedAt, windowStart, inputs.modules)...)
	anomalies = append(anomalies, buildRuntimeMetricAnomalies(observedAt, windowStart, inputs.runtimeSnapshot, inputs.trend)...)
	return anomalies
}

func buildDependencyAnomalies(
	observedAt time.Time,
	windowStart time.Time,
	dependencies generated.ServerStatusDependencies,
) []generated.ServerStatusAnomaly {
	anomalies := make([]generated.ServerStatusAnomaly, 0, serverDependencyCount)
	appendDependencyAnomaly := func(scopeRef string, dependency generated.ServerStatusDependency) {
		switch dependency.Status {
		case statusDegraded:
			anomalies = append(anomalies, generated.ServerStatusAnomaly{
				AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(monitorcontract.DependencyStatusDegraded),
				ScopeKind:  generated.ServerStatusAnomalyScopeKind(scopeKindDependency),
				ScopeRef:   scopeRef,
				Severity:   generated.ServerStatusAnomalySeverity(monitorcontract.SeverityCritical),
				Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
				ObservedAt: observedAt,
				Summary:    dependency.Detail,
				EvidenceLinks: []generated.EvidenceLink{
					unavailableEvidenceLink(windowStart, observedAt, "Audit evidence is not available for this dependency health issue."),
				},
			})
		case statusUnknown:
			anomalies = append(anomalies, generated.ServerStatusAnomaly{
				AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(monitorcontract.DependencyStatusUnknown),
				ScopeKind:  generated.ServerStatusAnomalyScopeKind(scopeKindDependency),
				ScopeRef:   scopeRef,
				Severity:   generated.ServerStatusAnomalySeverity(monitorcontract.SeverityWarning),
				Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
				ObservedAt: observedAt,
				Summary:    dependency.Detail,
				EvidenceLinks: []generated.EvidenceLink{
					unavailableEvidenceLink(windowStart, observedAt, "Audit evidence is not available for this dependency observability gap."),
				},
			})
		}
	}

	appendDependencyAnomaly("database", dependencies.Database)
	appendDependencyAnomaly("redis", dependencies.Redis)

	return anomalies
}

func buildModuleDependencyAnomalies(
	observedAt time.Time,
	windowStart time.Time,
	modules []generated.ServerStatusModule,
) []generated.ServerStatusAnomaly {
	anomalies := make([]generated.ServerStatusAnomaly, 0)
	for _, item := range modules {
		if item.MissingDependencies == nil || len(*item.MissingDependencies) == 0 {
			continue
		}
		anomalies = append(anomalies, generated.ServerStatusAnomaly{
			AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(monitorcontract.ModuleDependencyMissing),
			ScopeKind:  generated.ServerStatusAnomalyScopeKind(scopeKindModule),
			ScopeRef:   item.Name,
			Severity:   generated.ServerStatusAnomalySeverity(monitorcontract.SeverityCritical),
			Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
			ObservedAt: observedAt,
			Summary:    item.StatusDetail,
			EvidenceLinks: []generated.EvidenceLink{
				unavailableEvidenceLink(windowStart, observedAt, "Audit evidence is not available for this module dependency issue."),
			},
		})
	}
	return anomalies
}

func buildRuntimeMetricAnomalies(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
	trend generated.ServerStatusTrend,
) []generated.ServerStatusAnomaly {
	anomalies := make([]generated.ServerStatusAnomaly, 0)

	if cpuAnomaly, ok := buildCPUAnomaly(observedAt, windowStart, trend); ok {
		anomalies = append(anomalies, cpuAnomaly)
	}
	if memoryAnomaly, ok := buildMemoryAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, memoryAnomaly)
	}
	if diskAnomaly, ok := buildDiskAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, diskAnomaly)
	}
	if loadAnomaly, ok := buildLoadAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, loadAnomaly)
	}
	if goroutineAnomaly, ok := buildGoroutineAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, goroutineAnomaly)
	}
	if heapAnomaly, ok := buildHeapAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, heapAnomaly)
	}

	return anomalies
}

func buildCPUAnomaly(observedAt time.Time, windowStart time.Time, trend generated.ServerStatusTrend) (generated.ServerStatusAnomaly, bool) {
	cpuPercent, ok := latestTrendCPUPercent(trend)
	if !ok {
		return generated.ServerStatusAnomaly{}, false
	}
	severity, hit := classifyPercentSeverity(cpuPercent, cpuPressureWarningPercent, cpuPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.ResourceCPUPressure,
			scopeKind: scopeKindResource,
			scopeRef:  "runtime.cpu",
			severity:  severity,
			summary:   fmt.Sprintf("CPU usage reached %.1f%% in the current monitor window.", cpuPercent),
		},
	), true
}

func buildMemoryAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	severity, hit := classifyPercentSeverity(float64(runtimeSnapshot.HostMemoryUsedPercent), memoryPressureWarningPercent, memoryPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.ResourceMemoryPressure,
			scopeKind: scopeKindResource,
			scopeRef:  "runtime.host_memory",
			severity:  severity,
			summary:   fmt.Sprintf("Server memory usage reached %.1f%%.", float64(runtimeSnapshot.HostMemoryUsedPercent)),
		},
	), true
}

func buildDiskAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	if runtimeSnapshot.DiskUsage.TotalBytes <= 0 {
		return generated.ServerStatusAnomaly{}, false
	}
	severity, hit := classifyPercentSeverity(float64(runtimeSnapshot.DiskUsage.UsedPercent), diskPressureWarningPercent, diskPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.ResourceDiskPressure,
			scopeKind: scopeKindResource,
			scopeRef:  fmt.Sprintf("disk:%s", runtimeSnapshot.DiskUsage.Path),
			severity:  severity,
			summary:   fmt.Sprintf("Disk usage on %s reached %.1f%%.", runtimeSnapshot.DiskUsage.Path, float64(runtimeSnapshot.DiskUsage.UsedPercent)),
		},
	), true
}

func buildLoadAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	loadPercent := 0.0
	if runtimeSnapshot.CpuCores > 0 {
		loadPercent = (float64(runtimeSnapshot.LoadAverage.OneMinute) / float64(runtimeSnapshot.CpuCores)) * percentageScale
	}
	severity, hit := classifyPercentSeverity(loadPercent, loadPressureWarningPercent, loadPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.SystemLoadPressure,
			scopeKind: scopeKindRuntime,
			scopeRef:  "runtime.load",
			severity:  severity,
			summary:   fmt.Sprintf("1-minute load average reached %.2f against %d CPU cores.", float64(runtimeSnapshot.LoadAverage.OneMinute), runtimeSnapshot.CpuCores),
		},
	), true
}

func buildGoroutineAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	severity, hit := classifyCountSeverity(runtimeSnapshot.Goroutines, goroutinePressureWarningCount, goroutinePressureCriticalCount)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.RuntimeGoroutinePressure,
			scopeKind: scopeKindRuntime,
			scopeRef:  "runtime.goroutines",
			severity:  severity,
			summary:   fmt.Sprintf("Goroutine count reached %d.", runtimeSnapshot.Goroutines),
		},
	), true
}

func buildHeapAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	severity, hit := classifyInt64Severity(runtimeSnapshot.RuntimeHeapInUseBytes, runtimeHeapWarningBytes, runtimeHeapCriticalBytes)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.RuntimeHeapPressure,
			scopeKind: scopeKindRuntime,
			scopeRef:  "runtime.heap_in_use",
			severity:  severity,
			summary:   fmt.Sprintf("Runtime heap usage reached %d bytes.", runtimeSnapshot.RuntimeHeapInUseBytes),
		},
	), true
}

func newMetricAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	spec metricAnomalySpec,
) generated.ServerStatusAnomaly {
	return generated.ServerStatusAnomaly{
		AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(spec.key),
		ScopeKind:  generated.ServerStatusAnomalyScopeKind(spec.scopeKind),
		ScopeRef:   spec.scopeRef,
		Severity:   generated.ServerStatusAnomalySeverity(spec.severity),
		Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
		ObservedAt: observedAt,
		Summary:    spec.summary,
		EvidenceLinks: []generated.EvidenceLink{
			availableEvidenceLink(windowStart, observedAt, "Review related audit activity", "Check audit records from the same bounded monitor window."),
		},
	}
}

func latestTrendCPUPercent(trend generated.ServerStatusTrend) (float64, bool) {
	if len(trend.Points) == 0 {
		return 0, false
	}
	return float64(trend.Points[len(trend.Points)-1].CpuPercent), true
}

func classifyPercentSeverity(value float64, warningThreshold float64, criticalThreshold float64) (monitorcontract.Severity, bool) {
	if value >= criticalThreshold {
		return monitorcontract.SeverityCritical, true
	}
	if value >= warningThreshold {
		return monitorcontract.SeverityWarning, true
	}
	return "", false
}

func classifyCountSeverity(value int, warningThreshold int, criticalThreshold int) (monitorcontract.Severity, bool) {
	if value >= criticalThreshold {
		return monitorcontract.SeverityCritical, true
	}
	if value >= warningThreshold {
		return monitorcontract.SeverityWarning, true
	}
	return "", false
}

func classifyInt64Severity(value int64, warningThreshold int64, criticalThreshold int64) (monitorcontract.Severity, bool) {
	if value >= criticalThreshold {
		return monitorcontract.SeverityCritical, true
	}
	if value >= warningThreshold {
		return monitorcontract.SeverityWarning, true
	}
	return "", false
}

func availableEvidenceLink(windowStart time.Time, windowEnd time.Time, title string, reason string) generated.EvidenceLink {
	return generated.EvidenceLink{
		TargetKind: generated.EvidenceLinkTargetKind(evidenceTargetAudit),
		LinkState:  generated.EvidenceLinkLinkState(evidenceStateAvailable),
		Title:      title,
		Reason:     stringPointer(reason),
		TimeWindow: &generated.EvidenceLinkTimeWindow{
			CreatedFrom: windowStart,
			CreatedTo:   windowEnd,
		},
		AuditContext: &generated.AuditEvidenceContext{
			CreatedFrom: &windowStart,
			CreatedTo:   &windowEnd,
		},
	}
}

func unavailableEvidenceLink(windowStart time.Time, windowEnd time.Time, reason string) generated.EvidenceLink {
	return generated.EvidenceLink{
		TargetKind: generated.EvidenceLinkTargetKind(evidenceTargetAudit),
		LinkState:  generated.EvidenceLinkLinkState(evidenceStateUnavailable),
		Title:      "Audit evidence is unavailable",
		Reason:     stringPointer(reason),
		TimeWindow: &generated.EvidenceLinkTimeWindow{
			CreatedFrom: windowStart,
			CreatedTo:   windowEnd,
		},
	}
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func databaseHealth(ctx context.Context, instance *Module) (generated.ServerStatusDependency, error) {
	if instance == nil || instance.db == nil {
		return generated.ServerStatusDependency{
			Status: statusUnknown,
			Detail: "Database handle is unavailable",
		}, nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	startedAt := time.Now()
	if err := instance.db.PingContext(pingCtx); err != nil {
		logTrendWarning(instance, nil, "database ping failed", err)
		return generated.ServerStatusDependency{
			Status: statusDegraded,
			Detail: "Database ping failed",
		}, nil
	}

	latencyMs, err := toGeneratedFloat32(roundLatencyMilliseconds(time.Since(startedAt)), "database latency ms")
	if err != nil {
		return generated.ServerStatusDependency{}, fmt.Errorf("convert database latency: %w", err)
	}
	return generated.ServerStatusDependency{
		Status:    statusHealthy,
		Detail:    "Database ping succeeded",
		LatencyMs: &latencyMs,
	}, nil
}

func redisHealth(ctx context.Context, moduleCtx *module.Context) (generated.ServerStatusDependency, error) {
	if moduleCtx == nil || moduleCtx.Redis == nil {
		return generated.ServerStatusDependency{
			Status: statusDisabled,
			Detail: "Redis client is not configured",
		}, nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	startedAt := time.Now()
	if err := moduleCtx.Redis.Ping(pingCtx).Err(); err != nil {
		logTrendWarning(nil, moduleCtx, "redis ping failed", err)
		return generated.ServerStatusDependency{
			Status: statusDegraded,
			Detail: "Redis ping failed",
		}, nil
	}

	latencyMs, err := toGeneratedFloat32(roundLatencyMilliseconds(time.Since(startedAt)), "redis latency ms")
	if err != nil {
		return generated.ServerStatusDependency{}, fmt.Errorf("convert redis latency: %w", err)
	}
	return generated.ServerStatusDependency{
		Status:    statusHealthy,
		Detail:    "Redis ping succeeded",
		LatencyMs: &latencyMs,
	}, nil
}

func runtimeModuleSummaries(
	moduleCtx *module.Context,
	database generated.ServerStatusDependency,
	redis generated.ServerStatusDependency,
) []generated.ServerStatusModule {
	if moduleCtx == nil {
		return nil
	}

	descriptors := moduleCtx.RuntimeMetadata.OrderedModuleDescriptors()
	available := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		name := strings.TrimSpace(descriptor.Name)
		if name == "" {
			continue
		}
		available[name] = struct{}{}
	}

	platformStatus := deriveOverallStatus(database.Status, redis.Status, nil)
	items := make([]generated.ServerStatusModule, 0, len(descriptors))
	for _, descriptor := range descriptors {
		dependsOn := append([]string(nil), descriptor.DependsOn...)
		status, statusDetail, missingDependencies := deriveRuntimeModuleObservation(descriptor, available, platformStatus)
		item := generated.ServerStatusModule{
			Name:         descriptor.Name,
			Status:       status,
			StatusDetail: statusDetail,
			DependsOn:    dependsOn,
		}
		if len(missingDependencies) > 0 {
			missing := append([]string(nil), missingDependencies...)
			item.MissingDependencies = &missing
		}
		items = append(items, item)
	}

	return items
}

// deriveRuntimeModuleObservation keeps module runtime semantics explicit and narrow:
// a module is healthy only when its runtime metadata is complete, its declared
// module dependencies are present, and the current shared runtime signals are not
// degraded. When that cannot be confirmed, the returned detail explains the most
// useful operator-facing reason instead of collapsing everything into a coarse summary.
func deriveRuntimeModuleObservation(
	descriptor module.DescriptorSnapshot,
	available map[string]struct{},
	platformStatus string,
) (status string, detail string, missingDependencies []string) {
	if strings.TrimSpace(descriptor.Name) == "" {
		return statusUnknown, "Runtime metadata is incomplete", nil
	}

	for _, dependency := range descriptor.DependsOn {
		dependencyName := strings.TrimSpace(dependency)
		if dependencyName == "" {
			continue
		}
		if _, ok := available[dependencyName]; !ok {
			missingDependencies = append(missingDependencies, dependencyName)
		}
	}

	if len(missingDependencies) > 0 {
		return statusDegraded,
			fmt.Sprintf("Missing runtime dependencies: %s", strings.Join(missingDependencies, ", ")),
			missingDependencies
	}

	switch platformStatus {
	case statusHealthy:
		return statusHealthy, "Runtime metadata is present and platform signals are healthy", nil
	case statusDegraded:
		return statusDegraded, "Runtime metadata is present, but shared runtime signals are degraded", nil
	default:
		return statusUnknown, "Runtime status is not fully observable from shared platform signals", nil
	}
}

func buildServerStatusSummary(
	database generated.ServerStatusDependency,
	redis generated.ServerStatusDependency,
	modules []generated.ServerStatusModule,
) generated.ServerStatusSummary {
	summary := generated.ServerStatusSummary{
		TotalDependencies: len([]generated.ServerStatusDependency{database, redis}),
		TotalModules:      len(modules),
	}

	for _, dependency := range []generated.ServerStatusDependency{database, redis} {
		switch dependency.Status {
		case statusHealthy:
			summary.HealthyDependencies++
		case statusDegraded:
			summary.DegradedDependencies++
		case statusDisabled:
			summary.DisabledDependencies++
		default:
			summary.UnknownDependencies++
		}
	}

	for _, moduleSummary := range modules {
		if moduleSummary.Status == statusHealthy {
			summary.HealthyModules++
		}
	}

	return summary
}

func buildServerStatusTrend(
	ctx context.Context,
	moduleCtx *module.Context,
	instance *Module,
	observedAt time.Time,
	trendRange monitorcontract.TrendRange,
) generated.ServerStatusTrend {
	retention := trendRange.Duration()
	trend := generated.ServerStatusTrend{
		Range:                 generated.ServerStatusTrendRange(trendRange.String()),
		RetentionSeconds:      int64(retention.Seconds()),
		SampleIntervalSeconds: int64(trendSampleInterval.Seconds()),
		Points:                nil,
	}

	redisClient := resolveRedisClient(moduleCtx, instance)
	if redisClient == nil {
		return trend
	}

	points, err := loadTrendPoints(ctx, redisClient, trendStorageKey(resolveAppName(moduleCtx), resolveHostName()), observedAt, retention)
	if err != nil {
		logTrendWarning(instance, moduleCtx, "load redis trend points failed", err)
		return trend
	}

	trend.Points = points
	return trend
}

func resolveRedisClient(moduleCtx *module.Context, instance *Module) *redis.Client {
	if instance != nil && instance.redis != nil {
		return instance.redis
	}
	if moduleCtx != nil {
		return moduleCtx.Redis
	}
	return nil
}

func (p *Module) startTrendSampler(ctx *module.Context) {
	if p == nil || ctx == nil || ctx.Redis == nil || ctx.LifecycleContext == nil {
		return
	}

	p.samplerMu.Lock()
	defer p.samplerMu.Unlock()

	if p.samplerCancel != nil {
		return
	}

	runCtx, cancel := context.WithCancel(ctx.LifecycleContext)
	done := make(chan struct{})
	p.samplerCancel = cancel
	p.samplerDone = done

	storageKey := trendStorageKey(resolveAppName(ctx), resolveHostName())
	go func() {
		defer close(done)
		p.runTrendSampler(runCtx, ctx.Redis, storageKey)
	}()
}

func (p *Module) stopTrendSampler(ctx *module.Context) error {
	if p == nil {
		return nil
	}

	p.samplerMu.Lock()
	cancel := p.samplerCancel
	done := p.samplerDone
	p.samplerCancel = nil
	p.samplerDone = nil
	p.samplerMu.Unlock()

	if cancel == nil || done == nil {
		return nil
	}

	cancel()

	if ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("monitor trend sampler shutdown missing lifecycle context")
	}
	waitCtx := ctx.LifecycleContext

	select {
	case <-done:
		return nil
	case <-waitCtx.Done():
		return waitCtx.Err()
	case <-time.After(samplerShutdownTimeout):
		return errors.New("monitor trend sampler shutdown timed out")
	}
}

func (p *Module) runTrendSampler(ctx context.Context, redisClient *redis.Client, storageKey string) {
	var processHandle *process.Process
	processID, err := currentProcessID()
	if err != nil {
		logTrendWarning(p, nil, "resolve monitor cpu sampler pid failed", err)
	} else {
		processHandle, err = process.NewProcessWithContext(ctx, processID)
		if err != nil {
			logTrendWarning(p, nil, "initialize monitor cpu sampler failed", err)
			processHandle = nil
		}
	}

	// Prime the CPU sampler before the first stored sample.
	if processHandle != nil {
		_, _ = processHandle.CPUPercentWithContext(ctx)
	}

	p.recordTrendSample(ctx, redisClient, storageKey, processHandle)

	ticker := time.NewTicker(trendSampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.recordTrendSample(ctx, redisClient, storageKey, processHandle)
		}
	}
}

func (p *Module) recordTrendSample(
	ctx context.Context,
	redisClient *redis.Client,
	storageKey string,
	processHandle *process.Process,
) {
	if redisClient == nil {
		return
	}

	runtimeSnapshot, err := collectRuntimeSnapshot(ctx)
	if err != nil {
		logTrendWarning(p, nil, "collect monitor runtime snapshot failed", err)
		return
	}
	cpuPercent, err := toGeneratedFloat32(collectCPUPercent(ctx, processHandle), "cpu percent")
	if err != nil {
		logTrendWarning(p, nil, "convert monitor cpu sample failed", err)
		return
	}
	observedAt := time.Now().UTC()
	point := generated.ServerStatusTrendPoint{
		ObservedAt:                observedAt,
		CpuPercent:                cpuPercent,
		HostMemoryUsedPercent:     runtimeSnapshot.HostMemoryUsedPercent,
		LoadAverageOneMinute:      runtimeSnapshot.LoadAverage.OneMinute,
		LoadAverageFiveMinutes:    runtimeSnapshot.LoadAverage.FiveMinutes,
		LoadAverageFifteenMinutes: runtimeSnapshot.LoadAverage.FifteenMinutes,
		Goroutines:                runtimeSnapshot.Goroutines,
		RuntimeAllocBytes:         runtimeSnapshot.RuntimeAllocBytes,
		RuntimeHeapInUseBytes:     runtimeSnapshot.RuntimeHeapInUseBytes,
		RuntimeSysBytes:           runtimeSnapshot.RuntimeSysBytes,
	}

	if err := storeTrendPoint(ctx, redisClient, storageKey, observedAt, point); err != nil {
		logTrendWarning(p, nil, "store monitor trend sample failed", err)
	}
}

func collectCPUPercent(ctx context.Context, processHandle *process.Process) float64 {
	if processHandle == nil {
		return 0
	}

	percent, err := processHandle.CPUPercentWithContext(ctx)
	if err != nil {
		return 0
	}

	return roundCPUPercent(percent)
}

func storeTrendPoint(
	ctx context.Context,
	redisClient *redis.Client,
	storageKey string,
	observedAt time.Time,
	point generated.ServerStatusTrendPoint,
) error {
	payload, err := json.Marshal(point)
	if err != nil {
		return fmt.Errorf("marshal trend point: %w", err)
	}

	observedAtMillis := observedAt.UnixMilli()
	cutoffMillis := observedAt.Add(-maxTrendRetentionWindow).UnixMilli()
	pipe := redisClient.TxPipeline()
	pipe.ZAdd(ctx, storageKey, redis.Z{
		Score:  float64(observedAtMillis),
		Member: string(payload),
	})
	pipe.ZRemRangeByScore(ctx, storageKey, "-inf", strconv.FormatInt(cutoffMillis, 10))
	pipe.Expire(ctx, storageKey, trendStorageTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("exec redis trend pipeline: %w", err)
	}

	return nil
}

func loadTrendPoints(
	ctx context.Context,
	redisClient *redis.Client,
	storageKey string,
	observedAt time.Time,
	retention time.Duration,
) ([]generated.ServerStatusTrendPoint, error) {
	if redisClient == nil {
		return nil, nil
	}

	minScore := strconv.FormatInt(observedAt.Add(-retention).UnixMilli(), 10)
	maxScore := strconv.FormatInt(observedAt.UnixMilli(), 10)
	members, err := redisClient.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     storageKey,
		Start:   minScore,
		Stop:    maxScore,
		ByScore: true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("range redis trend points: %w", err)
	}

	points := make([]generated.ServerStatusTrendPoint, 0, len(members))
	for _, member := range members {
		var point generated.ServerStatusTrendPoint
		if err := json.Unmarshal([]byte(member), &point); err != nil {
			continue
		}
		points = append(points, point)
	}

	return points, nil
}

func trendStorageKey(appName string, hostName string) string {
	resolvedAppName := sanitizeTrendKeySegment(appName)
	if resolvedAppName == "" {
		resolvedAppName = "app"
	}

	resolvedHostName := sanitizeTrendKeySegment(hostName)
	if resolvedHostName == "" {
		resolvedHostName = "host"
	}

	return fmt.Sprintf("%s:%s:%s", trendStorageKeyPrefix, resolvedAppName, resolvedHostName)
}

func sanitizeTrendKeySegment(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", ".", "-")
	return replacer.Replace(trimmed)
}

func resolveHostName() string {
	hostName, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(hostName)
}

func currentProcessID() (int32, error) {
	pid := os.Getpid()
	if pid < 0 || int64(pid) > maxProcessIDInt32 {
		return 0, fmt.Errorf("current pid %d overflows int32", pid)
	}

	return int32(pid), nil
}

func collectRuntimeSnapshot(ctx context.Context) (generated.ServerStatusRuntime, error) {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)
	hostMemory := collectHostMemory(ctx)
	loadAverage, err := collectLoadAverage(ctx)
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	diskUsage, err := collectDiskUsage(ctx, defaultDiskUsagePath())
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	hostMemoryTotalBytes, err := mustConvertGeneratedInt64(hostMemory.Total, "host memory total bytes")
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	hostMemoryUsedBytes, err := mustConvertGeneratedInt64(hostMemory.Used, "host memory used bytes")
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	hostMemoryFreeBytes, err := mustConvertGeneratedInt64(hostMemory.Free, "host memory free bytes")
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	hostMemoryUsedPercent, err := toGeneratedFloat32(roundUsagePercent(hostMemory.UsedPercent), "host memory used percent")
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	runtimeAllocBytes, err := mustConvertGeneratedInt64(stats.Alloc, "runtime alloc bytes")
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	runtimeHeapInUseBytes, err := mustConvertGeneratedInt64(stats.HeapInuse, "runtime heap in use bytes")
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}
	runtimeSysBytes, err := mustConvertGeneratedInt64(stats.Sys, "runtime sys bytes")
	if err != nil {
		return generated.ServerStatusRuntime{}, err
	}

	return generated.ServerStatusRuntime{
		GoVersion:             runtime.Version(),
		HostName:              resolveHostName(),
		OperatingSystem:       runtime.GOOS,
		Architecture:          runtime.GOARCH,
		CpuCores:              runtime.NumCPU(),
		LoadAverage:           loadAverage,
		DiskUsage:             diskUsage,
		HostMemoryTotalBytes:  hostMemoryTotalBytes,
		HostMemoryUsedBytes:   hostMemoryUsedBytes,
		HostMemoryFreeBytes:   hostMemoryFreeBytes,
		HostMemoryUsedPercent: hostMemoryUsedPercent,
		Goroutines:            runtime.NumGoroutine(),
		RuntimeAllocBytes:     runtimeAllocBytes,
		RuntimeHeapInUseBytes: runtimeHeapInUseBytes,
		RuntimeSysBytes:       runtimeSysBytes,
		RuntimeGcCycles:       int(stats.NumGC),
	}, nil
}

func collectHostMemory(ctx context.Context) *mem.VirtualMemoryStat {
	if ctx == nil {
		return &mem.VirtualMemoryStat{}
	}

	snapshot, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || snapshot == nil {
		return &mem.VirtualMemoryStat{}
	}

	return snapshot
}

func collectLoadAverage(ctx context.Context) (generated.ServerStatusLoadAverage, error) {
	if ctx == nil {
		return generated.ServerStatusLoadAverage{}, nil
	}

	avg, err := load.AvgWithContext(ctx)
	if err != nil || avg == nil {
		return generated.ServerStatusLoadAverage{}, nil
	}

	oneMinute, err := toGeneratedFloat32(avg.Load1, "load average one minute")
	if err != nil {
		return generated.ServerStatusLoadAverage{}, err
	}
	fiveMinutes, err := toGeneratedFloat32(avg.Load5, "load average five minutes")
	if err != nil {
		return generated.ServerStatusLoadAverage{}, err
	}
	fifteenMinutes, err := toGeneratedFloat32(avg.Load15, "load average fifteen minutes")
	if err != nil {
		return generated.ServerStatusLoadAverage{}, err
	}

	return generated.ServerStatusLoadAverage{
		OneMinute:      oneMinute,
		FiveMinutes:    fiveMinutes,
		FifteenMinutes: fifteenMinutes,
	}, nil
}

func collectDiskUsage(ctx context.Context, path string) (generated.ServerStatusDiskUsage, error) {
	if ctx == nil {
		return generated.ServerStatusDiskUsage{Path: path}, nil
	}

	usage, err := disk.UsageWithContext(ctx, path)
	if err != nil || usage == nil {
		return generated.ServerStatusDiskUsage{Path: path}, nil
	}

	totalBytes, err := mustConvertGeneratedInt64(usage.Total, "disk total bytes")
	if err != nil {
		return generated.ServerStatusDiskUsage{}, err
	}
	usedBytes, err := mustConvertGeneratedInt64(usage.Used, "disk used bytes")
	if err != nil {
		return generated.ServerStatusDiskUsage{}, err
	}
	freeBytes, err := mustConvertGeneratedInt64(usage.Free, "disk free bytes")
	if err != nil {
		return generated.ServerStatusDiskUsage{}, err
	}
	usedPercent, err := toGeneratedFloat32(roundUsagePercent(usage.UsedPercent), "disk used percent")
	if err != nil {
		return generated.ServerStatusDiskUsage{}, err
	}

	return generated.ServerStatusDiskUsage{
		Path:        usage.Path,
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		FreeBytes:   freeBytes,
		UsedPercent: usedPercent,
	}, nil
}

func roundLatencyMilliseconds(duration time.Duration) float64 {
	return math.Round(duration.Seconds()*millisecondsPerSecond*latencyPrecisionScale) / latencyPrecisionScale
}

func roundCPUPercent(value float64) float64 {
	return math.Round(value*latencyPrecisionScale) / latencyPrecisionScale
}

func roundUsagePercent(value float64) float64 {
	return math.Round(value*latencyPrecisionScale) / latencyPrecisionScale
}

func toGeneratedFloat32(value float64, label string) (float32, error) {
	if value > math.MaxFloat32 || value < -math.MaxFloat32 {
		return 0, fmt.Errorf("%s exceeds float32: %v", label, value)
	}
	return float32(value), nil
}

func mustConvertGeneratedInt64(value uint64, label string) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds int64: %d", label, value)
	}
	return int64(value), nil
}

func resolveAppName(moduleCtx *module.Context) string {
	if moduleCtx == nil || moduleCtx.Config == nil {
		return ""
	}
	return strings.TrimSpace(moduleCtx.Config.App.Name)
}

func resolveAppEnv(moduleCtx *module.Context) string {
	if moduleCtx == nil || moduleCtx.Config == nil {
		return ""
	}
	return strings.TrimSpace(moduleCtx.Config.App.Env)
}

func deriveOverallStatus(databaseStatus string, redisStatus string, anomalies []generated.ServerStatusAnomaly) string {
	for _, status := range []string{databaseStatus, redisStatus} {
		if status == statusDegraded {
			return statusDegraded
		}
	}

	if len(anomalies) > 0 {
		return statusDegraded
	}

	if databaseStatus == statusHealthy || redisStatus == statusHealthy {
		return statusHealthy
	}

	return statusUnknown
}

func parseTrendRange(raw string) monitorcontract.TrendRange {
	switch monitorcontract.TrendRange(strings.TrimSpace(raw)) {
	case monitorcontract.TrendRange30Minutes:
		return monitorcontract.TrendRange30Minutes
	case monitorcontract.TrendRange1Hour:
		return monitorcontract.TrendRange1Hour
	default:
		return monitorcontract.TrendRange10Minutes
	}
}

func parseGeneratedTrendRange(raw *monitoropenapi.GetMonitorServerStatusParamsTrendRange) monitorcontract.TrendRange {
	if raw == nil {
		return monitorcontract.TrendRange10Minutes
	}

	return parseTrendRange(string(*raw))
}

func logTrendWarning(instance *Module, moduleCtx *module.Context, message string, err error) {
	switch {
	case instance != nil && instance.logger != nil:
		instance.logger.Warn(message, zap.Error(err))
	case moduleCtx != nil && moduleCtx.Logger != nil:
		moduleCtx.Logger.Warn(message, zap.Error(err))
	}
}

var _ module.Module = (*Module)(nil)
