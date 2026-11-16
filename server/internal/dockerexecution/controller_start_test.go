package dockerexecution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type controllerStartFailureCase struct {
	name            string
	startStatus     int
	inspectStatus   int
	state           string
	removeStatus    int
	expectedOutcome string
	expectInspect   bool
	expectRemove    bool
}

func TestControllerStartFailurePreservesUncertainEffects(t *testing.T) {
	for _, testCase := range []controllerStartFailureCase{
		{name: "transport lost", expectedOutcome: "needs_attention"},
		{name: "server error", startStatus: 500, expectedOutcome: "needs_attention"},
		{name: "confirmed created and removed", startStatus: 409, inspectStatus: 200, state: `{"Status":"created","Running":false,"StartedAt":"0001-01-01T00:00:00Z"}`, removeStatus: 204, expectedOutcome: "failed", expectInspect: true, expectRemove: true},
		{name: "running", startStatus: 409, inspectStatus: 200, state: `{"Status":"running","Running":true}`, expectedOutcome: "needs_attention", expectInspect: true},
		{name: "already exited", startStatus: 409, inspectStatus: 200, state: `{"Status":"exited","Running":false,"StartedAt":"2026-11-15T12:00:00Z"}`, expectedOutcome: "needs_attention", expectInspect: true},
		{name: "state missing", startStatus: 409, inspectStatus: 200, state: `null`, expectedOutcome: "needs_attention", expectInspect: true},
		{name: "timestamp missing", startStatus: 409, inspectStatus: 200, state: `{"Status":"created","Running":false}`, expectedOutcome: "needs_attention", expectInspect: true},
		{name: "inspect failed", startStatus: 409, inspectStatus: 500, expectedOutcome: "needs_attention", expectInspect: true},
		{name: "remove failed", startStatus: 409, inspectStatus: 200, state: `{"Status":"created","Running":false,"StartedAt":"0001-01-01T00:00:00Z"}`, removeStatus: 500, expectedOutcome: "needs_attention", expectInspect: true, expectRemove: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server, inspected, removed := newControllerStartFailureEngine(t, testCase)
			defer server.Close()
			request := Request{Capability: updateControllerCapability, Protocol: updateControllerProtocol, OperationID: updateControllerOperation, Input: json.RawMessage(`{"operation_id":"update-progress","source_version":"1.0.0","target_version":"1.1.0"}`)}
			result := Execute(context.Background(), controllerProgressEnvironment(t, server.URL), request)
			if result.Outcome != testCase.expectedOutcome {
				t.Fatalf("start failure result = %#v, want %s", result, testCase.expectedOutcome)
			}
			if result.Outcome == "needs_attention" && result.FailureCode != failureInterrupted {
				t.Fatalf("uncertain start failure code = %s", result.FailureCode)
			}
			if inspected.Load() != testCase.expectInspect || removed.Load() != testCase.expectRemove {
				t.Fatalf("cleanup effects inspected=%v removed=%v", inspected.Load(), removed.Load())
			}
		})
	}
}

func newControllerStartFailureEngine(t *testing.T, testCase controllerStartFailureCase) (*httptest.Server, *atomic.Bool, *atomic.Bool) {
	t.Helper()
	inspected, removed := &atomic.Bool{}, &atomic.Bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.55")
		case strings.HasSuffix(r.URL.Path, "/images/create"):
			_, _ = w.Write([]byte(`{"status":"complete"}`))
		case strings.HasSuffix(r.URL.Path, "/containers/create"):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id":"controller","Warnings":[]}`))
		case strings.HasSuffix(r.URL.Path, "/containers/controller/start"):
			if testCase.startStatus == 0 {
				connection, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Errorf("hijack start response: %v", err)
					return
				}
				_ = connection.Close()
				return
			}
			w.WriteHeader(testCase.startStatus)
			_, _ = w.Write([]byte(`{"message":"controller start rejected"}`))
		case strings.HasSuffix(r.URL.Path, "/containers/controller/json"):
			inspected.Store(true)
			if !testCase.expectInspect {
				t.Errorf("unexpected inspection after ambiguous start")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(testCase.inspectStatus)
			_, _ = w.Write([]byte(`{"Id":"controller","State":` + testCase.state + `}`))
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/containers/controller"):
			removed.Store(true)
			if !testCase.expectRemove {
				t.Errorf("unexpected removal without unstarted confirmation")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(testCase.removeStatus)
		default:
			t.Errorf("unexpected Docker request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, inspected, removed
}
