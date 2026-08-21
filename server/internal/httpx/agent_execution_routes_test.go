package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

type recordingRuntimeAgentGateway struct {
	claimed moduleapi.ExternalExecutionClaimRequest
	renewed moduleapi.ExternalExecutionLeaseHandle
	logs    moduleapi.ExternalExecutionLogBatch
	receipt moduleapi.ExternalExecutionReceipt
	lease   moduleapi.ExternalExecutionLease
	claims  int
	misses  int
}

func (g *recordingRuntimeAgentGateway) ClaimExternalExecution(_ context.Context, request moduleapi.ExternalExecutionClaimRequest) (moduleapi.ExternalExecutionLease, error) {
	g.claimed = request
	g.claims++
	if g.claims <= g.misses {
		return moduleapi.ExternalExecutionLease{}, moduleapi.ErrExternalExecutionNotFound
	}
	return g.lease, nil
}
func (g *recordingRuntimeAgentGateway) RenewExternalExecution(_ context.Context, handle moduleapi.ExternalExecutionLeaseHandle) (moduleapi.ExternalExecutionLease, error) {
	g.renewed = handle
	return g.lease, nil
}
func (g *recordingRuntimeAgentGateway) AppendExternalExecutionLogs(_ context.Context, batch moduleapi.ExternalExecutionLogBatch) error {
	g.logs = batch
	return nil
}
func (g *recordingRuntimeAgentGateway) SettleExternalExecution(_ context.Context, receipt moduleapi.ExternalExecutionReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	g.receipt = receipt
	return moduleapi.ExternalReceiptSettlement{TaskID: g.lease.TaskID, StageID: g.lease.StageID, Status: moduleapi.TaskStatusRunning}, nil
}
func (g *recordingRuntimeAgentGateway) ExpireExternalExecutions(context.Context, int) (int, error) {
	return 0, nil
}

type recordingAgentBindingReader struct {
	binding moduleapi.RuntimeTargetAgentBinding
}

func (r recordingAgentBindingReader) ReadAgentBinding(context.Context, int64, string) (moduleapi.RuntimeTargetAgentBinding, error) {
	return r.binding, nil
}

func TestConfigureExecutionRoutesRejectsCapabilityOutsideBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &recordingRuntimeAgentGateway{}
	server := newExecutionAgentTestServer()
	if err := server.ConfigureExecutionRoutes(gateway, recordingAgentBindingReader{binding: moduleapi.RuntimeTargetAgentBinding{TargetID: 7, AgentID: "agent-7", ProviderID: "docker", Capabilities: []string{"compose_execution"}, CapabilityVersion: "runtime/v1", Status: moduleapi.RuntimeTargetAgentStatusActive}}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, agentExecutionClaimPath, strings.NewReader(`{"provider_id":"docker","capability":"build_execution","capability_version":"runtime/v1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("claim status=%d", recorder.Code)
	}
}

func TestExecutionRoutesForwardFencedLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	gateway := &recordingRuntimeAgentGateway{lease: moduleapi.ExternalExecutionLease{ID: "lease-1", TaskID: 11, StageID: 12, FenceToken: "fence-1", LeaseTTL: time.Minute, LeaseExpiresAt: now.Add(time.Minute), AbsoluteDeadlineAt: now.Add(time.Hour)}}
	server := newExecutionAgentTestServer()
	bindings := recordingAgentBindingReader{binding: moduleapi.RuntimeTargetAgentBinding{TargetID: 7, AgentID: "agent-7", ProviderID: "docker", Capabilities: []string{"compose_execution"}, CapabilityVersion: "runtime/v1", Status: moduleapi.RuntimeTargetAgentStatusActive}}
	if err := server.ConfigureExecutionRoutes(gateway, bindings); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, agentExecutionClaimPath, strings.NewReader(`{"provider_id":"docker","capability":"compose_execution","capability_version":"runtime/v1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || gateway.claimed.Capability != "compose_execution" || gateway.claimed.RuntimeTargetID != 7 {
		t.Fatalf("claim status=%d request=%#v", recorder.Code, gateway.claimed)
	}
}

func TestLongPollAgentExecutionRetriesUntilWorkArrives(t *testing.T) {
	gateway := &recordingRuntimeAgentGateway{misses: 2, lease: moduleapi.ExternalExecutionLease{ID: "lease-delayed"}}
	lease, err := longPollAgentExecution(context.Background(), gateway, moduleapi.ExternalExecutionClaimRequest{RuntimeTargetID: 7, ProviderID: "docker", Capability: "compose_execution"})
	if err != nil {
		t.Fatal(err)
	}
	if lease.ID != "lease-delayed" || gateway.claims != 3 {
		t.Fatalf("lease=%#v claims=%d", lease, gateway.claims)
	}
}

func newExecutionAgentTestServer() *AgentServer {
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set(agentIdentityContextKey, AgentMTLSIdentity{AgentIdentity: moduleapi.AgentIdentity{TargetID: 7, AgentID: "agent-7"}})
		ctx.Next()
	})
	return &AgentServer{engine: engine}
}
