// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import "testing"

func TestToDetailMapsHealthcheckAndRuntimeStability(t *testing.T) {
	t.Parallel()

	detail := Detail{
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
		},
		EnvironmentPolicy: "masked",
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
		LastExitCode:     intPtrAllowZero(137),
		Mounts:           []Mount{},
		Networks:         []Network{},
		OOMKilled:        boolPtr(true),
		RuntimeInfo:      RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled", Endpoint: "local"},
		InspectUpdatedAt: "2026-06-17T01:32:00Z",
	}

	mapped := toDetail(detail)
	if mapped.Healthcheck == nil {
		t.Fatalf("expected mapped healthcheck")
	}
	if !mapped.Healthcheck.Configured || string(mapped.Healthcheck.Status) != containerHealthUnhealthy {
		t.Fatalf("unexpected mapped healthcheck %#v", mapped.Healthcheck)
	}
	if len(mapped.Healthcheck.Command) != 2 || mapped.Healthcheck.Command[1] != "curl -f http://localhost:8080/health || exit 1" {
		t.Fatalf("unexpected mapped healthcheck command %#v", mapped.Healthcheck.Command)
	}
	assertIntPtr(t, mapped.Healthcheck.ExitCode, 1, "mapped healthcheck exit code")
	assertIntPtr(t, mapped.Healthcheck.FailingStreak, 2, "mapped healthcheck failing streak")
	if mapped.Healthcheck.Output == nil || *mapped.Healthcheck.Output != "curl failed" {
		t.Fatalf("unexpected mapped healthcheck output %#v", mapped.Healthcheck.Output)
	}
	if mapped.Healthcheck.FailureMessage == nil || *mapped.Healthcheck.FailureMessage != "curl failed" {
		t.Fatalf("unexpected mapped healthcheck failure message %#v", mapped.Healthcheck.FailureMessage)
	}
	if mapped.Healthcheck.CheckedAt == nil || mapped.Healthcheck.CheckedAt.Format("2006-01-02T15:04:05Z07:00") != "2026-06-17T01:31:53Z" {
		t.Fatalf("unexpected mapped healthcheck checked_at %#v", mapped.Healthcheck.CheckedAt)
	}
	assertIntPtr(t, mapped.LastExitCode, 137, "mapped last exit code")
	if mapped.OomKilled == nil || !*mapped.OomKilled {
		t.Fatalf("expected mapped oom killed true, got %#v", mapped.OomKilled)
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
