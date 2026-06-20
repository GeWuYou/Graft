// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"testing"

	containergen "graft/server/internal/contract/openapi/generated"
	containercontract "graft/server/modules/container/contract"
)

func TestToDetailMapsHealthcheckAndRuntimeStability(t *testing.T) {
	t.Parallel()

	mapped := toDetail(detailWithHealthcheckAndRuntimeStability())
	assertMappedHealthcheck(t, mapped.Healthcheck)
	assertMappedOrchestrator(t, mapped.Orchestrator)
	assertIntPtr(t, mapped.LastExitCode, 137, "mapped last exit code")
	if mapped.OomKilled == nil || !*mapped.OomKilled {
		t.Fatalf("expected mapped oom killed true, got %#v", mapped.OomKilled)
	}
	assertMappedEnvironmentDisplayValue(t, mapped.Environment)
	assertMappedMountUsage(t, mapped.Mounts)
}

func detailWithHealthcheckAndRuntimeStability() Detail {
	return Detail{
		Summary: Summary{
			ID:            "abc123",
			ShortID:       "abc123",
			Name:          "web",
			Names:         []string{"web"},
			Image:         "nginx:latest",
			Runtime:       runtimeNameDocker,
			CreatedAt:     "2026-06-14T00:00:00Z",
			State:         "running",
			Status:        "running",
			Health:        containerHealthUnhealthy,
			RestartCount:  intPtrAllowZero(3),
			RestartPolicy: "unless-stopped",
			Orchestrator: OrchestratorInfo{
				Type:               containerOrchestratorCompose,
				Managed:            true,
				GroupScopeKind:     composeProjectScopeKind,
				GroupDisplayName:   "graft",
				GroupValue:         "graft",
				MemberScopeKind:    composeServiceScopeKind,
				MemberDisplayName:  "web",
				MemberValue:        "web",
				Project:            "graft",
				Service:            "web",
				DisplayName:        "graft",
				Confidence:         orchestratorConfidenceHigh,
				ActionLevel:        containercontract.ContainerOrchestratorActionLevelWarn.String(),
				BatchActionAllowed: false,
				Warnings:           []string{orchestratorWarningManagedActionRisk, orchestratorWarningBatchBlocked},
				RecommendedAction:  recommendedActionOpenController,
			},
		},
		EnvironmentPolicy: "masked",
		Environment: []EnvironmentVariable{
			{
				Key:          "API_TOKEN",
				Value:        "",
				CopyValue:    "secret-token",
				DisplayValue: maskedEnvironmentPlaceholder,
				ValueMasked:  true,
				Masked:       true,
				Sensitive:    true,
				Source:       dockerEnvironmentSource,
			},
		},
		Healthcheck: &Healthcheck{
			Configured:     true,
			Status:         containerHealthUnhealthy,
			Command:        []string{"CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"},
			ExitCode:       intPtrAllowZero(1),
			Output:         "curl failed",
			CheckedAt:      "2026-06-17T01:31:53Z",
			FailingStreak:  intPtrAllowZero(2),
			FailureMessage: "curl failed",
		},
		LastExitCode: intPtrAllowZero(137),
		Mounts: []Mount{
			{
				ID:          "m_abc123",
				Type:        "bind",
				Source:      "/srv/graft/data",
				Destination: "/app/data",
				Mode:        "rw",
				ReadOnly:    false,
				Usage: &MountUsage{
					MountID:     "m_abc123",
					ContainerID: "abc123",
					Type:        "bind",
					Source:      "/srv/graft/data",
					Destination: "/app/data",
					SizeBytes:   134637568,
					SizeDisplay: "128.4 MiB",
					Status:      containerMountUsageStatusMeasured,
					MeasuredAt:  "2026-06-17T08:30:21Z",
					Message:     "host path usage",
					SharedHint:  "shared host path",
				},
			},
		},
		Networks:         []Network{},
		OOMKilled:        boolPtr(true),
		RuntimeInfo:      RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled", Endpoint: "local"},
		InspectUpdatedAt: "2026-06-17T01:32:00Z",
	}
}

func assertMappedOrchestrator(t *testing.T, info *containergen.ContainerOrchestratorInfo) {
	t.Helper()

	if info == nil {
		t.Fatalf("expected mapped orchestrator info")
	}
	assertMappedOrchestratorIdentity(t, info)
	assertMappedOrchestratorLegacyFields(t, info)
	assertMappedOrchestratorScopeFields(t, info)
	assertMappedOrchestratorPolicy(t, info)
}

func assertMappedOrchestratorIdentity(t *testing.T, info *containergen.ContainerOrchestratorInfo) {
	t.Helper()
	if string(info.Type) != containerOrchestratorCompose || !info.Managed {
		t.Fatalf("unexpected orchestrator identity %#v", info)
	}
}

func assertMappedOrchestratorLegacyFields(t *testing.T, info *containergen.ContainerOrchestratorInfo) {
	t.Helper()
	if info.Project == nil || *info.Project != "graft" || info.Service == nil || *info.Service != "web" {
		t.Fatalf("unexpected orchestrator project/service %#v", info)
	}
}

func assertMappedOrchestratorScopeFields(t *testing.T, info *containergen.ContainerOrchestratorInfo) {
	t.Helper()
	if info.GroupScopeKind == nil || *info.GroupScopeKind != composeProjectScopeKind || info.GroupValue == nil || *info.GroupValue != "graft" {
		t.Fatalf("unexpected orchestrator group scope %#v", info)
	}
	if info.MemberScopeKind == nil || *info.MemberScopeKind != composeServiceScopeKind || info.MemberValue == nil || *info.MemberValue != "web" {
		t.Fatalf("unexpected orchestrator member scope %#v", info)
	}
}

func assertMappedOrchestratorPolicy(t *testing.T, info *containergen.ContainerOrchestratorInfo) {
	t.Helper()
	if string(info.ActionLevel) != containercontract.ContainerOrchestratorActionLevelWarn.String() || info.BatchActionAllowed {
		t.Fatalf("unexpected orchestrator policy %#v", info)
	}
	if len(info.Warnings) != 2 {
		t.Fatalf("expected orchestrator warnings, got %#v", info.Warnings)
	}
}

func assertMappedHealthcheck(t *testing.T, healthcheck *containergen.ContainerHealthcheck) {
	t.Helper()

	if healthcheck == nil {
		t.Fatalf("expected mapped healthcheck")
	}
	if !healthcheck.Configured || string(healthcheck.Status) != containerHealthUnhealthy {
		t.Fatalf("unexpected mapped healthcheck %#v", healthcheck)
	}
	if len(healthcheck.Command) != 2 || healthcheck.Command[1] != "curl -f http://localhost:8080/health || exit 1" {
		t.Fatalf("unexpected mapped healthcheck command %#v", healthcheck.Command)
	}
	assertIntPtr(t, healthcheck.ExitCode, 1, "mapped healthcheck exit code")
	assertIntPtr(t, healthcheck.FailingStreak, 2, "mapped healthcheck failing streak")
	if healthcheck.Output == nil || *healthcheck.Output != "curl failed" {
		t.Fatalf("unexpected mapped healthcheck output %#v", healthcheck.Output)
	}
	if healthcheck.FailureMessage == nil || *healthcheck.FailureMessage != "curl failed" {
		t.Fatalf("unexpected mapped healthcheck failure message %#v", healthcheck.FailureMessage)
	}
	if healthcheck.CheckedAt == nil || healthcheck.CheckedAt.Format("2006-01-02T15:04:05Z07:00") != "2026-06-17T01:31:53Z" {
		t.Fatalf("unexpected mapped healthcheck checked_at %#v", healthcheck.CheckedAt)
	}
}

func assertMappedEnvironmentDisplayValue(t *testing.T, environment *[]containergen.ContainerEnvironmentEntry) {
	t.Helper()

	if environment == nil || len(*environment) != 1 {
		t.Fatalf("expected one mapped environment entry, got %#v", environment)
	}
	if (*environment)[0].DisplayValue == nil || *(*environment)[0].DisplayValue != maskedEnvironmentPlaceholder {
		t.Fatalf("expected mapped environment display value, got %#v", environment)
	}
	if (*environment)[0].CopyValue == nil || *(*environment)[0].CopyValue != "secret-token" {
		t.Fatalf("expected mapped copy_value, got %#v", environment)
	}
	if (*environment)[0].ValueMasked == nil || !*(*environment)[0].ValueMasked {
		t.Fatalf("expected mapped value_masked marker, got %#v", environment)
	}
}

func assertMappedMountUsage(t *testing.T, mounts []containergen.ContainerMount) {
	t.Helper()

	if len(mounts) != 1 {
		t.Fatalf("expected one mapped mount, got %#v", mounts)
	}
	mount := mounts[0]
	if mount.MountId != "m_abc123" || mount.Usage == nil {
		t.Fatalf("expected mapped mount id and usage, got %#v", mount)
	}
	if mount.Usage.MountId != "m_abc123" || mount.Usage.SizeBytes == nil || *mount.Usage.SizeBytes != 134637568 {
		t.Fatalf("unexpected mapped mount usage %#v", mount.Usage)
	}
	if mount.Usage.MeasuredAt == nil || mount.Usage.MeasuredAt.Format("2006-01-02T15:04:05Z07:00") != "2026-06-17T08:30:21Z" {
		t.Fatalf("unexpected mapped mount usage measured_at %#v", mount.Usage.MeasuredAt)
	}
}

func TestToResourceSummaryMapsDockerStatsFields(t *testing.T) {
	t.Parallel()

	resource := ResourceSummary{
		Available:                  true,
		StatsAvailable:             true,
		CPUPercent:                 float64Ptr(12.5),
		OnlineCPUs:                 int64Ptr(4),
		SystemCPUUsage:             int64Ptr(1000),
		TotalCPUUsage:              int64Ptr(200),
		CPUUsageInUsermode:         int64Ptr(70),
		CPUUsageInKernelmode:       int64Ptr(30),
		ThrottlingPeriods:          int64Ptr(11),
		ThrottlingThrottledPeriods: int64Ptr(3),
		ThrottlingThrottledTime:    int64Ptr(900),
		MemoryUsageBytes:           int64Ptr(256),
		MemoryLimitBytes:           int64Ptr(1024),
		MemoryPercent:              float64Ptr(25),
		MemoryCache:                int64Ptr(10),
		MemoryRSS:                  int64Ptr(20),
		MemoryActiveFile:           int64Ptr(30),
		MemoryInactiveFile:         int64Ptr(40),
		MemoryPgfault:              int64Ptr(50),
		MemoryPgmajfault:           int64Ptr(60),
		RxBytes:                    int64Ptr(107),
		TxBytes:                    int64Ptr(208),
		RxPackets:                  int64Ptr(12),
		TxPackets:                  int64Ptr(14),
		RxErrors:                   int64Ptr(12),
		TxErrors:                   int64Ptr(14),
		RxDropped:                  int64Ptr(18),
		TxDropped:                  int64Ptr(20),
		PIDsCurrent:                int64Ptr(5),
		PIDsLimit:                  int64Ptr(128),
	}

	mapped := toResourceSummary(resource)
	if mapped == nil {
		t.Fatalf("expected mapped resource summary")
	}
	assertFloatPtr(t, mapped.CpuPercent, 12.5, "mapped CPU percent")
	assertInt64Ptr(t, mapped.OnlineCpus, 4, "mapped online CPUs")
	assertInt64Ptr(t, mapped.SystemCpuUsage, 1000, "mapped system CPU usage")
	assertInt64Ptr(t, mapped.TotalCpuUsage, 200, "mapped total CPU usage")
	assertInt64Ptr(t, mapped.CpuUsageInUsermode, 70, "mapped CPU user mode usage")
	assertInt64Ptr(t, mapped.CpuUsageInKernelmode, 30, "mapped CPU kernel mode usage")
	assertInt64Ptr(t, mapped.ThrottlingPeriods, 11, "mapped CPU throttling periods")
	assertInt64Ptr(t, mapped.ThrottlingThrottledPeriods, 3, "mapped CPU throttled periods")
	assertInt64Ptr(t, mapped.ThrottlingThrottledTime, 900, "mapped CPU throttled time")
	assertInt64Ptr(t, mapped.MemoryUsageBytes, 256, "mapped memory usage bytes")
	assertInt64Ptr(t, mapped.MemoryLimitBytes, 1024, "mapped memory limit bytes")
	assertFloatPtr(t, mapped.MemoryPercent, 25, "mapped memory percent")
	assertInt64Ptr(t, mapped.MemoryCache, 10, "mapped memory cache")
	assertInt64Ptr(t, mapped.MemoryRss, 20, "mapped memory rss")
	assertInt64Ptr(t, mapped.MemoryActiveFile, 30, "mapped memory active file")
	assertInt64Ptr(t, mapped.MemoryInactiveFile, 40, "mapped memory inactive file")
	assertInt64Ptr(t, mapped.MemoryPgfault, 50, "mapped memory pgfault")
	assertInt64Ptr(t, mapped.MemoryPgmajfault, 60, "mapped memory pgmajfault")
	assertInt64Ptr(t, mapped.RxBytes, 107, "mapped rx bytes")
	assertInt64Ptr(t, mapped.TxBytes, 208, "mapped tx bytes")
	assertInt64Ptr(t, mapped.RxPackets, 12, "mapped rx packets")
	assertInt64Ptr(t, mapped.TxPackets, 14, "mapped tx packets")
	assertInt64Ptr(t, mapped.RxErrors, 12, "mapped rx errors")
	assertInt64Ptr(t, mapped.TxErrors, 14, "mapped tx errors")
	assertInt64Ptr(t, mapped.RxDropped, 18, "mapped rx dropped")
	assertInt64Ptr(t, mapped.TxDropped, 20, "mapped tx dropped")
	assertInt64Ptr(t, mapped.PidsCurrent, 5, "mapped pids current")
	assertInt64Ptr(t, mapped.PidsLimit, 128, "mapped pids limit")
}

func int64Ptr(value int64) *int64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
