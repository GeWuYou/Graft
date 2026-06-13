// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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

	"graft/server/internal/eventbus"
	"graft/server/internal/moduleapi"
)

type explorerDeleteRepoRecorder struct {
	deletedIDs []uint64
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

func TestBindAppLogListQueryRejectsInvalidSorter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/api/app-log?sort=request_id:desc", nil)

	_, invalidField := bindAppLogListQuery(ginCtx)
	if invalidField != "sort" {
		t.Fatalf("expected invalid sort field, got %q", invalidField)
	}
}

func TestHandleDeleteAppLogPublishesAuditEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &explorerDeleteRepoRecorder{}
	bus := eventbus.New(zap.NewNop())
	events := make([]moduleapi.AuditEvent, 0, 1)
	if err := bus.Subscribe(string(moduleapi.AuditRecordEventName), func(_ context.Context, event eventbus.Event) error {
		payload, ok := event.Payload.(moduleapi.AuditEvent)
		if !ok {
			t.Fatalf("expected audit payload, got %T", event.Payload)
		}
		events = append(events, payload)
		return nil
	}); err != nil {
		t.Fatalf("subscribe audit event: %v", err)
	}

	router := gin.New()
	router.DELETE("/app-log/:id", handleDeleteAppLog(nil, repo, bus))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/app-log/42", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(repo.deletedIDs) != 1 || repo.deletedIDs[0] != 42 {
		t.Fatalf("expected app log 42 to be deleted, got %#v", repo.deletedIDs)
	}
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %#v", events)
	}
	event := events[0]
	if event.Action != appLogManualDeleteAction || event.ResourceType != appLogResourceType || event.ResourceID != "42" {
		t.Fatalf("unexpected audit event identity: %#v", event)
	}
	if event.Metadata["retentionOwner"] != string(AppLogRetentionOwnerLogger) {
		t.Fatalf("expected retention owner metadata, got %#v", event.Metadata)
	}
}

func TestHandleDeleteAppLogReturnsErrorWhenAuditPublishFails(t *testing.T) {
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
	router.DELETE("/app-log/:id", handleDeleteAppLog(nil, repo, bus))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/app-log/42", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(repo.deletedIDs) != 1 || repo.deletedIDs[0] != 42 {
		t.Fatalf("expected app log 42 to be deleted before audit failure, got %#v", repo.deletedIDs)
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
