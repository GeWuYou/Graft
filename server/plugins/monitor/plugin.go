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
	fallbackServerVersion   = "dev"
	healthCheckTimeout      = 2 * time.Second
	trendSampleInterval     = 5 * time.Second
	maxTrendRetentionWindow = time.Hour
	trendStorageTTL         = 2 * time.Hour
	samplerShutdownTimeout  = 3 * time.Second
	millisecondsPerSecond   = 1000
	latencyPrecisionScale   = 100
	trendStorageKeyPrefix   = "graft:monitor:server-status:trend"
	maxProcessIDInt32       = int64(1<<31 - 1)
	diskUsagePath           = "/"

	statusHealthy  = "healthy"
	statusDegraded = "degraded"
	statusDisabled = "disabled"
	statusUnknown  = "unknown"
)

// Plugin implements the monitor/server-status slice.
type Plugin struct {
	startedAtUnixNs atomic.Int64
	db              *sql.DB
	redis           *redis.Client
	logger          *zap.Logger
	authService     pluginapi.AuthService
	routeAuthorizer pluginapi.Authorizer

	samplerMu     sync.Mutex
	samplerCancel context.CancelFunc
	samplerDone   chan struct{}
}

type serverStatusResponse struct {
	Status       string                   `json:"status"`
	ObservedAt   string                   `json:"observed_at"`
	Server       serverStatusServer       `json:"server"`
	Runtime      serverStatusRuntime      `json:"runtime"`
	Dependencies serverStatusDependencies `json:"dependencies"`
	Summary      serverStatusSummary      `json:"summary"`
	Trend        serverStatusTrend        `json:"trend"`
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
	Status    string   `json:"status"`
	Detail    string   `json:"detail"`
	LatencyMs *float64 `json:"latency_ms"`
}

type serverStatusPlugin struct {
	Name                string   `json:"name"`
	Status              string   `json:"status"`
	StatusDetail        string   `json:"status_detail"`
	Version             string   `json:"version"`
	DependsOn           []string `json:"depends_on"`
	MissingDependencies []string `json:"missing_dependencies,omitempty"`
}

type serverStatusRuntime struct {
	GoVersion             string                  `json:"go_version"`
	HostName              string                  `json:"host_name"`
	OperatingSystem       string                  `json:"operating_system"`
	Architecture          string                  `json:"architecture"`
	CPUCores              int                     `json:"cpu_cores"`
	LoadAverage           serverStatusLoadAverage `json:"load_average"`
	DiskUsage             serverStatusDiskUsage   `json:"disk_usage"`
	HostMemoryTotalBytes  uint64                  `json:"host_memory_total_bytes"`
	HostMemoryUsedBytes   uint64                  `json:"host_memory_used_bytes"`
	HostMemoryFreeBytes   uint64                  `json:"host_memory_free_bytes"`
	HostMemoryUsedPercent float64                 `json:"host_memory_used_percent"`
	Goroutines            int                     `json:"goroutines"`
	RuntimeAllocBytes     uint64                  `json:"runtime_alloc_bytes"`
	RuntimeHeapInUseBytes uint64                  `json:"runtime_heap_in_use_bytes"`
	RuntimeSysBytes       uint64                  `json:"runtime_sys_bytes"`
	RuntimeGCCycles       uint32                  `json:"runtime_gc_cycles"`
}

type serverStatusLoadAverage struct {
	OneMinute      float64 `json:"one_minute"`
	FiveMinutes    float64 `json:"five_minutes"`
	FifteenMinutes float64 `json:"fifteen_minutes"`
}

type serverStatusDiskUsage struct {
	Path        string  `json:"path"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type serverStatusSummary struct {
	TotalDependencies    int `json:"total_dependencies"`
	HealthyDependencies  int `json:"healthy_dependencies"`
	DegradedDependencies int `json:"degraded_dependencies"`
	UnknownDependencies  int `json:"unknown_dependencies"`
	DisabledDependencies int `json:"disabled_dependencies"`
	TotalPlugins         int `json:"total_plugins"`
	HealthyPlugins       int `json:"healthy_plugins"`
}

type serverStatusTrend struct {
	Range                 string                   `json:"range"`
	RetentionSeconds      int64                    `json:"retention_seconds"`
	SampleIntervalSeconds int64                    `json:"sample_interval_seconds"`
	Points                []serverStatusTrendPoint `json:"points"`
}

type serverStatusTrendPoint struct {
	ObservedAt             string  `json:"observed_at"`
	CPUPercent             float64 `json:"cpu_percent"`
	HostMemoryUsedPercent  float64 `json:"host_memory_used_percent"`
	LoadAverageOneMinute   float64 `json:"load_average_one_minute"`
	LoadAverageFiveMinutes float64 `json:"load_average_five_minutes"`
	LoadAverageFifteenMins float64 `json:"load_average_fifteen_minutes"`
	Goroutines             int     `json:"goroutines"`
	RuntimeAllocBytes      uint64  `json:"runtime_alloc_bytes"`
	RuntimeHeapInUseBytes  uint64  `json:"runtime_heap_in_use_bytes"`
	RuntimeSysBytes        uint64  `json:"runtime_sys_bytes"`
}

// NewPlugin creates the monitor plugin.
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

// Boot records the first stable startup timestamp and starts the Redis-backed trend sampler.
func (p *Plugin) Boot(ctx *plugin.Context) error {
	p.startedAtUnixNs.CompareAndSwap(0, time.Now().UTC().UnixNano())
	if ctx != nil {
		p.redis = ctx.Redis
		p.logger = ctx.Logger
	}

	p.startTrendSampler(ctx)
	return nil
}

// Shutdown stops the owned trend sampler before shared runtime resources are released.
func (p *Plugin) Shutdown(ctx *plugin.Context) error {
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
				{Key: i18n.MessageKey(monitorcontract.MonitorSectionTitle.String()), Text: "服务器管理"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusMenuTitle.String()), Text: "服务器状态"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusOverviewMenuTitle.String()), Text: "概览"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusRuntimeMenuTitle.String()), Text: "运行时"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusDependenciesMenuTitle.String()), Text: "依赖服务"},
			},
		},
		{
			Namespace: "monitor",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(monitorcontract.MonitorSectionTitle.String()), Text: "Server Management"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusMenuTitle.String()), Text: "Server Status"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusOverviewMenuTitle.String()), Text: "Overview"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusRuntimeMenuTitle.String()), Text: "Runtime"},
				{Key: i18n.MessageKey(monitorcontract.ServerStatusDependenciesMenuTitle.String()), Text: "Dependencies"},
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
	p.redis = ctx.Redis
	p.logger = ctx.Logger

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
		Description: "Allows reading the server status overview.",
		Category:    "api",
		Plugin:      pluginName,
	})
}

func registerMonitorMenu(registry *menu.Registry, pluginName string) {
	if registry == nil {
		return
	}

	registry.Register(menu.Item{
		Code:       "monitor.section",
		Title:      "服务器管理",
		TitleKey:   monitorcontract.MonitorSectionTitle.String(),
		Path:       monitorcontract.MonitorGroup,
		Icon:       "server",
		Permission: "",
		Plugin:     pluginName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status",
		Title:      "服务器状态",
		TitleKey:   monitorcontract.ServerStatusMenuTitle.String(),
		Path:       monitorcontract.ServerStatusMenuPath,
		Icon:       "activity",
		Permission: "",
		Plugin:     pluginName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.overview",
		Title:      "概览",
		TitleKey:   monitorcontract.ServerStatusOverviewMenuTitle.String(),
		Path:       monitorcontract.ServerStatusOverviewMenuPath,
		Icon:       "dashboard",
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Plugin:     pluginName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.runtime",
		Title:      "运行时",
		TitleKey:   monitorcontract.ServerStatusRuntimeMenuTitle.String(),
		Path:       monitorcontract.ServerStatusRuntimeMenuPath,
		Icon:       "time",
		Permission: monitorcontract.ServerStatusReadPermission.String(),
		Plugin:     pluginName,
	})

	registry.Register(menu.Item{
		Code:       "monitor.server-status.dependencies",
		Title:      "依赖服务",
		TitleKey:   monitorcontract.ServerStatusDependenciesMenuTitle.String(),
		Path:       monitorcontract.ServerStatusDependenciesMenuPath,
		Icon:       "data-base",
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
		trendRange := parseTrendRange(ginCtx.Query(monitorcontract.TrendRangeQueryKey))
		payload, err := buildServerStatusResponse(ginCtx.Request.Context(), ctx, instance, trendRange)
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
	trendRange monitorcontract.TrendRange,
) (serverStatusResponse, error) {
	observedAt := time.Now().UTC()
	startedAt := observedAt
	if instance != nil {
		if startedAtUnixNs := instance.startedAtUnixNs.Load(); startedAtUnixNs > 0 {
			startedAt = time.Unix(0, startedAtUnixNs).UTC()
		}
	}

	runtimeSnapshot := collectRuntimeSnapshot(ctx)
	databaseStatus := databaseHealth(ctx, instance)
	redisStatus := redisHealth(ctx, pluginCtx)
	plugins := runtimePluginSummaries(pluginCtx, databaseStatus, redisStatus)
	summary := buildServerStatusSummary(databaseStatus, redisStatus, plugins)
	trend := buildServerStatusTrend(ctx, pluginCtx, instance, observedAt, trendRange)

	return serverStatusResponse{
		Status:     deriveOverallStatus(databaseStatus.Status, redisStatus.Status),
		ObservedAt: observedAt.Format(time.RFC3339),
		Server: serverStatusServer{
			Version:       fallbackServerVersion,
			StartedAt:     startedAt.Format(time.RFC3339),
			UptimeSeconds: int64(observedAt.Sub(startedAt).Seconds()),
			GoVersion:     runtime.Version(),
			AppName:       resolveAppName(pluginCtx),
			AppEnv:        resolveAppEnv(pluginCtx),
		},
		Runtime: runtimeSnapshot,
		Dependencies: serverStatusDependencies{
			Database: databaseStatus,
			Redis:    redisStatus,
		},
		Summary: summary,
		Trend:   trend,
		Plugins: plugins,
	}, nil
}

func databaseHealth(ctx context.Context, instance *Plugin) dependencyStatus {
	if instance == nil || instance.db == nil {
		return dependencyStatus{
			Status: statusUnknown,
			Detail: "Database handle is unavailable",
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	startedAt := time.Now()
	if err := instance.db.PingContext(pingCtx); err != nil {
		logTrendWarning(instance, nil, "database ping failed", err)
		return dependencyStatus{
			Status: statusDegraded,
			Detail: "Database ping failed",
		}
	}

	latencyMs := roundLatencyMilliseconds(time.Since(startedAt))
	return dependencyStatus{
		Status:    statusHealthy,
		Detail:    "Database ping succeeded",
		LatencyMs: &latencyMs,
	}
}

func redisHealth(ctx context.Context, pluginCtx *plugin.Context) dependencyStatus {
	if pluginCtx == nil || pluginCtx.Redis == nil {
		return dependencyStatus{
			Status: statusDisabled,
			Detail: "Redis client is not configured",
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	startedAt := time.Now()
	if err := pluginCtx.Redis.Ping(pingCtx).Err(); err != nil {
		logTrendWarning(nil, pluginCtx, "redis ping failed", err)
		return dependencyStatus{
			Status: statusDegraded,
			Detail: "Redis ping failed",
		}
	}

	latencyMs := roundLatencyMilliseconds(time.Since(startedAt))
	return dependencyStatus{
		Status:    statusHealthy,
		Detail:    "Redis ping succeeded",
		LatencyMs: &latencyMs,
	}
}

func runtimePluginSummaries(
	pluginCtx *plugin.Context,
	database dependencyStatus,
	redis dependencyStatus,
) []serverStatusPlugin {
	if pluginCtx == nil {
		return nil
	}

	descriptors := pluginCtx.RuntimeMetadata.OrderedPluginDescriptors()
	available := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		name := strings.TrimSpace(descriptor.Name)
		if name == "" {
			continue
		}
		available[name] = struct{}{}
	}

	platformStatus := deriveOverallStatus(database.Status, redis.Status)
	items := make([]serverStatusPlugin, 0, len(descriptors))
	for _, descriptor := range descriptors {
		dependsOn := append([]string(nil), descriptor.DependsOn...)
		status, statusDetail, missingDependencies := deriveRuntimePluginObservation(descriptor, available, platformStatus)
		items = append(items, serverStatusPlugin{
			Name:                descriptor.Name,
			Status:              status,
			StatusDetail:        statusDetail,
			Version:             descriptor.Version,
			DependsOn:           dependsOn,
			MissingDependencies: missingDependencies,
		})
	}

	return items
}

// deriveRuntimePluginObservation keeps plugin runtime semantics explicit and narrow:
// a plugin is healthy only when its runtime metadata is complete, its declared
// plugin dependencies are present, and the current shared runtime signals are not
// degraded. When that cannot be confirmed, the returned detail explains the most
// useful operator-facing reason instead of collapsing everything into a coarse summary.
func deriveRuntimePluginObservation(
	descriptor plugin.DescriptorSnapshot,
	available map[string]struct{},
	platformStatus string,
) (status string, detail string, missingDependencies []string) {
	if strings.TrimSpace(descriptor.Name) == "" || strings.TrimSpace(descriptor.Version) == "" {
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
	database dependencyStatus,
	redis dependencyStatus,
	plugins []serverStatusPlugin,
) serverStatusSummary {
	summary := serverStatusSummary{
		TotalDependencies: len([]dependencyStatus{database, redis}),
		TotalPlugins:      len(plugins),
	}

	for _, dependency := range []dependencyStatus{database, redis} {
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

	for _, plugin := range plugins {
		if plugin.Status == statusHealthy {
			summary.HealthyPlugins++
		}
	}

	return summary
}

func buildServerStatusTrend(
	ctx context.Context,
	pluginCtx *plugin.Context,
	instance *Plugin,
	observedAt time.Time,
	trendRange monitorcontract.TrendRange,
) serverStatusTrend {
	retention := trendRange.Duration()
	trend := serverStatusTrend{
		Range:                 trendRange.String(),
		RetentionSeconds:      int64(retention.Seconds()),
		SampleIntervalSeconds: int64(trendSampleInterval.Seconds()),
		Points:                nil,
	}

	redisClient := resolveRedisClient(pluginCtx, instance)
	if redisClient == nil {
		return trend
	}

	points, err := loadTrendPoints(ctx, redisClient, trendStorageKey(resolveAppName(pluginCtx), resolveHostName()), observedAt, retention)
	if err != nil {
		logTrendWarning(instance, pluginCtx, "load redis trend points failed", err)
		return trend
	}

	trend.Points = points
	return trend
}

func resolveRedisClient(pluginCtx *plugin.Context, instance *Plugin) *redis.Client {
	if instance != nil && instance.redis != nil {
		return instance.redis
	}
	if pluginCtx != nil {
		return pluginCtx.Redis
	}
	return nil
}

func (p *Plugin) startTrendSampler(ctx *plugin.Context) {
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

func (p *Plugin) stopTrendSampler(ctx *plugin.Context) error {
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

func (p *Plugin) runTrendSampler(ctx context.Context, redisClient *redis.Client, storageKey string) {
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

func (p *Plugin) recordTrendSample(
	ctx context.Context,
	redisClient *redis.Client,
	storageKey string,
	processHandle *process.Process,
) {
	if redisClient == nil {
		return
	}

	runtimeSnapshot := collectRuntimeSnapshot(ctx)
	observedAt := time.Now().UTC()
	point := serverStatusTrendPoint{
		ObservedAt:             observedAt.Format(time.RFC3339),
		CPUPercent:             collectCPUPercent(ctx, processHandle),
		HostMemoryUsedPercent:  runtimeSnapshot.HostMemoryUsedPercent,
		LoadAverageOneMinute:   runtimeSnapshot.LoadAverage.OneMinute,
		LoadAverageFiveMinutes: runtimeSnapshot.LoadAverage.FiveMinutes,
		LoadAverageFifteenMins: runtimeSnapshot.LoadAverage.FifteenMinutes,
		Goroutines:             runtimeSnapshot.Goroutines,
		RuntimeAllocBytes:      runtimeSnapshot.RuntimeAllocBytes,
		RuntimeHeapInUseBytes:  runtimeSnapshot.RuntimeHeapInUseBytes,
		RuntimeSysBytes:        runtimeSnapshot.RuntimeSysBytes,
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
	point serverStatusTrendPoint,
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
) ([]serverStatusTrendPoint, error) {
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

	points := make([]serverStatusTrendPoint, 0, len(members))
	for _, member := range members {
		var point serverStatusTrendPoint
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

func collectRuntimeSnapshot(ctx context.Context) serverStatusRuntime {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)
	hostMemory := collectHostMemory(ctx)

	return serverStatusRuntime{
		GoVersion:             runtime.Version(),
		HostName:              resolveHostName(),
		OperatingSystem:       runtime.GOOS,
		Architecture:          runtime.GOARCH,
		CPUCores:              runtime.NumCPU(),
		LoadAverage:           collectLoadAverage(ctx),
		DiskUsage:             collectDiskUsage(ctx, diskUsagePath),
		HostMemoryTotalBytes:  hostMemory.Total,
		HostMemoryUsedBytes:   hostMemory.Used,
		HostMemoryFreeBytes:   hostMemory.Available,
		HostMemoryUsedPercent: roundUsagePercent(hostMemory.UsedPercent),
		Goroutines:            runtime.NumGoroutine(),
		RuntimeAllocBytes:     stats.Alloc,
		RuntimeHeapInUseBytes: stats.HeapInuse,
		RuntimeSysBytes:       stats.Sys,
		RuntimeGCCycles:       stats.NumGC,
	}
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

func collectLoadAverage(ctx context.Context) serverStatusLoadAverage {
	if ctx == nil {
		return serverStatusLoadAverage{}
	}

	avg, err := load.AvgWithContext(ctx)
	if err != nil || avg == nil {
		return serverStatusLoadAverage{}
	}

	return serverStatusLoadAverage{
		OneMinute:      avg.Load1,
		FiveMinutes:    avg.Load5,
		FifteenMinutes: avg.Load15,
	}
}

func collectDiskUsage(ctx context.Context, path string) serverStatusDiskUsage {
	if ctx == nil {
		return serverStatusDiskUsage{Path: path}
	}

	usage, err := disk.UsageWithContext(ctx, path)
	if err != nil || usage == nil {
		return serverStatusDiskUsage{Path: path}
	}

	return serverStatusDiskUsage{
		Path:        usage.Path,
		TotalBytes:  usage.Total,
		UsedBytes:   usage.Used,
		FreeBytes:   usage.Free,
		UsedPercent: roundUsagePercent(usage.UsedPercent),
	}
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
		if status == statusDegraded {
			return statusDegraded
		}
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

func logTrendWarning(instance *Plugin, pluginCtx *plugin.Context, message string, err error) {
	switch {
	case instance != nil && instance.logger != nil:
		instance.logger.Warn(message, zap.Error(err))
	case pluginCtx != nil && pluginCtx.Logger != nil:
		pluginCtx.Logger.Warn(message, zap.Error(err))
	}
}

var _ plugin.Plugin = (*Plugin)(nil)
