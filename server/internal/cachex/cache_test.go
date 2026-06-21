package cachex

import (
	"context"
	"errors"
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

func TestCacheGetOrLoadIgnoresLeaderCancellation(t *testing.T) {
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
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	defer cancelLeader()

	started := make(chan struct{})
	release := make(chan struct{})
	loader := func(ctx context.Context) (Item, error) {
		close(started)
		<-release
		select {
		case <-ctx.Done():
			return Item{}, ctx.Err()
		default:
		}

		return NewItem([]byte("enabled"), 0), nil
	}

	leaderResult := make(chan error, 1)
	go func() {
		_, getErr := cache.GetOrLoad(leaderCtx, key, loader)
		leaderResult <- getErr
	}()

	<-started
	cancelLeader()

	followerResult := make(chan error, 1)
	go func() {
		item, getErr := cache.GetOrLoad(context.Background(), key, loader)
		if getErr == nil && string(item.Value) != "enabled" {
			getErr = errors.New("unexpected cached payload")
		}
		followerResult <- getErr
	}()

	close(release)

	if err := <-leaderResult; err != nil {
		t.Fatalf("leader get or load: %v", err)
	}
	if err := <-followerResult; err != nil {
		t.Fatalf("follower get or load: %v", err)
	}
}
