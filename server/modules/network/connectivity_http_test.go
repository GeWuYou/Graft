package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

func TestHandleLegacyDiagnosticWritesSanitizedCompatibilityContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	diagnostics := NewDiagnosticRegistry()
	if err := diagnostics.RegisterOutboundDiagnosticTarget(httpDiagnosticTargetStub{}); err != nil {
		t.Fatalf("register diagnostic target: %v", err)
	}
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/platform/network/diagnostics/legacy", nil)
	ginCtx.Params = gin.Params{{Key: "targetId", Value: "legacy"}}
	routeRuntime{service: NewService(nil, diagnostics, NewConsumerRegistry(), &diagnosticHistoryStoreStub{}, nil)}.handleLegacyDiagnostic(ginCtx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data legacyDiagnosticResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.TargetID != "legacy" || response.Data.Status != "connected" || response.Data.HTTPStatus != http.StatusOK || response.Data.Error != "" {
		t.Fatalf("unexpected response: %#v", response.Data)
	}
}

type httpDiagnosticTargetStub struct{}

func (httpDiagnosticTargetStub) Name() string        { return "legacy" }
func (httpDiagnosticTargetStub) DisplayName() string { return "legacy" }
func (httpDiagnosticTargetStub) ExecuteOutboundDiagnostic(context.Context) (moduleapi.OutboundDiagnosticResult, error) {
	return moduleapi.OutboundDiagnosticResult{Connected: true, HTTPStatus: http.StatusOK}, nil
}

func TestConnectivityCheckResponseKeepsOptionalHTTPStatusInSummaryProjection(t *testing.T) {
	httpStatus := 503
	withResponse, err := json.Marshal(toConnectivityCheckResponse(ConnectivityCheck{
		ID:         42,
		TargetID:   "github",
		Status:     moduleapi.ConnectivityReportStatusDegraded,
		Latency:    183 * time.Millisecond,
		HTTPStatus: &httpStatus,
		CheckedAt:  time.Date(2026, 8, 4, 14, 33, 0, 0, time.UTC),
	}))
	if err != nil || !strings.Contains(string(withResponse), `"http_status":503`) {
		t.Fatalf("expected HTTP status in summary response, body=%s err=%v", withResponse, err)
	}

	withoutResponse, err := json.Marshal(toConnectivityCheckResponse(ConnectivityCheck{ID: 43, TargetID: "smtp", Status: moduleapi.ConnectivityReportStatusFailed, CheckedAt: time.Now()}))
	if err != nil || !strings.Contains(string(withoutResponse), `"http_status":null`) {
		t.Fatalf("expected unavailable HTTP status to remain explicit, body=%s err=%v", withoutResponse, err)
	}
}
