package logger

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

	"graft/server/internal/eventbus"
	"graft/server/internal/moduleapi"
)

type explorerDeleteRepoRecorder struct {
	deletedIDs []uint64
}

type explorerListFailureRepo struct {
	explorerDeleteRepoRecorder
	err error
}

func (r explorerListFailureRepo) ListAppLogs(context.Context, AppLogListQuery) (AppLogListResult, error) {
	return AppLogListResult{}, r.err
}

func (r *explorerDeleteRepoRecorder) CreateAppLog(context.Context, CreateAppLogInput) (AppLogRecord, error) {
	return AppLogRecord{}, nil
}

func (r *explorerDeleteRepoRecorder) DeleteAppLogByID(_ context.Context, id uint64) (bool, error) {
	r.deletedIDs = append(r.deletedIDs, id)
	return true, nil
}

func (r *explorerDeleteRepoRecorder) DeleteAppLogsByIDs(_ context.Context, ids []uint64) (int64, error) {
	r.deletedIDs = append(r.deletedIDs, ids...)
	return int64(len(ids)), nil
}

func (r *explorerDeleteRepoRecorder) DeleteAppLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *explorerDeleteRepoRecorder) DeleteAppLogsBeforeLimit(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func (r *explorerDeleteRepoRecorder) ListAppLogs(context.Context, AppLogListQuery) (AppLogListResult, error) {
	return AppLogListResult{}, nil
}

func (r *explorerDeleteRepoRecorder) GetAppLogByID(context.Context, uint64) (AppLogRecord, error) {
	return AppLogRecord{}, ErrAppLogNotFound
}

func TestBindAppLogListQueryParsesSorters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest(
		"GET",
		"/api/app-log?sort=component:asc&sort=occurred_at:desc&sort=component:desc",
		nil,
	)

	query, invalidField := bindAppLogListQuery(ginCtx)
	if invalidField != "" {
		t.Fatalf("expected valid query, got invalid field %q", invalidField)
	}
	if len(query.Sorters) != 2 {
		t.Fatalf("expected duplicate sort field to be ignored, got %#v", query.Sorters)
	}
	if query.Sorters[0] != (AppLogSorter{Field: AppLogSortFieldComponent, Order: AppLogSortOrderAsc}) {
		t.Fatalf("unexpected first sorter: %#v", query.Sorters[0])
	}
	if query.Sorters[1] != (AppLogSorter{Field: AppLogSortFieldOccurredAt, Order: AppLogSortOrderDesc}) {
		t.Fatalf("unexpected second sorter: %#v", query.Sorters[1])
	}
}

// TestHandleListAppLogsLogsUnexpectedRepositoryFailure 验证 Explorer 不暴露仓储原因，并由统一 HTTP 边界记录一次 fallback。
func TestHandleListAppLogsLogsUnexpectedRepositoryFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zapcore.ErrorLevel)
	restoreGlobals := zap.ReplaceGlobals(zap.New(core))
	t.Cleanup(restoreGlobals)

	cause := errors.New("app log storage unavailable")
	router := gin.New()
	router.GET("/app-log", handleListAppLogs(nil, &explorerListFailureRepo{err: cause}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app-log", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), cause.Error()) {
		t.Fatalf("expected internal cause to stay out of response: %s", recorder.Body.String())
	}
	if len(observed.All()) != 1 || observed.All()[0].Message != "unreported internal error" {
		t.Fatalf("expected one HTTP fallback error, got %#v", observed.All())
	}
}

func TestBindAppLogListQueryRejectsInvalidSorter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/api/app-log?sort=request_id:desc", nil)

	_, invalidField := bindAppLogListQuery(ginCtx)
	if invalidField != "sort" {
		t.Fatalf("expected invalid sort field, got %q", invalidField)
	}
}

func TestBindAppLogListQueryParsesCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/api/app-log?category=runtime.metrics", nil)

	query, invalidField := bindAppLogListQuery(ginCtx)
	if invalidField != "" || query.Category != CategoryRuntimeMetrics {
		t.Fatalf("expected runtime.metrics category, got query=%#v invalid=%q", query, invalidField)
	}

	invalidContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	invalidContext.Request = httptest.NewRequest("GET", "/api/app-log?category=unknown", nil)
	_, invalidField = bindAppLogListQuery(invalidContext)
	if invalidField != "category" {
		t.Fatalf("expected category validation error, got %q", invalidField)
	}
}

func TestHandleBatchDeleteAppLogsValidatesIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &explorerDeleteRepoRecorder{}

	router := gin.New()
	router.POST("/app-log/batch-delete", handleBatchDeleteAppLogs(nil, repo, nil))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/app-log/batch-delete", strings.NewReader(`{"ids":[3,4,3]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(repo.deletedIDs) != 2 || repo.deletedIDs[0] != 3 || repo.deletedIDs[1] != 4 {
		t.Fatalf("expected normalized batch ids, got %#v", repo.deletedIDs)
	}

	var response struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success response, got %s", recorder.Body.String())
	}
}

func TestHandleBatchDeleteAppLogsReturnsErrorWhenAuditPublishFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &explorerDeleteRepoRecorder{}
	bus := eventbus.New(zap.NewNop())
	publishErr := errors.New("persist audit failed")
	if err := bus.Subscribe(string(moduleapi.AuditRecordEventName), func(context.Context, eventbus.Event) error {
		return publishErr
	}); err != nil {
		t.Fatalf("subscribe audit event: %v", err)
	}

	router := gin.New()
	router.POST("/app-log/batch-delete", handleBatchDeleteAppLogs(nil, repo, bus))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/app-log/batch-delete", strings.NewReader(`{"ids":[3,4,3]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(repo.deletedIDs) != 2 || repo.deletedIDs[0] != 3 || repo.deletedIDs[1] != 4 {
		t.Fatalf("expected normalized batch ids to be deleted before audit failure, got %#v", repo.deletedIDs)
	}
}
