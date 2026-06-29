package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/module"
	"graft/server/internal/statex"
	statexkeys "graft/server/internal/statex/keys"
	monitorcontract "graft/server/modules/monitor/contract"
)

// buildServerStatusTrend 构造服务器状态趋势对象，包含时间范围、保留时间和采样间隔。
// 若配置了趋势存储，则加载指定时间范围内的历史趋势数据点；若加载失败，返回不含数据点的趋势对象。
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

	trendStore := resolveTrendStore(moduleCtx, instance)
	if trendStore == nil {
		return trend
	}

	points, err := loadTrendPoints(ctx, trendStore, trendStorageKey(resolveAppName(moduleCtx), resolveHostName()), observedAt, retention)
	if err != nil {
		logTrendWarning(instance, moduleCtx, "load state trend points failed", err)
		return trend
	}

	trend.Points = points
	return trend
}

// resolveTrendStore 返回时间序列存储。若存储不可用或解析失败，返回 nil。
func resolveTrendStore(moduleCtx *module.Context, instance *Module) statex.TimeSeriesStore {
	if instance != nil && instance.trendStore != nil {
		return instance.trendStore
	}

	store, err := resolveOptionalTrendStore(moduleCtx)
	if err != nil {
		logTrendWarning(instance, moduleCtx, "resolve monitor trend store failed", err)
		return nil
	}

	return store
}

func (p *Module) startTrendSampler(ctx *module.Context) {
	if p == nil || ctx == nil || p.trendStore == nil || ctx.LifecycleContext == nil {
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
		p.runTrendSampler(runCtx, p.trendStore, storageKey)
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

func (p *Module) runTrendSampler(ctx context.Context, trendStore statex.TimeSeriesStore, storageKey string) {
	var previousCPUTimes *cpu.TimesStat

	p.recordTrendSample(ctx, trendStore, storageKey, &previousCPUTimes)

	ticker := time.NewTicker(trendSampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.recordTrendSample(ctx, trendStore, storageKey, &previousCPUTimes)
		}
	}
}

func (p *Module) recordTrendSample(
	ctx context.Context,
	trendStore statex.TimeSeriesStore,
	storageKey string,
	previousCPUTimes **cpu.TimesStat,
) {
	if trendStore == nil {
		return
	}

	runtimeSnapshot, err := collectRuntimeSnapshot(ctx)
	if err != nil {
		logTrendWarning(p, nil, "collect monitor runtime snapshot failed", err)
		return
	}
	cpuPercent, err := toGeneratedFloat32(collectCPUPercent(ctx, previousCPUTimes, p, storageKey), "cpu percent")
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

	if err := storeTrendPoint(ctx, trendStore, storageKey, observedAt, point); err != nil {
		logTrendWarning(p, nil, "store monitor trend sample failed", err)
	}
}

// collectCPUPercent 计算当前 CPU 使用百分比，基于与前一次采样的对比。若无法获取 CPU 数据或前一次采样数据为 nil，返回 0。
func collectCPUPercent(ctx context.Context, previousCPUTimes **cpu.TimesStat, instance *Module, storageKey string) float64 {
	if ctx == nil || previousCPUTimes == nil {
		return 0
	}

	samples, err := cpu.TimesWithContext(ctx, false)
	if err != nil || len(samples) == 0 {
		return 0
	}

	current := samples[0]
	percent := calculateHostCPUUsagePercent(*previousCPUTimes, &current, func(raw float64) {
		logTrendWarning(
			instance,
			nil,
			"normalize monitor host cpu sample",
			nil,
			zap.Float64("rawPercent", raw),
			zap.String("sampleContext", "server-status trend sample"),
			zap.String("storageKey", storageKey),
		)
	})
	*previousCPUTimes = &current

	return roundCPUPercent(percent)
}

// storeTrendPoint stores a server status trend point to the time-series store with retention policy applied to trim old entries and set expiration.
func storeTrendPoint(
	ctx context.Context,
	trendStore statex.TimeSeriesStore,
	storageKey string,
	observedAt time.Time,
	point generated.ServerStatusTrendPoint,
) error {
	payload, err := json.Marshal(point)
	if err != nil {
		return fmt.Errorf("marshal trend point: %w", err)
	}

	return trendStore.Append(ctx, storageKey, statex.TimeSeriesSample{
		ObservedAt: observedAt,
		Payload:    payload,
	}, statex.RetentionPolicy{
		TrimBefore:   observedAt.Add(-maxTrendRetentionWindow),
		ExpiresAfter: trendStorageTTL,
	})
}

// loadTrendPoints retrieves trend points from storage within a specified time range.
func loadTrendPoints(
	ctx context.Context,
	trendStore statex.TimeSeriesStore,
	storageKey string,
	observedAt time.Time,
	retention time.Duration,
) ([]generated.ServerStatusTrendPoint, error) {
	if trendStore == nil {
		return nil, nil
	}

	samples, err := trendStore.Range(ctx, storageKey, statex.TimeSeriesQuery{
		StartAt: observedAt.Add(-retention),
		EndAt:   observedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("range state trend points: %w", err)
	}

	points := make([]generated.ServerStatusTrendPoint, 0, len(samples))
	for _, sample := range samples {
		var point generated.ServerStatusTrendPoint
		if err := json.Unmarshal(sample.Payload, &point); err != nil {
			continue
		}
		points = append(points, point)
	}

	return points, nil
}

// trendStorageKey 使用给定的应用名和主机名构造服务器状态趋势点的存储键。
func trendStorageKey(appName string, hostName string) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		trendStorageKeyPrefix,
		statexkeys.Segment(appName, "app"),
		statexkeys.Segment(hostName, "host"),
	)
}

// resolveHostName returns the system hostname trimmed of whitespace, or an empty string on error.
func resolveHostName() string {
	hostName, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(hostName)
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

func calculateHostCPUUsagePercent(previous *cpu.TimesStat, current *cpu.TimesStat, onOutOfRange func(raw float64)) float64 {
	if previous == nil || current == nil {
		return 0
	}

	previousTotal, previousBusy := hostCPUTotalAndBusy(*previous)
	currentTotal, currentBusy := hostCPUTotalAndBusy(*current)
	totalDelta := currentTotal - previousTotal
	if totalDelta <= 0 {
		return 0
	}

	busyDelta := currentBusy - previousBusy
	raw := (busyDelta / totalDelta) * percentageScale
	return normalizeCPUPercent(raw, onOutOfRange)
}

func hostCPUTotalAndBusy(sample cpu.TimesStat) (float64, float64) {
	user := sample.User - sample.Guest
	nice := sample.Nice - sample.GuestNice
	total := user +
		sample.System +
		sample.Idle +
		nice +
		sample.Iowait +
		sample.Irq +
		sample.Softirq +
		sample.Steal
	busy := total - sample.Idle - sample.Iowait

	return total, busy
}

func normalizeCPUPercent(raw float64, onOutOfRange func(raw float64)) float64 {
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0
	}
	if raw < 0 {
		if onOutOfRange != nil {
			onOutOfRange(raw)
		}
		return 0
	}
	if raw > percentageScale {
		if onOutOfRange != nil {
			onOutOfRange(raw)
		}
		return percentageScale
	}

	return raw
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

func logTrendWarning(instance *Module, moduleCtx *module.Context, message string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	switch {
	case instance != nil && instance.logger != nil:
		instance.logger.Warn(message, fields...)
	case moduleCtx != nil && moduleCtx.Logger != nil:
		moduleCtx.Logger.Warn(message, fields...)
	}
}
