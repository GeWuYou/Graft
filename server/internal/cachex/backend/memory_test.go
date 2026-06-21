package backend

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBackendSetGetDeleteAndExpire(t *testing.T) {
	backend := NewMemory()
	backend.now = func() time.Time {
		return time.Unix(100, 0)
	}

	entry := Entry{
		Value:     []byte("cached"),
		ExpiresAt: time.Unix(101, 0),
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
	if string(stored.Value) != "cached" {
		t.Fatalf("expected cached payload, got %q", string(stored.Value))
	}

	backend.now = func() time.Time {
		return time.Unix(102, 0)
	}
	if _, ok, err := backend.Get(context.Background(), "alpha"); err != nil {
		t.Fatalf("get expired: %v", err)
	} else if ok {
		t.Fatal("expected expired entry to miss")
	}

	if err := backend.Set(context.Background(), "beta", Entry{Value: []byte("keep")}); err != nil {
		t.Fatalf("set beta: %v", err)
	}
	if err := backend.Delete(context.Background(), "beta"); err != nil {
		t.Fatalf("delete beta: %v", err)
	}
	if _, ok, err := backend.Get(context.Background(), "beta"); err != nil {
		t.Fatalf("get beta: %v", err)
	} else if ok {
		t.Fatal("expected deleted entry to miss")
	}
}
