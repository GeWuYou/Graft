package redisx

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"graft/server/internal/config"
)

func TestParseMetricsReadsOnlySupportedInfoFields(t *testing.T) {
	t.Parallel()

	metrics := parseMetrics("# Clients\nconnected_clients:4\nblocked_clients:0\nmaxclients:10000\n" +
		"# Memory\nused_memory:1024\nused_memory_peak:2048\nmaxmemory:4096\nmem_fragmentation_ratio:1.25\n" +
		"# Stats\ntotal_connections_received:11\ntotal_commands_processed:12\ninstantaneous_ops_per_sec:3.5\nkeyspace_hits:18\nkeyspace_misses:2\nexpired_keys:4\nevicted_keys:1\n" +
		"# Persistence\nrdb_last_save_time:1784368800\nrdb_bgsave_in_progress:0\naof_enabled:1\naof_rewrite_in_progress:0\n" +
		"# Replication\nrole:slave\nmaster_link_status:up\n" +
		"# Keyspace\ndb0:keys=10,expires=2,avg_ttl=500\ndb2:keys=0,expires=0,avg_ttl=0\nmaster_host:secret-host\n")

	if metrics.ConnectedClients == nil || *metrics.ConnectedClients != 4 || metrics.MemoryFragmentationRatio == nil || *metrics.MemoryFragmentationRatio != 1.25 {
		t.Fatalf("expected parsed client and memory metrics, got %#v", metrics)
	}
	if metrics.ReplicationRole == nil || *metrics.ReplicationRole != "replica" || metrics.MasterLinkStatus == nil || *metrics.MasterLinkStatus != "up" {
		t.Fatalf("expected sanitized replication metrics, got %#v", metrics)
	}
	if metrics.Keyspaces == nil || len(*metrics.Keyspaces) != 2 || (*metrics.Keyspaces)[0].Database != "db0" || (*metrics.Keyspaces)[1].Database != "db2" {
		t.Fatalf("expected only keyspace aggregates, got %#v", metrics.Keyspaces)
	}
	if (*metrics.Keyspaces)[1].Keys == nil || *(*metrics.Keyspaces)[1].Keys != 0 {
		t.Fatalf("expected observed zero-valued keyspace counter, got %#v", (*metrics.Keyspaces)[1])
	}
}

func TestOpenAppliesPoolOptions(t *testing.T) {
	server := miniredis.RunT(t)

	client, err := Open(context.Background(), config.RedisConfig{
		Addr:            server.Addr(),
		DB:              0,
		PoolSize:        17,
		MinIdleConns:    2,
		MaxIdleConns:    6,
		MaxActiveConns:  19,
		PoolTimeout:     1500 * time.Millisecond,
		ConnMaxIdleTime: 10 * time.Minute,
		ConnMaxLifetime: 45 * time.Minute,
	})
	if err != nil {
		t.Fatalf("open redis client: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Fatalf("close redis client: %v", closeErr)
		}
	})

	options := client.Options()
	assertEqual(t, "pool size", options.PoolSize, 17)
	assertEqual(t, "min idle connections", options.MinIdleConns, 2)
	assertEqual(t, "max idle connections", options.MaxIdleConns, 6)
	assertEqual(t, "max active connections", options.MaxActiveConns, 19)
	assertEqual(t, "pool timeout", options.PoolTimeout, 1500*time.Millisecond)
	assertEqual(t, "connection max idle time", options.ConnMaxIdleTime, 10*time.Minute)
	assertEqual(t, "connection max lifetime", options.ConnMaxLifetime, 45*time.Minute)
}

func TestHealthReporterReportsReachableRedis(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), PoolSize: 12})
	t.Cleanup(func() {
		_ = client.Close()
	})

	reporter := NewHealthReporter(client)
	report, err := reporter.Report(context.Background())
	if err != nil {
		t.Fatalf("report redis health: %v", err)
	}

	if !report.Configured {
		t.Fatal("expected redis health reporter to report configured state")
	}
	if !report.Reachable {
		t.Fatal("expected redis health reporter to report reachable state")
	}
	if report.Pool.Capacity != 12 {
		t.Fatalf("expected pool capacity 12, got %d", report.Pool.Capacity)
	}
}

func TestHealthReporterHandlesMissingClient(t *testing.T) {
	reporter := NewHealthReporter(nil)
	report, err := reporter.Report(context.Background())
	if err != nil {
		t.Fatalf("report missing client health: %v", err)
	}
	if report.Configured {
		t.Fatal("expected missing redis client to be reported as unconfigured")
	}
	if report.Reachable {
		t.Fatal("expected missing redis client to be unreachable")
	}
}

func assertEqual[T comparable](t *testing.T, label string, got T, want T) {
	t.Helper()

	if got != want {
		t.Fatalf("expected %s %v, got %v", label, want, got)
	}
}
