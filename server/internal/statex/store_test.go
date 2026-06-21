// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package statex

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisTimeSeriesStoreAppendAndRange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	store, err := NewRedisTimeSeriesStore(client)
	if err != nil {
		t.Fatalf("new redis time-series store: %v", err)
	}

	observedAt := time.Date(2026, 6, 21, 9, 0, 0, 0, time.UTC)
	for _, sample := range []TimeSeriesSample{
		{ObservedAt: observedAt.Add(-45 * time.Minute), Payload: []byte(`{"point":"oldest"}`)},
		{ObservedAt: observedAt.Add(-20 * time.Minute), Payload: []byte(`{"point":"middle"}`)},
		{ObservedAt: observedAt.Add(-5 * time.Minute), Payload: []byte(`{"point":"latest"}`)},
	} {
		if err := store.Append(ctx, "graft:monitor:trend:test-host", sample, RetentionPolicy{
			TrimBefore:   observedAt.Add(-time.Hour),
			ExpiresAfter: 2 * time.Hour,
		}); err != nil {
			t.Fatalf("append sample: %v", err)
		}
	}

	samples, err := store.Range(ctx, "graft:monitor:trend:test-host", TimeSeriesQuery{
		StartAt: observedAt.Add(-30 * time.Minute),
		EndAt:   observedAt,
	})
	if err != nil {
		t.Fatalf("range samples: %v", err)
	}

	if len(samples) != 2 {
		t.Fatalf("expected 2 samples in 30m window, got %d", len(samples))
	}
	if string(samples[0].Payload) != `{"point":"middle"}` {
		t.Fatalf("expected middle payload first, got %s", samples[0].Payload)
	}
	if !samples[0].ObservedAt.Equal(observedAt.Add(-20 * time.Minute)) {
		t.Fatalf("expected middle observedAt, got %v", samples[0].ObservedAt)
	}
	if string(samples[1].Payload) != `{"point":"latest"}` {
		t.Fatalf("expected latest payload second, got %s", samples[1].Payload)
	}
	if !samples[1].ObservedAt.Equal(observedAt.Add(-5 * time.Minute)) {
		t.Fatalf("expected latest observedAt, got %v", samples[1].ObservedAt)
	}
}

func TestRedisTimeSeriesStoreAppliesRetentionTrimAndExpiry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	store, err := NewRedisTimeSeriesStore(client)
	if err != nil {
		t.Fatalf("new redis time-series store: %v", err)
	}

	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	if err := store.Append(ctx, "graft:monitor:trend:trim-host", TimeSeriesSample{
		ObservedAt: now.Add(-2 * time.Hour),
		Payload:    []byte(`{"point":"expired"}`),
	}, RetentionPolicy{
		TrimBefore:   now.Add(-3 * time.Hour),
		ExpiresAfter: 2 * time.Hour,
	}); err != nil {
		t.Fatalf("append expired sample: %v", err)
	}
	if err := store.Append(ctx, "graft:monitor:trend:trim-host", TimeSeriesSample{
		ObservedAt: now,
		Payload:    []byte(`{"point":"current"}`),
	}, RetentionPolicy{
		TrimBefore:   now.Add(-time.Hour),
		ExpiresAfter: 2 * time.Hour,
	}); err != nil {
		t.Fatalf("append current sample: %v", err)
	}

	samples, err := store.Range(ctx, "graft:monitor:trend:trim-host", TimeSeriesQuery{
		StartAt: now.Add(-4 * time.Hour),
		EndAt:   now,
	})
	if err != nil {
		t.Fatalf("range trimmed samples: %v", err)
	}

	if len(samples) != 1 {
		t.Fatalf("expected 1 sample after trimming, got %d", len(samples))
	}
	if string(samples[0].Payload) != `{"point":"current"}` {
		t.Fatalf("expected current payload after trim, got %s", samples[0].Payload)
	}

	ttl := server.TTL("graft:monitor:trend:trim-host")
	if ttl <= 0 || ttl > 2*time.Hour {
		t.Fatalf("expected positive ttl within configured expiry, got %v", ttl)
	}
}

func TestRedisTimeSeriesStoreKeepsDuplicatePayloadsAtDifferentTimes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	store, err := NewRedisTimeSeriesStore(client)
	if err != nil {
		t.Fatalf("new redis time-series store: %v", err)
	}

	base := time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC)
	for _, observedAt := range []time.Time{
		base.Add(-2 * time.Minute),
		base.Add(-time.Minute),
	} {
		if err := store.Append(ctx, "graft:monitor:trend:dup-host", TimeSeriesSample{
			ObservedAt: observedAt,
			Payload:    []byte(`{"point":"same"}`),
		}, RetentionPolicy{
			TrimBefore:   base.Add(-time.Hour),
			ExpiresAfter: time.Hour,
		}); err != nil {
			t.Fatalf("append duplicate payload sample: %v", err)
		}
	}

	samples, err := store.Range(ctx, "graft:monitor:trend:dup-host", TimeSeriesQuery{
		StartAt: base.Add(-5 * time.Minute),
		EndAt:   base,
	})
	if err != nil {
		t.Fatalf("range duplicate payload samples: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 duplicate-payload samples, got %d", len(samples))
	}
	if !samples[0].ObservedAt.Equal(base.Add(-2 * time.Minute)) {
		t.Fatalf("expected first observedAt to be preserved, got %v", samples[0].ObservedAt)
	}
	if !samples[1].ObservedAt.Equal(base.Add(-time.Minute)) {
		t.Fatalf("expected second observedAt to be preserved, got %v", samples[1].ObservedAt)
	}
}

func TestRedisTimeSeriesStoreReadsLegacyMembers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	if err := client.ZAdd(ctx, "graft:monitor:trend:legacy-host", redis.Z{
		Score:  float64(time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC).UnixMilli()),
		Member: `{"point":"legacy"}`,
	}).Err(); err != nil {
		t.Fatalf("seed legacy member: %v", err)
	}

	store, err := NewRedisTimeSeriesStore(client)
	if err != nil {
		t.Fatalf("new redis time-series store: %v", err)
	}

	samples, err := store.Range(ctx, "graft:monitor:trend:legacy-host", TimeSeriesQuery{
		StartAt: time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2026, 6, 21, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("range legacy members: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected 1 legacy sample, got %d", len(samples))
	}
	if string(samples[0].Payload) != `{"point":"legacy"}` {
		t.Fatalf("expected legacy payload, got %q", string(samples[0].Payload))
	}
	if samples[0].ObservedAt.IsZero() {
		t.Fatal("expected legacy sample observedAt to be reconstructed from score")
	}
}
