package container

import (
	"context"
	"testing"

	"graft/server/modules/container/terminal"
)

func TestBuildContainerDashboardSummary(t *testing.T) {
	t.Parallel()

	cpuHeavy := fakeSummary()
	cpuHeavy.ID = "c-1"
	cpuHeavy.Name = "cpu-heavy"
	cpuHeavy.ShortID = "c1"
	cpuHeavy.Resource = resourceWithCPUAndMemory(82.5, 512, 1024)

	memoryHeavy := fakeSummary()
	memoryHeavy.ID = "c-2"
	memoryHeavy.Name = "memory-heavy"
	memoryHeavy.ShortID = "c2"
	memoryHeavy.Resource = resourceWithCPUAndMemory(10, 2048, 4096)

	unhealthy := fakeSummary()
	unhealthy.ID = "c-3"
	unhealthy.Name = "unhealthy"
	unhealthy.ShortID = "c3"
	unhealthy.Health = containerHealthUnhealthy
	unhealthy.Resource = resourceWithCPUAndMemory(1, 128, 256)

	exited := fakeSummary()
	exited.ID = "c-4"
	exited.Name = "exited"
	exited.ShortID = "c4"
	exited.State = "exited"
	exited.Status = "Exited"
	exited.Resource = resourceWithCPUAndMemory(0, 64, 256)

	result := buildContainerDashboardSummary([]Summary{cpuHeavy, memoryHeavy, unhealthy, exited})

	if result.Overview.RunningContainers != 3 {
		t.Fatalf("expected 3 running containers, got %d", result.Overview.RunningContainers)
	}
	if result.Overview.AbnormalContainers != 2 {
		t.Fatalf("expected 2 abnormal containers, got %d", result.Overview.AbnormalContainers)
	}
	if len(result.Hotspots.CPUTop) != 3 || result.Hotspots.CPUTop[0].ID != "c-1" {
		t.Fatalf("expected cpu top ordered by cpu usage, got %#v", result.Hotspots.CPUTop)
	}
	if len(result.Hotspots.MemoryTop) != 3 || result.Hotspots.MemoryTop[0].ID != "c-2" {
		t.Fatalf("expected memory top ordered by memory usage, got %#v", result.Hotspots.MemoryTop)
	}
	if len(result.Anomalies) != 2 {
		t.Fatalf("expected only abnormal containers included, got %#v", result.Anomalies)
	}
	if result.Anomalies[0].ID != "c-3" {
		t.Fatalf("expected unhealthy anomaly first, got %#v", result.Anomalies)
	}
	if result.Anomalies[1].ID != "c-4" {
		t.Fatalf("expected exited anomaly before hotspot-only rows, got %#v", result.Anomalies)
	}
	if result.Overview.MemoryTotalPercent == nil {
		t.Fatalf("expected aggregated memory total percent")
	}
}

func TestBuildContainerDashboardSummaryExcludesNormalLoadFromAnomalies(t *testing.T) {
	t.Parallel()

	cpuHeavy := fakeSummary()
	cpuHeavy.ID = "cpu"
	cpuHeavy.Name = "cpu"
	cpuHeavy.Resource = resourceWithCPUAndMemory(90, 128, 256)

	memoryHeavy := fakeSummary()
	memoryHeavy.ID = "mem"
	memoryHeavy.Name = "mem"
	memoryHeavy.Resource = resourceWithCPUAndMemory(8, 4096, 8192)

	result := buildContainerDashboardSummary([]Summary{cpuHeavy, memoryHeavy})
	if len(result.Anomalies) != 0 {
		t.Fatalf("expected hotspot-only rows to stay out of anomalies, got %#v", result.Anomalies)
	}
}

func TestServiceDashboardSummaryUsesRuntimeList(t *testing.T) {
	t.Parallel()

	summaryA := fakeSummary()
	summaryA.ID = "dashboard-a"
	summaryA.Name = "dashboard-a"
	summaryA.ShortID = "da"
	summaryA.Resource = resourceWithCPUAndMemory(20, 100, 200)

	summaryB := fakeSummary()
	summaryB.ID = "dashboard-b"
	summaryB.Name = "dashboard-b"
	summaryB.ShortID = "db"
	summaryB.State = "restarting"
	summaryB.Resource = resourceWithCPUAndMemory(5, 50, 200)

	service, err := newTestService(containerServiceOptions{
		runtime:                 listOnlyRuntime{items: []Summary{summaryA, summaryB}},
		enabled:                 true,
		dangerousActionsEnabled: true,
		shellEnabled:            true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
		authorizer:              fakeAuthorizer{},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DashboardSummary(context.Background(), dashboardSummaryQuery{})
	if err != nil {
		t.Fatalf("dashboard summary: %v", err)
	}
	if len(result.Hotspots.CPUTop) != 2 {
		t.Fatalf("expected 2 cpu hotspots, got %d", len(result.Hotspots.CPUTop))
	}
	if len(result.Anomalies) != 1 {
		t.Fatalf("expected only abnormal containers in anomalies, got %d", len(result.Anomalies))
	}
	if result.Anomalies[0].ID != "dashboard-b" {
		t.Fatalf("expected restarting container to remain the only anomaly, got %#v", result.Anomalies)
	}
}

func resourceWithCPUAndMemory(cpu float64, usage int64, limit int64) ResourceSummary {
	memoryPercent := 0.0
	if limit > 0 {
		memoryPercent = (float64(usage) / float64(limit)) * 100
	}
	return ResourceSummary{
		Available:        true,
		StatsAvailable:   true,
		CPUPercent:       &cpu,
		MemoryUsageBytes: &usage,
		MemoryLimitBytes: &limit,
		MemoryPercent:    &memoryPercent,
	}
}

type listOnlyRuntime struct {
	items []Summary
}

func (r listOnlyRuntime) Info(context.Context) (RuntimeInfo, error) {
	return fakeRuntime{}.Info(context.Background())
}
func (r listOnlyRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return append([]Summary(nil), r.items...), nil
}
func (r listOnlyRuntime) Detail(context.Context, Ref) (Detail, error)  { return Detail{}, nil }
func (r listOnlyRuntime) Mounts(context.Context, Ref) ([]Mount, error) { return nil, nil }
func (r listOnlyRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (r listOnlyRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) { return Logs{}, nil }
func (r listOnlyRuntime) StreamLogs(context.Context, Ref, LogQuery, func(LogChunk) error) error {
	return nil
}
func (r listOnlyRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return newStubTerminalSession(), nil
}
func (r listOnlyRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r listOnlyRuntime) Stop(context.Context, Ref) (ActionResult, error) { return ActionResult{}, nil }
func (r listOnlyRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r listOnlyRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r listOnlyRuntime) Close() error { return nil }
