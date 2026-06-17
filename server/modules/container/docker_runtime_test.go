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
	"time"

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
		stats: richDockerStatsFixture(),
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
	assertRichDockerResourceStats(t, item.Resource)
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

func TestDockerRuntimeListCollectsStatsWithBoundedConcurrency(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		stats: container.StatsResponse{
			MemoryStats: container.MemoryStats{Usage: 64, Limit: 128},
		},
		statsDelay: 40 * time.Millisecond,
	}
	for _, id := range []string{"one123456789", "two123456789", "three12345678", "four123456789"} {
		client.list = append(client.list, container.Summary{
			ID:      id,
			Names:   []string{"/" + id},
			Image:   "nginx:latest",
			State:   container.StateRunning,
			Status:  "Up",
			Created: 1781409600,
		})
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	items, err := runtime.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != len(client.list) {
		t.Fatalf("expected %d items, got %#v", len(client.list), items)
	}
	if calls := client.statsCalls.Load(); calls != int64(len(client.list)) {
		t.Fatalf("expected one stats call per item, got %d", calls)
	}
	if maxConcurrent := client.maxConcurrentStats.Load(); maxConcurrent < 2 {
		t.Fatalf("expected concurrent stats collection, got max concurrency %d", maxConcurrent)
	} else if bound := int64(min(len(client.list), dockerStatsListWorkers)); maxConcurrent > bound {
		t.Fatalf("expected stats concurrency bounded by %d, got max concurrency %d", bound, maxConcurrent)
	}
	for index, item := range items {
		if item.ID != client.list[index].ID {
			t.Fatalf("expected stable list order, got item %d as %#v", index, item)
		}
		assertInt64Ptr(t, item.Resource.MemoryUsageBytes, 64, "memory usage bytes")
		assertInt64Ptr(t, item.Resource.MemoryLimitBytes, 128, "memory limit bytes")
	}
}

func TestDockerRuntimeDetailCollectsResourceStats(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:      "1234567890abcdef",
				Name:    "/graft-web",
				State:   &container.State{Status: container.StateRunning},
				Created: "2026-06-14T00:00:00Z",
			},
			Config: &container.Config{Image: "graft/web:latest"},
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

	detail, err := runtime.Detail(context.Background(), Ref{Value: "graft-web"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}

	if !detail.Resource.Available || !detail.Resource.StatsAvailable {
		t.Fatalf("expected available detail resource stats, got %#v", detail.Resource)
	}
	assertFloatPtr(t, detail.Resource.CPUPercent, 40, "detail CPU percent")
	assertInt64Ptr(t, detail.Resource.MemoryUsageBytes, 256, "detail memory usage bytes")
	assertInt64Ptr(t, detail.Resource.MemoryLimitBytes, 1024, "detail memory limit bytes")
	assertFloatPtr(t, detail.Resource.MemoryPercent, 25, "detail memory percent")
	if calls := client.inspectCalls.Load(); calls != 1 {
		t.Fatalf("expected detail to inspect once, got %d", calls)
	}
	if calls := client.statsCalls.Load(); calls != 1 {
		t.Fatalf("expected detail to collect stats once, got %d", calls)
	}
}

func TestDockerRuntimeRejectsActionsWhenKnownStateDisallowsThem(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		state  container.ContainerState
		action func(context.Context, *DockerRuntime) (ActionResult, error)
		calls  func(*countingDockerClient) int64
	}{
		{
			name:  "start running",
			state: container.StateRunning,
			action: func(ctx context.Context, runtime *DockerRuntime) (ActionResult, error) {
				return runtime.Start(ctx, Ref{Value: "web"})
			},
			calls: func(client *countingDockerClient) int64 {
				return client.startCalls.Load()
			},
		},
		{
			name:  "start paused",
			state: container.StatePaused,
			action: func(ctx context.Context, runtime *DockerRuntime) (ActionResult, error) {
				return runtime.Start(ctx, Ref{Value: "web"})
			},
			calls: func(client *countingDockerClient) int64 {
				return client.startCalls.Load()
			},
		},
		{
			name:  "stop exited",
			state: container.StateExited,
			action: func(ctx context.Context, runtime *DockerRuntime) (ActionResult, error) {
				return runtime.Stop(ctx, Ref{Value: "web"})
			},
			calls: func(client *countingDockerClient) int64 {
				return client.stopCalls.Load()
			},
		},
		{
			name:  "restart dead",
			state: container.StateDead,
			action: func(ctx context.Context, runtime *DockerRuntime) (ActionResult, error) {
				return runtime.Restart(ctx, Ref{Value: "web"})
			},
			calls: func(client *countingDockerClient) int64 {
				return client.restartCalls.Load()
			},
		},
		{
			name:  "remove running without force",
			state: container.StateRunning,
			action: func(ctx context.Context, runtime *DockerRuntime) (ActionResult, error) {
				return runtime.Remove(ctx, Ref{Value: "web"}, RemoveOptions{Force: false})
			},
			calls: func(client *countingDockerClient) int64 {
				return client.removeCalls.Load()
			},
		},
		{
			name:  "remove paused without force",
			state: container.StatePaused,
			action: func(ctx context.Context, runtime *DockerRuntime) (ActionResult, error) {
				return runtime.Remove(ctx, Ref{Value: "web"}, RemoveOptions{Force: false})
			},
			calls: func(client *countingDockerClient) int64 {
				return client.removeCalls.Load()
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &countingDockerClient{
				inspect: container.InspectResponse{
					ContainerJSONBase: &container.ContainerJSONBase{
						ID:    "abc123",
						Name:  "/web",
						State: &container.State{Status: tc.state},
					},
					Config: &container.Config{Image: "nginx:latest"},
				},
			}
			runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

			result, err := tc.action(context.Background(), runtime)
			if !errors.Is(err, errInvalidContainerState) {
				t.Fatalf("expected invalid state, got %v", err)
			}
			if result.StatusBefore == "" || result.StatusAfter != result.StatusBefore {
				t.Fatalf("expected status context, got %#v", result)
			}
			if calls := tc.calls(client); calls != 0 {
				t.Fatalf("expected runtime action not to be called, got %d", calls)
			}
		})
	}
}

func TestDockerResourceSummaryKeepsZeroMemoryValues(t *testing.T) {
	t.Parallel()

	resource := dockerResourceSummary(container.StatsResponse{
		MemoryStats: container.MemoryStats{Usage: 0, Limit: 0},
	})
	if !resource.Available || !resource.StatsAvailable {
		t.Fatalf("expected zero memory values to count as available stats, got %#v", resource)
	}
	assertInt64Ptr(t, resource.MemoryUsageBytes, 0, "memory usage bytes")
	assertInt64Ptr(t, resource.MemoryLimitBytes, 0, "memory limit bytes")
	if resource.MemoryPercent != nil {
		t.Fatalf("expected zero limit to skip memory percent, got %#v", resource.MemoryPercent)
	}
}

func TestDockerResourceSummarySkipsUnknownOnlineCPUs(t *testing.T) {
	t.Parallel()

	stats := richDockerStatsFixture()
	stats.CPUStats.OnlineCPUs = 0

	resource := dockerResourceSummary(stats)

	if !resource.Available || !resource.StatsAvailable {
		t.Fatalf("expected per-CPU stats to keep resource available, got %#v", resource)
	}
	assertFloatPtr(t, resource.CPUPercent, 40, "computed CPU percent")
	if resource.OnlineCPUs != nil {
		t.Fatalf("expected unknown online CPUs to stay absent, got %#v", resource.OnlineCPUs)
	}
}

func TestDockerResourceSummarySkipsOverflowedNetworkTotals(t *testing.T) {
	t.Parallel()

	resource := dockerResourceSummary(container.StatsResponse{
		MemoryStats: container.MemoryStats{Usage: 1, Limit: 2},
		Networks: map[string]container.NetworkStats{
			"one": {RxBytes: ^uint64(0), TxBytes: 10},
			"two": {RxBytes: 1, TxBytes: 20},
		},
	})

	if !resource.Available || !resource.StatsAvailable {
		t.Fatalf("expected memory stats to keep resource available, got %#v", resource)
	}
	if resource.RxBytes != nil {
		t.Fatalf("expected overflowed rx bytes to stay absent, got %#v", resource.RxBytes)
	}
	assertInt64Ptr(t, resource.TxBytes, 30, "aggregated tx bytes")
}

func TestDockerDetailParsesEnvironmentVariables(t *testing.T) {
	t.Parallel()

	inspect := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:      "abc123",
			Name:    "/web",
			State:   &container.State{Status: container.StateRunning},
			Created: "2026-06-14T00:00:00Z",
		},
		Config: &container.Config{
			Image: "nginx:latest",
			Env: []string{
				"APP_ENV=prod",
				"PASSWORD=s3cr3t",
				"EMPTY=",
				"malformed",
			},
		},
	}

	detail := dockerDetail(inspect, RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled"})
	if len(detail.Environment) != 3 {
		t.Fatalf("expected three parsed env entries, got %#v", detail.Environment)
	}
	if detail.Environment[0].Key != "APP_ENV" || detail.Environment[0].Value != "prod" || detail.Environment[0].Sensitive {
		t.Fatalf("unexpected non-sensitive env entry %#v", detail.Environment[0])
	}
	if detail.Environment[1].Key != "PASSWORD" || detail.Environment[1].Value != "s3cr3t" || !detail.Environment[1].Sensitive {
		t.Fatalf("unexpected sensitive env entry %#v", detail.Environment[1])
	}
	if detail.Environment[2].Key != "EMPTY" || detail.Environment[2].Value != "" {
		t.Fatalf("unexpected empty env entry %#v", detail.Environment[2])
	}
	for _, item := range detail.Environment {
		if item.Source != dockerEnvironmentSource {
			t.Fatalf("expected docker env source, got %#v", item)
		}
	}
}

func TestDockerDetailMapsHealthcheckAndRuntimeStability(t *testing.T) {
	t.Parallel()

	checkedAt := time.Date(2026, 6, 17, 1, 31, 53, 0, time.UTC)
	inspect := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:   "abc123",
			Name: "/web",
			State: &container.State{
				Status:    container.StateRunning,
				ExitCode:  137,
				OOMKilled: true,
				Health: &container.Health{
					Status:        container.Unhealthy,
					FailingStreak: 2,
					Log: []*container.HealthcheckResult{
						{
							End:      checkedAt,
							ExitCode: 1,
							Output:   "curl failed\n",
						},
					},
				},
			},
			Created: "2026-06-14T00:00:00Z",
		},
		Config: &container.Config{
			Image: "nginx:latest",
			Healthcheck: &container.HealthConfig{
				Test: []string{"CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"},
			},
		},
	}

	detail := dockerDetail(inspect, RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled"})

	if detail.Health != containerHealthUnhealthy {
		t.Fatalf("expected unhealthy summary health, got %q", detail.Health)
	}
	if detail.Healthcheck == nil {
		t.Fatalf("expected mapped healthcheck")
	}
	if !detail.Healthcheck.Configured || detail.Healthcheck.Status != containerHealthUnhealthy {
		t.Fatalf("unexpected healthcheck status %#v", detail.Healthcheck)
	}
	if !reflect.DeepEqual(detail.Healthcheck.Command, []string{"CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"}) {
		t.Fatalf("unexpected healthcheck command %#v", detail.Healthcheck.Command)
	}
	if detail.Healthcheck.CheckedAt != "2026-06-17T01:31:53Z" {
		t.Fatalf("unexpected healthcheck checked_at %q", detail.Healthcheck.CheckedAt)
	}
	if detail.Healthcheck.Output != "curl failed" || detail.Healthcheck.FailureMessage != "curl failed" {
		t.Fatalf("unexpected healthcheck output %#v", detail.Healthcheck)
	}
	assertIntPtr(t, detail.Healthcheck.ExitCode, 1, "healthcheck exit code")
	assertIntPtr(t, detail.Healthcheck.FailingStreak, 2, "healthcheck failing streak")
	assertIntPtr(t, detail.LastExitCode, 137, "last exit code")
	if detail.OOMKilled == nil || !*detail.OOMKilled {
		t.Fatalf("expected oom killed true, got %#v", detail.OOMKilled)
	}
}

func TestDockerDetailOmitsDisabledHealthcheck(t *testing.T) {
	t.Parallel()

	detail := dockerDetail(container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    "abc123",
			Name:  "/web",
			State: &container.State{Status: container.StateRunning},
		},
		Config: &container.Config{
			Image:       "nginx:latest",
			Healthcheck: &container.HealthConfig{Test: []string{"NONE"}},
		},
	}, RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled"})

	if detail.Healthcheck != nil {
		t.Fatalf("expected disabled healthcheck to be omitted, got %#v", detail.Healthcheck)
	}
	if detail.Health != containerHealthNone {
		t.Fatalf("expected no healthcheck health, got %q", detail.Health)
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

func richDockerStatsFixture() container.StatsResponse {
	return container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage: container.CPUUsage{
				TotalUsage:        200,
				PercpuUsage:       []uint64{100, 100},
				UsageInUsermode:   70,
				UsageInKernelmode: 30,
			},
			SystemUsage: 1000,
			OnlineCPUs:  2,
			ThrottlingData: container.ThrottlingData{
				Periods:          11,
				ThrottledPeriods: 3,
				ThrottledTime:    900,
			},
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 100},
			SystemUsage: 500,
		},
		MemoryStats: richDockerMemoryStatsFixture(),
		Networks:    richDockerNetworkStatsFixture(),
		PidsStats:   container.PidsStats{Current: 5, Limit: 128},
	}
}

func richDockerMemoryStatsFixture() container.MemoryStats {
	return container.MemoryStats{
		Usage: 256,
		Limit: 1024,
		Stats: map[string]uint64{
			"cache":         10,
			"rss":           20,
			"active_file":   30,
			"inactive_file": 40,
			"pgfault":       50,
			"pgmajfault":    60,
		},
	}
}

func richDockerNetworkStatsFixture() map[string]container.NetworkStats {
	return map[string]container.NetworkStats{
		"bridge": {
			RxBytes:   100,
			TxBytes:   200,
			RxPackets: 3,
			TxPackets: 4,
			RxErrors:  1,
			TxErrors:  2,
			RxDropped: 5,
			TxDropped: 6,
		},
		"frontend": {
			RxBytes:   7,
			TxBytes:   8,
			RxPackets: 9,
			TxPackets: 10,
			RxErrors:  11,
			TxErrors:  12,
			RxDropped: 13,
			TxDropped: 14,
		},
	}
}

func assertRichDockerResourceStats(t *testing.T, resource ResourceSummary) {
	t.Helper()

	assertFloatPtr(t, resource.CPUPercent, 40, "computed CPU percent")
	assertInt64Ptr(t, resource.OnlineCPUs, 2, "online CPUs")
	assertInt64Ptr(t, resource.SystemCPUUsage, 1000, "system CPU usage")
	assertInt64Ptr(t, resource.TotalCPUUsage, 200, "total CPU usage")
	assertInt64Ptr(t, resource.CPUUsageInUsermode, 70, "CPU user mode usage")
	assertInt64Ptr(t, resource.CPUUsageInKernelmode, 30, "CPU kernel mode usage")
	assertInt64Ptr(t, resource.ThrottlingPeriods, 11, "CPU throttling periods")
	assertInt64Ptr(t, resource.ThrottlingThrottledPeriods, 3, "CPU throttled periods")
	assertInt64Ptr(t, resource.ThrottlingThrottledTime, 900, "CPU throttled time")
	assertRichDockerMemoryStats(t, resource)
	assertRichDockerNetworkStats(t, resource)
	assertInt64Ptr(t, resource.PIDsCurrent, 5, "pids current")
	assertInt64Ptr(t, resource.PIDsLimit, 128, "pids limit")
}

func assertRichDockerMemoryStats(t *testing.T, resource ResourceSummary) {
	t.Helper()

	assertInt64Ptr(t, resource.MemoryUsageBytes, 256, "memory usage bytes")
	assertInt64Ptr(t, resource.MemoryLimitBytes, 1024, "memory limit bytes")
	assertFloatPtr(t, resource.MemoryPercent, 25, "computed memory percent")
	assertInt64Ptr(t, resource.MemoryCache, 10, "memory cache")
	assertInt64Ptr(t, resource.MemoryRSS, 20, "memory rss")
	assertInt64Ptr(t, resource.MemoryActiveFile, 30, "memory active file")
	assertInt64Ptr(t, resource.MemoryInactiveFile, 40, "memory inactive file")
	assertInt64Ptr(t, resource.MemoryPgfault, 50, "memory pgfault")
	assertInt64Ptr(t, resource.MemoryPgmajfault, 60, "memory pgmajfault")
}

func assertRichDockerNetworkStats(t *testing.T, resource ResourceSummary) {
	t.Helper()

	assertInt64Ptr(t, resource.RxBytes, 107, "aggregated rx bytes")
	assertInt64Ptr(t, resource.TxBytes, 208, "aggregated tx bytes")
	assertInt64Ptr(t, resource.RxPackets, 12, "aggregated rx packets")
	assertInt64Ptr(t, resource.TxPackets, 14, "aggregated tx packets")
	assertInt64Ptr(t, resource.RxErrors, 12, "aggregated rx errors")
	assertInt64Ptr(t, resource.TxErrors, 14, "aggregated tx errors")
	assertInt64Ptr(t, resource.RxDropped, 18, "aggregated rx dropped")
	assertInt64Ptr(t, resource.TxDropped, 20, "aggregated tx dropped")
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

func assertIntPtr(t *testing.T, actual *int, expected int, label string) {
	t.Helper()

	if actual == nil || *actual != expected {
		t.Fatalf("expected %s %v, got %#v", label, expected, actual)
	}
}

type countingDockerClient struct {
	infoCalls          atomic.Int64
	inspectCalls       atomic.Int64
	logCalls           atomic.Int64
	listCalls          atomic.Int64
	statsCalls         atomic.Int64
	startCalls         atomic.Int64
	stopCalls          atomic.Int64
	restartCalls       atomic.Int64
	removeCalls        atomic.Int64
	removeForce        atomic.Bool
	logReader          io.ReadCloser
	inspect            container.InspectResponse
	list               []container.Summary
	stats              container.StatsResponse
	statsErr           error
	statsDelay         time.Duration
	activeStats        int64
	maxConcurrentStats atomic.Int64
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
	active := atomic.AddInt64(&c.activeStats, 1)
	for {
		current := c.maxConcurrentStats.Load()
		if active <= current || c.maxConcurrentStats.CompareAndSwap(current, active) {
			break
		}
	}
	if c.statsDelay > 0 {
		time.Sleep(c.statsDelay)
	}
	defer atomic.AddInt64(&c.activeStats, -1)
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
	c.startCalls.Add(1)
	return nil
}

func (c *countingDockerClient) ContainerStop(context.Context, string, container.StopOptions) error {
	c.stopCalls.Add(1)
	return nil
}

func (c *countingDockerClient) ContainerRestart(context.Context, string, container.StopOptions) error {
	c.restartCalls.Add(1)
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
