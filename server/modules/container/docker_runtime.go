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

// DockerRuntime 将官方 Docker SDK 适配到容器模块边界，并在返回前完成字段脱敏、统计缓存和错误映射。
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
	Close() error
}

type systemInfo interface {
	dockerSystemInfo()
}

// NewDockerRuntime 创建 Docker 容器运行时适配器。staleWindow 指定统计刷新失败或进行中时仍可返回旧快照的窗口。
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

// Info 读取 Docker 运行时元数据并返回可用于 API 响应的脱敏投影。
func (r *DockerRuntime) Info(ctx context.Context) (RuntimeInfo, error) {
	info, err := r.client.Info(ctx)
	if err != nil {
		return RuntimeInfo{}, mapDockerError(err)
	}
	return dockerInfoToRuntimeInfo(info, safeEndpointLabel(r.endpoint)), nil
}

// List 读取 Docker 容器概要；它不执行原始 inspect，也不把日志或环境变量带入列表响应。
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

// Detail 将 Docker inspect 转换为脱敏详情；环境变量由策略层决定展示形式，原始敏感字段不得越过模块边界。
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

// Mounts 从 Docker inspect 提取脱敏挂载元数据，不接受调用方任意指定主机路径。
func (r *DockerRuntime) Mounts(ctx context.Context, ref Ref) ([]Mount, error) {
	inspect, err := r.client.ContainerInspect(ctx, ref.Value)
	if err != nil {
		return nil, mapDockerError(err)
	}
	return dockerMounts(inspect.Mounts), nil
}

// MountUsage 只测量当前 inspect 中匹配到的挂载源，拒绝任意路径输入以避免把运行时接口变成主机文件探测入口。
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

// Logs 按模块的尾部上限和时间条件读取 Docker 日志，并将 Docker 帧转换为规范化日志条目。
func (r *DockerRuntime) Logs(ctx context.Context, ref Ref, query LogQuery) (Logs, error) {
	since, err := parseLogSince(query.Since)
	if err != nil {
		return Logs{}, fmt.Errorf("%w: %w", errInvalidLogQuery, err)
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

// StreamLogs 跟随 Docker 增量日志流并逐行发出规范化消息；调用方取消上下文或 Docker 流结束时返回。
func (r *DockerRuntime) StreamLogs(ctx context.Context, ref Ref, query LogQuery, emit func(LogChunk) error) error {
	if emit == nil {
		return errors.New("container log stream emitter is required")
	}
	since, err := parseLogSince(query.Since)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidLogQuery, err)
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

// Close releases the Docker SDK client resources.
func (r *DockerRuntime) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}
