package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/httpheader"
	"graft/server/internal/httpx"
)

func TestBindGeneratedAuditReadHeadersTrimsValues(t *testing.T) {
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs", nil)
	request.Header.Set(string(httpheader.Locale), "  zh-CN  ")
	request.Header.Set(httpx.RequestIDHeader, "  req-123  ")
	ginCtx.Request = request

	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	if locale == nil || *locale != "zh-CN" {
		t.Fatalf("expected trimmed locale header, got %#v", locale)
	}
	if requestID == nil || *requestID != "req-123" {
		t.Fatalf("expected trimmed request id header, got %#v", requestID)
	}
}

func TestAuditHeaderPointerTreatsBlankAsNil(t *testing.T) {
	if value := auditHeaderPointer("   "); value != nil {
		t.Fatalf("expected blank header to map to nil, got %#v", value)
	}
}
