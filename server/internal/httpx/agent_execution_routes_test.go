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
func (g *recordingRuntimeAgentGateway) InspectExternalExecution(context.Context, moduleapi.ExternalExecutionLeaseHandle) (moduleapi.ExternalExecutionLease, error) {
	return g.lease, nil
}
func (g *recordingRuntimeAgentGateway) RenewExternalExecution(_ context.Context, handle moduleapi.ExternalExecutionLeaseHandle) (moduleapi.ExternalExecutionLease, error) {
	g.renewed = handle
	return g.lease, nil
}
func (g *recordingRuntimeAgentGateway) ResolveExternalExecutionMaterial(context.Context, moduleapi.ExternalExecutionLeaseHandle) (moduleapi.ExternalExecutionMaterial, error) {
	return moduleapi.ExternalExecutionMaterial{Protocol: "test/v1", Payload: []byte(`{}`)}, nil
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

func TestConfigureExecutionRoutesRejectsCapabilityVersionMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &recordingRuntimeAgentGateway{}
	server := newExecutionAgentTestServer()
	binding := moduleapi.RuntimeTargetAgentBinding{TargetID: 7, AgentID: "agent-7", ProviderID: "docker", Capabilities: []string{"compose_execution"}, CapabilityVersion: "docker/v1", Status: moduleapi.RuntimeTargetAgentStatusActive}
	if err := server.ConfigureExecutionRoutes(gateway, recordingAgentBindingReader{binding: binding}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, agentExecutionClaimPath, strings.NewReader(`{"provider_id":"docker","capability":"compose_execution","capability_version":"docker/v2"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || gateway.claims != 0 {
		t.Fatalf("claim status=%d claims=%d", recorder.Code, gateway.claims)
	}
}

func TestConfigureExecutionRoutesRejectsRetiredCertificateGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &recordingRuntimeAgentGateway{}
	identity := AgentMTLSIdentity{AgentIdentity: moduleapi.AgentIdentity{IdentityID: "identity-7", TargetID: 7, AgentID: "agent-7", Generation: 1, CertificateSerial: "serial-old", PublicKeyFingerprint: "fingerprint-old"}}
	server := newExecutionAgentTestServerWithIdentity(identity)
	binding := moduleapi.RuntimeTargetAgentBinding{IdentityID: "identity-7", TargetID: 7, AgentID: "agent-7", Generation: 2, CertificateSerial: "serial-new", PublicKeyFingerprint: "fingerprint-new", ProviderID: "docker", Capabilities: []string{"compose_execution"}, CapabilityVersion: "docker/v1", Status: moduleapi.RuntimeTargetAgentStatusActive}
	if err := server.ConfigureExecutionRoutes(gateway, recordingAgentBindingReader{binding: binding}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, agentExecutionClaimPath, strings.NewReader(`{"provider_id":"docker","capability":"compose_execution","capability_version":"docker/v1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || gateway.claims != 0 {
		t.Fatalf("claim status=%d claims=%d", recorder.Code, gateway.claims)
	}
	renewRequest := httptest.NewRequest(http.MethodPost, "/agent/v1/execution-leases/lease-old/renew", strings.NewReader(`{"fence_token":"fence-old"}`))
	renewRequest.Header.Set("Content-Type", "application/json")
	renewRecorder := httptest.NewRecorder()
	server.engine.ServeHTTP(renewRecorder, renewRequest)
	if renewRecorder.Code != http.StatusUnauthorized || gateway.renewed.LeaseID != "" {
		t.Fatalf("renew status=%d handle=%#v", renewRecorder.Code, gateway.renewed)
	}
}

func TestExecutionRoutesRejectLeaseCapabilityRemovedFromCurrentGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	gateway := &recordingRuntimeAgentGateway{lease: moduleapi.ExternalExecutionLease{
		ID: "lease-compose", RuntimeTargetID: 7, ProviderID: "docker", Capability: "compose_execution",
		CapabilityVersion: "docker/v1", LeaseExpiresAt: now.Add(time.Minute), AbsoluteDeadlineAt: now.Add(time.Hour),
	}}
	server := newExecutionAgentTestServer()
	binding := moduleapi.RuntimeTargetAgentBinding{TargetID: 7, AgentID: "agent-7", ProviderID: "docker", Capabilities: []string{"container_execution"}, CapabilityVersion: "docker/v1", Status: moduleapi.RuntimeTargetAgentStatusActive}
	if err := server.ConfigureExecutionRoutes(gateway, recordingAgentBindingReader{binding: binding}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/agent/v1/execution-leases/lease-compose/renew", strings.NewReader(`{"fence_token":"fence-compose"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized || gateway.renewed.LeaseID != "" {
		t.Fatalf("renew status=%d handle=%#v", recorder.Code, gateway.renewed)
	}
}

func TestExecutionRoutesForwardFencedLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	gateway := &recordingRuntimeAgentGateway{lease: moduleapi.ExternalExecutionLease{ID: "lease-1", TaskID: 11, StageID: 12, RuntimeTargetID: 7, ProviderID: "docker", Capability: "compose_execution", FenceToken: "fence-1", CapabilityVersion: "runtime/v1", LeaseTTL: time.Minute, LeaseExpiresAt: now.Add(time.Minute), AbsoluteDeadlineAt: now.Add(time.Hour)}}
	server := newExecutionAgentTestServer()
	bindings := recordingAgentBindingReader{binding: moduleapi.RuntimeTargetAgentBinding{TargetID: 7, AgentID: "agent-7", ProviderID: "docker", Capabilities: []string{"compose_execution"}, CapabilityVersion: "runtime/v1", Status: moduleapi.RuntimeTargetAgentStatusActive}}
	if err := server.ConfigureExecutionRoutes(gateway, bindings); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, agentExecutionClaimPath, strings.NewReader(`{"provider_id":"docker","capability":"compose_execution","capability_version":"runtime/v1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || gateway.claimed.Capability != "compose_execution" || gateway.claimed.CapabilityVersion != "runtime/v1" || gateway.claimed.RuntimeTargetID != 7 {
		t.Fatalf("claim status=%d request=%#v", recorder.Code, gateway.claimed)
	}
	materialRequest := httptest.NewRequest(http.MethodPost, "/agent/v1/execution-leases/lease-1/material", strings.NewReader(`{"fence_token":"fence-1"}`))
	materialRequest.Header.Set("Content-Type", "application/json")
	materialRecorder := httptest.NewRecorder()
	server.engine.ServeHTTP(materialRecorder, materialRequest)
	if materialRecorder.Code != http.StatusOK || !strings.Contains(materialRecorder.Body.String(), `"protocol":"test/v1"`) {
		t.Fatalf("material status=%d body=%s", materialRecorder.Code, materialRecorder.Body.String())
	}
}

func TestLongPollAgentExecutionRetriesUntilWorkArrives(t *testing.T) {
	gateway := &recordingRuntimeAgentGateway{misses: 2, lease: moduleapi.ExternalExecutionLease{ID: "lease-delayed"}}
	lease, err := longPollAgentExecution(context.Background(), gateway, moduleapi.ExternalExecutionClaimRequest{RuntimeTargetID: 7, ProviderID: "docker", Capability: "compose_execution", CapabilityVersion: "runtime/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if lease.ID != "lease-delayed" || gateway.claims != 3 {
		t.Fatalf("lease=%#v claims=%d", lease, gateway.claims)
	}
}

func newExecutionAgentTestServer() *AgentServer {
	return newExecutionAgentTestServerWithIdentity(AgentMTLSIdentity{AgentIdentity: moduleapi.AgentIdentity{TargetID: 7, AgentID: "agent-7"}})
}

func newExecutionAgentTestServerWithIdentity(identity AgentMTLSIdentity) *AgentServer {
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set(agentIdentityContextKey, identity)
		ctx.Next()
	})
	return &AgentServer{engine: engine}
}
