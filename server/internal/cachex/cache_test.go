// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cachex

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"graft/server/internal/cachex/backend"
	"graft/server/internal/cachex/keys"
)

func TestCacheGetOrLoadCollapsesConcurrentMisses(t *testing.T) {
	manager, err := NewManager(ManagerOptions{
		Backend:   backend.NewMemory(),
		Namespace: "runtime",
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	cache, err := manager.NewCache("settings", WithTTL(time.Minute))
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}

	key := keys.MustNew("system-config", "effective", "auth")
	var loaderCalls atomic.Int32
	var wg sync.WaitGroup
	results := make(chan Item, 8)
	errs := make(chan error, 8)

	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, getErr := cache.GetOrLoad(context.Background(), key, func(context.Context) (Item, error) {
				loaderCalls.Add(1)
				time.Sleep(10 * time.Millisecond)
				return NewItem([]byte("enabled"), 0), nil
			})
			if getErr != nil {
				errs <- getErr
				return
			}
			results <- item
		}()
	}

	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("get or load: %v", err)
		}
	}
	if got := loaderCalls.Load(); got != 1 {
		t.Fatalf("expected exactly one loader call, got %d", got)
	}

	for item := range results {
		if string(item.Value) != "enabled" {
			t.Fatalf("expected cached payload, got %q", string(item.Value))
		}
	}
}
