package project

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/httpx"
	"graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	projectstore "graft/server/modules/project/store"
)

type failingLifecycleTaskService struct {
	err error
}

func (s failingLifecycleTaskService) Submit(context.Context, moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	return moduleapi.TaskReceipt{}, s.err
}

func (failingLifecycleTaskService) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, nil
}

func (failingLifecycleTaskService) Cancel(context.Context, uint64) error { return nil }

func (failingLifecycleTaskService) RetryStage(context.Context, uint64, uint64) error { return nil }

func TestRedeployTaskSubmissionFailureIsReportedOnceAndReturnsSafeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cause := errors.New("task store is unavailable")
	core, observed := observer.New(zapcore.ErrorLevel)
	baseLogger := zap.New(core)

	repository := &stubProjectRepository{aggregate: projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ApplicationRecordID:   41,
			ApplicationID:         "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			ComposeProjectName:    "demo",
			WorkspacePath:         t.TempDir(),
			LifecycleReviewStatus: "confirmed",
			LifecycleStrategyKind: "standard",
		},
		Snapshot: &projectstore.Snapshot{ApplicationRecordID: 41, ConfigHash: "cfg-1", RefreshedAt: time.Now().UTC()},
	}}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	service.SetTaskService(failingLifecycleTaskService{err: cause})
	service.SetAppLogger(logger.NewAppLogger(baseLogger).Named("modules.project.lifecycle"))

	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/ops/applications/app_01ARZ3NDEKTSV4RRFFQ69G5FAV/redeploy", nil)
	request = request.WithContext(moduleapi.WithRequestAuthContext(httpx.WithRequestAuditContext(request.Context(), httpx.RequestAuditContext{
		RequestID: "req-redeploy-1",
		TraceID:   "trace-redeploy-1",
		Route:     "/api/ops/applications/:applicationId/redeploy",
		Method:    http.MethodPost,
	}), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}}))
	ginCtx.Request = request

	result, err := service.Redeploy(ginCtx.Request.Context(), 41, nil)
	if !errors.Is(err, cause) {
		t.Fatalf("expected original submit cause, got %v", err)
	}
	(routeRuntime{ctx: &module.Context{Logger: baseLogger}}).writeRouteErrorWithAction(ginCtx, err, result)

	assertSafeRedeployResponse(t, recorder, cause)
	assertLifecycleErrorLog(t, observed.All(), cause)
}

func assertSafeRedeployResponse(t *testing.T, recorder *httptest.ResponseRecorder, cause error) {
	t.Helper()
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"] != "COMMON_INTERNAL_ERROR" || strings.Contains(recorder.Body.String(), cause.Error()) {
		t.Fatalf("expected safe internal response, got %s", recorder.Body.String())
	}
}

func assertLifecycleErrorLog(t *testing.T, entries []observer.LoggedEntry, cause error) {
	t.Helper()
	if len(entries) != 1 {
		t.Fatalf("expected one business error record, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields[logger.FieldOperation] != "submit_application_lifecycle_task" || fields["application_id"] != "app_01ARZ3NDEKTSV4RRFFQ69G5FAV" || fields[logger.FieldRequestID] != "req-redeploy-1" || fields[logger.FieldTraceID] != "trace-redeploy-1" || fields["lifecycle_action"] != "redeploy" || fields["error_kind"] != "internal" || fields["error_code"] != "COMMON_INTERNAL_ERROR" || fields["error_type"] == "" || fields["error_fingerprint"] == "" || fields["error_fingerprint"] == cause.Error() || fields[logger.FieldError] == cause.Error() {
		t.Fatalf("unexpected lifecycle error fields: %#v", fields)
	}
}
