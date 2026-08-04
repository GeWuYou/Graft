package build

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildListQueryBindsBuildOwnedHistoryFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest("GET", "/api/build/jobs?limit=50&offset=100&application_id=42&image_repository=example%2Fapp&image_tag=v1&created_after=2026-08-01T00%3A00%3A00Z&created_before=2026-08-02T00%3A00%3A00Z", nil)
	context.Request = request

	query, ok := buildListQuery(context)
	if !ok {
		t.Fatal("expected valid Build history query")
	}
	if query.Limit != 50 || query.Offset != 100 || query.ApplicationID == nil || *query.ApplicationID != 42 {
		t.Fatalf("unexpected pagination or application filter: %#v", query)
	}
	if query.ImageRepository == nil || *query.ImageRepository != "example/app" || query.ImageTag == nil || *query.ImageTag != "v1" {
		t.Fatalf("unexpected image filters: %#v", query)
	}
	if query.CreatedAfter == nil || query.CreatedBefore == nil || !query.CreatedAfter.Before(*query.CreatedBefore) {
		t.Fatalf("unexpected creation range: %#v", query)
	}
}

func TestBuildListQueryRejectsInvalidHistoryRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/api/build/jobs?created_after=2026-08-02T00%3A00%3A00Z&created_before=2026-08-01T00%3A00%3A00Z", nil)

	if _, ok := buildListQuery(context); ok {
		t.Fatal("expected reverse creation range to be rejected")
	}
}
