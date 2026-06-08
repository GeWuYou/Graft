package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/config"
	"graft/server/internal/container"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	monitorcontract "graft/server/modules/monitor/contract"
)

func TestBuildServerStatusResponseIncludesCurrentSliceFields(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	startedAt := time.Now().UTC().Add(-5 * time.Second).Truncate(time.Second)
	response, err := buildServerStatusResponseWithRuntimeSnapshot(context.Background(), &module.Context{
		Config: &config.Config{
			App: config.AppConfig{
				Name: " graft ",
				Env:  " prod ",
			},
		},
		RuntimeMetadata: module.NewRuntimeMetadata([]module.Spec{
			{ID: "audit"},
			{ID: "user"},
			{ID: "rbac", Dependencies: []string{"user"}},
			{ID: moduleID, Dependencies: []string{"user", "rbac"}},
		}),
	}, moduleWithStartedAt(db, startedAt), monitorcontract.TrendRange10Minutes, stableRuntimeSnapshot())
	if err != nil {
		t.Fatalf("build server status response: %v", err)
	}

	assertCurrentSliceResponseStatus(t, response, startedAt)
	assertCurrentSliceRuntimeSnapshot(t, response)
	assertCurrentSliceTrendSnapshot(t, response)
	assertCurrentSliceSummary(t, response)
	assertCurrentSliceModuleSummaries(t, response.Modules)
}

func TestRegisterMonitorDashboardWidgetRegistersSystemHealthInsight(t *testing.T) {
	registry := dashboard.NewRegistry()
	moduleCtx := &module.Context{DashboardRegistry: registry, Services: container.New()}

	if err := registerMonitorDashboardWidget(moduleCtx, nil); err != nil {
		t.Fatalf("register monitor dashboard widget: %v", err)
	}

	widget, ok := registry.Get(monitorSystemHealthWidgetID)
	if !ok {
		t.Fatalf("expected monitor system health dashboard widget to be registered")
	}
	if widget.Type != dashboard.WidgetTypeHealth {
		t.Fatalf("expected health widget, got %q", widget.Type)
	}
	if widget.RouteLocation != monitorcontract.ServerStatusOverviewMenuPath {
		t.Fatalf("expected monitor overview route, got %q", widget.RouteLocation)
	}
	if len(widget.RequiredPermissions) != 1 || widget.RequiredPermissions[0] != monitorcontract.ServerStatusReadPermission.String() {
		t.Fatalf("unexpected required permissions: %#v", widget.RequiredPermissions)
	}
}

func TestMonitorSystemHealthDashboardWidgetLoadsHealthPayload(t *testing.T) {
	payload, err := loadMonitorSystemHealthWidget(context.Background(), &module.Context{Services: container.New()}, nil)
	if err != nil {
		t.Fatalf("load monitor system health widget: %v", err)
	}

	summary, ok := payload["summary"].(dashboard.HealthSummaryItem)
	if !ok {
		t.Fatalf("expected health summary payload, got %#v", payload["summary"])
	}
	if summary.Status == "" {
		t.Fatalf("expected summary status to be populated")
	}
	items, ok := payload["items"].([]dashboard.HealthItem)
	if !ok {
		t.Fatalf("expected health items payload, got %#v", payload["items"])
	}
	if len(items) != 3 {
		t.Fatalf("expected database, redis, and anomalies health items, got %d", len(items))
	}
}

func TestBuildServerStatusResponseUsesUnknownWhenDatabaseServiceIsAbsent(t *testing.T) {
	t.Parallel()

	response, err := buildServerStatusResponseWithRuntimeSnapshot(context.Background(), &module.Context{
		Services: container.New(),
	}, nil, monitorcontract.TrendRange10Minutes, stableRuntimeSnapshot())
	if err != nil {
		t.Fatalf("build server status response: %v", err)
	}

	if response.Dependencies.Database.Status != "unknown" {
		t.Fatalf("expected database status unknown, got %q", response.Dependencies.Database.Status)
	}
	if response.Dependencies.Database.Detail != "Database handle is unavailable" {
		t.Fatalf("expected database detail for missing handle, got %q", response.Dependencies.Database.Detail)
	}
	if response.Dependencies.Redis.Status != "disabled" {
		t.Fatalf("expected redis status disabled, got %q", response.Dependencies.Redis.Status)
	}
	if response.Dependencies.Redis.Detail != "Redis client is not configured" {
		t.Fatalf("expected redis detail for disabled client, got %q", response.Dependencies.Redis.Detail)
	}
	if response.Status != "degraded" {
		t.Fatalf("expected overall status degraded when dependency observability is missing, got %q", response.Status)
	}
	if len(response.Anomalies) != 1 {
		t.Fatalf("expected one dependency anomaly for missing database handle, got %d", len(response.Anomalies))
	}
	if string(response.Anomalies[0].AnomalyKey) != string(monitorcontract.DependencyStatusUnknown) {
		t.Fatalf("expected dependency_status_unknown anomaly, got %q", response.Anomalies[0].AnomalyKey)
	}
	if string(response.Trend.Range) != monitorcontract.TrendRange10Minutes.String() {
		t.Fatalf("expected default trend range in response, got %q", response.Trend.Range)
	}
}

func TestBuildServerStatusResponseReportsDegradedOnDatabasePingError(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close sqlite database: %v", err)
	}

	response, err := buildServerStatusResponseWithRuntimeSnapshot(context.Background(), &module.Context{}, &Module{db: db}, monitorcontract.TrendRange10Minutes, stableRuntimeSnapshot())
	if err != nil {
		t.Fatalf("build server status response: %v", err)
	}

	if response.Dependencies.Database.Status != "degraded" {
		t.Fatalf("expected database status degraded on ping error, got %q", response.Dependencies.Database.Status)
	}
	if response.Dependencies.Database.Detail != "Database ping failed" {
		t.Fatalf("expected degraded detail to be sanitized, got %q", response.Dependencies.Database.Detail)
	}
	if response.Status != "degraded" {
		t.Fatalf("expected overall status degraded on ping error, got %q", response.Status)
	}
}

func TestRuntimeModuleSummariesFollowPlatformStatus(t *testing.T) {
	t.Parallel()

	moduleCtx := &module.Context{
		RuntimeMetadata: module.NewRuntimeMetadata([]module.Spec{
			{ID: "user"},
			{ID: "rbac", Dependencies: []string{"user"}},
			{ID: moduleID, Dependencies: []string{"user", "rbac"}},
		}),
	}

	healthy := runtimeModuleSummaries(
		moduleCtx,
		generated.ServerStatusDependency{Status: statusHealthy},
		generated.ServerStatusDependency{Status: statusDisabled},
	)
	assertModuleSummaries(t, healthy, []generated.ServerStatusModule{
		{Name: "user", Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: nil},
		{Name: "rbac", Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: []string{"user"}},
		{Name: moduleID, Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: []string{"user", "rbac"}},
	})

	degraded := runtimeModuleSummaries(
		moduleCtx,
		generated.ServerStatusDependency{Status: statusDegraded},
		generated.ServerStatusDependency{Status: statusHealthy},
	)
	assertModuleSummaries(t, degraded, []generated.ServerStatusModule{
		{Name: "user", Status: statusDegraded, StatusDetail: "Runtime metadata is present, but shared runtime signals are degraded", DependsOn: nil},
		{Name: "rbac", Status: statusDegraded, StatusDetail: "Runtime metadata is present, but shared runtime signals are degraded", DependsOn: []string{"user"}},
		{Name: moduleID, Status: statusDegraded, StatusDetail: "Runtime metadata is present, but shared runtime signals are degraded", DependsOn: []string{"user", "rbac"}},
	})
}

func TestRuntimeModuleSummariesDegradeWhenDependencyMetadataIsMissing(t *testing.T) {
	t.Parallel()

	moduleCtx := &module.Context{
		RuntimeMetadata: module.NewRuntimeMetadata([]module.Spec{
			{ID: "audit"},
			{ID: moduleID, Dependencies: []string{"user", "rbac"}},
		}),
	}

	actual := runtimeModuleSummaries(
		moduleCtx,
		generated.ServerStatusDependency{Status: statusHealthy},
		generated.ServerStatusDependency{Status: statusDisabled},
	)

	assertModuleSummaries(t, actual, []generated.ServerStatusModule{
		{Name: "audit", Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: nil},
		{
			Name:                moduleID,
			Status:              statusDegraded,
			StatusDetail:        "Missing runtime dependencies: user, rbac",
			DependsOn:           []string{"user", "rbac"},
			MissingDependencies: stringSlicePointer("user", "rbac"),
		},
	})
}

func TestStopTrendSamplerRequiresLifecycleContext(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	moduleInstance := &Module{
		samplerCancel: func() {
			close(done)
		},
		samplerDone: done,
	}

	err := moduleInstance.stopTrendSampler(&module.Context{})
	if err == nil {
		t.Fatalf("expected missing lifecycle context error")
	}
	if !strings.Contains(err.Error(), "missing lifecycle context") {
		t.Fatalf("expected missing lifecycle context error, got %v", err)
	}
}

func TestLoadTrendPointsHonorsRequestedRange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisServer.Addr(),
	})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	storageKey := trendStorageKey("graft", "trend-host")
	observedAt := time.Date(2026, 5, 20, 9, 0, 0, 0, time.UTC)
	points := []generated.ServerStatusTrendPoint{
		{
			ObservedAt:                observedAt.Add(-45 * time.Minute),
			CpuPercent:                9.2,
			HostMemoryUsedPercent:     37.5,
			LoadAverageOneMinute:      0.21,
			LoadAverageFiveMinutes:    0.18,
			LoadAverageFifteenMinutes: 0.16,
			Goroutines:                11,
			RuntimeAllocBytes:         32 * 1024 * 1024,
			RuntimeHeapInUseBytes:     18 * 1024 * 1024,
			RuntimeSysBytes:           64 * 1024 * 1024,
		},
		{
			ObservedAt:                observedAt.Add(-20 * time.Minute),
			CpuPercent:                14.4,
			HostMemoryUsedPercent:     41.2,
			LoadAverageOneMinute:      0.33,
			LoadAverageFiveMinutes:    0.26,
			LoadAverageFifteenMinutes: 0.22,
			Goroutines:                17,
			RuntimeAllocBytes:         48 * 1024 * 1024,
			RuntimeHeapInUseBytes:     28 * 1024 * 1024,
			RuntimeSysBytes:           80 * 1024 * 1024,
		},
		{
			ObservedAt:                observedAt.Add(-5 * time.Minute),
			CpuPercent:                21.8,
			HostMemoryUsedPercent:     46.9,
			LoadAverageOneMinute:      0.57,
			LoadAverageFiveMinutes:    0.44,
			LoadAverageFifteenMinutes: 0.38,
			Goroutines:                23,
			RuntimeAllocBytes:         60 * 1024 * 1024,
			RuntimeHeapInUseBytes:     34 * 1024 * 1024,
			RuntimeSysBytes:           96 * 1024 * 1024,
		},
	}

	for _, point := range points {
		if err := storeTrendPoint(ctx, redisClient, storageKey, point.ObservedAt, point); err != nil {
			t.Fatalf("store trend point: %v", err)
		}
	}

	thirtyMinutePoints, err := loadTrendPoints(ctx, redisClient, storageKey, observedAt, monitorcontract.TrendRange30Minutes.Duration())
	if err != nil {
		t.Fatalf("load 30m trend points: %v", err)
	}
	if len(thirtyMinutePoints) != 2 {
		t.Fatalf("expected 2 trend points in 30m window, got %d", len(thirtyMinutePoints))
	}
	assertEqual(t, "30m oldest point", thirtyMinutePoints[0].ObservedAt, points[1].ObservedAt)
	assertEqual(t, "30m latest point", thirtyMinutePoints[1].ObservedAt, points[2].ObservedAt)

	tenMinutePoints, err := loadTrendPoints(ctx, redisClient, storageKey, observedAt, monitorcontract.TrendRange10Minutes.Duration())
	if err != nil {
		t.Fatalf("load 10m trend points: %v", err)
	}
	if len(tenMinutePoints) != 1 {
		t.Fatalf("expected 1 trend point in 10m window, got %d", len(tenMinutePoints))
	}
	assertEqual(t, "10m point", tenMinutePoints[0].ObservedAt, points[2].ObservedAt)
}

func TestBuildServerStatusResponseLoadsRedisTrendPoints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisServer.Addr(),
	})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	observedAt := time.Now().UTC()
	storageKey := trendStorageKey("graft", resolveHostName())
	for _, point := range []generated.ServerStatusTrendPoint{
		{
			ObservedAt:                observedAt.Add(-25 * time.Minute),
			CpuPercent:                12.4,
			HostMemoryUsedPercent:     39.1,
			LoadAverageOneMinute:      0.28,
			LoadAverageFiveMinutes:    0.24,
			LoadAverageFifteenMinutes: 0.19,
			Goroutines:                15,
			RuntimeAllocBytes:         40 * 1024 * 1024,
			RuntimeHeapInUseBytes:     22 * 1024 * 1024,
			RuntimeSysBytes:           72 * 1024 * 1024,
		},
		{
			ObservedAt:                observedAt.Add(-8 * time.Minute),
			CpuPercent:                18.7,
			HostMemoryUsedPercent:     44.3,
			LoadAverageOneMinute:      0.49,
			LoadAverageFiveMinutes:    0.35,
			LoadAverageFifteenMinutes: 0.27,
			Goroutines:                19,
			RuntimeAllocBytes:         55 * 1024 * 1024,
			RuntimeHeapInUseBytes:     30 * 1024 * 1024,
			RuntimeSysBytes:           88 * 1024 * 1024,
		},
	} {
		if err := storeTrendPoint(ctx, redisClient, storageKey, point.ObservedAt, point); err != nil {
			t.Fatalf("store redis trend point: %v", err)
		}
	}

	response, err := buildServerStatusResponse(ctx, &module.Context{
		Config: &config.Config{
			App: config.AppConfig{
				Name: "graft",
			},
		},
		Redis: redisClient,
	}, moduleWithStartedAt(db, observedAt.Add(-5*time.Minute)), monitorcontract.TrendRange30Minutes)
	if err != nil {
		t.Fatalf("build response with redis trend: %v", err)
	}

	assertEqual(t, "redis trend range", string(response.Trend.Range), monitorcontract.TrendRange30Minutes.String())
	if len(response.Trend.Points) != 2 {
		t.Fatalf("expected 2 redis-backed trend points, got %d", len(response.Trend.Points))
	}
	if response.Trend.Points[1].CpuPercent != 18.7 {
		t.Fatalf("expected cpu percent from redis-backed trend point, got %v", response.Trend.Points[1].CpuPercent)
	}
	if response.Trend.Points[1].HostMemoryUsedPercent != 44.3 {
		t.Fatalf("expected host memory percent from redis-backed trend point, got %v", response.Trend.Points[1].HostMemoryUsedPercent)
	}
	if response.Trend.Points[1].LoadAverageOneMinute != 0.49 {
		t.Fatalf("expected one-minute load average from redis-backed trend point, got %v", response.Trend.Points[1].LoadAverageOneMinute)
	}
}

func TestIncidentEvidenceCapabilityReturnsExpiredWhenWindowExceedsRetention(t *testing.T) {
	t.Parallel()

	capability := incidentEvidenceCapability{module: &Module{}, ctx: &module.Context{}}
	now := time.Now().UTC()
	resolved, err := capability.ResolveAuditIncidentMonitorEvidence(context.Background(), moduleapi.ResolveAuditIncidentMonitorEvidenceInput{
		IncidentStartedAt: now.Add(-2 * time.Hour),
		IncidentEndedAt:   now.Add(-90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("resolve monitor incident evidence: %v", err)
	}
	if resolved.Availability != moduleapi.MonitorEvidenceExpired {
		t.Fatalf("expected expired availability, got %q", resolved.Availability)
	}
}

func TestParseTrendRange(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected monitorcontract.TrendRange
	}{
		{name: "default empty", input: "", expected: monitorcontract.TrendRange10Minutes},
		{name: "10m", input: "10m", expected: monitorcontract.TrendRange10Minutes},
		{name: "30m", input: "30m", expected: monitorcontract.TrendRange30Minutes},
		{name: "1h", input: "1h", expected: monitorcontract.TrendRange1Hour},
		{name: "invalid fallback", input: "24h", expected: monitorcontract.TrendRange10Minutes},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := parseTrendRange(testCase.input)
			if actual != testCase.expected {
				t.Fatalf("parseTrendRange(%q) = %q, want %q", testCase.input, actual, testCase.expected)
			}
		})
	}
}

func TestDeriveOverallStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		databaseStatus string
		redisStatus    string
		expected       string
	}{
		{
			name:           "degraded dominates",
			databaseStatus: "healthy",
			redisStatus:    "degraded",
			expected:       "degraded",
		},
		{
			name:           "healthy survives disabled dependency",
			databaseStatus: "healthy",
			redisStatus:    "disabled",
			expected:       "healthy",
		},
		{
			name:           "unknown when no dependency is healthy",
			databaseStatus: "unknown",
			redisStatus:    "disabled",
			expected:       "unknown",
		},
		{
			name:           "healthy redis survives unknown database",
			databaseStatus: "unknown",
			redisStatus:    "healthy",
			expected:       "healthy",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if actual := deriveOverallStatus(testCase.databaseStatus, testCase.redisStatus, nil); actual != testCase.expected {
				t.Fatalf(
					"deriveOverallStatus(%q, %q) = %q, want %q",
					testCase.databaseStatus,
					testCase.redisStatus,
					actual,
					testCase.expected,
				)
			}
		})
	}
}

func TestDeriveOverallStatusDegradesWhenAnomaliesExist(t *testing.T) {
	t.Parallel()

	actual := deriveOverallStatus("healthy", "disabled", []generated.ServerStatusAnomaly{
		{
			AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(monitorcontract.ResourceCPUPressure),
		},
	})
	if actual != "degraded" {
		t.Fatalf("expected anomaly-backed overall status degraded, got %q", actual)
	}
}

func assertEqual[T comparable](t *testing.T, field string, actual T, expected T) {
	t.Helper()

	if actual != expected {
		t.Fatalf("expected %s %v, got %v", field, expected, actual)
	}
}

func assertCurrentSliceResponseStatus(t *testing.T, response generated.ServerStatusResponse, startedAt time.Time) {
	t.Helper()

	assertEqual(t, "overall status", response.Status, "healthy")
	assertEqual(t, "database status", response.Dependencies.Database.Status, "healthy")
	assertEqual(t, "database detail", response.Dependencies.Database.Detail, "Database ping succeeded")
	if response.Dependencies.Database.LatencyMs == nil {
		t.Fatalf("expected database latency to be recorded")
	}
	assertEqual(t, "redis status", response.Dependencies.Redis.Status, "disabled")
	assertEqual(t, "redis detail", response.Dependencies.Redis.Detail, "Redis client is not configured")
	assertEqual(t, "server version", response.Server.Version, fallbackServerVersion)
	assertEqual(t, "started_at", response.Server.StartedAt, startedAt)
	assertEqual(t, "go version", response.Server.GoVersion, runtime.Version())
	assertEqual(t, "app name", response.Server.AppName, "graft")
	assertEqual(t, "app env", response.Server.AppEnv, "prod")
	if len(response.Anomalies) != 0 {
		t.Fatalf("expected current slice happy-path response to stay anomaly-free, got %d", len(response.Anomalies))
	}
	if response.Server.UptimeSeconds < 5 {
		t.Fatalf("expected uptime to be at least 5 seconds, got %d", response.Server.UptimeSeconds)
	}
}

func assertCurrentSliceRuntimeSnapshot(t *testing.T, response generated.ServerStatusResponse) {
	t.Helper()

	assertEqual(t, "runtime go version", response.Runtime.GoVersion, runtime.Version())
	assertEqual(t, "runtime operating system", response.Runtime.OperatingSystem, runtime.GOOS)
	assertEqual(t, "runtime architecture", response.Runtime.Architecture, runtime.GOARCH)
	assertEqual(t, "runtime disk path", response.Runtime.DiskUsage.Path, defaultDiskUsagePath())
	if response.Runtime.CpuCores < 1 {
		t.Fatalf("expected cpu cores to be positive, got %d", response.Runtime.CpuCores)
	}
	if response.Runtime.Goroutines < 1 {
		t.Fatalf("expected goroutines to be positive, got %d", response.Runtime.Goroutines)
	}
	if response.Runtime.HostMemoryTotalBytes == 0 {
		t.Fatalf("expected host memory total bytes to be positive")
	}
	if response.Runtime.HostMemoryUsedBytes > response.Runtime.HostMemoryTotalBytes {
		t.Fatalf("expected host memory used bytes to be within total bytes")
	}
	if response.Runtime.HostMemoryFreeBytes > response.Runtime.HostMemoryTotalBytes {
		t.Fatalf("expected host memory free bytes to be within total bytes")
	}
	if response.Runtime.HostMemoryUsedPercent < 0 {
		t.Fatalf("expected host memory used percent to be non-negative")
	}
	if response.Runtime.RuntimeSysBytes == 0 {
		t.Fatalf("expected runtime sys bytes to be positive")
	}
	if response.Runtime.DiskUsage.TotalBytes == 0 {
		t.Fatalf("expected runtime disk usage total bytes to be positive")
	}
	if response.Runtime.DiskUsage.UsedBytes > response.Runtime.DiskUsage.TotalBytes {
		t.Fatalf("expected runtime disk used bytes to be within total bytes")
	}
	if response.Runtime.DiskUsage.UsedPercent < 0 {
		t.Fatalf("expected runtime disk usage percent to be non-negative")
	}
	if response.Runtime.LoadAverage.OneMinute < 0 ||
		response.Runtime.LoadAverage.FiveMinutes < 0 ||
		response.Runtime.LoadAverage.FifteenMinutes < 0 {
		t.Fatalf("expected runtime load averages to be non-negative")
	}
}

func assertCurrentSliceTrendSnapshot(t *testing.T, response generated.ServerStatusResponse) {
	t.Helper()

	assertEqual(t, "trend range", string(response.Trend.Range), monitorcontract.TrendRange10Minutes.String())
	assertEqual(t, "trend retention seconds", response.Trend.RetentionSeconds, int64(monitorcontract.TrendRange10Minutes.Duration().Seconds()))
	assertEqual(t, "trend sample interval seconds", response.Trend.SampleIntervalSeconds, int64(trendSampleInterval.Seconds()))
	if len(response.Trend.Points) != 0 {
		t.Fatalf("expected empty trend points without redis sampler, got %d", len(response.Trend.Points))
	}
}

func assertCurrentSliceSummary(t *testing.T, response generated.ServerStatusResponse) {
	t.Helper()

	assertEqual(t, "summary total dependencies", response.Summary.TotalDependencies, 2)
	assertEqual(t, "summary healthy dependencies", response.Summary.HealthyDependencies, 1)
	assertEqual(t, "summary disabled dependencies", response.Summary.DisabledDependencies, 1)
	assertEqual(t, "summary degraded dependencies", response.Summary.DegradedDependencies, 0)
	assertEqual(t, "summary unknown dependencies", response.Summary.UnknownDependencies, 0)
	assertEqual(t, "summary total modules", response.Summary.TotalModules, 4)
	assertEqual(t, "summary healthy modules", response.Summary.HealthyModules, 4)
}

func assertCurrentSliceModuleSummaries(t *testing.T, actual []generated.ServerStatusModule) {
	t.Helper()

	assertModuleSummaries(t, actual, []generated.ServerStatusModule{
		{Name: "audit", Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: nil},
		{Name: "user", Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: nil},
		{Name: "rbac", Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: []string{"user"}},
		{Name: moduleID, Status: statusHealthy, StatusDetail: "Runtime metadata is present and platform signals are healthy", DependsOn: []string{"user", "rbac"}},
	})
}

func assertModuleSummaries(t *testing.T, actual []generated.ServerStatusModule, expected []generated.ServerStatusModule) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("expected %d module summaries, got %d", len(expected), len(actual))
	}

	for index, want := range expected {
		got := actual[index]
		if got.Name != want.Name ||
			got.Status != want.Status ||
			got.StatusDetail != want.StatusDetail ||
			!sameStrings(got.DependsOn, want.DependsOn) ||
			!sameOptionalStrings(got.MissingDependencies, want.MissingDependencies) {
			t.Fatalf(
				"expected module summary %s at index %d to be %s, got %s",
				want.Name,
				index,
				formatModuleSummary(want),
				formatModuleSummary(got),
			)
		}
	}
}

func sameStrings(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}

	for index := range actual {
		if actual[index] != expected[index] {
			return false
		}
	}

	return true
}

func sameOptionalStrings(actual *[]string, expected *[]string) bool {
	switch {
	case actual == nil && expected == nil:
		return true
	case actual == nil || expected == nil:
		return false
	default:
		return sameStrings(*actual, *expected)
	}
}

func stringSlicePointer(values ...string) *[]string {
	if len(values) == 0 {
		return nil
	}
	items := append([]string(nil), values...)
	return &items
}

func formatModuleSummary(value generated.ServerStatusModule) string {
	return fmt.Sprintf(
		"{name:%s status:%s status_detail:%s depends_on:%v missing_dependencies:%v}",
		value.Name,
		value.Status,
		value.StatusDetail,
		value.DependsOn,
		value.MissingDependencies,
	)
}

func moduleWithStartedAt(db *sql.DB, startedAt time.Time) *Module {
	moduleInstance := &Module{db: db}
	moduleInstance.startedAtUnixNs.Store(startedAt.UnixNano())
	return moduleInstance
}

func stableRuntimeSnapshot() generated.ServerStatusRuntime {
	return generated.ServerStatusRuntime{
		GoVersion:       runtime.Version(),
		HostName:        "test-host",
		OperatingSystem: runtime.GOOS,
		Architecture:    runtime.GOARCH,
		CpuCores:        8,
		LoadAverage: generated.ServerStatusLoadAverage{
			OneMinute:      0.24,
			FiveMinutes:    0.19,
			FifteenMinutes: 0.15,
		},
		DiskUsage: generated.ServerStatusDiskUsage{
			Path:        defaultDiskUsagePath(),
			TotalBytes:  512 * 1024 * 1024 * 1024,
			UsedBytes:   128 * 1024 * 1024 * 1024,
			FreeBytes:   384 * 1024 * 1024 * 1024,
			UsedPercent: 25,
		},
		HostMemoryTotalBytes:  32 * 1024 * 1024 * 1024,
		HostMemoryUsedBytes:   12 * 1024 * 1024 * 1024,
		HostMemoryFreeBytes:   20 * 1024 * 1024 * 1024,
		HostMemoryUsedPercent: 37.5,
		Goroutines:            12,
		RuntimeAllocBytes:     48 * 1024 * 1024,
		RuntimeHeapInUseBytes: 28 * 1024 * 1024,
		RuntimeSysBytes:       96 * 1024 * 1024,
		RuntimeGcCycles:       4,
	}
}

func TestDefaultDiskUsagePathForGOOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		goos     string
		envValue string
		want     string
	}{
		{name: "non-windows", goos: "linux", want: "/"},
		{name: "windows uses system drive", goos: "windows", envValue: "D:", want: "D:\\"},
		{name: "windows defaults to c drive", goos: "windows", want: "C:\\"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := config.DefaultDiskUsagePathForGOOS(tc.goos, func(string) string {
				return tc.envValue
			})

			if got != tc.want {
				t.Fatalf("expected disk usage path %q, got %q", tc.want, got)
			}
		})
	}
}
