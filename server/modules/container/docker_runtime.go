package container

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
	"graft/server/modules/container/terminal"
)

const (
	dockerSocketScheme       = "unix"
	dockerLogScannerInitSize = 64 * 1024
	dockerLogScannerMaxSize  = 1024 * 1024
	dockerStatsListTimeout   = 2 * time.Second
	dockerStatsListWorkers   = 8
	dockerStatsPercentScale  = 100.0
	dockerEnvironmentSource  = "docker"
)

var errInvalidLogQuery = errors.New("invalid log query parameter")

// DockerRuntime adapts the official Docker SDK to the container module runtime boundary.
type DockerRuntime struct {
	client            dockerClient
	endpoint          string
	logger            *zap.Logger
	mountUsageScanner mountUsageScanner
	resourceStats     *resourceStatsCache
	cpuBaselinesMu    sync.Mutex
	cpuBaselines      map[string]dockerCPUStatsBaseline
}

type dockerCPUStatsBaseline struct {
	totalUsage  uint64
	systemUsage uint64
	onlineCPUs  uint32
	collectedAt time.Time
}

type dockerClient interface {
	Info(context.Context) (systemInfo, error)
	ContainerList(context.Context, mobyclient.ContainerListOptions) ([]container.Summary, error)
	ContainerInspect(context.Context, string) (container.InspectResponse, error)
	ContainerLogs(context.Context, string, mobyclient.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerStatsOneShot(context.Context, string) (mobyclient.ContainerStatsResult, error)
	ContainerExecCreate(context.Context, string, mobyclient.ExecCreateOptions) (mobyclient.ExecCreateResult, error)
	ContainerExecAttach(context.Context, string, mobyclient.ExecAttachOptions) (mobyclient.HijackedResponse, error)
	ContainerExecResize(context.Context, string, mobyclient.ExecResizeOptions) error
	ContainerStart(context.Context, string, mobyclient.ContainerStartOptions) error
	ContainerStop(context.Context, string, mobyclient.ContainerStopOptions) error
	ContainerRestart(context.Context, string, mobyclient.ContainerRestartOptions) error
	ContainerRemove(context.Context, string, mobyclient.ContainerRemoveOptions) error
	Close() error
}

type systemInfo interface {
	dockerSystemInfo()
}

// NewDockerRuntime 创建 Docker 容器运行时适配器。
// staleWindow 指定缓存允许返回过期数据的时间窗口。
func NewDockerRuntime(endpoint string, logger *zap.Logger, cacheTTL time.Duration, staleWindow time.Duration) (*DockerRuntime, error) {
	endpoint = firstNonEmpty(endpoint, defaultContainerDockerEndpoint)
	cli, err := mobyclient.New(mobyclient.WithHost(endpoint))
	if err != nil {
		return nil, mapDockerError(err)
	}
	return &DockerRuntime{
		client:        dockerClientAdapter{Client: cli},
		endpoint:      endpoint,
		logger:        logger,
		resourceStats: newResourceStatsCache(cacheTTL, staleWindow),
		cpuBaselines:  make(map[string]dockerCPUStatsBaseline),
	}, nil
}

// Info returns sanitized Docker runtime metadata for API responses.
func (r *DockerRuntime) Info(ctx context.Context) (RuntimeInfo, error) {
	info, err := r.client.Info(ctx)
	if err != nil {
		return RuntimeInfo{}, mapDockerError(err)
	}
	return dockerInfoToRuntimeInfo(info, safeEndpointLabel(r.endpoint)), nil
}

// List returns Docker container summaries without raw inspect, logs, or env leakage.
func (r *DockerRuntime) List(ctx context.Context, _ ListQuery) ([]Summary, error) {
	items, err := r.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return nil, mapDockerError(err)
	}
	summaries := make([]Summary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, dockerSummary(item))
	}
	r.collectListResourceSummaries(ctx, summaries)
	return summaries, nil
}

// Detail returns a sanitized Docker inspect view without environment variables or raw sensitive fields.
func (r *DockerRuntime) Detail(ctx context.Context, ref Ref) (Detail, error) {
	inspect, err := r.client.ContainerInspect(ctx, ref.Value)
	if err != nil {
		return Detail{}, mapDockerError(err)
	}
	info, err := r.Info(ctx)
	if err != nil {
		return Detail{}, err
	}
	detail := dockerDetail(inspect, info)
	detail.Resource = r.currentResourceSummary(firstNonEmpty(detail.ID, ref.Value))
	return detail, nil
}

// Mounts returns sanitized mount metadata from Docker inspect.
func (r *DockerRuntime) Mounts(ctx context.Context, ref Ref) ([]Mount, error) {
	inspect, err := r.client.ContainerInspect(ctx, ref.Value)
	if err != nil {
		return nil, mapDockerError(err)
	}
	return dockerMounts(inspect.Mounts), nil
}

// MountUsage measures one inspect-derived mount source without accepting arbitrary paths.
func (r *DockerRuntime) MountUsage(ctx context.Context, ref Ref, mountID string) (MountUsage, error) {
	inspect, err := r.client.ContainerInspect(ctx, ref.Value)
	if err != nil {
		return MountUsage{}, mapDockerError(err)
	}
	mount, ok := findMountByID(dockerMounts(inspect.Mounts), mountID)
	if !ok {
		return MountUsage{}, errContainerMountNotFound
	}
	if !mountUsageSupported(mount) {
		return mountUsageFromMount(strings.TrimSpace(inspect.ID), mount, containerMountUsageStatusUnsupported, 0, ""), nil
	}
	scanner := r.mountUsageScanner
	if scanner == nil {
		scanner = filesystemMountUsageScanner{}
	}
	size, err := scanner.ScanUsage(ctx, mount.Source)
	if err != nil {
		return mountUsageFromScanError(strings.TrimSpace(inspect.ID), mount, err), nil
	}
	return mountUsageFromMount(strings.TrimSpace(inspect.ID), mount, containerMountUsageStatusMeasured, size, time.Now().UTC().Format(time.RFC3339)), nil
}

// Logs reads bounded Docker logs according to the module log guardrails.
func (r *DockerRuntime) Logs(ctx context.Context, ref Ref, query LogQuery) (Logs, error) {
	since, err := parseLogSince(query.Since)
	if err != nil {
		return Logs{}, fmt.Errorf("%w: %v", errInvalidLogQuery, err)
	}
	reader, err := r.client.ContainerLogs(ctx, ref.Value, mobyclient.ContainerLogsOptions{
		ShowStdout: query.Stdout,
		ShowStderr: query.Stderr,
		Since:      since,
		Timestamps: query.Timestamps,
		Tail:       strconv.Itoa(query.Tail),
	})
	if err != nil {
		return Logs{}, mapDockerError(err)
	}
	defer func() {
		_ = reader.Close()
	}()

	entries, truncated, err := readDockerLogEntries(ctx, reader, query.Tail, query.Timestamps)
	if err != nil {
		return Logs{}, mapDockerError(err)
	}
	name := ""
	id := ref.Value
	if inspect, inspectErr := r.client.ContainerInspect(ctx, ref.Value); inspectErr == nil {
		if trimmedID := strings.TrimSpace(inspect.ID); trimmedID != "" {
			id = trimmedID
		}
		name = firstContainerName([]string{strings.TrimPrefix(strings.TrimSpace(inspect.Name), "/")})
	}
	return Logs{
		ID:         id,
		Name:       name,
		Runtime:    runtimeNameDocker,
		Entries:    entries,
		Tail:       query.Tail,
		Since:      query.Since,
		Timestamps: query.Timestamps,
		Stdout:     query.Stdout,
		Stderr:     query.Stderr,
		Truncated:  truncated,
	}, nil
}

// StreamLogs follows incremental Docker logs and emits one normalized log chunk
// per line until the caller context is canceled or the runtime stream ends.
func (r *DockerRuntime) StreamLogs(ctx context.Context, ref Ref, query LogQuery, emit func(LogChunk) error) error {
	if emit == nil {
		return errors.New("container log stream emitter is required")
	}
	since, err := parseLogSince(query.Since)
	if err != nil {
		return fmt.Errorf("%w: %v", errInvalidLogQuery, err)
	}
	reader, err := r.client.ContainerLogs(ctx, ref.Value, mobyclient.ContainerLogsOptions{
		ShowStdout: query.Stdout,
		ShowStderr: query.Stderr,
		Since:      since,
		Timestamps: query.Timestamps,
		Follow:     true,
		Tail:       strconv.Itoa(query.Tail),
	})
	if err != nil {
		return mapDockerError(err)
	}
	defer func() {
		_ = reader.Close()
	}()
	return streamDockerLogLines(ctx, reader, query.Timestamps, emit)
}

// Shell opens one interactive exec session inside the target container.
func (r *DockerRuntime) Shell(ctx context.Context, ref Ref, command string) (terminal.Session, error) {
	inspect, err := r.client.ContainerInspect(ctx, ref.Value)
	if err != nil {
		return nil, mapDockerShellError(err)
	}
	if strings.TrimSpace(inspect.ID) == "" {
		return nil, errContainerNotFound
	}
	return newDockerExecSession(r.client, inspect.ID, command), nil
}

// mapDockerShellError 将 Docker Shell 执行错误映射为特定领域的错误类型。
func mapDockerShellError(err error) error {
	if err == nil {
		return nil
	}
	mapped := mapDockerError(err)
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case errors.Is(mapped, errContainerNotFound):
		return errContainerNotFound
	case strings.Contains(message, "executable file not found"),
		strings.Contains(message, "not found in $path"):
		return errShellCommandNotFound
	default:
		if mapped != nil {
			return mapped
		}
		return errShellSessionFailed
	}
}

// Start starts one Docker container by id or name.
func (r *DockerRuntime) Start(ctx context.Context, ref Ref) (ActionResult, error) {
	before, _ := r.Detail(ctx, ref)
	if before.State != "" && !canStartState(before.State) {
		return actionResultFromDetail(before, ref, containerActionStart, before.State), errInvalidContainerState
	}
	if err := r.client.ContainerStart(ctx, ref.Value, mobyclient.ContainerStartOptions{}); err != nil {
		return actionResultFromDetail(before, ref, containerActionStart, ""), mapDockerError(err)
	}
	r.invalidateResourceSummary(ref.Value, before.ID)
	after, _ := r.Detail(ctx, ref)
	return actionResultFromDetail(after, ref, containerActionStart, before.State), nil
}

// Stop stops one Docker container by id or name.
func (r *DockerRuntime) Stop(ctx context.Context, ref Ref) (ActionResult, error) {
	return r.runTimedStateAction(ctx, ref, containerActionStop, canStopState, func(ctx context.Context, id string, timeout *int) error {
		return r.client.ContainerStop(ctx, id, mobyclient.ContainerStopOptions{Timeout: timeout})
	})
}

// Restart restarts one Docker container by id or name.
func (r *DockerRuntime) Restart(ctx context.Context, ref Ref) (ActionResult, error) {
	return r.runTimedStateAction(ctx, ref, containerActionRestart, canRestartState, func(ctx context.Context, id string, timeout *int) error {
		return r.client.ContainerRestart(ctx, id, mobyclient.ContainerRestartOptions{Timeout: timeout})
	})
}

func (r *DockerRuntime) runTimedStateAction(
	ctx context.Context,
	ref Ref,
	action string,
	allowed func(string) bool,
	run func(context.Context, string, *int) error,
) (ActionResult, error) {
	before, _ := r.Detail(ctx, ref)
	if before.State != "" && !allowed(before.State) {
		return actionResultFromDetail(before, ref, action, before.State), errInvalidContainerState
	}
	timeout := 10
	if err := run(ctx, ref.Value, &timeout); err != nil {
		return actionResultFromDetail(before, ref, action, ""), mapDockerError(err)
	}
	r.invalidateResourceSummary(ref.Value, before.ID)
	after, _ := r.Detail(ctx, ref)
	return actionResultFromDetail(after, ref, action, before.State), nil
}

// Remove removes one Docker container by id or name.
func (r *DockerRuntime) Remove(ctx context.Context, ref Ref, options RemoveOptions) (ActionResult, error) {
	before, err := r.Detail(ctx, ref)
	if err != nil {
		return actionResultFromDetail(before, ref, containerActionRemove, ""), err
	}
	if !canRemoveState(before.State) || (!options.Force && !canRemoveWithoutForce(before.State)) {
		return actionResultFromDetail(before, ref, containerActionRemove, before.State), errInvalidContainerState
	}
	if err := r.client.ContainerRemove(ctx, ref.Value, mobyclient.ContainerRemoveOptions{Force: options.Force}); err != nil {
		return actionResultFromDetail(before, ref, containerActionRemove, before.State), mapDockerError(err)
	}
	r.invalidateResourceSummary(ref.Value, before.ID)
	result := actionResultFromDetail(before, ref, containerActionRemove, before.State)
	result.StatusAfter = actionStatusRemoved
	result.Result = actionResultCompleted
	return result, nil
}

// Close releases the Docker SDK client resources.
func (r *DockerRuntime) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// dockerSummary converts a Docker container summary to the module Summary type, normalizing metadata and detecting orchestrator information.
func dockerSummary(item container.Summary) Summary {
	names := cleanDockerNames(item.Names)
	networks := dockerSummaryNetworks(item)
	primaryIP := primaryNetworkIP(networks)
	state := normalizeContainerState(string(item.State))
	labels := cloneLabels(item.Labels)
	orchestrator := dockerOrchestratorFromLabels(labels)
	return Summary{
		ID:             strings.TrimSpace(item.ID),
		ShortID:        shortRuntimeID(item.ID),
		Name:           firstNonEmpty(firstContainerName(names), shortRuntimeID(item.ID), strings.TrimSpace(item.ID)),
		Names:          names,
		Image:          strings.TrimSpace(item.Image),
		ImageID:        strings.TrimSpace(item.ImageID),
		Labels:         labels,
		Ports:          dockerPorts(item.Ports),
		PrimaryIP:      primaryIP,
		Networks:       networks,
		NetworkSummary: networkSummary(networks),
		Resource: ResourceSummary{
			Available:         false,
			UnavailableReason: containerStatsNotCollectedReason,
			StatsAvailable:    false,
			StatsErrorKey:     containerStatsNotCollectedReason,
			StatsErrorMessage: "Container stats were not collected.",
		},
		Runtime:        runtimeNameDocker,
		CreatedAt:      time.Unix(item.Created, 0).UTC().Format(time.RFC3339),
		State:          state,
		Status:         strings.TrimSpace(item.Status),
		Health:         containerHealthUnavailable,
		ComposeProject: strings.TrimSpace(labels[composeProjectLabel]),
		ComposeService: strings.TrimSpace(labels[composeServiceLabel]),
		Orchestrator:   orchestrator,
		CanStart:       canStartState(state),
		CanStop:        canStopState(state),
		CanRestart:     canRestartState(state),
		CanRemove:      canRemoveState(state),
	}
}

func (r *DockerRuntime) containerResourceSummary(ctx context.Context, id string) ResourceSummary {
	ref := strings.TrimSpace(id)
	if ref == "" {
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}
	statsCtx, cancel := context.WithTimeout(ctx, dockerStatsListTimeout)
	defer cancel()

	reader, err := r.client.ContainerStatsOneShot(statsCtx, ref)
	if err != nil {
		return unavailableResourceSummary(resourceStatsErrorReason(err))
	}
	defer func() {
		if reader.Body != nil {
			_ = reader.Body.Close()
		}
	}()
	if reader.Body == nil {
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}

	var stats container.StatsResponse
	if err := json.NewDecoder(reader.Body).Decode(&stats); err != nil {
		return unavailableResourceSummary(resourceStatsErrorReason(err))
	}
	return r.dockerResourceSummary(ref, stats)
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

func (r *DockerRuntime) collectListResourceSummaries(ctx context.Context, summaries []Summary) {
	_ = ctx
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

	workers := min(len(items), dockerStatsListWorkers)
	if workers < 1 {
		workers = 1
	}

	indexes := make(chan int, len(items))
	for index := range items {
		indexes <- index
	}
	close(indexes)

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for index := range indexes {
				summary := dockerSummary(items[index])
				resource := r.collectCachedResourceSummary(ctx, summary.ID)
				snapshotCollectedAt := collectedAt
				if parsedCollectedAt, ok := parseResourceCollectedAt(resource.CollectedAt); ok {
					snapshotCollectedAt = parsedCollectedAt
				} else if strings.TrimSpace(resource.CollectedAt) == "" {
					resource.CollectedAt = collectedAt.Format(time.RFC3339)
				}
				snapshots[index] = StatsSnapshot{
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
		}()
	}
	wg.Wait()
	return snapshots, nil
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
//
// @returns 在线 CPU 数量的指针；当无法确定时返回 nil。
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
// @returns 累加后的和；如果输入已溢出或加法会溢出，则返回 0 和 true。
func addUint64(total uint64, value uint64, overflow bool) (uint64, bool) {
	if overflow || total > ^uint64(0)-value {
		return 0, true
	}
	return total + value, false
}

// 优先使用运行时维护的上一帧 one-shot 样本计算 CPU 百分比。
// 当基线缺失或当前采样不完整时返回 false，并在可行时更新基线供下一帧使用。
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

// dockerCurrentCPUStatsBaseline 提取用于计算 CPU 百分比的当前基线。
// 当可用 CPU 数或系统 CPU 使用量为零时，返回 false。
// dockerCurrentCPUStatsBaseline 提取当前容器 CPU 统计基线。
// 当可用在线 CPU 数或系统 CPU 使用量为 0 时，返回无效基线。
// @returns 有效的 CPU 基线及其是否可用。
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
		delete(r.cpuBaselines, normalizedID)
	}
}

// dockerStatsOnlineCPUs 返回统计信息中的在线 CPU 数。
// 当在线 CPU 数不可用时，它会使用每个 CPU 的使用率条目数量作为估算值；
// 如果也无法获得该数量，或数量超出 uint32 范围，则返回 0。
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
// 摘要会使用给定原因，或在原因为空时回退到默认的资源统计不可用原因。
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

// resourceStatsErrorMessage 将资源统计原因转换为用户可读的错误消息。
// reason 对应的消息包括“未收集”“数据不完整”“收集超时”和通用不可用说明。
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
// 成功时返回转换为 UTC 的时间和 true；解析失败时返回零值和 false。
func parseResourceCollectedAt(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

// uint64ToInt64 将 uint64 值转换为 int64，并在超出范围时返回失败。
// @returns 转换后的 int64 值，以及一个布尔值；当原值可以安全表示为 int64 时为 `true`，否则为 `false`。
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

// dockerDetail 从 Docker 容器检查输出和运行时元数据构建 Detail 结构体。
func dockerDetail(inspect container.InspectResponse, info RuntimeInfo) Detail {
	state, status, startedAt := dockerState(inspect)
	labels := dockerLabels(inspect)
	orchestrator := dockerOrchestratorFromLabels(labels)
	summary := Summary{
		ID:             strings.TrimSpace(inspect.ID),
		ShortID:        shortRuntimeID(inspect.ID),
		Names:          []string{strings.TrimPrefix(strings.TrimSpace(inspect.Name), "/")},
		Image:          dockerImage(inspect),
		ImageID:        strings.TrimSpace(inspect.Image),
		Labels:         labels,
		Ports:          dockerInspectPorts(inspect),
		Networks:       dockerNetworks(inspect),
		Resource:       unavailableResourceSummary(containerStatsNotCollectedReason),
		Runtime:        runtimeNameDocker,
		CreatedAt:      parseDockerTimeString(inspect.Created),
		StartedAt:      startedAt,
		State:          state,
		Status:         status,
		Health:         dockerHealth(inspect),
		ComposeProject: strings.TrimSpace(labels[composeProjectLabel]),
		ComposeService: strings.TrimSpace(labels[composeServiceLabel]),
		Orchestrator:   orchestrator,
		CanStart:       canStartState(state),
		CanStop:        canStopState(state),
		CanRestart:     canRestartState(state),
		CanRemove:      canRemoveState(state),
	}
	summary.Name = firstNonEmpty(firstContainerName(summary.Names), summary.ShortID, summary.ID)
	summary.PrimaryIP = primaryNetworkIP(summary.Networks)
	summary.NetworkSummary = networkSummary(summary.Networks)
	summary.RestartCount = intPtrAllowZero(inspect.RestartCount)
	detail := Detail{
		Summary:          summary,
		Healthcheck:      dockerHealthcheck(inspect),
		LastExitCode:     dockerLastExitCode(inspect),
		Mounts:           dockerMounts(inspect.Mounts),
		Networks:         dockerNetworks(inspect),
		OOMKilled:        dockerOOMKilled(inspect),
		RuntimeInfo:      info,
		InspectUpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if inspect.Config != nil {
		detail.Command = []string(inspect.Config.Cmd)
		detail.Entrypoint = []string(inspect.Config.Entrypoint)
		detail.Environment = dockerEnvironmentVariables(inspect.Config.Env)
		detail.WorkingDir = strings.TrimSpace(inspect.Config.WorkingDir)
	}
	if inspect.HostConfig != nil {
		detail.RestartPolicy = string(inspect.HostConfig.RestartPolicy.Name)
	}
	return detail
}

// dockerOrchestratorFromLabels 从容器标签检测编排器类型，并返回该编排器的元数据。
func dockerOrchestratorFromLabels(labels map[string]string) OrchestratorInfo {
	labels = cloneLabels(labels)
	if len(labels) == 0 {
		return OrchestratorInfo{
			Type:            containerOrchestratorStandalone,
			Managed:         false,
			Confidence:      orchestratorConfidenceHigh,
			GroupScopeKind:  "",
			MemberScopeKind: "",
		}
	}

	typeCount := 0
	info := OrchestratorInfo{
		Type:            containerOrchestratorStandalone,
		Managed:         false,
		Confidence:      orchestratorConfidenceHigh,
		GroupScopeKind:  "",
		MemberScopeKind: "",
	}

	if metadata, ok := kubernetesMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorKubernetes
		info.Managed = true
		info.GroupScopeKind = kubernetesNamespaceScopeKind
		info.GroupValue = metadata.Namespace
		info.GroupDisplayName = metadata.Namespace
		info.MemberScopeKind = kubernetesPodScopeKind
		info.MemberValue = metadata.Pod
		info.MemberDisplayName = metadata.Pod
		info.Namespace = metadata.Namespace
		info.Pod = metadata.Pod
		info.Container = metadata.Container
		info.DisplayName = firstNonEmpty(metadata.Namespace, metadata.Pod, "kubernetes")
		info.Confidence = orchestratorConfidenceHigh
	}
	if stack, task, ok := swarmMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorSwarm
		info.Managed = true
		info.GroupScopeKind = swarmStackScopeKind
		info.GroupValue = stack
		info.GroupDisplayName = stack
		info.MemberScopeKind = swarmTaskScopeKind
		info.MemberValue = task
		info.MemberDisplayName = task
		info.Stack = stack
		info.Task = task
		info.DisplayName = firstNonEmpty(stack, task, "swarm")
		info.Confidence = orchestratorConfidenceHigh
	}
	if project, service, ok := composeMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorCompose
		info.Managed = true
		info.GroupScopeKind = composeProjectScopeKind
		info.GroupValue = project
		info.GroupDisplayName = project
		info.MemberScopeKind = composeServiceScopeKind
		info.MemberValue = service
		info.MemberDisplayName = service
		info.Project = project
		info.Service = service
		info.DisplayName = firstNonEmpty(project, service, "compose")
		info.Confidence = orchestratorConfidenceHigh
	}
	if typeCount == 0 {
		return info
	}
	if typeCount > 1 {
		return OrchestratorInfo{
			Type:            containerOrchestratorUnknown,
			Managed:         true,
			Confidence:      orchestratorConfidenceLow,
			GroupScopeKind:  "",
			MemberScopeKind: "",
		}
	}
	return info
}

type kubernetesOrchestratorMetadata struct {
	Namespace string
	Pod       string
	Container string
}

// kubernetesMetadata extracts Kubernetes metadata from the provided labels, returning the extracted metadata and whether any Kubernetes metadata was found.
func kubernetesMetadata(labels map[string]string) (kubernetesOrchestratorMetadata, bool) {
	metadata := kubernetesOrchestratorMetadata{
		Namespace: strings.TrimSpace(labels["io.kubernetes.pod.namespace"]),
		Pod:       strings.TrimSpace(labels["io.kubernetes.pod.name"]),
		Container: strings.TrimSpace(labels["io.kubernetes.container.name"]),
	}
	ok := metadata.Namespace != "" || metadata.Pod != "" || metadata.Container != ""
	return metadata, ok
}

// swarmMetadata extracts Docker Swarm metadata from container labels. It returns the stack namespace, task name, and a flag indicating whether valid metadata was found.
func swarmMetadata(labels map[string]string) (stack string, task string, ok bool) {
	stack = strings.TrimSpace(labels["com.docker.stack.namespace"])
	task = strings.TrimSpace(labels["com.docker.swarm.task.name"])
	ok = stack != "" || task != ""
	return stack, task, ok
}

// composeMetadata 从容器标签中检测 Docker Compose 的项目和服务名称。
// 返回项目名、服务名以及是否检测到 Compose 编排器的布尔值。
func composeMetadata(labels map[string]string) (project string, service string, ok bool) {
	project = strings.TrimSpace(labels[composeProjectLabel])
	service = strings.TrimSpace(labels[composeServiceLabel])
	ok = project != "" || service != ""
	return project, service, ok
}

// dockerEnvironmentVariables 将原始环境变量字符串列表解析为 EnvironmentVariable 对象。跳过格式错误（无等号分隔符）或键为空的条目。返回的每个 EnvironmentVariable 对象包含其敏感性判断和来源标记。
func dockerEnvironmentVariables(values []string) []EnvironmentVariable {
	if len(values) == 0 {
		return nil
	}
	environment := make([]EnvironmentVariable, 0, len(values))
	for _, raw := range values {
		key, value, ok := strings.Cut(raw, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		environment = append(environment, EnvironmentVariable{
			Key:       key,
			Value:     value,
			Sensitive: isSensitiveEnvironmentKey(key),
			Source:    dockerEnvironmentSource,
		})
	}
	return environment
}

func dockerSummaryNetworks(item container.Summary) []Network {
	if item.NetworkSettings == nil || len(item.NetworkSettings.Networks) == 0 {
		return nil
	}
	networks := make([]Network, 0, len(item.NetworkSettings.Networks))
	for name, network := range item.NetworkSettings.Networks {
		if mapped, ok := dockerEndpointNetwork(name, network); ok {
			networks = append(networks, mapped)
		}
	}
	return networks
}

func primaryNetworkIP(networks []Network) string {
	for _, network := range networks {
		if strings.TrimSpace(network.IPAddress) != "" {
			return strings.TrimSpace(network.IPAddress)
		}
	}
	return ""
}

func networkSummary(networks []Network) string {
	names := make([]string, 0, len(networks))
	for _, network := range networks {
		if strings.TrimSpace(network.Name) != "" {
			names = append(names, strings.TrimSpace(network.Name))
		}
	}
	return strings.Join(names, ", ")
}

// dockerHealth derives the container's normalized health status from inspection data.
// It returns a constant representing no health check, starting, healthy, unhealthy, or unavailable.
func dockerHealth(inspect container.InspectResponse) string {
	if inspect.State == nil || inspect.State.Health == nil {
		return containerHealthNone
	}
	switch inspect.State.Health.Status {
	case container.NoHealthcheck:
		return containerHealthNone
	case container.Starting:
		return containerHealthStarting
	case container.Healthy:
		return containerHealthHealthy
	case container.Unhealthy:
		return containerHealthUnhealthy
	default:
		return containerHealthUnavailable
	}
}

// health state is unavailable, status is set to unavailable.
func dockerHealthcheck(inspect container.InspectResponse) *Healthcheck {
	command := dockerHealthcheckCommand(inspect)
	if len(command) == 0 {
		return nil
	}
	result := &Healthcheck{
		Configured: true,
		Status:     dockerHealth(inspect),
		Command:    command,
	}
	if inspect.State == nil || inspect.State.Health == nil {
		result.Status = containerHealthUnavailable
		return result
	}

	health := inspect.State.Health
	result.FailingStreak = intPtrAllowZero(health.FailingStreak)
	if len(health.Log) == 0 {
		return result
	}
	last := health.Log[len(health.Log)-1]
	if last == nil {
		return result
	}
	result.ExitCode = intPtrAllowZero(last.ExitCode)
	result.Output = strings.TrimSpace(last.Output)
	if !last.End.IsZero() {
		result.CheckedAt = last.End.UTC().Format(time.RFC3339)
	} else if !last.Start.IsZero() {
		result.CheckedAt = last.Start.UTC().Format(time.RFC3339)
	}
	if last.ExitCode != 0 {
		result.FailureMessage = result.Output
	}
	return result
}

// dockerHealthcheckCommand extracts the healthcheck test command from a container's configuration, returning the trimmed command parts or nil if no healthcheck is configured, disabled, or contains only empty items.
func dockerHealthcheckCommand(inspect container.InspectResponse) []string {
	if inspect.Config == nil || inspect.Config.Healthcheck == nil {
		return nil
	}
	test := inspect.Config.Healthcheck.Test
	if len(test) == 0 {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(test[0]), "NONE") {
		return nil
	}
	command := make([]string, 0, len(test))
	for _, item := range test {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			command = append(command, trimmed)
		}
	}
	if len(command) == 0 {
		return nil
	}
	return command
}

// dockerLastExitCode returns a pointer to the container's exit code from the inspection state, or nil if the state is unavailable.
func dockerLastExitCode(inspect container.InspectResponse) *int {
	if inspect.State == nil {
		return nil
	}
	return intPtrAllowZero(inspect.State.ExitCode)
}

// dockerOOMKilled returns a pointer to the OOMKilled flag from the container's state, or nil if the state is unavailable.
func dockerOOMKilled(inspect container.InspectResponse) *bool {
	if inspect.State == nil {
		return nil
	}
	value := inspect.State.OOMKilled
	return &value
}

// ShortRuntimeID returns the runtime ID truncated to containerShortIDLength characters.
func shortRuntimeID(id string) string {
	value := strings.TrimSpace(id)
	if len(value) <= containerShortIDLength {
		return value
	}
	return value[:containerShortIDLength]
}

func canStartState(state string) bool {
	return state != "running" && state != "paused" && state != "removing"
}

func canStopState(state string) bool {
	return state == "running"
}

func canRestartState(state string) bool {
	return state != "removing" && state != "dead"
}

func canRemoveState(state string) bool {
	return state != "" && state != "unknown" && state != "removing"
}

func canRemoveWithoutForce(state string) bool {
	return canRemoveState(state) && state != "running" && state != "paused" && state != "restarting"
}

func dockerInfoToRuntimeInfo(info systemInfo, endpoint string) RuntimeInfo {
	value, ok := info.(dockerClientSystemInfo)
	if !ok {
		return RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled", Endpoint: endpoint}
	}
	return RuntimeInfo{
		Runtime:           runtimeNameDocker,
		Status:            "enabled",
		Endpoint:          endpoint,
		APIVersion:        value.APIVersion,
		ServerVersion:     value.ServerVersion,
		OperatingSystem:   value.OperatingSystem,
		Architecture:      value.Architecture,
		ContainersTotal:   value.Containers,
		ContainersRunning: value.ContainersRunning,
	}
}

func mapDockerError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errContainerRuntimeTimeout
	}
	if cerrdefs.IsNotFound(err) {
		return errContainerNotFound
	}
	if errors.Is(err, os.ErrNotExist) {
		return errRuntimeSocketMissing
	}
	if errors.Is(err, os.ErrPermission) {
		return errRuntimePermissionDenied
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errContainerRuntimeTimeout
	}
	if mapped := mapSyscallDockerError(err); mapped != nil {
		return mapped
	}
	return mapDockerMessageError(err.Error())
}

func mapSyscallDockerError(err error) error {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return nil
	}
	switch errno {
	case syscall.EACCES, syscall.EPERM:
		return errRuntimePermissionDenied
	case syscall.ENOENT:
		return errRuntimeSocketMissing
	case syscall.ECONNREFUSED, syscall.ECONNRESET:
		return errRuntimeDaemonUnavailable
	default:
		return nil
	}
}

// mapDockerMessageError 根据错误消息中的关键片段映射容器运行时错误。
// 如果消息匹配已知规则，返回对应错误；否则返回运行时守护进程不可用错误。
func mapDockerMessageError(message string) error {
	normalized := strings.ToLower(message)
	for _, rule := range dockerErrorMessageRules {
		if strings.Contains(normalized, rule.fragment) {
			return rule.err
		}
	}
	return errRuntimeDaemonUnavailable
}

// readDockerLogEntries 使用调用方上下文读取并截取容器日志条目。
//
// 返回按时间顺序排列的结构化日志条目，以及是否因尾部截断而丢弃了更早的内容。
// readDockerLogEntries 读取 Docker 日志并将其转换为日志条目，按指定尾部条数保留最近的记录。
// 读取过程中发生错误时返回错误，并指示结果是否被截断。
func readDockerLogEntries(ctx context.Context, reader io.Reader, tail int, timestamps bool) ([]LogEntry, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	limit := tail
	if limit > defaultContainerLogsMaxTail {
		limit = defaultContainerLogsMaxTail
	}
	truncated := false
	if limit <= 0 {
		err := streamDockerLogLines(ctx, reader, timestamps, func(_ LogChunk) error {
			truncated = true
			return nil
		})
		if err != nil {
			return nil, false, err
		}
		return nil, truncated, nil
	}

	buffer := make([]LogEntry, limit)
	count := 0
	head := 0
	err := streamDockerLogLines(ctx, reader, timestamps, func(chunk LogChunk) error {
		entry := logEntryFromChunk(chunk)
		if count < limit {
			buffer[count] = entry
			count++
			return nil
		}
		if count == limit {
			truncated = true
			buffer[head] = entry
			head = (head + 1) % limit
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return materializeBoundedLogEntries(buffer, count, limit, head), truncated, nil
}

func materializeBoundedLogEntries(buffer []LogEntry, count int, limit int, head int) []LogEntry {
	if count == 0 {
		return nil
	}
	if count < limit {
		entries := make([]LogEntry, count)
		copy(entries, buffer[:count])
		return entries
	}

	entries := make([]LogEntry, limit)
	// Reconstruct the full ring so callers still receive oldest-to-newest order.
	for index := range entries {
		entries[index] = buffer[(head+index)%limit]
	}
	return entries
}

// 当上下文取消、回调返回错误或读取过程发生错误时返回相应错误。
func streamDockerLogLines(ctx context.Context, reader io.Reader, timestamps bool, emit func(LogChunk) error) error {
	if reader == nil {
		return nil
	}
	chunkEmitter := newDockerLogChunkEmitter(ctx, timestamps, emit)
	_, err := stdcopy.StdCopy(chunkEmitter.stdoutWriter(), chunkEmitter.stderrWriter(), reader)
	if flushErr := chunkEmitter.flush(); flushErr != nil {
		return flushErr
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

type dockerLogChunkEmitter struct {
	ctx        context.Context
	timestamps bool
	emit       func(LogChunk) error
	err        error
	stdout     dockerLogStreamWriter
	stderr     dockerLogStreamWriter
}

type dockerLogStreamWriter struct {
	parent *dockerLogChunkEmitter
	stream string
	buffer []byte
}

func newDockerLogChunkEmitter(ctx context.Context, timestamps bool, emit func(LogChunk) error) *dockerLogChunkEmitter {
	emitter := &dockerLogChunkEmitter{
		ctx:        ctx,
		timestamps: timestamps,
		emit:       emit,
	}
	emitter.stdout = dockerLogStreamWriter{parent: emitter, stream: "stdout", buffer: make([]byte, 0, dockerLogScannerInitSize)}
	emitter.stderr = dockerLogStreamWriter{parent: emitter, stream: "stderr", buffer: make([]byte, 0, dockerLogScannerInitSize)}
	return emitter
}

func (e *dockerLogChunkEmitter) stdoutWriter() io.Writer {
	return &e.stdout
}

func (e *dockerLogChunkEmitter) stderrWriter() io.Writer {
	return &e.stderr
}

func (e *dockerLogChunkEmitter) emitChunk(stream string, line string) error {
	if e.err != nil {
		return e.err
	}
	select {
	case <-e.ctx.Done():
		e.err = e.ctx.Err()
		return e.err
	default:
	}
	chunk := LogChunk{
		Line:   line,
		Stream: stream,
	}
	if e.timestamps {
		chunk.Timestamp, chunk.Line = parseDockerLogChunkTimestamp(line)
	}
	if err := e.emit(chunk); err != nil {
		e.err = err
		return err
	}
	return nil
}

// logEntryFromChunk 将日志块转换为日志条目，并将发生时间规范为 UTC；
// 当时间为空时，使用当前 UTC 时间。
func logEntryFromChunk(chunk LogChunk) LogEntry {
	occurredAt := chunk.Timestamp.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return LogEntry{
		Line:       chunk.Line,
		Stream:     strings.TrimSpace(chunk.Stream),
		OccurredAt: occurredAt,
	}
}

func (e *dockerLogChunkEmitter) flush() error {
	if err := e.stdout.flushRemainder(); err != nil {
		return err
	}
	if err := e.stderr.flushRemainder(); err != nil {
		return err
	}
	return e.err
}

func (w *dockerLogStreamWriter) Write(p []byte) (int, error) {
	if w.parent.err != nil {
		return 0, w.parent.err
	}
	w.buffer = append(w.buffer, p...)
	for {
		index := bytes.IndexByte(w.buffer, '\n')
		if index < 0 {
			if len(w.buffer) > dockerLogScannerMaxSize {
				w.parent.err = bufio.ErrTooLong
				return 0, w.parent.err
			}
			return len(p), nil
		}
		line := string(w.buffer[:index])
		w.buffer = w.buffer[index+1:]
		if err := w.parent.emitChunk(w.stream, line); err != nil {
			return 0, err
		}
	}
}

func (w *dockerLogStreamWriter) flushRemainder() error {
	if len(w.buffer) == 0 {
		return w.parent.err
	}
	if len(w.buffer) > dockerLogScannerMaxSize {
		w.parent.err = bufio.ErrTooLong
		return w.parent.err
	}
	line := string(w.buffer)
	w.buffer = nil
	return w.parent.emitChunk(w.stream, line)
}

func parseDockerLogChunkTimestamp(line string) (time.Time, string) {
	rawTimestamp, rest, ok := strings.Cut(line, " ")
	if !ok {
		return time.Time{}, line
	}
	timestamp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(rawTimestamp))
	if err != nil {
		return time.Time{}, line
	}
	return timestamp.UTC(), rest
}

// actionResultFromDetail 根据容器详情构造操作结果，并标记状态是否发生变化。
// 当操作前状态与当前状态一致时，结果为未变化；否则为已完成。
func actionResultFromDetail(detail Detail, ref Ref, action string, statusBefore string) ActionResult {
	statusAfter := detail.State
	result := actionResultCompleted
	if statusBefore != "" && statusBefore == statusAfter {
		result = actionResultUnchanged
	}
	return ActionResult{
		ID:           firstNonEmpty(detail.ID, ref.Value),
		Name:         firstContainerName(detail.Names),
		Image:        detail.Image,
		Action:       action,
		Result:       result,
		Runtime:      runtimeNameDocker,
		StatusBefore: statusBefore,
		StatusAfter:  statusAfter,
	}
}

func dockerState(inspect container.InspectResponse) (string, string, string) {
	if inspect.State == nil {
		return "unknown", "", ""
	}
	startedAt := parseDockerTimeString(inspect.State.StartedAt)
	return normalizeContainerState(string(inspect.State.Status)), strings.TrimSpace(string(inspect.State.Status)), startedAt
}

func dockerImage(inspect container.InspectResponse) string {
	if inspect.Config != nil && strings.TrimSpace(inspect.Config.Image) != "" {
		return strings.TrimSpace(inspect.Config.Image)
	}
	return strings.TrimSpace(inspect.Image)
}

func dockerLabels(inspect container.InspectResponse) map[string]string {
	if inspect.Config == nil {
		return nil
	}
	return cloneLabels(inspect.Config.Labels)
}

func dockerInspectPorts(inspect container.InspectResponse) []Port {
	if inspect.NetworkSettings == nil {
		return nil
	}
	ports := make([]Port, 0, len(inspect.NetworkSettings.Ports))
	for port, bindings := range inspect.NetworkSettings.Ports {
		privatePort, _ := strconv.Atoi(port.Port())
		if len(bindings) == 0 {
			ports = append(ports, Port{PrivatePort: privatePort, Type: string(port.Proto())})
			continue
		}
		for _, binding := range bindings {
			publicPort, _ := strconv.Atoi(binding.HostPort)
			ports = append(ports, Port{
				IP:          strings.TrimSpace(binding.HostIP.String()),
				PrivatePort: privatePort,
				PublicPort:  intPtr(publicPort),
				Type:        string(port.Proto()),
			})
		}
	}
	return ports
}

func dockerPorts(ports []container.PortSummary) []Port {
	mapped := make([]Port, 0, len(ports))
	for _, port := range ports {
		privatePort := int(port.PrivatePort)
		publicPort := int(port.PublicPort)
		item := Port{
			IP:          strings.TrimSpace(port.IP.String()),
			PrivatePort: privatePort,
			Type:        strings.TrimSpace(port.Type),
		}
		if publicPort > 0 {
			item.PublicPort = &publicPort
		}
		mapped = append(mapped, item)
	}
	return mapped
}

// dockerMounts converts Docker mount points to Mount structures, computing a stable mount identifier for each mount.
func dockerMounts(mounts []container.MountPoint) []Mount {
	mapped := make([]Mount, 0, len(mounts))
	for _, mount := range mounts {
		item := Mount{
			Type:        string(mount.Type),
			Name:        strings.TrimSpace(mount.Name),
			Source:      strings.TrimSpace(mount.Source),
			Destination: strings.TrimSpace(mount.Destination),
			Mode:        strings.TrimSpace(mount.Mode),
			ReadOnly:    !mount.RW,
		}
		item.ID = stableMountID(item)
		mapped = append(mapped, item)
	}
	return mapped
}

// findMountByID locates a mount by its stable ID.
// It returns the mount and true if found, false otherwise.
func findMountByID(mounts []Mount, mountID string) (Mount, bool) {
	mountID = strings.TrimSpace(mountID)
	for _, mount := range mounts {
		if mount.ID == mountID {
			return mount, true
		}
	}
	return Mount{}, false
}

// mountUsageSupported reports whether a mount supports usage measurement.
// It returns true if the mount type is bind or volume with a non-empty source.
func mountUsageSupported(mount Mount) bool {
	switch strings.TrimSpace(strings.ToLower(mount.Type)) {
	case "bind", "volume":
		return strings.TrimSpace(mount.Source) != ""
	default:
		return false
	}
}

// mountUsageFromMount constructs mount usage information from mount details and measurement metadata.
func mountUsageFromMount(containerID string, mount Mount, status string, size int64, measuredAt string) MountUsage {
	usage := MountUsage{
		MountID:     mount.ID,
		ContainerID: containerID,
		Type:        strings.TrimSpace(mount.Type),
		Name:        strings.TrimSpace(mount.Name),
		Source:      strings.TrimSpace(mount.Source),
		Destination: strings.TrimSpace(mount.Destination),
		Status:      firstNonEmpty(strings.TrimSpace(status), containerMountUsageStatusNotMeasured),
		MeasuredAt:  strings.TrimSpace(measuredAt),
	}
	if strings.TrimSpace(mount.Name) != "" {
		usage.SharedHint = "named volume may be shared by multiple containers"
	}
	if usage.Status == containerMountUsageStatusMeasured {
		usage.SizeBytes = size
		usage.SizeDisplay = formatIECBytes(size)
	}
	if usage.Status == containerMountUsageStatusUnsupported {
		usage.Message = "Mount usage is not supported for this mount."
	}
	return usage
}

// mountUsageFromScanError maps mount scan errors into mount usage information with
// appropriate status and message. The input error is translated to a MountUsage status
// and message rather than being returned as a Go error.
func mountUsageFromScanError(containerID string, mount Mount, err error) MountUsage {
	status := containerMountUsageStatusError
	message := "Mount usage measurement failed."
	switch {
	case errors.Is(err, errRuntimePermissionDenied):
		status = containerMountUsageStatusPermissionDenied
		message = "Permission denied while measuring mount usage."
	case errors.Is(err, errContainerMountNotFound):
		status = containerMountUsageStatusNotFound
		message = "Mount source was not found while measuring usage."
	case errors.Is(err, errContainerRuntimeTimeout):
		status = containerMountUsageStatusTimeout
		message = "Mount usage measurement timed out."
	}
	usage := mountUsageFromMount(containerID, mount, status, 0, "")
	usage.Message = message
	return usage
}

// dockerNetworks converts networks from a Docker container inspection response into the module's Network format.
func dockerNetworks(inspect container.InspectResponse) []Network {
	if inspect.NetworkSettings == nil || len(inspect.NetworkSettings.Networks) == 0 {
		return nil
	}
	networks := make([]Network, 0, len(inspect.NetworkSettings.Networks))
	for name, network := range inspect.NetworkSettings.Networks {
		if mapped, ok := dockerEndpointNetwork(name, network); ok {
			networks = append(networks, mapped)
		}
	}
	return networks
}

func dockerEndpointNetwork(name string, network *network.EndpointSettings) (Network, bool) {
	if network == nil {
		return Network{}, false
	}
	return Network{
		Name:       strings.TrimSpace(name),
		NetworkID:  strings.TrimSpace(network.NetworkID),
		EndpointID: strings.TrimSpace(network.EndpointID),
		Gateway:    strings.TrimSpace(network.Gateway.String()),
		IPAddress:  strings.TrimSpace(network.IPAddress.String()),
		MacAddress: strings.TrimSpace(network.MacAddress.String()),
	}, true
}

func cleanDockerNames(names []string) []string {
	cleaned := make([]string, 0, len(names))
	for _, name := range names {
		if trimmed := strings.TrimPrefix(strings.TrimSpace(name), "/"); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func firstContainerName(names []string) string {
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}

func normalizeContainerState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "created", "running", "paused", "restarting", "removing", "exited", "dead":
		return strings.ToLower(strings.TrimSpace(state))
	default:
		return "unknown"
	}
}

func parseDockerTimeString(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "0001-") {
		return ""
	}
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return timestamp.UTC().Format(time.RFC3339)
}

func safeEndpointLabel(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return runtimeNameDocker
	}
	if parsed.Scheme == dockerSocketScheme {
		return "unix://" + parsed.Path
	}
	return parsed.Scheme
}

func intPtr(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func intPtrAllowZero(value int) *int {
	return &value
}

type dockerClientAdapter struct {
	*mobyclient.Client
}

func (dockerClientAdapter) dockerSystemInfo() {}

type dockerClientSystemInfo struct {
	APIVersion        string
	ServerVersion     string
	OperatingSystem   string
	Architecture      string
	Containers        int
	ContainersRunning int
}

func (dockerClientSystemInfo) dockerSystemInfo() {}

func (d dockerClientAdapter) Info(ctx context.Context) (systemInfo, error) {
	info, err := d.Client.Info(ctx, mobyclient.InfoOptions{})
	if err != nil {
		return nil, err
	}
	return dockerClientSystemInfo{
		APIVersion:        d.ClientVersion(),
		ServerVersion:     info.Info.ServerVersion,
		OperatingSystem:   info.Info.OperatingSystem,
		Architecture:      info.Info.Architecture,
		Containers:        info.Info.Containers,
		ContainersRunning: info.Info.ContainersRunning,
	}, nil
}

func (d dockerClientAdapter) ContainerList(ctx context.Context, options mobyclient.ContainerListOptions) ([]container.Summary, error) {
	result, err := d.Client.ContainerList(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (d dockerClientAdapter) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	result, err := d.Client.ContainerInspect(ctx, containerID, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return result.Container, nil
}

func (d dockerClientAdapter) ContainerLogs(ctx context.Context, containerID string, options mobyclient.ContainerLogsOptions) (io.ReadCloser, error) {
	return d.Client.ContainerLogs(ctx, containerID, options)
}

func (d dockerClientAdapter) ContainerStatsOneShot(ctx context.Context, containerID string) (mobyclient.ContainerStatsResult, error) {
	return d.ContainerStats(ctx, containerID, mobyclient.ContainerStatsOptions{
		Stream:                false,
		IncludePreviousSample: true,
	})
}

func (d dockerClientAdapter) ContainerExecCreate(ctx context.Context, containerID string, options mobyclient.ExecCreateOptions) (mobyclient.ExecCreateResult, error) {
	return d.ExecCreate(ctx, containerID, options)
}

func (d dockerClientAdapter) ContainerExecAttach(ctx context.Context, execID string, config mobyclient.ExecAttachOptions) (mobyclient.HijackedResponse, error) {
	result, err := d.ExecAttach(ctx, execID, config)
	if err != nil {
		return mobyclient.HijackedResponse{}, err
	}
	return result.HijackedResponse, nil
}

func (d dockerClientAdapter) ContainerExecResize(ctx context.Context, execID string, options mobyclient.ExecResizeOptions) error {
	_, err := d.ExecResize(ctx, execID, options)
	return err
}

func (d dockerClientAdapter) ContainerStart(ctx context.Context, containerID string, options mobyclient.ContainerStartOptions) error {
	_, err := d.Client.ContainerStart(ctx, containerID, options)
	return err
}

func (d dockerClientAdapter) ContainerStop(ctx context.Context, containerID string, options mobyclient.ContainerStopOptions) error {
	_, err := d.Client.ContainerStop(ctx, containerID, options)
	return err
}

func (d dockerClientAdapter) ContainerRestart(ctx context.Context, containerID string, options mobyclient.ContainerRestartOptions) error {
	_, err := d.Client.ContainerRestart(ctx, containerID, options)
	return err
}

func (d dockerClientAdapter) ContainerRemove(ctx context.Context, containerID string, options mobyclient.ContainerRemoveOptions) error {
	_, err := d.Client.ContainerRemove(ctx, containerID, options)
	return err
}

var dockerErrorMessageRules = []struct {
	fragment string
	err      error
}{
	{fragment: "permission denied", err: errRuntimePermissionDenied},
	{fragment: "no such file", err: errRuntimeSocketMissing},
	{fragment: "cannot connect", err: errRuntimeDaemonUnavailable},
	{fragment: "connection refused", err: errRuntimeDaemonUnavailable},
	{fragment: "not found", err: errContainerNotFound},
	{fragment: "is already", err: errInvalidContainerState},
	{fragment: "not running", err: errInvalidContainerState},
}

func (r *DockerRuntime) String() string {
	return fmt.Sprintf("DockerRuntime(%s)", safeEndpointLabel(r.endpoint))
}

var _ Runtime = (*DockerRuntime)(nil)
