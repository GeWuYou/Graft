package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/redisx"
	"graft/server/internal/statex"
	monitorcontract "graft/server/modules/monitor/contract"
)

func TestCollectPostgreSQLMetricsCollectsApprovedDiagnostics(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta(postgresActivityMetricsQuery)).WillReturnRows(
		sqlmock.NewRows([]string{"active", "idle", "idle_in_transaction", "waiting"}).AddRow(2, 5, 1, 3),
	)
	mock.ExpectQuery(regexp.QuoteMeta(postgresMaxConnectionsQuery)).WillReturnRows(
		sqlmock.NewRows([]string{"max_connections"}).AddRow(100),
	)
	mock.ExpectQuery(regexp.QuoteMeta(postgresDatabaseMetricsQuery)).WillReturnRows(
		sqlmock.NewRows([]string{
			"database_size", "commits", "rollbacks", "blocks_read", "blocks_hit", "tuples_returned", "tuples_fetched",
			"tuples_inserted", "tuples_updated", "tuples_deleted", "conflicts", "deadlocks", "temp_files", "temp_bytes",
		}).AddRow(1024, 12, 2, 3, 27, 100, 90, 8, 4, 1, 0, 0, 0, 0),
	)

	metrics, err := collectPostgreSQLMetrics(context.Background(), db)
	if err != nil {
		t.Fatalf("collect PostgreSQL metrics: %v", err)
	}
	if metrics == nil || metrics.ActiveConnections == nil || *metrics.ActiveConnections != 2 {
		t.Fatalf("expected activity metrics, got %#v", metrics)
	}
	if metrics.DatabaseSizeBytes == nil || *metrics.DatabaseSizeBytes != 1024 {
		t.Fatalf("expected database size, got %#v", metrics.DatabaseSizeBytes)
	}
	if metrics.CacheHitPercent == nil || *metrics.CacheHitPercent != 90 {
		t.Fatalf("expected cache hit percentage 90, got %#v", metrics.CacheHitPercent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verify SQL expectations: %v", err)
	}
}

func TestCollectPostgreSQLMetricsPreservesPartialResults(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta(postgresActivityMetricsQuery)).WillReturnError(errors.New("statistics denied"))
	mock.ExpectQuery(regexp.QuoteMeta(postgresMaxConnectionsQuery)).WillReturnRows(
		sqlmock.NewRows([]string{"max_connections"}).AddRow(100),
	)
	mock.ExpectQuery(regexp.QuoteMeta(postgresDatabaseMetricsQuery)).WillReturnError(errors.New("database statistics denied"))

	metrics, err := collectPostgreSQLMetrics(context.Background(), db)
	if err != nil {
		t.Fatalf("partial PostgreSQL metrics should remain usable: %v", err)
	}
	if metrics == nil || metrics.MaxConnections == nil || *metrics.MaxConnections != 100 {
		t.Fatalf("expected retained connection limit, got %#v", metrics)
	}
	if metrics.ActiveConnections != nil || metrics.DatabaseSizeBytes != nil {
		t.Fatalf("expected failed metric groups to remain null, got %#v", metrics)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verify SQL expectations: %v", err)
	}
}

func TestDatabaseHealthRetainsSnapshotWhenMetricsAreUnavailable(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectPing()
	mock.ExpectQuery(regexp.QuoteMeta(postgresActivityMetricsQuery)).WillReturnError(errors.New("statistics unavailable"))
	mock.ExpectQuery(regexp.QuoteMeta(postgresMaxConnectionsQuery)).WillReturnError(errors.New("settings unavailable"))
	mock.ExpectQuery(regexp.QuoteMeta(postgresDatabaseMetricsQuery)).WillReturnError(errors.New("database unavailable"))

	dependency, err := databaseHealth(context.Background(), &Module{db: db})
	if err != nil {
		t.Fatalf("database health: %v", err)
	}
	if dependency.Status != statusHealthy || dependency.LatencyMs == nil || dependency.PostgresqlMetrics != nil {
		t.Fatalf("expected healthy snapshot with null optional metrics, got %#v", dependency)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verify SQL expectations: %v", err)
	}
}

func TestRedisHealthRetainsSnapshotWhenMetricsAreUnavailable(t *testing.T) {
	t.Parallel()

	reporter := monitorRedisHealthReporterStub{
		report:     redisx.HealthReport{Configured: true, Reachable: true, Latency: time.Millisecond},
		metricsErr: errors.New("INFO unavailable"),
	}
	dependency, err := redisHealth(context.Background(), nil, &Module{redisHealth: reporter})
	if err != nil {
		t.Fatalf("redis health: %v", err)
	}
	if dependency.Status != statusHealthy || dependency.LatencyMs == nil || dependency.RedisMetrics != nil {
		t.Fatalf("expected healthy snapshot with null optional metrics, got %#v", dependency)
	}
}

func TestRedisHealthMapsCollectedMetrics(t *testing.T) {
	t.Parallel()

	clients := int64(4)
	keyspaces := []redisx.KeyspaceMetrics{{Database: "db0", Keys: int64Ptr(10)}}
	reporter := monitorRedisHealthReporterStub{
		report: redisx.HealthReport{Configured: true, Reachable: true, Latency: time.Millisecond},
		metrics: redisx.Metrics{
			ConnectedClients: &clients,
			Keyspaces:        &keyspaces,
		},
	}
	dependency, err := redisHealth(context.Background(), nil, &Module{redisHealth: reporter})
	if err != nil {
		t.Fatalf("redis health: %v", err)
	}
	if dependency.RedisMetrics == nil || dependency.RedisMetrics.ConnectedClients == nil || *dependency.RedisMetrics.ConnectedClients != clients {
		t.Fatalf("expected Redis metrics in dependency snapshot, got %#v", dependency)
	}
	if dependency.RedisMetrics.Keyspaces == nil || len(*dependency.RedisMetrics.Keyspaces) != 1 || (*dependency.RedisMetrics.Keyspaces)[0].Database != "db0" {
		t.Fatalf("expected mapped Redis keyspace metrics, got %#v", dependency.RedisMetrics.Keyspaces)
	}
}

func TestRecordDependencyHistorySamplesStoresProbeAggregatesWithBoundedRetention(t *testing.T) {
	t.Parallel()

	store := &monitorTrendStoreStub{}
	observedAt := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	latency := float32(7.5)
	recordDependencyHistorySamples(
		context.Background(),
		store,
		dependencyHistorySampleInput{
			appName:    "graft",
			hostName:   "host-a",
			observedAt: observedAt,
			database:   generated.ServerStatusDependency{Status: statusHealthy, LatencyMs: &latency},
			redis:      generated.ServerStatusDependency{Status: statusDegraded},
		},
	)

	if len(store.appended) != 2 {
		t.Fatalf("expected PostgreSQL and Redis samples, got %d", len(store.appended))
	}
	assertDependencyHistoryRetention(t, store.appended[0], observedAt)
	assertDependencyHistoryKeys(t, store.appended)
	assertDependencyHistoryAggregates(t, store.appended, latency)
}

func TestRecordDependencyHistorySamplesSkipsDisabledAndUnknownDependencies(t *testing.T) {
	t.Parallel()

	store := &monitorTrendStoreStub{}
	recordDependencyHistorySamples(
		context.Background(),
		store,
		dependencyHistorySampleInput{
			appName:    "graft",
			hostName:   "host-a",
			observedAt: time.Now().UTC(),
			database:   generated.ServerStatusDependency{Status: statusUnknown},
			redis:      generated.ServerStatusDependency{Status: statusDisabled},
		},
	)
	if len(store.appended) != 0 {
		t.Fatalf("expected no synthetic samples for unavailable probes, got %d", len(store.appended))
	}
}

func jsonUnmarshalHistoryPoint(sample statex.TimeSeriesSample, point *generated.ServerStatusDependencyHistoryPoint) error {
	return json.Unmarshal(sample.Payload, point)
}

func assertDependencyHistoryRetention(t *testing.T, sample appendedTrendSample, observedAt time.Time) {
	t.Helper()
	if sample.policy.TrimBefore != observedAt.Add(-time.Hour) || sample.policy.ExpiresAfter != trendStorageTTL {
		t.Fatalf("unexpected dependency history retention: %#v", sample.policy)
	}
}

func assertDependencyHistoryKeys(t *testing.T, samples []appendedTrendSample) {
	t.Helper()
	if samples[0].key != dependencyHistoryStorageKey("graft", "host-a", monitorcontract.DependencyKindPostgreSQL) {
		t.Fatalf("unexpected PostgreSQL history key %q", samples[0].key)
	}
	if samples[1].key != dependencyHistoryStorageKey("graft", "host-a", monitorcontract.DependencyKindRedis) {
		t.Fatalf("unexpected Redis history key %q", samples[1].key)
	}
}

func assertDependencyHistoryAggregates(t *testing.T, samples []appendedTrendSample, latency float32) {
	t.Helper()
	healthy := mustDependencyHistoryPoint(t, samples[0].sample)
	degraded := mustDependencyHistoryPoint(t, samples[1].sample)
	if healthy.ProbeCount == nil || *healthy.ProbeCount != 1 || healthy.AvailabilityPercent == nil || *healthy.AvailabilityPercent != 100 || healthy.LatencyP95Ms == nil || *healthy.LatencyP95Ms != latency {
		t.Fatalf("unexpected healthy aggregate: %#v", healthy)
	}
	if degraded.ProbeCount == nil || *degraded.ProbeCount != 1 || degraded.FailureCount == nil || *degraded.FailureCount != 1 || degraded.LatencyAverageMs != nil {
		t.Fatalf("unexpected degraded aggregate: %#v", degraded)
	}
}

func mustDependencyHistoryPoint(t *testing.T, sample statex.TimeSeriesSample) generated.ServerStatusDependencyHistoryPoint {
	t.Helper()
	var point generated.ServerStatusDependencyHistoryPoint
	if err := jsonUnmarshalHistoryPoint(sample, &point); err != nil {
		t.Fatalf("decode dependency history sample: %v", err)
	}
	return point
}
