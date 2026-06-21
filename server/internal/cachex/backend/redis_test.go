// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisBackendSetGetDelete(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	now := time.Unix(100, 0)
	backend, err := NewRedis(client, RedisOptions{
		Prefix: "graft-cache",
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new redis backend: %v", err)
	}

	entry := Entry{
		Value:     []byte("payload"),
		ExpiresAt: now.Add(2 * time.Minute),
	}
	if err := backend.Set(context.Background(), "alpha", entry); err != nil {
		t.Fatalf("set: %v", err)
	}

	stored, ok, err := backend.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(stored.Value) != "payload" {
		t.Fatalf("expected payload, got %q", string(stored.Value))
	}
	if stored.ExpiresAt.IsZero() {
		t.Fatal("expected redis backend to recover ttl-based expiration")
	}

	if err := backend.Delete(context.Background(), "alpha"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, err := backend.Get(context.Background(), "alpha"); err != nil {
		t.Fatalf("get after delete: %v", err)
	} else if ok {
		t.Fatal("expected delete to remove entry")
	}
}
