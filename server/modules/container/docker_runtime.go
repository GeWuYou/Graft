package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	mobyclient "github.com/moby/moby/client"
	"go.uber.org/zap"

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

// StreamRuntimeEvents follows Docker daemon events and emits canonical container runtime event candidates.
func (r *DockerRuntime) StreamRuntimeEvents(ctx context.Context, emit func(RuntimeEventCandidate) error) error {
	if emit == nil {
		return errors.New("container runtime event emitter is required")
	}
	eventClient, ok := any(r.client).(interface {
		Events(context.Context, mobyclient.EventsListOptions) mobyclient.EventsResult
	})
	if !ok {
		return errRuntimeEventHistoryUnavailable
	}
	result := eventClient.Events(ctx, mobyclient.EventsListOptions{Filters: dockerRuntimeEventFilters()})

	for {
		done, err := consumeDockerRuntimeEvents(ctx, &result, emit)
		if done || err != nil {
			return err
		}
	}
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
