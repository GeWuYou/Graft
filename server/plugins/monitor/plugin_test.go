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
	"graft/server/internal/plugin"
	monitorcontract "graft/server/plugins/monitor/contract"
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
	response, err := buildServerStatusResponse(context.Background(), &plugin.Context{
		Config: &config.Config{
			App: config.AppConfig{
				Name: " graft ",
				Env:  " prod ",
			},
		},
		RuntimeMetadata: plugin.NewRuntimeMetadata([]plugin.Descriptor{
			{ID: "audit", PluginVersion: "0.1.0"},
			{ID: "user", PluginVersion: "0.2.0"},
			{ID: "rbac", PluginVersion: "0.3.0", Dependencies: []string{"user"}},
			{ID: pluginID, PluginVersion: pluginVersion, Dependencies: []string{"user", "rbac"}},
		}),
	}, pluginWithStartedAt(db, startedAt), monitorcontract.TrendRange10Minutes)
	if err != nil {
		t.Fatalf("build server status response: %v", err)
	}

	assertEqual(t, "overall status", response.Status, "healthy")
	assertEqual(t, "database status", response.Dependencies.Database.Status, "healthy")
	assertEqual(t, "database detail", response.Dependencies.Database.Detail, "Database ping succeeded")
	if response.Dependencies.Database.LatencyMs == nil {
		t.Fatalf("expected database latency to be recorded")
	}
	assertEqual(t, "redis status", response.Dependencies.Redis.Status, "disabled")
	assertEqual(t, "redis detail", response.Dependencies.Redis.Detail, "Redis client is not configured")
	assertEqual(t, "server version", response.Server.Version, fallbackServerVersion)
	assertEqual(t, "started_at", response.Server.StartedAt, startedAt.Format(time.RFC3339))
	assertEqual(t, "go version", response.Server.GoVersion, runtime.Version())
	assertEqual(t, "app name", response.Server.AppName, "graft")
	assertEqual(t, "app env", response.Server.AppEnv, "prod")
	assertEqual(t, "runtime go version", response.Runtime.GoVersion, runtime.Version())
	assertEqual(t, "runtime operating system", response.Runtime.OperatingSystem, runtime.GOOS)
	assertEqual(t, "runtime architecture", response.Runtime.Architecture, runtime.GOARCH)
	assertEqual(t, "trend range", response.Trend.Range, monitorcontract.TrendRange10Minutes.String())
	assertEqual(t, "trend retention seconds", response.Trend.RetentionSeconds, int64(monitorcontract.TrendRange10Minutes.Duration().Seconds()))
	assertEqual(t, "trend sample interval seconds", response.Trend.SampleIntervalSeconds, int64(trendSampleInterval.Seconds()))
	if len(response.Trend.Points) != 0 {
		t.Fatalf("expected empty trend points without redis sampler, got %d", len(response.Trend.Points))
	}

	if response.Runtime.CPUCores < 1 {
		t.Fatalf("expected cpu cores to be positive, got %d", response.Runtime.CPUCores)
	}
	if response.Runtime.Goroutines < 1 {
		t.Fatalf("expected goroutines to be positive, got %d", response.Runtime.Goroutines)
	}
	if response.Runtime.SystemMemoryBytes == 0 {
		t.Fatalf("expected runtime system memory to be positive")
	}
	if response.Server.UptimeSeconds < 5 {
		t.Fatalf("expected uptime to be at least 5 seconds, got %d", response.Server.UptimeSeconds)
	}

	assertEqual(t, "summary total dependencies", response.Summary.TotalDependencies, 2)
	assertEqual(t, "summary healthy dependencies", response.Summary.HealthyDependencies, 1)
	assertEqual(t, "summary disabled dependencies", response.Summary.DisabledDependencies, 1)
	assertEqual(t, "summary degraded dependencies", response.Summary.DegradedDependencies, 0)
	assertEqual(t, "summary unknown dependencies", response.Summary.UnknownDependencies, 0)
	assertEqual(t, "summary total plugins", response.Summary.TotalPlugins, 4)
	assertEqual(t, "summary healthy plugins", response.Summary.HealthyPlugins, 4)

	expectedPlugins := []serverStatusPlugin{
		{Name: "audit", Status: statusHealthy, Version: "0.1.0", DependsOn: nil},
		{Name: "user", Status: statusHealthy, Version: "0.2.0", DependsOn: nil},
		{Name: "rbac", Status: statusHealthy, Version: "0.3.0", DependsOn: []string{"user"}},
		{Name: pluginID, Status: statusHealthy, Version: pluginVersion, DependsOn: []string{"user", "rbac"}},
	}
	assertPluginSummaries(t, response.Plugins, expectedPlugins)
}

func TestBuildServerStatusResponseUsesUnknownWhenDatabaseServiceIsAbsent(t *testing.T) {
	t.Parallel()

	response, err := buildServerStatusResponse(context.Background(), &plugin.Context{
		Services: container.New(),
	}, nil, monitorcontract.TrendRange10Minutes)
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
	if response.Status != "unknown" {
		t.Fatalf("expected overall status unknown, got %q", response.Status)
	}
	if response.Trend.Range != monitorcontract.TrendRange10Minutes.String() {
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

	response, err := buildServerStatusResponse(context.Background(), &plugin.Context{}, &Plugin{db: db}, monitorcontract.TrendRange10Minutes)
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

func TestRuntimePluginSummariesFollowPlatformStatus(t *testing.T) {
	t.Parallel()

	pluginCtx := &plugin.Context{
		RuntimeMetadata: plugin.NewRuntimeMetadata([]plugin.Descriptor{
			{ID: "user", PluginVersion: "0.2.0"},
			{ID: "rbac", PluginVersion: "0.3.0", Dependencies: []string{"user"}},
			{ID: pluginID, PluginVersion: pluginVersion, Dependencies: []string{"user", "rbac"}},
		}),
	}

	healthy := runtimePluginSummaries(
		pluginCtx,
		dependencyStatus{Status: statusHealthy},
		dependencyStatus{Status: statusDisabled},
	)
	assertPluginSummaries(t, healthy, []serverStatusPlugin{
		{Name: "user", Status: statusHealthy, Version: "0.2.0", DependsOn: nil},
		{Name: "rbac", Status: statusHealthy, Version: "0.3.0", DependsOn: []string{"user"}},
		{Name: pluginID, Status: statusHealthy, Version: pluginVersion, DependsOn: []string{"user", "rbac"}},
	})

	degraded := runtimePluginSummaries(
		pluginCtx,
		dependencyStatus{Status: statusDegraded},
		dependencyStatus{Status: statusHealthy},
	)
	assertPluginSummaries(t, degraded, []serverStatusPlugin{
		{Name: "user", Status: statusDegraded, Version: "0.2.0", DependsOn: nil},
		{Name: "rbac", Status: statusDegraded, Version: "0.3.0", DependsOn: []string{"user"}},
		{Name: pluginID, Status: statusDegraded, Version: pluginVersion, DependsOn: []string{"user", "rbac"}},
	})
}

func TestRuntimePluginSummariesDegradeWhenDependencyMetadataIsMissing(t *testing.T) {
	t.Parallel()

	pluginCtx := &plugin.Context{
		RuntimeMetadata: plugin.NewRuntimeMetadata([]plugin.Descriptor{
			{ID: "audit", PluginVersion: "0.1.0"},
			{ID: pluginID, PluginVersion: pluginVersion, Dependencies: []string{"user", "rbac"}},
		}),
	}

	actual := runtimePluginSummaries(
		pluginCtx,
		dependencyStatus{Status: statusHealthy},
		dependencyStatus{Status: statusDisabled},
	)

	assertPluginSummaries(t, actual, []serverStatusPlugin{
		{Name: "audit", Status: statusHealthy, Version: "0.1.0", DependsOn: nil},
		{Name: pluginID, Status: statusDegraded, Version: pluginVersion, DependsOn: []string{"user", "rbac"}},
	})
}

func TestStopTrendSamplerRequiresLifecycleContext(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	pluginInstance := &Plugin{
		samplerCancel: func() {
			close(done)
		},
		samplerDone: done,
	}

	err := pluginInstance.stopTrendSampler(&plugin.Context{})
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
	points := []serverStatusTrendPoint{
		{
			ObservedAt:        observedAt.Add(-45 * time.Minute).Format(time.RFC3339),
			CPUPercent:        9.2,
			Goroutines:        11,
			AllocBytes:        32 * 1024 * 1024,
			HeapInUseBytes:    18 * 1024 * 1024,
			SystemMemoryBytes: 64 * 1024 * 1024,
		},
		{
			ObservedAt:        observedAt.Add(-20 * time.Minute).Format(time.RFC3339),
			CPUPercent:        14.4,
			Goroutines:        17,
			AllocBytes:        48 * 1024 * 1024,
			HeapInUseBytes:    28 * 1024 * 1024,
			SystemMemoryBytes: 80 * 1024 * 1024,
		},
		{
			ObservedAt:        observedAt.Add(-5 * time.Minute).Format(time.RFC3339),
			CPUPercent:        21.8,
			Goroutines:        23,
			AllocBytes:        60 * 1024 * 1024,
			HeapInUseBytes:    34 * 1024 * 1024,
			SystemMemoryBytes: 96 * 1024 * 1024,
		},
	}

	for _, point := range points {
		pointTime, err := time.Parse(time.RFC3339, point.ObservedAt)
		if err != nil {
			t.Fatalf("parse observed_at: %v", err)
		}
		if err := storeTrendPoint(ctx, redisClient, storageKey, pointTime, point); err != nil {
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
	for _, point := range []serverStatusTrendPoint{
		{
			ObservedAt:        observedAt.Add(-25 * time.Minute).Format(time.RFC3339),
			CPUPercent:        12.4,
			Goroutines:        15,
			AllocBytes:        40 * 1024 * 1024,
			HeapInUseBytes:    22 * 1024 * 1024,
			SystemMemoryBytes: 72 * 1024 * 1024,
		},
		{
			ObservedAt:        observedAt.Add(-8 * time.Minute).Format(time.RFC3339),
			CPUPercent:        18.7,
			Goroutines:        19,
			AllocBytes:        55 * 1024 * 1024,
			HeapInUseBytes:    30 * 1024 * 1024,
			SystemMemoryBytes: 88 * 1024 * 1024,
		},
	} {
		pointTime, err := time.Parse(time.RFC3339, point.ObservedAt)
		if err != nil {
			t.Fatalf("parse trend point time: %v", err)
		}
		if err := storeTrendPoint(ctx, redisClient, storageKey, pointTime, point); err != nil {
			t.Fatalf("store redis trend point: %v", err)
		}
	}

	response, err := buildServerStatusResponse(ctx, &plugin.Context{
		Config: &config.Config{
			App: config.AppConfig{
				Name: "graft",
			},
		},
		Redis: redisClient,
	}, pluginWithStartedAt(db, observedAt.Add(-5*time.Minute)), monitorcontract.TrendRange30Minutes)
	if err != nil {
		t.Fatalf("build response with redis trend: %v", err)
	}

	assertEqual(t, "redis trend range", response.Trend.Range, monitorcontract.TrendRange30Minutes.String())
	if len(response.Trend.Points) != 2 {
		t.Fatalf("expected 2 redis-backed trend points, got %d", len(response.Trend.Points))
	}
	if response.Trend.Points[1].CPUPercent != 18.7 {
		t.Fatalf("expected cpu percent from redis-backed trend point, got %v", response.Trend.Points[1].CPUPercent)
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

			if actual := deriveOverallStatus(testCase.databaseStatus, testCase.redisStatus); actual != testCase.expected {
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

func assertEqual[T comparable](t *testing.T, field string, actual T, expected T) {
	t.Helper()

	if actual != expected {
		t.Fatalf("expected %s %v, got %v", field, expected, actual)
	}
}

func assertPluginSummaries(t *testing.T, actual []serverStatusPlugin, expected []serverStatusPlugin) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("expected %d plugin summaries, got %d", len(expected), len(actual))
	}

	for index, want := range expected {
		got := actual[index]
		if got.Name != want.Name || got.Status != want.Status || got.Version != want.Version || !sameStrings(got.DependsOn, want.DependsOn) {
			t.Fatalf(
				"expected plugin summary %s at index %d to be %s, got %s",
				want.Name,
				index,
				formatPluginSummary(want),
				formatPluginSummary(got),
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

func formatPluginSummary(value serverStatusPlugin) string {
	return fmt.Sprintf("{name:%s status:%s version:%s depends_on:%v}", value.Name, value.Status, value.Version, value.DependsOn)
}

func pluginWithStartedAt(db *sql.DB, startedAt time.Time) *Plugin {
	pluginInstance := &Plugin{db: db}
	pluginInstance.startedAtUnixNs.Store(startedAt.UnixNano())
	return pluginInstance
}
