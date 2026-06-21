// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package kvx

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisPutGetDelete(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	store, err := NewRedis(client, RedisOptions{
		Prefix: "graft-kv",
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new redis store: %v", err)
	}

	if err := store.Put(context.Background(), "alpha", []byte("payload"), 2*time.Minute); err != nil {
		t.Fatalf("put: %v", err)
	}

	item, ok, err := store.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected stored value")
	}
	if string(item.Value) != "payload" {
		t.Fatalf("unexpected value %q", string(item.Value))
	}
	if item.ExpiresAt.IsZero() {
		t.Fatal("expected redis expiry timestamp")
	}

	if err := store.Delete(context.Background(), "alpha"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, err := store.Get(context.Background(), "alpha"); err != nil {
		t.Fatalf("get after delete: %v", err)
	} else if ok {
		t.Fatal("expected deleted value to be absent")
	}
}

func TestRedisCompareAndSwapAndExpiry(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	store, err := NewRedis(client, RedisOptions{Prefix: "graft-kv"})
	if err != nil {
		t.Fatalf("new redis store: %v", err)
	}

	if err := store.Put(context.Background(), "alpha", []byte("payload"), time.Second); err != nil {
		t.Fatalf("put: %v", err)
	}

	swapped, err := store.CompareAndSwap(context.Background(), "alpha", []byte("payload"), []byte("used"), time.Second)
	if err != nil {
		t.Fatalf("compare and swap: %v", err)
	}
	if !swapped {
		t.Fatal("expected compare and swap success")
	}

	item, ok, err := store.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("get after swap: %v", err)
	}
	if !ok || string(item.Value) != "used" {
		t.Fatalf("unexpected item after swap: %#v ok=%v", item, ok)
	}

	server.FastForward(2 * time.Second)
	if _, ok, err := store.Get(context.Background(), "alpha"); err != nil {
		t.Fatalf("get after expiry: %v", err)
	} else if ok {
		t.Fatal("expected expired value to be absent")
	}
}
