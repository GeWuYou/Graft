package httpx

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindAccessLogListQueryRejectsOutOfRangeNumericFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		rawQuery     string
		wantInvalid  string
	}{
		{name: "status code below valid range", rawQuery: "status_code=99", wantInvalid: "status_code"},
		{name: "status code above valid range", rawQuery: "status_code=512", wantInvalid: "status_code"},
		{name: "negative duration min", rawQuery: "duration_min_ms=-1", wantInvalid: "duration_min_ms"},
		{name: "negative duration max", rawQuery: "duration_max_ms=-1", wantInvalid: "duration_max_ms"},
		{name: "duration min greater than max", rawQuery: "duration_min_ms=50&duration_max_ms=10", wantInvalid: "duration_min_ms"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/access-log?"+testCase.rawQuery, nil)

			_, invalidField := bindAccessLogListQuery(ctx)
			if invalidField != testCase.wantInvalid {
				t.Fatalf("expected invalid field %q, got %q", testCase.wantInvalid, invalidField)
			}
		})
	}
}

func TestBindAccessLogListQueryAcceptsValidNumericFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/access-log?status_code=404&duration_min_ms=10&duration_max_ms=50", nil)

	query, invalidField := bindAccessLogListQuery(ctx)
	if invalidField != "" {
		t.Fatalf("expected valid query, got invalid field %q", invalidField)
	}
	if query.StatusCode == nil || *query.StatusCode != 404 {
		t.Fatalf("expected status code 404, got %#v", query.StatusCode)
	}
	if query.DurationMinMS == nil || *query.DurationMinMS != 10 {
		t.Fatalf("expected duration min 10, got %#v", query.DurationMinMS)
	}
	if query.DurationMaxMS == nil || *query.DurationMaxMS != 50 {
		t.Fatalf("expected duration max 50, got %#v", query.DurationMaxMS)
	}
}
