package container

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/netip"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
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
			ID:   "full-id",
			Name: "/web",
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

func TestMapDockerShellErrorPreservesMappedRuntimeErrors(t *testing.T) {
	t.Parallel()

	err := mapDockerShellError(timeoutError{})
	if !errors.Is(err, errContainerRuntimeTimeout) {
		t.Fatalf("expected runtime timeout mapping, got %v", err)
	}
}

func TestMapDockerShellErrorDoesNotTreatGenericNoSuchFileAsCommandNotFound(t *testing.T) {
	t.Parallel()

	err := mapDockerShellError(errors.New("dial unix /var/run/docker.sock: connect: no such file or directory"))
	if errors.Is(err, errShellCommandNotFound) {
		t.Fatalf("expected socket path failure to avoid shell command mapping")
	}
	if !errors.Is(err, errRuntimeSocketMissing) && !strings.Contains(err.Error(), "runtime") {
		t.Fatalf("expected runtime-oriented mapping, got %v", err)
	}
}

func TestDockerExecSessionCloseWithoutStartDoesNotBlock(t *testing.T) {
	t.Parallel()

	session := newDockerExecSession(&countingDockerClient{}, "abc123", "sh")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- session.Close(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("close returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("close blocked before start")
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
				Ports:  []container.PortSummary{{IP: mustAddr(t, "0.0.0.0"), PrivatePort: 80, PublicPort: 8080, Type: "tcp"}},
				State:  container.StateRunning,
				Status: "Up 10 minutes",
				NetworkSettings: &container.NetworkSettingsSummary{
					Networks: map[string]*network.EndpointSettings{
						"bridge": {
							NetworkID:  "net-1",
							EndpointID: "endpoint-1",
							Gateway:    mustAddr(t, "172.18.0.1"),
							IPAddress:  mustAddr(t, "172.18.0.2"),
							MacAddress: mustHardwareAddr(t, "02:42:ac:12:00:02"),
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
		if item.Resource.Available || item.Resource.StatsAvailable {
			t.Fatalf("expected cold-start partial list stats to degrade as unavailable, got %#v", item.Resource)
		}
		if item.Resource.CPUPercent != nil || item.Resource.MemoryUsageBytes != nil || item.Resource.MemoryLimitBytes != nil || item.Resource.MemoryPercent != nil {
			t.Fatalf("expected cold-start list miss to avoid field-level partial stats, got %#v", item.Resource)
		}
	}
}

func TestDockerRuntimeDetailCollectsResourceStats(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		inspect: container.InspectResponse{
			ID:      "1234567890abcdef",
			Name:    "/graft-web",
			State:   &container.State{Status: container.StateRunning},
			Created: "2026-06-14T00:00:00Z",
			Config:  &container.Config{Image: "graft/web:latest"},
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

func TestDockerRuntimeDetailReusesCachedResourceStats(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		inspect: container.InspectResponse{
			ID:      "1234567890abcdef",
			Name:    "/graft-web",
			State:   &container.State{Status: container.StateRunning},
			Created: "2026-06-14T00:00:00Z",
			Config:  &container.Config{Image: "graft/web:latest"},
		},
		stats: richDockerStatsFixture(),
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	first, err := runtime.Detail(context.Background(), Ref{Value: "graft-web"})
	if err != nil {
		t.Fatalf("first detail: %v", err)
	}
	second, err := runtime.Detail(context.Background(), Ref{Value: "graft-web"})
	if err != nil {
		t.Fatalf("second detail: %v", err)
	}
	if !first.Resource.Available || !second.Resource.Available {
		t.Fatalf("expected cached resource stats to remain available, got %#v %#v", first.Resource, second.Resource)
	}
	if calls := client.statsCalls.Load(); calls != 1 {
		t.Fatalf("expected repeated detail reads to reuse cached stats, got %d stats calls", calls)
	}
}

func TestDockerRuntimeResourceStatsCacheCollapsesConcurrentMiss(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		stats: container.StatsResponse{
			MemoryStats: container.MemoryStats{Usage: 128, Limit: 256},
		},
		statsDelay: 40 * time.Millisecond,
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	var wg sync.WaitGroup
	results := make([]ResourceSummary, 6)
	for index := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = runtime.cachedContainerResourceSummary(context.Background(), "container-1")
		}(index)
	}
	wg.Wait()

	for _, summary := range results {
		if summary.Available || summary.StatsAvailable {
			t.Fatalf("expected cold-start partial stats to degrade as unavailable, got %#v", summary)
		}
		if summary.UnavailableReason == "" || summary.StatsErrorKey == "" || summary.StatsErrorMessage == "" {
			t.Fatalf("expected unavailable summary context, got %#v", summary)
		}
		if summary.CPUPercent != nil || summary.MemoryUsageBytes != nil || summary.MemoryLimitBytes != nil || summary.MemoryPercent != nil {
			t.Fatalf("expected cold-start miss to avoid field-level partial stats, got %#v", summary)
		}
	}
	if calls := client.statsCalls.Load(); calls != 1 {
		t.Fatalf("expected concurrent miss collapse to issue one stats call, got %d", calls)
	}
}

func TestDockerRuntimeListReusesCachedResourceStatsAcrossPolls(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		list: []container.Summary{
			{
				ID:      "1234567890abcdef",
				Names:   []string{"/graft-web"},
				Image:   "graft/web:latest",
				State:   container.StateRunning,
				Status:  "Up 10 minutes",
				Created: 1781409600,
			},
		},
		stats: richDockerStatsFixture(),
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	first, err := runtime.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("first list: %v", err)
	}
	second, err := runtime.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("second list: %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected one item per list call, got %#v %#v", first, second)
	}
	if calls := client.statsCalls.Load(); calls != 1 {
		t.Fatalf("expected repeated list polls to reuse cached stats, got %d stats calls", calls)
	}
}

func TestDockerRuntimeActionInvalidatesCachedResourceStats(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		inspect: container.InspectResponse{
			ID:      "1234567890abcdef",
			Name:    "/graft-web",
			State:   &container.State{Status: container.StateRunning},
			Created: "2026-06-14T00:00:00Z",
			Config:  &container.Config{Image: "graft/web:latest"},
		},
		stats: richDockerStatsFixture(),
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	if _, err := runtime.Detail(context.Background(), Ref{Value: "graft-web"}); err != nil {
		t.Fatalf("seed detail: %v", err)
	}
	if calls := client.statsCalls.Load(); calls != 1 {
		t.Fatalf("expected seeded cache to collect one stats sample, got %d", calls)
	}

	client.stats = container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 400, PercpuUsage: []uint64{200, 200}},
			SystemUsage: 2000,
			OnlineCPUs:  2,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 100},
			SystemUsage: 1000,
		},
		MemoryStats: container.MemoryStats{Usage: 512, Limit: 1024},
	}

	if _, err := runtime.Restart(context.Background(), Ref{Value: "graft-web"}); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if calls := client.statsCalls.Load(); calls != 2 {
		t.Fatalf("expected restart to reuse cached pre-action stats and recollect post-action stats, got %d", calls)
	}

	detail, err := runtime.Detail(context.Background(), Ref{Value: "graft-web"})
	if err != nil {
		t.Fatalf("detail after restart: %v", err)
	}
	assertInt64Ptr(t, detail.Resource.MemoryUsageBytes, 512, "memory usage after restart")
	if calls := client.statsCalls.Load(); calls != 2 {
		t.Fatalf("expected post-restart detail to use refreshed cache, got %d", calls)
	}
}

func TestDockerOrchestratorFromLabels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		labels      map[string]string
		wantType    string
		wantManaged bool
		wantProject string
		wantService string
		wantStack   string
		wantTask    string
		wantPod     string
		wantNS      string
		wantConf    string
	}{
		{
			name:        "standalone",
			labels:      nil,
			wantType:    containerOrchestratorStandalone,
			wantManaged: false,
			wantConf:    orchestratorConfidenceHigh,
		},
		{
			name: "compose",
			labels: map[string]string{
				composeProjectLabel: "graft",
				composeServiceLabel: "web",
			},
			wantType:    containerOrchestratorCompose,
			wantManaged: true,
			wantProject: "graft",
			wantService: "web",
			wantConf:    orchestratorConfidenceHigh,
		},
		{
			name: "kubernetes",
			labels: map[string]string{
				"io.kubernetes.pod.namespace":  "default",
				"io.kubernetes.pod.name":       "web-7c9f",
				"io.kubernetes.container.name": "web",
			},
			wantType:    containerOrchestratorKubernetes,
			wantManaged: true,
			wantPod:     "web-7c9f",
			wantNS:      "default",
			wantConf:    orchestratorConfidenceHigh,
		},
		{
			name: "swarm",
			labels: map[string]string{
				"com.docker.stack.namespace": "edge",
				"com.docker.swarm.task.name": "edge.1.abcd",
			},
			wantType:    containerOrchestratorSwarm,
			wantManaged: true,
			wantStack:   "edge",
			wantTask:    "edge.1.abcd",
			wantConf:    orchestratorConfidenceHigh,
		},
		{
			name: "conflicting labels become unknown",
			labels: map[string]string{
				composeProjectLabel:          "graft",
				"com.docker.stack.namespace": "edge",
				"com.docker.swarm.task.name": "edge.1.abcd",
			},
			wantType:    containerOrchestratorUnknown,
			wantManaged: true,
			wantConf:    orchestratorConfidenceLow,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			info := dockerOrchestratorFromLabels(tc.labels)
			assertDockerOrchestratorBase(t, info, tc)
			assertDockerOrchestratorMetadata(t, info, tc)
			assertDockerOrchestratorScopes(t, info, tc)
		})
	}
}

func assertDockerOrchestratorBase(t *testing.T, info OrchestratorInfo, tc struct {
	name        string
	labels      map[string]string
	wantType    string
	wantManaged bool
	wantProject string
	wantService string
	wantStack   string
	wantTask    string
	wantPod     string
	wantNS      string
	wantConf    string
}) {
	t.Helper()
	if info.Type != tc.wantType || info.Managed != tc.wantManaged || info.Confidence != tc.wantConf {
		t.Fatalf("unexpected orchestrator info %#v", info)
	}
}

func assertDockerOrchestratorMetadata(t *testing.T, info OrchestratorInfo, tc struct {
	name        string
	labels      map[string]string
	wantType    string
	wantManaged bool
	wantProject string
	wantService string
	wantStack   string
	wantTask    string
	wantPod     string
	wantNS      string
	wantConf    string
}) {
	t.Helper()
	if info.Project != tc.wantProject || info.Service != tc.wantService || info.Stack != tc.wantStack {
		t.Fatalf("unexpected project/service/stack %#v", info)
	}
	if info.Task != tc.wantTask {
		t.Fatalf("unexpected swarm task metadata %#v", info)
	}
	if info.Pod != tc.wantPod || info.Namespace != tc.wantNS {
		t.Fatalf("unexpected kubernetes metadata %#v", info)
	}
}

func assertDockerOrchestratorScopes(t *testing.T, info OrchestratorInfo, tc struct {
	name        string
	labels      map[string]string
	wantType    string
	wantManaged bool
	wantProject string
	wantService string
	wantStack   string
	wantTask    string
	wantPod     string
	wantNS      string
	wantConf    string
}) {
	t.Helper()
	if tc.wantType == containerOrchestratorCompose {
		assertComposeScopeSemantics(t, info, tc.wantProject, tc.wantService)
	}
	if tc.wantType == containerOrchestratorSwarm {
		assertSwarmScopeSemantics(t, info, tc.wantStack, tc.wantTask)
	}
	if tc.wantType == containerOrchestratorKubernetes {
		assertKubernetesScopeSemantics(t, info, tc.wantNS, tc.wantPod)
	}
}

func assertComposeScopeSemantics(t *testing.T, info OrchestratorInfo, project string, service string) {
	t.Helper()
	if info.GroupScopeKind != composeProjectScopeKind || info.GroupValue != project {
		t.Fatalf("unexpected compose group scope %#v", info)
	}
	if info.MemberScopeKind != composeServiceScopeKind || info.MemberValue != service {
		t.Fatalf("unexpected compose member scope %#v", info)
	}
}

func assertKubernetesScopeSemantics(t *testing.T, info OrchestratorInfo, namespace string, pod string) {
	t.Helper()
	if info.GroupScopeKind != kubernetesNamespaceScopeKind || info.GroupValue != namespace {
		t.Fatalf("unexpected kubernetes group scope %#v", info)
	}
	if info.MemberScopeKind != kubernetesPodScopeKind || info.MemberValue != pod {
		t.Fatalf("unexpected kubernetes member scope %#v", info)
	}
}

func assertSwarmScopeSemantics(t *testing.T, info OrchestratorInfo, stack string, task string) {
	t.Helper()
	if info.GroupScopeKind != swarmStackScopeKind || info.GroupValue != stack {
		t.Fatalf("unexpected swarm group scope %#v", info)
	}
	if info.MemberScopeKind != swarmTaskScopeKind || info.MemberValue != task {
		t.Fatalf("unexpected swarm member scope %#v", info)
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
					ID:     "abc123",
					Name:   "/web",
					State:  &container.State{Status: tc.state},
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

	runtime := &DockerRuntime{}
	resource := runtime.dockerResourceSummary("container-1", container.StatsResponse{
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

	runtime := &DockerRuntime{}
	stats := richDockerStatsFixture()
	stats.CPUStats.OnlineCPUs = 0

	resource := runtime.dockerResourceSummary("container-1", stats)

	if !resource.Available || !resource.StatsAvailable {
		t.Fatalf("expected per-CPU stats to keep resource available, got %#v", resource)
	}
	assertFloatPtr(t, resource.CPUPercent, 40, "computed CPU percent")
	assertInt64Ptr(t, resource.OnlineCPUs, 2, "fallback online CPUs")
}

func TestDockerResourceSummaryMatchesDockerCLICompatibleCPUPercent(t *testing.T) {
	t.Parallel()

	runtime := &DockerRuntime{}
	resource := runtime.dockerResourceSummary("container-1", container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage: container.CPUUsage{
				TotalUsage:  1_400,
				PercpuUsage: []uint64{350, 350, 350, 350},
			},
			SystemUsage: 10_000,
			OnlineCPUs:  4,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 1_000},
			SystemUsage: 9_000,
		},
		MemoryStats: container.MemoryStats{Usage: 1, Limit: 2},
	})

	if !resource.Available || !resource.StatsAvailable {
		t.Fatalf("expected normalized cpu stats to remain available, got %#v", resource)
	}
	assertFloatPtr(t, resource.CPUPercent, 160, "docker CLI compatible CPU percent")
	assertInt64Ptr(t, resource.OnlineCPUs, 4, "online CPUs")
}

func TestDockerResourceSummarySkipsOverflowedNetworkTotals(t *testing.T) {
	t.Parallel()

	runtime := &DockerRuntime{}
	resource := runtime.dockerResourceSummary("container-1", container.StatsResponse{
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
		ID:      "abc123",
		Name:    "/web",
		State:   &container.State{Status: container.StateRunning},
		Created: "2026-06-14T00:00:00Z",
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
		ID:    "abc123",
		Name:  "/web",
		State: &container.State{Status: container.StateRunning},
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
			ID:     "abc123",
			Name:   "/web",
			State:  &container.State{Status: container.StateRunning},
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

func TestDockerRuntimeMountUsageUsesInspectDerivedBindSource(t *testing.T) {
	t.Parallel()

	mount := dockerTestMount("bind", "/host/data", "/data", "")
	scanner := &recordingMountUsageScanner{size: 2 * 1024 * 1024}
	runtime := &DockerRuntime{
		client:            &countingDockerClient{inspect: dockerInspectWithMounts("abc123", mount)},
		endpoint:          "unix:///var/run/docker.sock",
		mountUsageScanner: scanner,
	}
	mountID := dockerMounts([]container.MountPoint{mount})[0].ID

	usage, err := runtime.MountUsage(context.Background(), Ref{Value: "web"}, mountID)
	if err != nil {
		t.Fatalf("mount usage: %v", err)
	}
	if usage.MountID != mountID || usage.Status != containerMountUsageStatusMeasured || usage.SizeBytes != 2*1024*1024 || usage.SizeDisplay != "2 MiB" {
		t.Fatalf("unexpected mount usage %#v", usage)
	}
	if scanner.calls.Load() != 1 || scanner.paths[0] != "/host/data" {
		t.Fatalf("expected inspect-derived source scan, got calls=%d paths=%#v", scanner.calls.Load(), scanner.paths)
	}
}

func TestDockerRuntimeMountUsageMapsScannerStatuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		err    error
		status string
	}{
		{name: "not found", err: errContainerMountNotFound, status: containerMountUsageStatusNotFound},
		{name: "permission denied", err: errRuntimePermissionDenied, status: containerMountUsageStatusPermissionDenied},
		{name: "timeout", err: errContainerRuntimeTimeout, status: containerMountUsageStatusTimeout},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mount := dockerTestMount("bind", "/host/data", "/data", "")
			runtime := &DockerRuntime{
				client:            &countingDockerClient{inspect: dockerInspectWithMounts("abc123", mount)},
				mountUsageScanner: &recordingMountUsageScanner{err: tc.err},
			}
			mountID := dockerMounts([]container.MountPoint{mount})[0].ID

			usage, err := runtime.MountUsage(context.Background(), Ref{Value: "web"}, mountID)
			if err != nil {
				t.Fatalf("mount usage should return status record, got %v", err)
			}
			if usage.Status != tc.status || usage.Message == "" {
				t.Fatalf("expected status %q with message, got %#v", tc.status, usage)
			}
		})
	}
}

func TestDockerRuntimeMountUsageVolumeAndUnsupportedMounts(t *testing.T) {
	t.Parallel()

	volume := dockerTestMount("volume", "/var/lib/docker/volumes/data/_data", "/data", "data")
	tmpfs := dockerTestMount("tmpfs", "", "/tmp", "")
	emptyVolume := dockerTestMount("volume", "", "/empty", "empty")
	runtime := &DockerRuntime{
		client:            &countingDockerClient{inspect: dockerInspectWithMounts("abc123", volume, tmpfs, emptyVolume)},
		mountUsageScanner: &recordingMountUsageScanner{size: 3 * 1024 * 1024 * 1024},
	}
	mounts := dockerMounts([]container.MountPoint{volume, tmpfs, emptyVolume})

	usage, err := runtime.MountUsage(context.Background(), Ref{Value: "web"}, mounts[0].ID)
	if err != nil {
		t.Fatalf("volume mount usage: %v", err)
	}
	if usage.Status != containerMountUsageStatusMeasured || usage.SizeDisplay != "3 GiB" || usage.SharedHint == "" {
		t.Fatalf("unexpected volume usage %#v", usage)
	}
	for _, mount := range mounts[1:] {
		usage, err := runtime.MountUsage(context.Background(), Ref{Value: "web"}, mount.ID)
		if err != nil {
			t.Fatalf("unsupported mount should return a status record for %#v, got %v", mount, err)
		}
		if usage.Status != containerMountUsageStatusUnsupported {
			t.Fatalf("expected unsupported for %#v, got %#v", mount, usage)
		}
	}
}

func TestDockerRuntimeMountUsageRequiresCurrentInspectMountID(t *testing.T) {
	t.Parallel()

	current := dockerTestMount("bind", "/host/current", "/data", "")
	old := dockerTestMount("bind", "/host/old", "/data", "")
	currentID := dockerMounts([]container.MountPoint{current})[0].ID
	oldID := dockerMounts([]container.MountPoint{old})[0].ID
	runtime := &DockerRuntime{
		client:            &countingDockerClient{inspect: dockerInspectWithMounts("abc123", current)},
		mountUsageScanner: &recordingMountUsageScanner{size: 1024},
	}

	if _, err := runtime.MountUsage(context.Background(), Ref{Value: "web"}, oldID); !errors.Is(err, errContainerMountNotFound) {
		t.Fatalf("expected old mount id to miss current inspect mounts, got %v", err)
	}
	if _, err := runtime.MountUsage(context.Background(), Ref{Value: "web"}, "/host/current"); !errors.Is(err, errContainerMountNotFound) {
		t.Fatalf("expected arbitrary path to miss current inspect mount ids, got %v", err)
	}
	if usage, err := runtime.MountUsage(context.Background(), Ref{Value: "web"}, currentID); err != nil || usage.Status != containerMountUsageStatusMeasured {
		t.Fatalf("expected current mount id to measure, got usage=%#v err=%v", usage, err)
	}
}

func dockerLogStream(t *testing.T, chunks ...string) io.Reader {
	t.Helper()

	var output bytes.Buffer
	for _, chunk := range chunks {
		writeStdcopyFrame(t, &output, stdcopy.Stdout, []byte(chunk))
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
	execCreateCalls    atomic.Int64
	execAttachCalls    atomic.Int64
	execResizeCalls    atomic.Int64
	removeForce        atomic.Bool
	logReader          io.ReadCloser
	inspect            container.InspectResponse
	list               []container.Summary
	stats              container.StatsResponse
	statsErr           error
	statsDelay         time.Duration
	execCreate         mobyclient.ExecCreateResult
	execAttach         mobyclient.HijackedResponse
	execCreateErr      error
	execAttachErr      error
	execResizeErr      error
	activeStats        int64
	maxConcurrentStats atomic.Int64
}

func (c *countingDockerClient) Info(context.Context) (systemInfo, error) {
	c.infoCalls.Add(1)
	return dockerClientSystemInfo{}, nil
}

func (c *countingDockerClient) ContainerList(context.Context, mobyclient.ContainerListOptions) ([]container.Summary, error) {
	c.listCalls.Add(1)
	return c.list, nil
}

func (c *countingDockerClient) ContainerInspect(context.Context, string) (container.InspectResponse, error) {
	c.inspectCalls.Add(1)
	return c.inspect, nil
}

func (c *countingDockerClient) ContainerLogs(context.Context, string, mobyclient.ContainerLogsOptions) (io.ReadCloser, error) {
	c.logCalls.Add(1)
	if c.logReader == nil {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	return c.logReader, nil
}

func (c *countingDockerClient) ContainerStatsOneShot(context.Context, string) (mobyclient.ContainerStatsResult, error) {
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
		return mobyclient.ContainerStatsResult{}, c.statsErr
	}
	var output bytes.Buffer
	if err := json.NewEncoder(&output).Encode(c.stats); err != nil {
		return mobyclient.ContainerStatsResult{}, err
	}
	return mobyclient.ContainerStatsResult{Body: io.NopCloser(&output)}, nil
}

func (c *countingDockerClient) ContainerExecCreate(context.Context, string, mobyclient.ExecCreateOptions) (mobyclient.ExecCreateResult, error) {
	c.execCreateCalls.Add(1)
	if c.execCreateErr != nil {
		return mobyclient.ExecCreateResult{}, c.execCreateErr
	}
	if c.execCreate.ID == "" {
		c.execCreate = mobyclient.ExecCreateResult{ID: "exec-1"}
	}
	return c.execCreate, nil
}

func (c *countingDockerClient) ContainerExecAttach(context.Context, string, mobyclient.ExecAttachOptions) (mobyclient.HijackedResponse, error) {
	c.execAttachCalls.Add(1)
	if c.execAttachErr != nil {
		return mobyclient.HijackedResponse{}, c.execAttachErr
	}
	if c.execAttach.Reader == nil {
		c.execAttach = mobyclient.HijackedResponse{Reader: bufio.NewReader(bytes.NewReader(nil))}
	}
	return c.execAttach, nil
}

func (c *countingDockerClient) ContainerExecResize(context.Context, string, mobyclient.ExecResizeOptions) error {
	c.execResizeCalls.Add(1)
	return c.execResizeErr
}

func (c *countingDockerClient) ContainerStart(context.Context, string, mobyclient.ContainerStartOptions) error {
	c.startCalls.Add(1)
	return nil
}

func (c *countingDockerClient) ContainerStop(context.Context, string, mobyclient.ContainerStopOptions) error {
	c.stopCalls.Add(1)
	return nil
}

func (c *countingDockerClient) ContainerRestart(context.Context, string, mobyclient.ContainerRestartOptions) error {
	c.restartCalls.Add(1)
	return nil
}

func (c *countingDockerClient) ContainerRemove(_ context.Context, _ string, options mobyclient.ContainerRemoveOptions) error {
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

type recordingMountUsageScanner struct {
	calls atomic.Int64
	paths []string
	size  int64
	err   error
}

func (s *recordingMountUsageScanner) ScanUsage(_ context.Context, path string) (int64, error) {
	s.calls.Add(1)
	s.paths = append(s.paths, path)
	if s.err != nil {
		return 0, s.err
	}
	return s.size, nil
}

func dockerInspectWithMounts(id string, mounts ...container.MountPoint) container.InspectResponse {
	return container.InspectResponse{
		ID:     id,
		Name:   "/web",
		State:  &container.State{Status: container.StateRunning},
		Config: &container.Config{Image: "nginx:latest"},
		Mounts: mounts,
	}
}

func dockerTestMount(mountType string, source string, destination string, name string) container.MountPoint {
	return container.MountPoint{
		Type:        mount.Type(mountType),
		Name:        name,
		Source:      source,
		Destination: destination,
		RW:          true,
	}
}

func mustAddr(t *testing.T, value string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(value)
	if err != nil {
		t.Fatalf("parse addr %q: %v", value, err)
	}
	return addr
}

func mustHardwareAddr(t *testing.T, value string) network.HardwareAddr {
	t.Helper()
	var addr network.HardwareAddr
	if err := addr.UnmarshalText([]byte(value)); err != nil {
		t.Fatalf("parse hardware addr %q: %v", value, err)
	}
	return addr
}

func writeStdcopyFrame(t *testing.T, w io.Writer, stream stdcopy.StdType, payload []byte) {
	t.Helper()
	size := uint64(len(payload))
	if size > math.MaxUint32 {
		t.Fatalf("payload too large: %d", len(payload))
	}
	header := [8]byte{}
	header[0] = byte(stream)
	var sizeBuf [8]byte
	binary.BigEndian.PutUint64(sizeBuf[:], size)
	copy(header[4:], sizeBuf[4:])
	if _, err := w.Write(header[:]); err != nil {
		t.Fatalf("write stdcopy header: %v", err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("write stdcopy payload: %v", err)
	}
}
