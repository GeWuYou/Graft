package update

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/httpx"
	"graft/server/internal/logger"
)

func TestRolloutStartFailureKeepsCauseAndSafeDetails(t *testing.T) {
	sensitiveInput := rolloutDiagnosticSensitiveInput(t, 0)
	cause := errors.New("docker rejected " + sensitiveInput)
	err := newRolloutStartFailure(rolloutFailureOperationStartFailed, "runner_launch", "update-91", cause)

	if !errors.Is(err, cause) {
		t.Fatal("expected rollout failure to preserve the original cause")
	}
	code, stage, operationID := rolloutFailureDetails(err)
	if code != rolloutFailureOperationStartFailed || stage != "runner_launch" || operationID != "update-91" {
		t.Fatalf("unexpected rollout failure details: %q / %q / %q", code, stage, operationID)
	}
	if got := sanitizeRolloutError(err); strings.Contains(got, sensitiveInput) || !strings.Contains(got, "[REDACTED]") {
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
	sensitiveInput := rolloutDiagnosticSensitiveInput(t, 3)
	cause := errors.New("docker create failed: " + sensitiveInput)
	err := newRolloutStartFailure(rolloutFailureOperationStartFailed, "runner_launch", "update-91", cause)

	fixture.handler.writeStartFailure(fixture.context, 7, "0.11.0-beta.9", "candidate-91", err)
	assertSafeStartFailureResponse(t, fixture.recorder, sensitiveInput)
	assertSanitizedStartFailureLog(t, fixture.entries)
	if strings.Contains(fixture.diagnostics.value.Detail, sensitiveInput) || !strings.Contains(fixture.diagnostics.value.Detail, "[REDACTED]") {
		t.Fatalf("expected sanitized diagnostic detail, got %q", fixture.diagnostics.value.Detail)
	}
}

func TestGetFailureDiagnosticReturnsStoredSanitizedEvidence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "requestID", Value: "request-91"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/platform/updates/diagnostics/request-91", nil)
	store := &failureDiagnosticStoreRecorder{value: FailureDiagnostic{RequestID: "request-91", TargetVersion: "0.11.0-beta.9", FailureCode: rolloutFailureOperationStartFailed, FailureStage: "runner_launch", Summary: updateFailureDiagnosticSummary, Detail: "docker launch failed: [REDACTED]", OccurredAt: time.Now().UTC()}}

	updateRouteHandlers{diagnostics: store}.getFailureDiagnostic(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected diagnostic response status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "docker launch failed: [REDACTED]") {
		t.Fatalf("expected stored sanitized detail, got %s", recorder.Body.String())
	}
}

func TestGetOperationFailureDiagnosticUsesOperationIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "operationID", Value: "update-92"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/platform/updates/operations/update-92/diagnostic", nil)
	store := &failureDiagnosticStoreRecorder{value: FailureDiagnostic{RequestID: "request-92", OperationID: "update-92", TargetVersion: "0.11.0-beta.9", FailureCode: rolloutFailureRunnerTerminal, FailureStage: "runner_receipt", Summary: runnerFailureDiagnosticSummary, Detail: "runner reported a terminal failure (receipt_write_failed)", OccurredAt: time.Now().UTC()}}

	updateRouteHandlers{diagnostics: store}.getOperationFailureDiagnostic(ctx)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "update-92") {
		t.Fatalf("expected operation diagnostic response, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

type startFailureTestFixture struct {
	recorder    *httptest.ResponseRecorder
	context     *gin.Context
	handler     updateRouteHandlers
	entries     *observer.ObservedLogs
	diagnostics *failureDiagnosticStoreRecorder
}

type failureDiagnosticStoreRecorder struct {
	value FailureDiagnostic
	err   error
}

func (r *failureDiagnosticStoreRecorder) CreateFailureDiagnostic(_ context.Context, value FailureDiagnostic, _ uint64) error {
	r.value = value
	return nil
}

func (r *failureDiagnosticStoreRecorder) GetFailureDiagnostic(context.Context, string) (FailureDiagnostic, error) {
	if r.err != nil {
		return FailureDiagnostic{}, r.err
	}
	return r.value, nil
}

func (r *failureDiagnosticStoreRecorder) GetFailureDiagnosticByOperation(context.Context, string) (FailureDiagnostic, error) {
	if r.err != nil {
		return FailureDiagnostic{}, r.err
	}
	return r.value, nil
}

func newStartFailureTestHandler(t *testing.T) startFailureTestFixture {
	t.Helper()
	core, entries := observer.New(zap.ErrorLevel)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/platform/updates/operations", nil)
	request.Header.Set(httpx.RequestIDHeader, "request-91")
	request.Header.Set("X-Trace-Id", "trace-91")
	request = request.WithContext(httpx.WithRequestAuditContext(request.Context(), httpx.RequestAuditContext{RequestID: "request-91", TraceID: "trace-91", Method: http.MethodPost, Route: "/api/platform/updates/operations"}))
	ctx.Request = request
	diagnostics := &failureDiagnosticStoreRecorder{}
	return startFailureTestFixture{recorder: recorder, context: ctx, handler: updateRouteHandlers{diagnostics: diagnostics, appLogger: logger.NewAppLogger(zap.New(core))}, entries: entries, diagnostics: diagnostics}
}

func assertSafeStartFailureResponse(t *testing.T, recorder *httptest.ResponseRecorder, sensitiveInput string) {
	t.Helper()

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal failure status, got %d", recorder.Code)
	}
	if recorder.Header().Get(httpx.RequestIDHeader) != "request-91" {
		t.Fatalf("expected request ID response header, got %q", recorder.Header().Get(httpx.RequestIDHeader))
	}
	if strings.Contains(recorder.Body.String(), sensitiveInput) {
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
		"request_id": "request-91", "trace_id": "trace-91", "actor_id": uint64(7), "target_version": "0.11.0-beta.9", "failure_code": rolloutFailureOperationStartFailed, "failure_stage": "runner_launch", "operation_id": "update-91",
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
