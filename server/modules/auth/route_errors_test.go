// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package auth

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

type routeAppLogRecorder struct {
	mu      sync.Mutex
	records []applog.CreateAppLogInput
	seen    chan struct{}
}

func (r *routeAppLogRecorder) CreateAppLog(_ context.Context, input applog.CreateAppLogInput) (applog.AppLogRecord, error) {
	r.mu.Lock()
	r.records = append(r.records, input)
	r.mu.Unlock()
	select {
	case r.seen <- struct{}{}:
	default:
	}
	return applog.AppLogRecord{}, nil
}

func (r *routeAppLogRecorder) DeleteAppLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *routeAppLogRecorder) DeleteAppLogsBeforeLimit(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func (r *routeAppLogRecorder) ListAppLogs(context.Context, applog.AppLogListQuery) (applog.AppLogListResult, error) {
	return applog.AppLogListResult{}, nil
}

func (r *routeAppLogRecorder) GetAppLogByID(context.Context, uint64) (applog.AppLogRecord, error) {
	return applog.AppLogRecord{}, applog.ErrAppLogNotFound
}

func TestRouteRuntimeUsesResolvedAppLoggerForResponseMappingErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink := &routeAppLogRecorder{seen: make(chan struct{}, 1)}
	services := container.New()
	if err := services.RegisterSingleton((*applog.AppLogger)(nil), func(container.Resolver) (any, error) {
		return applog.NewAppLogger(zap.NewNop(), applog.WithAppLogRepository(sink)), nil
	}); err != nil {
		t.Fatalf("register app logger: %v", err)
	}

	runtime := authRouteRegistrar{
		ctx: &module.Context{
			Logger:   zap.NewNop(),
			Services: services,
		},
		moduleName: moduleID,
	}.runtime()

	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/api/auth/bootstrap", nil)
	runtime.writeResponseMappingError(ginCtx, "map bootstrap response", errors.New("bad payload"))

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
	if record.Component != "modules.auth.route" || record.Message != "map bootstrap response" {
		t.Fatalf("unexpected persisted app log record: %#v", record)
	}
	if record.Fields["module"] != moduleID || record.Error != "bad payload" {
		t.Fatalf("expected module and error fields, got %#v", record)
	}
}
