// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package kvx

import (
	"context"
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	return c.now
}

func TestMemoryPutGetDelete(t *testing.T) {
	t.Parallel()

	store := NewMemory(MemoryOptions{
		Clock: &fixedClock{now: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)},
	})

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
		t.Fatal("expected expiry timestamp")
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

func TestMemoryCompareAndDelete(t *testing.T) {
	t.Parallel()

	store := NewMemory(MemoryOptions{
		Clock: &fixedClock{now: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)},
	})

	if err := store.Put(context.Background(), "alpha", []byte("payload"), time.Minute); err != nil {
		t.Fatalf("put: %v", err)
	}

	deleted, err := store.CompareAndDelete(context.Background(), "alpha", []byte("payload"))
	if err != nil {
		t.Fatalf("compare and delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected compare and delete success")
	}

	if _, ok, err := store.Get(context.Background(), "alpha"); err != nil {
		t.Fatalf("get after compare delete: %v", err)
	} else if ok {
		t.Fatal("expected deleted value to be absent")
	}
}

func TestMemoryExpiresEntries(t *testing.T) {
	t.Parallel()

	clock := &fixedClock{now: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)}
	store := NewMemory(MemoryOptions{Clock: clock})

	if err := store.Put(context.Background(), "alpha", []byte("payload"), time.Second); err != nil {
		t.Fatalf("put: %v", err)
	}

	clock.now = clock.now.Add(2 * time.Second)
	if _, ok, err := store.Get(context.Background(), "alpha"); err != nil {
		t.Fatalf("get after expiry: %v", err)
	} else if ok {
		t.Fatal("expected expired value to be absent")
	}
}

func TestMemoryPreservesEmptyByteSlices(t *testing.T) {
	t.Parallel()

	store := NewMemory(MemoryOptions{
		Clock: &fixedClock{now: time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)},
	})

	if err := store.Put(context.Background(), "alpha", []byte{}, 0); err != nil {
		t.Fatalf("put empty bytes: %v", err)
	}

	item, ok, err := store.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("get empty bytes: %v", err)
	}
	if !ok {
		t.Fatal("expected stored empty value")
	}
	if item.Value == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(item.Value) != 0 {
		t.Fatalf("expected zero-length slice, got %d", len(item.Value))
	}
}
