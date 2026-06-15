// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
)

func TestReadDockerLogLinesDoesNotReportExactTailAsTruncated(t *testing.T) {
	t.Parallel()

	lines, truncated, err := readDockerLogLines(dockerLogStream(t, "one\n", "two\n"), 2)
	if err != nil {
		t.Fatalf("read log lines: %v", err)
	}
	if truncated {
		t.Fatalf("expected exactly tail lines to avoid truncation")
	}
	if !reflect.DeepEqual(lines, []string{"one", "two"}) {
		t.Fatalf("unexpected lines %#v", lines)
	}
}

func TestReadDockerLogLinesTruncatesOnlyAfterDiscardingLines(t *testing.T) {
	t.Parallel()

	lines, truncated, err := readDockerLogLines(dockerLogStream(t, "one\n", "two\n", "three\n"), 2)
	if err != nil {
		t.Fatalf("read log lines: %v", err)
	}
	if !truncated {
		t.Fatalf("expected more than tail lines to report truncation")
	}
	if !reflect.DeepEqual(lines, []string{"two", "three"}) {
		t.Fatalf("unexpected lines %#v", lines)
	}
}

func TestReadDockerLogLinesAvoidsUserSizedPreallocation(t *testing.T) {
	t.Parallel()

	const excessiveTail = int(^uint(0) >> 1)
	lines, truncated, err := readDockerLogLines(dockerLogStream(t, "one\n"), excessiveTail)
	if err != nil {
		t.Fatalf("read log lines: %v", err)
	}
	if truncated {
		t.Fatalf("expected one line to avoid truncation")
	}
	if !reflect.DeepEqual(lines, []string{"one"}) {
		t.Fatalf("unexpected lines %#v", lines)
	}
}

func TestDockerRuntimeLogsAvoidsRuntimeInfoCall(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		logReader: dockerLogReadCloser(t, "one\n", "two\n"),
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:   "full-id",
				Name: "/web",
			},
		},
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	logs, err := runtime.Logs(context.Background(), Ref{Value: "web"}, LogQuery{
		Tail:   2,
		Stdout: true,
	})
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if logs.ID != "full-id" || logs.Name != "web" {
		t.Fatalf("unexpected log metadata %#v", logs)
	}
	if calls := client.infoCalls.Load(); calls != 0 {
		t.Fatalf("expected logs to avoid Info calls, got %d", calls)
	}
	if calls := client.inspectCalls.Load(); calls != 1 {
		t.Fatalf("expected one inspect call for log metadata, got %d", calls)
	}
	if calls := client.logCalls.Load(); calls != 1 {
		t.Fatalf("expected one log call, got %d", calls)
	}
}

func TestDockerRuntimeLogsReturnsInvalidLogQueryError(t *testing.T) {
	t.Parallel()

	runtime := &DockerRuntime{client: &countingDockerClient{}, endpoint: "unix:///var/run/docker.sock"}

	_, err := runtime.Logs(context.Background(), Ref{Value: "web"}, LogQuery{
		Tail:   1,
		Since:  "not-a-time",
		Stdout: true,
	})
	if !errors.Is(err, errInvalidLogQuery) {
		t.Fatalf("expected invalid log query error, got %v", err)
	}
}

func TestDockerRuntimeListUsesCheapSummaryFields(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		list: []container.Summary{
			{
				ID:      "1234567890abcdef",
				Names:   []string{"/graft-web"},
				Image:   "graft/web:latest",
				ImageID: "sha256:web",
				Labels: map[string]string{
					composeProjectLabel: "graft",
					composeServiceLabel: "web",
				},
				Ports:  []container.Port{{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"}},
				State:  container.StateRunning,
				Status: "Up 10 minutes",
				NetworkSettings: &container.NetworkSettingsSummary{
					Networks: map[string]*network.EndpointSettings{
						"bridge": {
							NetworkID:  "net-1",
							EndpointID: "endpoint-1",
							Gateway:    "172.18.0.1",
							IPAddress:  "172.18.0.2",
							MacAddress: "02:42:ac:12:00:02",
						},
					},
				},
				Created: 1781409600,
			},
		},
		stats: container.StatsResponse{
			CPUStats: container.CPUStats{
				CPUUsage:    container.CPUUsage{TotalUsage: 200, PercpuUsage: []uint64{100, 100}},
				SystemUsage: 1000,
				OnlineCPUs:  2,
			},
			PreCPUStats: container.CPUStats{
				CPUUsage:    container.CPUUsage{TotalUsage: 100},
				SystemUsage: 500,
			},
			MemoryStats: container.MemoryStats{Usage: 256, Limit: 1024},
		},
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	items, err := runtime.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %#v", items)
	}
	item := items[0]
	assertListIdentity(t, item, "1234567890ab", "graft-web")
	assertListNetwork(t, item, "172.18.0.2", "bridge")
	if item.Health != containerHealthUnavailable || !item.Resource.Available || !item.Resource.StatsAvailable {
		t.Fatalf("expected available resource stats semantics, got %#v", item)
	}
	assertFloatPtr(t, item.Resource.CPUPercent, 40, "computed CPU percent")
	assertInt64Ptr(t, item.Resource.MemoryUsageBytes, 256, "memory usage bytes")
	assertInt64Ptr(t, item.Resource.MemoryLimitBytes, 1024, "memory limit bytes")
	assertFloatPtr(t, item.Resource.MemoryPercent, 25, "computed memory percent")
	assertListCompose(t, item, "graft", "web")
	assertListActions(t, item, false, true, true)
	if calls := client.inspectCalls.Load(); calls != 0 {
		t.Fatalf("expected list to avoid inspect calls, got %d", calls)
	}
	if calls := client.statsCalls.Load(); calls != 1 {
		t.Fatalf("expected list to collect stats once, got %d", calls)
	}
}

func TestDockerRuntimeListDegradesWhenStatsUnavailable(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		list: []container.Summary{
			{
				ID:      "abcdef1234567890",
				Names:   []string{"/graft-api"},
				Image:   "graft/api:latest",
				State:   container.StateRunning,
				Status:  "Up 5 minutes",
				Created: 1781409600,
			},
		},
		statsErr: timeoutError{},
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	items, err := runtime.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %#v", items)
	}
	resource := items[0].Resource
	if resource.Available || resource.StatsAvailable {
		t.Fatalf("expected unavailable resource stats, got %#v", resource)
	}
	if resource.UnavailableReason != containerStatsTimeoutReason || resource.StatsErrorKey != containerStatsTimeoutReason {
		t.Fatalf("expected sanitized timeout reason, got %#v", resource)
	}
	if resource.StatsErrorMessage == "" || resource.StatsErrorMessage == "i/o timeout" {
		t.Fatalf("expected sanitized stats error message, got %#v", resource)
	}
	if resource.CPUPercent != nil || resource.MemoryUsageBytes != nil || resource.MemoryLimitBytes != nil || resource.MemoryPercent != nil {
		t.Fatalf("expected no partial stats on failure, got %#v", resource)
	}
	if calls := client.statsCalls.Load(); calls != 1 {
		t.Fatalf("expected list to attempt stats once, got %d", calls)
	}
}

func TestDockerRuntimeRemoveRejectsRunningWithoutForce(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:    "abc123",
				Name:  "/web",
				State: &container.State{Status: container.StateRunning},
			},
			Config: &container.Config{Image: "nginx:latest"},
		},
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	result, err := runtime.Remove(context.Background(), Ref{Value: "web"}, RemoveOptions{Force: false})
	if !errors.Is(err, errInvalidContainerState) {
		t.Fatalf("expected invalid state, got %v", err)
	}
	if result.StatusBefore != "running" || result.StatusAfter != "running" {
		t.Fatalf("expected running status context, got %#v", result)
	}
	if calls := client.removeCalls.Load(); calls != 0 {
		t.Fatalf("expected remove not to be called, got %d", calls)
	}
}

func TestDockerRuntimeRemoveForceCallsDockerRemove(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:    "abc123",
				Name:  "/web",
				State: &container.State{Status: container.StateRunning},
			},
			Config: &container.Config{Image: "nginx:latest"},
		},
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	result, err := runtime.Remove(context.Background(), Ref{Value: "web"}, RemoveOptions{Force: true})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !client.removeForce.Load() {
		t.Fatalf("expected force remove option")
	}
	if result.Action != containerActionRemove || result.StatusBefore != "running" || result.StatusAfter != actionStatusRemoved {
		t.Fatalf("unexpected remove result %#v", result)
	}
	if calls := client.removeCalls.Load(); calls != 1 {
		t.Fatalf("expected one remove call, got %d", calls)
	}
}

func dockerLogStream(t *testing.T, chunks ...string) io.Reader {
	t.Helper()

	var output bytes.Buffer
	writer := stdcopy.NewStdWriter(&output, stdcopy.Stdout)
	for _, chunk := range chunks {
		if _, err := writer.Write([]byte(chunk)); err != nil {
			t.Fatalf("write log chunk: %v", err)
		}
	}
	return bytes.NewReader(output.Bytes())
}

func dockerLogReadCloser(t *testing.T, chunks ...string) io.ReadCloser {
	t.Helper()

	return io.NopCloser(dockerLogStream(t, chunks...))
}

func assertListIdentity(t *testing.T, item Summary, shortID string, name string) {
	t.Helper()

	if item.ShortID != shortID || item.Name != name {
		t.Fatalf("unexpected identity fields %#v", item)
	}
}

func assertListNetwork(t *testing.T, item Summary, primaryIP string, networkSummary string) {
	t.Helper()

	if item.PrimaryIP != primaryIP || item.NetworkSummary != networkSummary {
		t.Fatalf("unexpected network fields %#v", item)
	}
}

func assertListCompose(t *testing.T, item Summary, project string, service string) {
	t.Helper()

	if item.ComposeProject != project || item.ComposeService != service {
		t.Fatalf("unexpected compose fields %#v", item)
	}
}

func assertListActions(t *testing.T, item Summary, canStart bool, canStop bool, canRestart bool) {
	t.Helper()

	if item.CanStart != canStart || item.CanStop != canStop || item.CanRestart != canRestart {
		t.Fatalf("unexpected action availability %#v", item)
	}
}

func assertFloatPtr(t *testing.T, actual *float64, expected float64, label string) {
	t.Helper()

	if actual == nil || math.Abs(*actual-expected) > 0.0001 {
		t.Fatalf("expected %s %v, got %#v", label, expected, actual)
	}
}

func assertInt64Ptr(t *testing.T, actual *int64, expected int64, label string) {
	t.Helper()

	if actual == nil || *actual != expected {
		t.Fatalf("expected %s %v, got %#v", label, expected, actual)
	}
}

type countingDockerClient struct {
	infoCalls    atomic.Int64
	inspectCalls atomic.Int64
	logCalls     atomic.Int64
	listCalls    atomic.Int64
	statsCalls   atomic.Int64
	removeCalls  atomic.Int64
	removeForce  atomic.Bool
	logReader    io.ReadCloser
	inspect      container.InspectResponse
	list         []container.Summary
	stats        container.StatsResponse
	statsErr     error
}

func (c *countingDockerClient) Info(context.Context) (systemInfo, error) {
	c.infoCalls.Add(1)
	return dockerClientSystemInfo{}, nil
}

func (c *countingDockerClient) ContainerList(context.Context, container.ListOptions) ([]container.Summary, error) {
	c.listCalls.Add(1)
	return c.list, nil
}

func (c *countingDockerClient) ContainerInspect(context.Context, string) (container.InspectResponse, error) {
	c.inspectCalls.Add(1)
	return c.inspect, nil
}

func (c *countingDockerClient) ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error) {
	c.logCalls.Add(1)
	if c.logReader == nil {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	return c.logReader, nil
}

func (c *countingDockerClient) ContainerStatsOneShot(context.Context, string) (container.StatsResponseReader, error) {
	c.statsCalls.Add(1)
	if c.statsErr != nil {
		return container.StatsResponseReader{}, c.statsErr
	}
	var output bytes.Buffer
	if err := json.NewEncoder(&output).Encode(c.stats); err != nil {
		return container.StatsResponseReader{}, err
	}
	return container.StatsResponseReader{Body: io.NopCloser(&output)}, nil
}

func (c *countingDockerClient) ContainerStart(context.Context, string, container.StartOptions) error {
	return nil
}

func (c *countingDockerClient) ContainerStop(context.Context, string, container.StopOptions) error {
	return nil
}

func (c *countingDockerClient) ContainerRestart(context.Context, string, container.StopOptions) error {
	return nil
}

func (c *countingDockerClient) ContainerRemove(_ context.Context, _ string, options container.RemoveOptions) error {
	c.removeCalls.Add(1)
	c.removeForce.Store(options.Force)
	return nil
}

func (c *countingDockerClient) Close() error {
	return nil
}

type timeoutError struct{}

func (timeoutError) Error() string {
	return "i/o timeout"
}

func (timeoutError) Timeout() bool {
	return true
}

func (timeoutError) Temporary() bool {
	return true
}

var _ net.Error = timeoutError{}
