// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package redisx

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"graft/server/internal/config"
)

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
