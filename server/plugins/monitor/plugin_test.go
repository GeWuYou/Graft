package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/plugin"
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
	}, pluginWithStartedAt(db, startedAt))
	if err != nil {
		t.Fatalf("build server status response: %v", err)
	}

	assertEqual(t, "overall status", response.Status, "healthy")
	assertEqual(t, "database status", response.Dependencies.Database.Status, "healthy")
	assertEqual(t, "redis status", response.Dependencies.Redis.Status, "disabled")
	assertEqual(t, "server version", response.Server.Version, fallbackServerVersion)
	assertEqual(t, "started_at", response.Server.StartedAt, startedAt.Format(time.RFC3339))
	assertEqual(t, "go version", response.Server.GoVersion, runtime.Version())
	assertEqual(t, "app name", response.Server.AppName, "graft")
	assertEqual(t, "app env", response.Server.AppEnv, "prod")

	if response.Server.UptimeSeconds < 5 {
		t.Fatalf("expected uptime to be at least 5 seconds, got %d", response.Server.UptimeSeconds)
	}

	expectedPlugins := []serverStatusPlugin{
		{Name: "audit", Status: "unknown", Version: "0.1.0"},
		{Name: "user", Status: "unknown", Version: "0.2.0"},
		{Name: "rbac", Status: "unknown", Version: "0.3.0"},
		{Name: pluginID, Status: "unknown", Version: pluginVersion},
	}
	assertPluginSummaries(t, response.Plugins, expectedPlugins)
}

func TestBuildServerStatusResponseUsesUnknownWhenDatabaseServiceIsAbsent(t *testing.T) {
	t.Parallel()

	response, err := buildServerStatusResponse(context.Background(), &plugin.Context{
		Services: container.New(),
	}, nil)
	if err != nil {
		t.Fatalf("build server status response: %v", err)
	}

	if response.Dependencies.Database.Status != "unknown" {
		t.Fatalf("expected database status unknown, got %q", response.Dependencies.Database.Status)
	}
	if response.Dependencies.Redis.Status != "disabled" {
		t.Fatalf("expected redis status disabled, got %q", response.Dependencies.Redis.Status)
	}
	if response.Status != "unknown" {
		t.Fatalf("expected overall status unknown, got %q", response.Status)
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

	response, err := buildServerStatusResponse(context.Background(), &plugin.Context{}, &Plugin{db: db})
	if err != nil {
		t.Fatalf("build server status response: %v", err)
	}

	if response.Dependencies.Database.Status != "degraded" {
		t.Fatalf("expected database status degraded on ping error, got %q", response.Dependencies.Database.Status)
	}
	if response.Status != "degraded" {
		t.Fatalf("expected overall status degraded on ping error, got %q", response.Status)
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
		if actual[index] != want {
			t.Fatalf(
				"expected plugin summary %s at index %d to be %s, got %s",
				want.Name,
				index,
				formatPluginSummary(want),
				formatPluginSummary(actual[index]),
			)
		}
	}
}

func formatPluginSummary(value serverStatusPlugin) string {
	return fmt.Sprintf("{name:%s status:%s version:%s}", value.Name, value.Status, value.Version)
}

func pluginWithStartedAt(db *sql.DB, startedAt time.Time) *Plugin {
	pluginInstance := &Plugin{db: db}
	pluginInstance.startedAtUnixNs.Store(startedAt.UnixNano())
	return pluginInstance
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
