package container

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	mobyclient "github.com/moby/moby/client"
	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
)

func (r *DockerRuntime) containerResourceSummary(ctx context.Context, id string) ResourceSummary {
	ref := strings.TrimSpace(id)
	if ref == "" {
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}
	statsCtx, cancel := context.WithTimeout(ctx, dockerStatsListTimeout)
	defer cancel()

	reader, err := r.client.ContainerStatsOneShot(statsCtx, ref)
	if err != nil {
		r.logResourceStatsFailure("collect docker container stats failed", ref, err)
		return unavailableResourceSummary(resourceStatsErrorReason(err))
	}
	if reader.Body == nil {
		r.logResourceStatsFailure("docker container stats response body missing", ref, nil)
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}
	defer closeDockerStatsReaderBody(r.logger, reader.Body)

	stats, err := r.decodeDockerStatsResponse(ref, reader.Body)
	if err != nil {
		return unavailableResourceSummary(resourceStatsErrorReason(err))
	}
	return r.dockerResourceSummary(ref, stats)
}

func (r *DockerRuntime) decodeDockerStatsResponse(containerID string, reader io.Reader) (container.StatsResponse, error) {
	var stats container.StatsResponse
	if err := json.NewDecoder(reader).Decode(&stats); err != nil {
		r.logResourceStatsFailure("decode docker container stats failed", containerID, err)
		return container.StatsResponse{}, err
	}
	return stats, nil
}

func (r *DockerRuntime) currentResourceSummary(id string) ResourceSummary {
	ref := strings.TrimSpace(id)
	if ref == "" {
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}
	cache := r.ensureResourceStatsCache()
	if r == nil || cache == nil {
		return unavailableResourceSummary(containerStatsNotCollectedReason)
	}
	return cache.current(ref)
}

func (r *DockerRuntime) invalidateResourceSummary(ids ...string) {
	cache := r.ensureResourceStatsCache()
	if r == nil || cache == nil {
		return
	}
	cache.invalidate(ids...)
	r.clearCPUStatsBaselines(ids...)
}

func (r *DockerRuntime) ensureResourceStatsCache() *resourceStatsCache {
	if r == nil {
		return nil
	}
	if r.resourceStats == nil {
		r.resourceStats = newResourceStatsCache(containerResourceStatsCacheTTL, containerResourceStatsCacheStaleWindow)
	}
	return r.resourceStats
}

func (r *DockerRuntime) updateResourceStatsCachePolicy(ttl time.Duration, staleWindow time.Duration) {
	if r == nil {
		return
	}
	r.resourceStats = newResourceStatsCache(ttl, staleWindow)
}

func (r *DockerRuntime) collectListResourceSummaries(_ context.Context, summaries []Summary) {
	if len(summaries) == 0 {
		return
	}
	for index := range summaries {
		summaries[index].Resource = r.currentResourceSummary(summaries[index].ID)
	}
}

// CollectStatsSnapshots collects one bounded batch of Docker stats snapshots for publish.
func (r *DockerRuntime) CollectStatsSnapshots(ctx context.Context) ([]StatsSnapshot, error) {
	items, err := r.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return nil, mapDockerError(err)
	}
	snapshots := make([]StatsSnapshot, len(items))
	collectedAt := time.Now().UTC()
	if len(items) == 0 {
		return snapshots, nil
	}
	r.populateStatsSnapshots(ctx, items, snapshots, collectedAt)
	return snapshots, nil
}

func (r *DockerRuntime) populateStatsSnapshots(
	ctx context.Context,
	items []container.Summary,
	snapshots []StatsSnapshot,
	collectedAt time.Time,
) {
	indexes := dockerStatsWorkQueue(len(items))
	for index := range items {
		indexes <- index
	}
	close(indexes)

	var wg sync.WaitGroup
	workers := dockerStatsWorkerCount(len(items))
	wg.Add(workers)
	for workerIndex := 0; workerIndex < workers; workerIndex++ {
		go func() {
			defer wg.Done()
			for index := range indexes {
				snapshots[index] = r.collectStatsSnapshot(ctx, items[index], collectedAt)
			}
		}()
	}
	wg.Wait()
}

func dockerStatsWorkQueue(itemCount int) chan int {
	return make(chan int, itemCount)
}

func dockerStatsWorkerCount(itemCount int) int {
	workers := min(itemCount, dockerStatsListWorkers)
	if workers < 1 {
		return 1
	}
	return workers
}

func (r *DockerRuntime) collectStatsSnapshot(
	ctx context.Context,
	item container.Summary,
	collectedAt time.Time,
) StatsSnapshot {
	summary := dockerSummary(item)
	resource := r.collectCachedResourceSummary(ctx, summary.ID)
	snapshotCollectedAt, resource := normalizeResourceCollectedAt(resource, collectedAt)
	return StatsSnapshot{
		ContainerID:  summary.ID,
		Name:         summary.Name,
		ShortID:      summary.ShortID,
		Image:        summary.Image,
		Runtime:      summary.Runtime,
		State:        summary.State,
		Status:       summary.Status,
		Health:       summary.Health,
		RestartCount: summary.RestartCount,
		Resource:     resource,
		CollectedAt:  snapshotCollectedAt,
	}
}

func normalizeResourceCollectedAt(resource ResourceSummary, fallback time.Time) (time.Time, ResourceSummary) {
	if parsedCollectedAt, ok := parseResourceCollectedAt(resource.CollectedAt); ok {
		return parsedCollectedAt, resource
	}
	if strings.TrimSpace(resource.CollectedAt) == "" {
		resource.CollectedAt = fallback.Format(time.RFC3339)
	}
	return fallback, resource
}

func (r *DockerRuntime) collectCachedResourceSummary(ctx context.Context, id string) ResourceSummary {
	ref := strings.TrimSpace(id)
	if ref == "" {
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}
	cache := r.ensureResourceStatsCache()
	if cache == nil {
		return unavailableResourceSummary(containerStatsNotCollectedReason)
	}
	return cache.get(ctx, ref, func(loadCtx context.Context) ResourceSummary {
		return r.containerResourceSummary(loadCtx, ref)
	})
}

func (r *DockerRuntime) dockerResourceSummary(containerID string, stats container.StatsResponse) ResourceSummary {
	resource := ResourceSummary{
		Available:      true,
		StatsAvailable: true,
	}
	if cpuPercent, ok := r.dockerCPUPercent(containerID, stats); ok {
		resource.CPUPercent = &cpuPercent
		r.logDockerCPUCalculation(containerID, stats, cpuPercent, true)
	} else {
		r.logDockerCPUCalculation(containerID, stats, 0, false)
	}
	resource.OnlineCPUs = dockerOnlineCPUs(stats)
	resource.SystemCPUUsage = uint64ToInt64Ptr(stats.CPUStats.SystemUsage)
	resource.TotalCPUUsage = uint64ToInt64Ptr(stats.CPUStats.CPUUsage.TotalUsage)
	resource.CPUUsageInUsermode = uint64ToInt64Ptr(stats.CPUStats.CPUUsage.UsageInUsermode)
	resource.CPUUsageInKernelmode = uint64ToInt64Ptr(stats.CPUStats.CPUUsage.UsageInKernelmode)
	resource.ThrottlingPeriods = uint64ToInt64Ptr(stats.CPUStats.ThrottlingData.Periods)
	resource.ThrottlingThrottledPeriods = uint64ToInt64Ptr(stats.CPUStats.ThrottlingData.ThrottledPeriods)
	resource.ThrottlingThrottledTime = uint64ToInt64Ptr(stats.CPUStats.ThrottlingData.ThrottledTime)
	if usage, ok := uint64ToInt64(stats.MemoryStats.Usage); ok {
		resource.MemoryUsageBytes = &usage
	}
	if limit, ok := uint64ToInt64(stats.MemoryStats.Limit); ok {
		resource.MemoryLimitBytes = &limit
	}
	if resource.MemoryUsageBytes != nil && resource.MemoryLimitBytes != nil && *resource.MemoryLimitBytes > 0 {
		memoryPercent := (float64(*resource.MemoryUsageBytes) / float64(*resource.MemoryLimitBytes)) * dockerStatsPercentScale
		resource.MemoryPercent = &memoryPercent
	}
	resource.MemoryCache = dockerMemoryStat(stats, "cache")
	resource.MemoryRSS = dockerMemoryStat(stats, "rss")
	resource.MemoryActiveFile = dockerMemoryStat(stats, "active_file")
	resource.MemoryInactiveFile = dockerMemoryStat(stats, "inactive_file")
	resource.MemoryPgfault = dockerMemoryStat(stats, "pgfault")
	resource.MemoryPgmajfault = dockerMemoryStat(stats, "pgmajfault")
	applyDockerNetworkStats(stats, &resource)
	resource.PIDsCurrent = uint64ToInt64Ptr(stats.PidsStats.Current)
	resource.PIDsLimit = uint64ToInt64Ptr(stats.PidsStats.Limit)
	if resource.CPUPercent == nil && resource.MemoryUsageBytes == nil && resource.MemoryLimitBytes == nil {
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}
	return resource
}

// dockerOnlineCPUs 返回容器的在线 CPU 数量。
func dockerOnlineCPUs(stats container.StatsResponse) *int64 {
	onlineCPUs := dockerStatsOnlineCPUs(stats)
	if onlineCPUs == 0 {
		return nil
	}
	return uint32ToInt64Ptr(onlineCPUs)
}

// 如果对应统计项不存在或无法转换为 int64，则返回 nil。
func dockerMemoryStat(stats container.StatsResponse, key string) *int64 {
	if len(stats.MemoryStats.Stats) == 0 {
		return nil
	}
	value, ok := stats.MemoryStats.Stats[key]
	if !ok {
		return nil
	}
	return uint64ToInt64Ptr(value)
}

func applyDockerNetworkStats(stats container.StatsResponse, resource *ResourceSummary) {
	if len(stats.Networks) == 0 || resource == nil {
		return
	}
	var totals dockerNetworkTotals
	for _, networkStats := range stats.Networks {
		totals.add(networkStats)
	}
	resource.RxBytes = totals.int64Ptr(totals.rxBytes, totals.rxBytesOverflow)
	resource.TxBytes = totals.int64Ptr(totals.txBytes, totals.txBytesOverflow)
	resource.RxPackets = totals.int64Ptr(totals.rxPackets, totals.rxPacketsOverflow)
	resource.TxPackets = totals.int64Ptr(totals.txPackets, totals.txPacketsOverflow)
	resource.RxErrors = totals.int64Ptr(totals.rxErrors, totals.rxErrorsOverflow)
	resource.TxErrors = totals.int64Ptr(totals.txErrors, totals.txErrorsOverflow)
	resource.RxDropped = totals.int64Ptr(totals.rxDropped, totals.rxDroppedOverflow)
	resource.TxDropped = totals.int64Ptr(totals.txDropped, totals.txDroppedOverflow)
}

type dockerNetworkTotals struct {
	rxBytes, txBytes, rxPackets, txPackets, rxErrors, txErrors, rxDropped, txDropped uint64
	rxBytesOverflow, txBytesOverflow, rxPacketsOverflow, txPacketsOverflow           bool
	rxErrorsOverflow, txErrorsOverflow, rxDroppedOverflow, txDroppedOverflow         bool
}

func (t *dockerNetworkTotals) add(stats container.NetworkStats) {
	t.rxBytes, t.rxBytesOverflow = addUint64(t.rxBytes, stats.RxBytes, t.rxBytesOverflow)
	t.txBytes, t.txBytesOverflow = addUint64(t.txBytes, stats.TxBytes, t.txBytesOverflow)
	t.rxPackets, t.rxPacketsOverflow = addUint64(t.rxPackets, stats.RxPackets, t.rxPacketsOverflow)
	t.txPackets, t.txPacketsOverflow = addUint64(t.txPackets, stats.TxPackets, t.txPacketsOverflow)
	t.rxErrors, t.rxErrorsOverflow = addUint64(t.rxErrors, stats.RxErrors, t.rxErrorsOverflow)
	t.txErrors, t.txErrorsOverflow = addUint64(t.txErrors, stats.TxErrors, t.txErrorsOverflow)
	t.rxDropped, t.rxDroppedOverflow = addUint64(t.rxDropped, stats.RxDropped, t.rxDroppedOverflow)
	t.txDropped, t.txDroppedOverflow = addUint64(t.txDropped, stats.TxDropped, t.txDroppedOverflow)
}

func (dockerNetworkTotals) int64Ptr(value uint64, overflow bool) *int64 {
	if overflow {
		return nil
	}
	return uint64ToInt64Ptr(value)
}

// addUint64 将 value 累加到 total，并在发生溢出时标记结果无效。
func addUint64(total uint64, value uint64, overflow bool) (uint64, bool) {
	if overflow || total > ^uint64(0)-value {
		return 0, true
	}
	return total + value, false
}

// 优先使用运行时维护的上一帧 one-shot 样本计算 CPU 百分比。
func (r *DockerRuntime) dockerCPUPercent(containerID string, stats container.StatsResponse) (float64, bool) {
	normalizedID := strings.TrimSpace(containerID)
	current, ok := dockerCurrentCPUStatsBaseline(stats)
	if !ok {
		return 0, false
	}
	previous, hasPrevious := r.cpuStatsBaseline(normalizedID)
	r.recordCPUStatsBaseline(normalizedID, current)
	if !hasPrevious {
		return 0, false
	}
	if current.systemUsage <= previous.systemUsage {
		return 0, false
	}
	if current.totalUsage <= previous.totalUsage {
		return 0, true
	}
	cpuDelta := float64(current.totalUsage - previous.totalUsage)
	systemDelta := float64(current.systemUsage - previous.systemUsage)
	onlineCPUs := current.onlineCPUs
	if onlineCPUs == 0 {
		return 0, false
	}
	return (cpuDelta / systemDelta) * float64(onlineCPUs) * dockerStatsPercentScale, true
}

// dockerCurrentCPUStatsBaseline 提取当前容器 CPU 统计基线。
func dockerCurrentCPUStatsBaseline(stats container.StatsResponse) (dockerCPUStatsBaseline, bool) {
	onlineCPUs := dockerStatsOnlineCPUs(stats)
	if onlineCPUs == 0 || stats.CPUStats.SystemUsage == 0 {
		return dockerCPUStatsBaseline{}, false
	}
	return dockerCPUStatsBaseline{
		totalUsage:  stats.CPUStats.CPUUsage.TotalUsage,
		systemUsage: stats.CPUStats.SystemUsage,
		onlineCPUs:  onlineCPUs,
	}, true
}

func (r *DockerRuntime) cpuStatsBaseline(containerID string) (dockerCPUStatsBaseline, bool) {
	if r == nil || strings.TrimSpace(containerID) == "" {
		return dockerCPUStatsBaseline{}, false
	}
	r.cpuBaselinesMu.Lock()
	defer r.cpuBaselinesMu.Unlock()
	baseline, ok := r.cpuBaselines[strings.TrimSpace(containerID)]
	return baseline, ok
}

func (r *DockerRuntime) recordCPUStatsBaseline(containerID string, baseline dockerCPUStatsBaseline) {
	if r == nil || strings.TrimSpace(containerID) == "" {
		return
	}
	r.cpuBaselinesMu.Lock()
	defer r.cpuBaselinesMu.Unlock()
	if r.cpuBaselines == nil {
		r.cpuBaselines = make(map[string]dockerCPUStatsBaseline)
	}
	baseline.collectedAt = time.Now().UTC()
	r.cpuBaselines[strings.TrimSpace(containerID)] = baseline
}

func (r *DockerRuntime) clearCPUStatsBaselines(ids ...string) {
	if r == nil || len(ids) == 0 {
		return
	}
	r.cpuBaselinesMu.Lock()
	defer r.cpuBaselinesMu.Unlock()
	for _, id := range ids {
		normalizedID := strings.TrimSpace(id)
		if normalizedID == "" {
			continue
		}
		delete /* cpu baseline */ (r.cpuBaselines, normalizedID)
	}
}

// dockerStatsOnlineCPUs 返回统计信息中的在线 CPU 数。
func dockerStatsOnlineCPUs(stats container.StatsResponse) uint32 {
	if stats.CPUStats.OnlineCPUs > 0 {
		return stats.CPUStats.OnlineCPUs
	}
	if len(stats.CPUStats.CPUUsage.PercpuUsage) == 0 {
		return 0
	}
	perCPUUsageCount := uint64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	if perCPUUsageCount > math.MaxUint32 {
		return 0
	}
	return uint32(perCPUUsageCount)
}

type dockerCPUCalculation struct {
	containerID    string
	totalUsage     uint64
	preTotalUsage  uint64
	systemUsage    uint64
	preSystemUsage uint64
	onlineCPUs     uint32
	cpuDelta       uint64
	systemDelta    uint64
	cpuPercent     float64
}

func (r *DockerRuntime) logDockerCPUCalculation(containerID string, stats container.StatsResponse, cpuPercent float64, ok bool) {
	if r == nil || r.logger == nil || !r.logger.Core().Enabled(zap.DebugLevel) {
		return
	}
	calculation := dockerCPUCalculation{
		containerID:    strings.TrimSpace(containerID),
		totalUsage:     stats.CPUStats.CPUUsage.TotalUsage,
		preTotalUsage:  stats.PreCPUStats.CPUUsage.TotalUsage,
		systemUsage:    stats.CPUStats.SystemUsage,
		preSystemUsage: stats.PreCPUStats.SystemUsage,
		onlineCPUs:     dockerStatsOnlineCPUs(stats),
		cpuPercent:     cpuPercent,
	}
	if stats.CPUStats.CPUUsage.TotalUsage > stats.PreCPUStats.CPUUsage.TotalUsage {
		calculation.cpuDelta = stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
	}
	if stats.CPUStats.SystemUsage > stats.PreCPUStats.SystemUsage {
		calculation.systemDelta = stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage
	}
	logsafe.Debug(r.logger, "container cpu stats calculation",
		zap.String("container", calculation.containerID),
		zap.Uint64("totalUsage", calculation.totalUsage),
		zap.Uint64("preTotalUsage", calculation.preTotalUsage),
		zap.Uint64("systemUsage", calculation.systemUsage),
		zap.Uint64("preSystemUsage", calculation.preSystemUsage),
		zap.Uint32("onlineCPUs", calculation.onlineCPUs),
		zap.Uint64("cpuDelta", calculation.cpuDelta),
		zap.Uint64("systemDelta", calculation.systemDelta),
		zap.Float64("cpuPercent", calculation.cpuPercent),
		zap.Bool("calculated", ok),
	)
}

// unavailableResourceSummary 生成一个不可用的资源摘要。
func unavailableResourceSummary(reason string) ResourceSummary {
	reason = firstNonEmpty(strings.TrimSpace(reason), containerStatsUnavailableReason)
	return ResourceSummary{
		Available:         false,
		UnavailableReason: reason,
		StatsAvailable:    false,
		StatsErrorKey:     reason,
		StatsErrorMessage: resourceStatsErrorMessage(reason),
	}
}

func resourceStatsErrorReason(err error) string {
	if err == nil {
		return containerStatsUnavailableReason
	}
	mapped := mapDockerError(err)
	if errors.Is(mapped, errContainerRuntimeTimeout) {
		return containerStatsTimeoutReason
	}
	return containerStatsUnavailableReason
}

func closeDockerStatsReaderBody(logger *zap.Logger, body interface{ Close() error }) {
	if body == nil {
		return
	}
	if err := body.Close(); err != nil && logger != nil {
		logsafe.Debug(logger, "close docker stats reader failed", zap.Error(err))
	}
}

func (r *DockerRuntime) logResourceStatsFailure(message string, containerID string, err error) {
	if r == nil || r.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("container", strings.TrimSpace(containerID)),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	logsafe.Debug(r.logger, message, fields...)
}

// resourceStatsErrorMessage 将资源统计原因转换为用户可读的错误消息。
func resourceStatsErrorMessage(reason string) string {
	switch reason {
	case containerStatsNotCollectedReason:
		return "Container stats were not collected."
	case containerStatsIncompleteReason:
		return "Container stats did not include CPU or memory data."
	case containerStatsTimeoutReason:
		return "Container stats collection timed out."
	default:
		return "Container stats are unavailable."
	}
}

// parseResourceCollectedAt 解析资源采集时间。
func parseResourceCollectedAt(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

// uint64ToInt64 将 uint64 值转换为 int64，并在超出范围时返回失败。
func uint64ToInt64(value uint64) (int64, bool) {
	if value > uint64(^uint64(0)>>1) {
		return 0, false
	}
	return int64(value), true
}

func uint64ToInt64Ptr(value uint64) *int64 {
	converted, ok := uint64ToInt64(value)
	if !ok {
		return nil
	}
	return &converted
}

func uint32ToInt64Ptr(value uint32) *int64 {
	converted := int64(value)
	return &converted
}
