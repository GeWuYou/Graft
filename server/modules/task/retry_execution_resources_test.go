package task

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/dockerexecution"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

func TestFailedTaskRetryReacquiresFrozenExecutionResources(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	runtime.localExecution.execute = func(context.Context, dockerexecution.Environment, dockerexecution.Request) dockerexecution.Result {
		return dockerexecution.Result{Outcome: "failed", FailureCode: "synthetic_failure"}
	}
	failed, err := runtime.Submit(context.Background(), localSubmitInput("update", "platform-update"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	stages, err := repository.ListStages(context.Background(), failed.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	other, err := runtime.Submit(context.Background(), localSubmitInput("build", "image:docker.io/team/api:latest"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.RetryStage(context.Background(), failed.TaskID, stages[0].ID); !errors.Is(err, moduleapi.ErrTaskOwnerBusy) {
		t.Fatalf("retry with another active resource claim error = %v", err)
	}
	if mustTask(t, repository, failed.TaskID).Status != moduleapi.TaskStatusFailed {
		t.Fatal("rejected resource admission resumed the failed Task")
	}
	assertRetryHTTPConflict(t, runtime, failed.TaskID, stages[0].ID, "COMMON_TASK_RESOURCE_BUSY")
	if err := runtime.Cancel(context.Background(), other.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := runtime.RetryStage(context.Background(), failed.TaskID, stages[0].ID); err != nil {
		t.Fatalf("retry after conflicting Task released its claim: %v", err)
	}
	if _, err := runtime.Submit(context.Background(), localSubmitInput("next-build", "image:docker.io/team/api:latest")); !errors.Is(err, moduleapi.ErrTaskOwnerBusy) {
		t.Fatalf("retried Task did not reacquire its resource claim: %v", err)
	}
}

func TestFailedTaskRetryRejectsChangedExecutionMode(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	runtime.localExecution.execute = func(context.Context, dockerexecution.Environment, dockerexecution.Request) dockerexecution.Result {
		return dockerexecution.Result{Outcome: "failed", FailureCode: "synthetic_failure"}
	}
	receipt, err := runtime.Submit(context.Background(), localSubmitInput("previous-mode", "compose:previous"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CheckExecutionMode(context.Background(), "agent"); err != nil {
		t.Fatal(err)
	}
	assertRetryHTTPConflict(t, runtime, receipt.TaskID, stages[0].ID, "COMMON_TASK_EXECUTION_UNAVAILABLE")
	if mustTask(t, repository, receipt.TaskID).Status != moduleapi.TaskStatusFailed {
		t.Fatal("mode mismatch resumed the historical failed Task")
	}
}

type retryResourceAuthorizer struct{}

func (retryResourceAuthorizer) OwnerType() string { return "test" }
func (retryResourceAuthorizer) AuthorizeTaskOwner(context.Context, *moduleapi.CurrentUser, moduleapi.TaskOwnerAction, moduleapi.TaskOwner) error {
	return nil
}

func assertRetryHTTPConflict(t *testing.T, runtime *Runtime, taskID, stageID uint64, code string) {
	t.Helper()
	if err := runtime.RegisterTaskOwnerAuthorizer(retryResourceAuthorizer{}); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	routes := taskRoutes{runtime: runtime, ctx: &module.Context{}}
	engine.POST("/tasks/:taskId/stages/:stageId/retry", routes.retry)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tasks/%d/stages/%d/retry", taskID, stageID), nil)
	request = request.WithContext(moduleapi.WithRequestAuthContext(request.Context(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}}))
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), code) {
		t.Fatalf("retry response = %d %s, want 409 %s", response.Code, response.Body.String(), code)
	}
}

func TestUnknownTaskRetryExcludesItsOwnResourceClaim(t *testing.T) {
	runtime, repository := localRuntimeForTest(t)
	runtime.localExecution.execute = func(context.Context, dockerexecution.Environment, dockerexecution.Request) dockerexecution.Result {
		return dockerexecution.Result{Outcome: "needs_attention", FailureCode: "agent_execution_interrupted"}
	}
	receipt, err := runtime.Submit(context.Background(), localSubmitInput("uncertain", "compose:retained"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.RetryStage(context.Background(), receipt.TaskID, stages[0].ID); err != nil {
		t.Fatalf("unknown Task conflicted with its own retained resources: %v", err)
	}
}
