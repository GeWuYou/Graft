package update

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/httpx"
)

func TestRolloutStartFailureKeepsCauseAndSafeDetails(t *testing.T) {
	cause := errors.New("docker rejected " + rolloutDiagnosticSensitiveInput(t, 0))
	err := newRolloutStartFailure(rolloutFailureOperationStartFailed, "runner_launch", "update-91", cause)

	if !errors.Is(err, cause) {
		t.Fatal("expected rollout failure to preserve the original cause")
	}
	code, stage, operationID := rolloutFailureDetails(err)
	if code != rolloutFailureOperationStartFailed || stage != "runner_launch" || operationID != "update-91" {
		t.Fatalf("unexpected rollout failure details: %q / %q / %q", code, stage, operationID)
	}
	if got := sanitizeRolloutError(err); strings.Contains(got, cause.Error()) || !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected sensitive error value to be redacted, got %q", got)
	}
}

func TestSanitizeRolloutErrorRedactsAuthorizationCredentials(t *testing.T) {
	for index, name := range []string{"bearer", "basic"} {
		credential := rolloutDiagnosticSensitiveInput(t, index+1)
		t.Run(name, func(t *testing.T) {
			got := sanitizeRolloutError(errors.New("docker rejected " + credential))
			if strings.Contains(got, credential) || !strings.Contains(got, "[REDACTED]") {
				t.Fatalf("expected authorization credential to be redacted, got %q", got)
			}
		})
	}
}

func TestWriteStartFailureLogsSanitizedCauseAndReturnsSafeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := newStartFailureTestHandler(t)
	cause := errors.New("docker create failed: " + rolloutDiagnosticSensitiveInput(t, 3))
	err := newRolloutStartFailure(rolloutFailureOperationStartFailed, "runner_launch", "update-91", cause)

	fixture.handler.writeStartFailure(fixture.context, 7, "0.11.0-beta.9", "candidate-91", err)
	assertSafeStartFailureResponse(t, fixture.recorder, cause)
	assertSanitizedStartFailureLog(t, fixture.entries)
}

type startFailureTestFixture struct {
	recorder *httptest.ResponseRecorder
	context  *gin.Context
	handler  updateRouteHandlers
	entries  *observer.ObservedLogs
}

func newStartFailureTestHandler(t *testing.T) startFailureTestFixture {
	t.Helper()
	core, entries := observer.New(zap.ErrorLevel)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/platform/updates/operations", nil)
	request.Header.Set(httpx.RequestIDHeader, "request-91")
	request.Header.Set("X-Trace-Id", "trace-91")
	ctx.Request = request
	return startFailureTestFixture{recorder: recorder, context: ctx, handler: updateRouteHandlers{logger: zap.New(core)}, entries: entries}
}

func assertSafeStartFailureResponse(t *testing.T, recorder *httptest.ResponseRecorder, cause error) {
	t.Helper()

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal failure status, got %d", recorder.Code)
	}
	if recorder.Header().Get(httpx.RequestIDHeader) != "request-91" {
		t.Fatalf("expected request ID response header, got %q", recorder.Header().Get(httpx.RequestIDHeader))
	}
	if strings.Contains(recorder.Body.String(), cause.Error()) {
		t.Fatalf("response leaked original cause: %s", recorder.Body.String())
	}
	var payload httpx.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Code != rolloutFailureOperationStartFailed || payload.MessageKey != "update.operation.start.operation_start_failed" {
		t.Fatalf("unexpected safe error response: %#v", payload)
	}
	data, ok := payload.Data.(map[string]any)
	if !ok || data["reason"] != rolloutFailureOperationStartFailed {
		t.Fatalf("expected safe failure reason, got %#v", payload.Data)
	}
}

func assertSanitizedStartFailureLog(t *testing.T, entries *observer.ObservedLogs) {
	t.Helper()
	logs := entries.FilterMessage("platform update rollout start failed").All()
	if len(logs) != 1 {
		t.Fatalf("expected one rollout failure log, got %#v", logs)
	}
	fields := logs[0].ContextMap()
	for key, want := range map[string]any{
		"request_id": "request-91", "trace_id": "trace-91", "actor_id": uint64(7), "target_version": "0.11.0-beta.9", "compose_candidate_key": "candidate-91", "failure_code": rolloutFailureOperationStartFailed, "failure_stage": "runner_launch", "operation_id": "update-91",
	} {
		if fields[key] != want {
			t.Fatalf("log field %s = %#v, want %#v", key, fields[key], want)
		}
	}
	loggedCause, _ := fields["error"].(string)
	if strings.Contains(loggedCause, rolloutDiagnosticSensitiveInput(t, 3)) || !strings.Contains(loggedCause, "[REDACTED]") {
		t.Fatalf("expected sanitized log cause, got %q", loggedCause)
	}
}

func rolloutDiagnosticSensitiveInput(t *testing.T, index int) string {
	t.Helper()
	contents, err := os.ReadFile("testdata/rollout-diagnostics/sensitive-inputs.txt")
	if err != nil {
		t.Fatalf("read rollout diagnostics sensitive inputs: %v", err)
	}
	inputs := strings.Split(strings.TrimSpace(string(contents)), "\n")
	if index >= len(inputs) {
		t.Fatalf("sensitive input index %d is unavailable", index)
	}
	return inputs[index]
}

func TestRolloutPreflightFailureCodesAreStable(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{err: errRolloutInvalidArgument, want: rolloutFailureInvalidTarget},
		{err: errRolloutCatalogStale, want: rolloutFailureCatalogStale},
		{err: errRolloutInstallationUnavailable, want: rolloutFailureInstallationUnavailable},
		{err: errRolloutSourceVersionUnsupported, want: rolloutFailureSourceVersionUnsupported},
		{err: errRolloutComposeCandidateInvalid, want: rolloutFailureComposeCandidateInvalid},
		{err: errRolloutComposePreflightFailed, want: rolloutFailureComposePreflightFailed},
	}
	for _, testCase := range tests {
		if got := classifyRolloutPreflightFailure(testCase.err); got != testCase.want {
			t.Fatalf("classify %v = %q, want %q", testCase.err, got, testCase.want)
		}
	}
}
