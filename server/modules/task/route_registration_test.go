package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	taskstore "graft/server/modules/task/store"
)

func TestTaskResponseAdaptersUseOpenAPIFieldNames(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 10, 21, 36, 29, 0, time.UTC)
	stageID := uint64(8)
	responses := []map[string]any{
		taskSummaryResponse(moduleapi.TaskView{ID: 7, Type: "application.compose.redeploy", Owner: moduleapi.TaskOwner{Type: "application", ID: "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"}, Status: moduleapi.TaskStatusSuccess, CreatedAt: now}),
		taskStageResponse(moduleapi.TaskStageView{ID: stageID, Key: "up", Sequence: 3, ExecutorType: "application.compose.up", Status: moduleapi.StageStatusSuccess, Attempt: 1, MaxAttempts: 1, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile}),
		taskEventResponse(moduleapi.TaskEventView{ID: 9, Sequence: 4, Type: "task.created", Payload: []byte(`{"source":"test"}`), CreatedAt: now}),
		taskLogResponse(moduleapi.TaskLogView{ID: 10, StageID: &stageID, Sequence: 5, Stream: "stdout", Level: "info", Line: "started", OccurredAt: now}),
	}
	for _, response := range responses {
		encoded, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("marshal task response: %v", err)
		}
		var fields map[string]any
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatalf("unmarshal task response: %v", err)
		}
		for field := range fields {
			if field[0] >= 'A' && field[0] <= 'Z' {
				t.Fatalf("response leaked Go field name %q: %s", field, encoded)
			}
		}
	}
	if stage := responses[1]; stage["status"] != moduleapi.StageStatusSuccess || stage["executor_type"] != moduleapi.StageExecutorType("application.compose.up") {
		t.Fatalf("stage response = %#v", stage)
	}
	if log := responses[3]; log["stage_id"] != &stageID || log["occurred_at"] != now {
		t.Fatalf("log response = %#v", log)
	}
}

func TestTaskListFilterParsesOptionalTypeAndStatus(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/tasks?type=application.compose.redeploy&status=failed", nil)
	filter, err := taskListFilter(context, moduleapi.TaskOwner{Type: "application", ID: "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"})
	if err != nil {
		t.Fatalf("parse task list filter: %v", err)
	}
	if filter.Type == nil || *filter.Type != "application.compose.redeploy" || filter.Status == nil || *filter.Status != moduleapi.TaskStatusFailed {
		t.Fatalf("filter = %#v", filter)
	}
}

func TestTaskListFilterRejectsUnknownStatus(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/tasks?status=unsupported", nil)
	if _, err := taskListFilter(context, moduleapi.TaskOwner{Type: "application", ID: "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"}); err == nil {
		t.Fatal("unknown task status accepted")
	}
}

func TestTaskLogPageSupportsLegacyTailAndBeforeQueries(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		query      string
		wantMode   taskLogPageMode
		wantCursor int64
		wantLimit  int
	}{
		{name: "legacy forward cursor", query: "after_sequence=5&limit=20", wantMode: taskLogPageAfter, wantCursor: 5, wantLimit: 20},
		{name: "tail", query: "tail=true&limit=10", wantMode: taskLogPageTail, wantLimit: 10},
		{name: "before cursor", query: "before_sequence=5&limit=10", wantMode: taskLogPageBefore, wantCursor: 5, wantLimit: 10},
		{name: "false tail is absent", query: "tail=false&after_sequence=5", wantMode: taskLogPageAfter, wantCursor: 5, wantLimit: defaultTaskLogLimit},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodGet, "/tasks/1/logs?"+testCase.query, nil)
			page, err := taskLogPage(context, defaultTaskLogLimit, maxTaskLogLimit)
			if err != nil {
				t.Fatalf("parse task log page: %v", err)
			}
			if page.mode != testCase.wantMode || page.cursor != testCase.wantCursor || page.limit != testCase.wantLimit {
				t.Fatalf("page = %#v", page)
			}
		})
	}
}

func TestTaskLogPageRejectsConflictingCursorQueries(t *testing.T) {
	t.Parallel()
	for _, query := range []string{
		"after_sequence=1&before_sequence=2",
		"after_sequence=1&tail=true",
		"before_sequence=2&tail=true",
		"before_sequence=0",
		"tail=not-a-bool",
	} {
		t.Run(query, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodGet, "/tasks/1/logs?"+query, nil)
			if _, err := taskLogPage(context, defaultTaskLogLimit, maxTaskLogLimit); !errors.Is(err, errTaskInvalidArgument) {
				t.Fatalf("query %q error = %v, want invalid argument", query, err)
			}
		})
	}
}

func TestTaskRouteWritesNotFoundContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/tasks?owner_type=application&owner_id=NaN", nil)
	context.Request.Header.Set(i18n.LocaleHeader, "en-US")

	routes := taskRoutes{ctx: &module.Context{I18n: i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})}}
	routes.writeError(context, http.StatusNotFound, taskstore.ErrNotFound)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "COMMON_NOT_FOUND" || payload.MessageKey != messagecontract.CommonNotFound.String() {
		t.Fatalf("unexpected not-found contract payload: %#v", payload)
	}
	if payload.Locale != "en-US" || payload.Message != "Requested resource not found" {
		t.Fatalf("unexpected localized message: %#v", payload)
	}
}

func TestTaskRouteDoesNotExposeInternalCause(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/tasks/7", nil)

	routes := taskRoutes{ctx: &module.Context{}}
	routes.writeError(context, http.StatusInternalServerError, errors.New("database password leaked"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if got := recorder.Body.String(); strings.Contains(got, "database password leaked") || strings.Contains(got, "\"error\"") {
		t.Fatalf("task response exposed internal cause: %s", got)
	}
}
