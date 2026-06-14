// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	dockerSocketScheme       = "unix"
	dockerLogScannerInitSize = 64 * 1024
	dockerLogScannerMaxSize  = 1024 * 1024
)

var errInvalidLogQuery = errors.New("invalid log query parameter")

// DockerRuntime adapts the official Docker SDK to the container module runtime boundary.
type DockerRuntime struct {
	client   dockerClient
	endpoint string
}

type dockerClient interface {
	Info(context.Context) (systemInfo, error)
	ContainerList(context.Context, container.ListOptions) ([]container.Summary, error)
	ContainerInspect(context.Context, string) (container.InspectResponse, error)
	ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error)
	ContainerStart(context.Context, string, container.StartOptions) error
	ContainerStop(context.Context, string, container.StopOptions) error
	ContainerRestart(context.Context, string, container.StopOptions) error
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

// List returns Docker container summaries without raw inspect payloads.
func (r *DockerRuntime) List(ctx context.Context, _ ListQuery) ([]Summary, error) {
	items, err := r.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, mapDockerError(err)
	}
	summaries := make([]Summary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, dockerSummary(item))
	}
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
	return dockerDetail(inspect, info), nil
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

// Start starts one Docker container by id or name.
func (r *DockerRuntime) Start(ctx context.Context, ref Ref) (ActionResult, error) {
	before, _ := r.Detail(ctx, ref)
	if err := r.client.ContainerStart(ctx, ref.Value, container.StartOptions{}); err != nil {
		return actionResultFromDetail(before, ref, containerActionStart, ""), mapDockerError(err)
	}
	after, _ := r.Detail(ctx, ref)
	return actionResultFromDetail(after, ref, containerActionStart, before.State), nil
}

// Stop stops one Docker container by id or name.
func (r *DockerRuntime) Stop(ctx context.Context, ref Ref) (ActionResult, error) {
	before, _ := r.Detail(ctx, ref)
	timeout := 10
	if err := r.client.ContainerStop(ctx, ref.Value, container.StopOptions{Timeout: &timeout}); err != nil {
		return actionResultFromDetail(before, ref, containerActionStop, ""), mapDockerError(err)
	}
	after, _ := r.Detail(ctx, ref)
	return actionResultFromDetail(after, ref, containerActionStop, before.State), nil
}

// Restart restarts one Docker container by id or name.
func (r *DockerRuntime) Restart(ctx context.Context, ref Ref) (ActionResult, error) {
	before, _ := r.Detail(ctx, ref)
	timeout := 10
	if err := r.client.ContainerRestart(ctx, ref.Value, container.StopOptions{Timeout: &timeout}); err != nil {
		return actionResultFromDetail(before, ref, containerActionRestart, ""), mapDockerError(err)
	}
	after, _ := r.Detail(ctx, ref)
	return actionResultFromDetail(after, ref, containerActionRestart, before.State), nil
}

// Close releases the Docker SDK client resources.
func (r *DockerRuntime) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func dockerSummary(item container.Summary) Summary {
	return Summary{
		ID:        strings.TrimSpace(item.ID),
		Names:     cleanDockerNames(item.Names),
		Image:     strings.TrimSpace(item.Image),
		ImageID:   strings.TrimSpace(item.ImageID),
		Labels:    cloneLabels(item.Labels),
		Ports:     dockerPorts(item.Ports),
		Runtime:   runtimeNameDocker,
		CreatedAt: time.Unix(item.Created, 0).UTC().Format(time.RFC3339),
		State:     normalizeContainerState(string(item.State)),
		Status:    strings.TrimSpace(item.Status),
	}
}

func dockerDetail(inspect container.InspectResponse, info RuntimeInfo) Detail {
	state, status, startedAt := dockerState(inspect)
	summary := Summary{
		ID:        strings.TrimSpace(inspect.ID),
		Names:     []string{strings.TrimPrefix(strings.TrimSpace(inspect.Name), "/")},
		Image:     dockerImage(inspect),
		ImageID:   strings.TrimSpace(inspect.Image),
		Labels:    dockerLabels(inspect),
		Ports:     dockerInspectPorts(inspect),
		Runtime:   runtimeNameDocker,
		CreatedAt: parseDockerTimeString(inspect.Created),
		StartedAt: startedAt,
		State:     state,
		Status:    status,
	}
	detail := Detail{
		Summary:          summary,
		Mounts:           dockerMounts(inspect.Mounts),
		Networks:         dockerNetworks(inspect),
		RuntimeInfo:      info,
		InspectUpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if inspect.Config != nil {
		detail.Command = []string(inspect.Config.Cmd)
		detail.Entrypoint = []string(inspect.Config.Entrypoint)
		detail.WorkingDir = strings.TrimSpace(inspect.Config.WorkingDir)
	}
	if inspect.HostConfig != nil {
		detail.RestartPolicy = string(inspect.HostConfig.RestartPolicy.Name)
	}
	return detail
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

func dockerMounts(mounts []container.MountPoint) []Mount {
	mapped := make([]Mount, 0, len(mounts))
	for _, mount := range mounts {
		mapped = append(mapped, Mount{
			Type:        string(mount.Type),
			Name:        strings.TrimSpace(mount.Name),
			Source:      strings.TrimSpace(mount.Source),
			Destination: strings.TrimSpace(mount.Destination),
			Mode:        strings.TrimSpace(mount.Mode),
			ReadOnly:    !mount.RW,
		})
	}
	return mapped
}

func dockerNetworks(inspect container.InspectResponse) []Network {
	if inspect.NetworkSettings == nil || len(inspect.NetworkSettings.Networks) == 0 {
		return nil
	}
	networks := make([]Network, 0, len(inspect.NetworkSettings.Networks))
	for name, network := range inspect.NetworkSettings.Networks {
		if network == nil {
			continue
		}
		networks = append(networks, Network{
			Name:       strings.TrimSpace(name),
			NetworkID:  strings.TrimSpace(network.NetworkID),
			EndpointID: strings.TrimSpace(network.EndpointID),
			Gateway:    strings.TrimSpace(network.Gateway),
			IPAddress:  strings.TrimSpace(network.IPAddress),
			MacAddress: strings.TrimSpace(network.MacAddress),
		})
	}
	return networks
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
