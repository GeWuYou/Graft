package user

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/container"
	applog "graft/server/internal/logger"
	"graft/server/internal/module"
)

type userRouteAppLogRecorder struct {
	mu      sync.Mutex
	records []applog.CreateAppLogInput
	seen    chan struct{}
}

func (r *userRouteAppLogRecorder) CreateAppLog(_ context.Context, input applog.CreateAppLogInput) (applog.AppLogRecord, error) {
	r.mu.Lock()
	r.records = append(r.records, input)
	r.mu.Unlock()
	select {
	case r.seen <- struct{}{}:
	default:
	}
	return applog.AppLogRecord{}, nil
}

func (r *userRouteAppLogRecorder) DeleteAppLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *userRouteAppLogRecorder) ListAppLogs(context.Context, applog.AppLogListQuery) (applog.AppLogListResult, error) {
	return applog.AppLogListResult{}, nil
}

func (r *userRouteAppLogRecorder) GetAppLogByID(context.Context, uint64) (applog.AppLogRecord, error) {
	return applog.AppLogRecord{}, applog.ErrAppLogNotFound
}

func TestUserRouteRuntimeUsesResolvedAppLoggerForResponseMappingErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink := &userRouteAppLogRecorder{seen: make(chan struct{}, 1)}
	services := container.New()
	if err := services.RegisterSingleton((*applog.AppLogger)(nil), func(container.Resolver) (any, error) {
		return applog.NewAppLogger(zap.NewNop(), applog.WithAppLogRepository(sink)), nil
	}); err != nil {
		t.Fatalf("register app logger: %v", err)
	}

	runtime := userRouteRegistrar{
		ctx: &module.Context{
			Logger:   zap.NewNop(),
			Services: services,
		},
		moduleName: moduleID,
	}.runtime()

	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/api/users", nil)
	runtime.writeResponseMappingError(ginCtx, "map user list response failed", errors.New("bad payload"))

	select {
	case <-sink.seen:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for persisted app log record")
	}
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.records) != 1 {
		t.Fatalf("expected one persisted app log record, got %#v", sink.records)
	}
	record := sink.records[0]
	if record.Component != "modules.user.route" || record.Message != "map user list response failed" {
		t.Fatalf("unexpected persisted app log record: %#v", record)
	}
	if record.Fields["module"] != moduleID || record.Error != "bad payload" {
		t.Fatalf("expected module and error fields, got %#v", record)
	}
}
