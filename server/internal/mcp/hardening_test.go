package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRuntimeLifecycleRejectsRequestsAfterClose(t *testing.T) {
	lifecycle, err := newRuntimeLifecycle(testRuntimeLimits(), nil)
	if err != nil {
		t.Fatalf("new lifecycle: %v", err)
	}
	handler := lifecycle.httpHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, HTTPPath, nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("initial request status = %d, want %d", first.Code, http.StatusNoContent)
	}
	lifecycle.Close()
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, HTTPPath, nil))
	if second.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed runtime status = %d, want %d", second.Code, http.StatusServiceUnavailable)
	}
	metrics := lifecycle.Metrics()
	if metrics.RequestsTotal != 1 || metrics.ActiveRequests != 0 {
		t.Fatalf("unexpected lifecycle metrics: %#v", metrics)
	}
}

func TestRuntimeLifecycleLimitsConcurrentInvocations(t *testing.T) {
	limits := testRuntimeLimits()
	limits.MaxConcurrentRequests = 1
	lifecycle, err := newRuntimeLifecycle(limits, nil)
	if err != nil {
		t.Fatalf("new lifecycle: %v", err)
	}
	lifecycle.sem <- struct{}{}
	t.Cleanup(func() { <-lifecycle.sem })
	if err := lifecycle.acquire(t.Context()); err != errRuntimeOverloaded {
		t.Fatalf("acquire under concurrency limit = %v, want %v", err, errRuntimeOverloaded)
	}
	if got := lifecycle.Metrics().RequestsRejected; got != 1 {
		t.Fatalf("rejected metric = %d, want 1", got)
	}
}

func TestRuntimeLifecycleExpiresSessionBudget(t *testing.T) {
	limits := testRuntimeLimits()
	limits.MaxSessions = 1
	limits.SessionTimeout = time.Millisecond
	lifecycle, err := newRuntimeLifecycle(limits, nil)
	if err != nil {
		t.Fatalf("new lifecycle: %v", err)
	}
	lifecycle.mu.Lock()
	lifecycle.sessions["old"] = time.Now().Add(-time.Second)
	lifecycle.mu.Unlock()
	if !lifecycle.canOpenSession() {
		t.Fatal("expired session must not consume the admission budget")
	}
}
