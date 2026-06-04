package logger

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
