package httpx

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestRunRejectsConcurrentStart verifies the lifecycle guard rejects a second
// start while the first server instance still owns the runtime slot.
func TestRunRejectsConcurrentStart(t *testing.T) {
	server := NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- server.Run(ctx, "127.0.0.1:0")
	}()

	waitForRunningServer(t, server)

	if err := server.Run(context.Background(), "127.0.0.1:0"); err == nil {
		t.Fatal("expected concurrent run to fail")
	} else if err.Error() != "http server already running" {
		t.Fatalf("expected already running error, got %v", err)
	}

	cancel()

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("expected first run to shut down cleanly, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server run to exit")
	}
}

func waitForRunningServer(t *testing.T, server *Server) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		server.mu.Lock()
		running := server.server
		server.mu.Unlock()
		if running != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("timed out waiting for running server")
}

// TestDetachRunningServerClearsPointer verifies lifecycle teardown detaches the
// running pointer exactly once so later calls observe a clean state.
func TestDetachRunningServerClearsPointer(t *testing.T) {
	server := NewServer()
	running := &http.Server{}
	if err := server.bindRunningServer(running); err != nil {
		t.Fatalf("bind running server: %v", err)
	}

	first := server.detachRunningServer()
	if first != running {
		t.Fatalf("expected first detach to return bound server, got %v", first)
	}

	second := server.detachRunningServer()
	if second != nil {
		t.Fatalf("expected second detach to observe empty state, got %v", second)
	}
}
