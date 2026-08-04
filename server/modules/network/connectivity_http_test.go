package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
