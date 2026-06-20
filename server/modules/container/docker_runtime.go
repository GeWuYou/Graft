// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

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
	mountUsageScanner mountUsageScanner
}

type dockerClient interface {
	Info(context.Context) (systemInfo, error)
	ContainerList(context.Context, container.ListOptions) ([]container.Summary, error)
	ContainerInspect(context.Context, string) (container.InspectResponse, error)
	ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error)
	ContainerStatsOneShot(context.Context, string) (container.StatsResponseReader, error)
	ContainerExecCreate(context.Context, string, container.ExecOptions) (container.ExecCreateResponse, error)
	ContainerExecAttach(context.Context, string, container.ExecAttachOptions) (dockertypes.HijackedResponse, error)
	ContainerExecResize(context.Context, string, container.ResizeOptions) error
	ContainerStart(context.Context, string, container.StartOptions) error
	ContainerStop(context.Context, string, container.StopOptions) error
	ContainerRestart(context.Context, string, container.StopOptions) error
	ContainerRemove(context.Context, string, container.RemoveOptions) error
	Close() error
}

type systemInfo interface {
	dockerSystemInfo()
}

// NewDockerRuntime creates the first local container runtime adapter.
func NewDockerRuntime(endpoint string) (*DockerRuntime, error) {
	endpoint = firstNonEmpty(endpoint, defaultContainerDockerEndpoint)
	cli, err := client.NewClientWithOpts(client.WithHost(endpoint), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, mapDockerError(err)
	}
	return &DockerRuntime{client: dockerClientAdapter{Client: cli}, endpoint: endpoint}, nil
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
	items, err := r.client.ContainerList(ctx, container.ListOptions{All: true})
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
	detail.Resource = r.containerResourceSummary(ctx, firstNonEmpty(detail.ID, ref.Value))
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
	reader, err := r.client.ContainerLogs(ctx, ref.Value, container.LogsOptions{
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

	lines, truncated, err := readDockerLogLines(reader, query.Tail)
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
		Lines:      lines,
		Tail:       query.Tail,
		Since:      query.Since,
		Timestamps: query.Timestamps,
		Stdout:     query.Stdout,
		Stderr:     query.Stderr,
		Truncated:  truncated,
	}, nil
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
	if err := r.client.ContainerStart(ctx, ref.Value, container.StartOptions{}); err != nil {
		return actionResultFromDetail(before, ref, containerActionStart, ""), mapDockerError(err)
	}
	after, _ := r.Detail(ctx, ref)
	return actionResultFromDetail(after, ref, containerActionStart, before.State), nil
}

// Stop stops one Docker container by id or name.
func (r *DockerRuntime) Stop(ctx context.Context, ref Ref) (ActionResult, error) {
	return r.runTimedStateAction(ctx, ref, containerActionStop, canStopState, r.client.ContainerStop)
}

// Restart restarts one Docker container by id or name.
func (r *DockerRuntime) Restart(ctx context.Context, ref Ref) (ActionResult, error) {
	return r.runTimedStateAction(ctx, ref, containerActionRestart, canRestartState, r.client.ContainerRestart)
}

func (r *DockerRuntime) runTimedStateAction(
	ctx context.Context,
	ref Ref,
	action string,
	allowed func(string) bool,
	run func(context.Context, string, container.StopOptions) error,
) (ActionResult, error) {
	before, _ := r.Detail(ctx, ref)
	if before.State != "" && !allowed(before.State) {
		return actionResultFromDetail(before, ref, action, before.State), errInvalidContainerState
	}
	timeout := 10
	if err := run(ctx, ref.Value, container.StopOptions{Timeout: &timeout}); err != nil {
		return actionResultFromDetail(before, ref, action, ""), mapDockerError(err)
	}
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
	if err := r.client.ContainerRemove(ctx, ref.Value, container.RemoveOptions{Force: options.Force}); err != nil {
		return actionResultFromDetail(before, ref, containerActionRemove, before.State), mapDockerError(err)
	}
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
	return dockerResourceSummary(stats)
}

func (r *DockerRuntime) collectListResourceSummaries(ctx context.Context, summaries []Summary) {
	if len(summaries) == 0 {
		return
	}
	workerCount := min(len(summaries), dockerStatsListWorkers)
	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for index := range jobs {
				summaries[index].Resource = r.containerResourceSummary(ctx, summaries[index].ID)
			}
		}()
	}
	for index := range summaries {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
}

func dockerResourceSummary(stats container.StatsResponse) ResourceSummary {
	resource := ResourceSummary{
		Available:      true,
		StatsAvailable: true,
	}
	if cpuPercent, ok := dockerCPUPercent(stats); ok {
		resource.CPUPercent = &cpuPercent
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

func dockerOnlineCPUs(stats container.StatsResponse) *int64 {
	if stats.CPUStats.OnlineCPUs == 0 {
		return nil
	}
	return uint32ToInt64Ptr(stats.CPUStats.OnlineCPUs)
}

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

func addUint64(total uint64, value uint64, overflow bool) (uint64, bool) {
	if overflow || total > ^uint64(0)-value {
		return 0, true
	}
	return total + value, false
}

func dockerCPUPercent(stats container.StatsResponse) (float64, bool) {
	if stats.CPUStats.CPUUsage.TotalUsage <= stats.PreCPUStats.CPUUsage.TotalUsage ||
		stats.CPUStats.SystemUsage <= stats.PreCPUStats.SystemUsage {
		return 0, false
	}
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		return 0, false
	}
	return (cpuDelta / systemDelta) * onlineCPUs * dockerStatsPercentScale, true
}

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

// dockerDetail builds a Detail struct from Docker container inspect output and runtime metadata.
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
	if inspect.ContainerJSONBase != nil {
		summary.RestartCount = intPtrAllowZero(inspect.RestartCount)
	}
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

func dockerOrchestratorFromLabels(labels map[string]string) OrchestratorInfo {
	labels = cloneLabels(labels)
	if len(labels) == 0 {
		return OrchestratorInfo{
			Type:       containerOrchestratorStandalone,
			Managed:    false,
			Confidence: orchestratorConfidenceHigh,
		}
	}

	typeCount := 0
	info := OrchestratorInfo{
		Type:       containerOrchestratorStandalone,
		Managed:    false,
		Confidence: orchestratorConfidenceHigh,
	}

	if metadata, ok := kubernetesMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorKubernetes
		info.Managed = true
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
		info.Stack = stack
		info.Task = task
		info.DisplayName = firstNonEmpty(stack, task, "swarm")
		info.Confidence = orchestratorConfidenceHigh
	}
	if project, service, ok := composeMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorCompose
		info.Managed = true
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
			Type:       containerOrchestratorUnknown,
			Managed:    true,
			Confidence: orchestratorConfidenceLow,
		}
	}
	return info
}

type kubernetesOrchestratorMetadata struct {
	Namespace string
	Pod       string
	Container string
}

func kubernetesMetadata(labels map[string]string) (kubernetesOrchestratorMetadata, bool) {
	metadata := kubernetesOrchestratorMetadata{
		Namespace: strings.TrimSpace(labels["io.kubernetes.pod.namespace"]),
		Pod:       strings.TrimSpace(labels["io.kubernetes.pod.name"]),
		Container: strings.TrimSpace(labels["io.kubernetes.container.name"]),
	}
	ok := metadata.Namespace != "" || metadata.Pod != "" || metadata.Container != ""
	return metadata, ok
}

func swarmMetadata(labels map[string]string) (stack string, task string, ok bool) {
	stack = strings.TrimSpace(labels["com.docker.stack.namespace"])
	task = strings.TrimSpace(labels["com.docker.swarm.task.name"])
	ok = stack != "" || task != ""
	return stack, task, ok
}

func composeMetadata(labels map[string]string) (project string, service string, ok bool) {
	project = strings.TrimSpace(labels[composeProjectLabel])
	service = strings.TrimSpace(labels[composeServiceLabel])
	ok = project != "" || service != ""
	return project, service, ok
}

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
	switch strings.TrimSpace(inspect.State.Health.Status) {
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

func mapDockerMessageError(message string) error {
	normalized := strings.ToLower(message)
	for _, rule := range dockerErrorMessageRules {
		if strings.Contains(normalized, rule.fragment) {
			return rule.err
		}
	}
	return errRuntimeDaemonUnavailable
}

func readDockerLogLines(reader io.Reader, tail int) ([]string, bool, error) {
	var output bytes.Buffer
	if _, err := stdcopy.StdCopy(&output, &output, reader); err != nil {
		return nil, false, err
	}
	limit := tail
	if limit > defaultContainerLogsMaxTail {
		limit = defaultContainerLogsMaxTail
	}
	scanner := bufio.NewScanner(&output)
	scanner.Buffer(make([]byte, 0, dockerLogScannerInitSize), dockerLogScannerMaxSize)
	lines := make([]string, 0)
	truncated := false
	for scanner.Scan() {
		line := scanner.Text()
		if limit <= 0 {
			truncated = true
			continue
		}
		if len(lines) == limit {
			truncated = true
			copy(lines, lines[1:])
			lines[limit-1] = line
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, false, err
	}
	return lines, truncated, nil
}

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
			ports = append(ports, Port{PrivatePort: privatePort, Type: port.Proto()})
			continue
		}
		for _, binding := range bindings {
			publicPort, _ := strconv.Atoi(binding.HostPort)
			ports = append(ports, Port{
				IP:          strings.TrimSpace(binding.HostIP),
				PrivatePort: privatePort,
				PublicPort:  intPtr(publicPort),
				Type:        port.Proto(),
			})
		}
	}
	return ports
}

func dockerPorts(ports []container.Port) []Port {
	mapped := make([]Port, 0, len(ports))
	for _, port := range ports {
		privatePort := int(port.PrivatePort)
		publicPort := int(port.PublicPort)
		item := Port{
			IP:          strings.TrimSpace(port.IP),
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
		Gateway:    strings.TrimSpace(network.Gateway),
		IPAddress:  strings.TrimSpace(network.IPAddress),
		MacAddress: strings.TrimSpace(network.MacAddress),
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
	*client.Client
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
	info, err := d.Client.Info(ctx)
	if err != nil {
		return nil, err
	}
	return dockerClientSystemInfo{
		APIVersion:        d.ClientVersion(),
		ServerVersion:     info.ServerVersion,
		OperatingSystem:   info.OperatingSystem,
		Architecture:      info.Architecture,
		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
	}, nil
}

func (d dockerClientAdapter) ContainerExecCreate(ctx context.Context, containerID string, options container.ExecOptions) (container.ExecCreateResponse, error) {
	return d.Client.ContainerExecCreate(ctx, containerID, options)
}

func (d dockerClientAdapter) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (dockertypes.HijackedResponse, error) {
	return d.Client.ContainerExecAttach(ctx, execID, config)
}

func (d dockerClientAdapter) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	return d.Client.ContainerExecResize(ctx, execID, options)
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
