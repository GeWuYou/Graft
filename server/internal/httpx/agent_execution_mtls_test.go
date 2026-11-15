package httpx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

func TestExecutionRoutesResolveVerifiedCertificateBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	certificate := testAgentCertificate(t, "spiffe://graft/runtime-target/7/runtime-agent/agent-7")
	gateway := &recordingRuntimeAgentGateway{lease: moduleapi.ExternalExecutionLease{ID: "lease-1", RuntimeTargetID: 7, ProviderID: "docker", Capability: "compose_execution", CapabilityVersion: "runtime/v1"}}
	engine := gin.New()
	engine.Use(RequireAgentMTLSIdentity())
	var observedIdentity AgentMTLSIdentity
	engine.Use(func(ctx *gin.Context) {
		ctx.Next()
		observedIdentity, _ = AgentMTLSIdentityFromContext(ctx.Request.Context())
	})
	server := &AgentServer{engine: engine}
	binding := executionCertificateBinding(t, certificate)
	if err := server.ConfigureExecutionRoutes(gateway, recordingAgentBindingReader{binding: binding}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		suffix, body string
		status       int
	}{
		{"claim", `{"provider_id":"docker","capability":"compose_execution","capability_version":"runtime/v1"}`, http.StatusOK},
		{"lease-1/renew", `{"fence_token":"fence-1"}`, http.StatusOK},
		{"lease-1/material", `{"fence_token":"fence-1"}`, http.StatusOK},
		{"lease-1/logs", `{"fence_token":"fence-1","entries":[]}`, http.StatusNoContent},
		{"lease-1/results", `{"fence_token":"fence-1","protocol":"test/v1","payload":{}}`, http.StatusNoContent},
		{"lease-1/receipts", `{"fence_token":"fence-1","outcome":"success"}`, http.StatusOK},
	} {
		t.Run(test.suffix, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/agent/v1/execution-leases/"+test.suffix, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{certificate}}}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if observedIdentity.IdentityID != binding.IdentityID || observedIdentity.Generation != binding.Generation || observedIdentity.CertificateSerial != binding.CertificateSerial || observedIdentity.PublicKeyFingerprint != binding.PublicKeyFingerprint {
				t.Fatalf("resolved identity=%#v", observedIdentity)
			}
		})
	}
	if gateway.claims != 1 || gateway.materials != 1 || gateway.renewed.LeaseID != "lease-1" || gateway.receipt.Handle.LeaseID != "lease-1" {
		t.Fatalf("gateway=%#v", gateway)
	}
}

func executionCertificateBinding(t *testing.T, certificate *x509.Certificate) moduleapi.RuntimeTargetAgentBinding {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{certificate}}}
	identity, err := agentMTLSIdentityFromRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	return moduleapi.RuntimeTargetAgentBinding{IdentityID: "identity-7", TargetID: identity.TargetID, AgentID: identity.AgentID, Generation: 2, CertificateSerial: identity.CertificateSerial, PublicKeyFingerprint: identity.PublicKeyFingerprint, ProviderID: "docker", Capabilities: []string{"compose_execution"}, CapabilityVersion: "runtime/v1", Status: moduleapi.RuntimeTargetAgentStatusActive, ExpiresAt: time.Now().Add(time.Hour)}
}

func TestExecutionRoutesRejectInvalidCertificateBinding(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*moduleapi.RuntimeTargetAgentBinding)
	}{
		{"retired-certificate", func(b *moduleapi.RuntimeTargetAgentBinding) { b.CertificateSerial = "retired" }},
		{"wrong-key", func(b *moduleapi.RuntimeTargetAgentBinding) { b.PublicKeyFingerprint = "wrong-key" }},
		{"wrong-target", func(b *moduleapi.RuntimeTargetAgentBinding) { b.TargetID++ }},
		{"wrong-agent", func(b *moduleapi.RuntimeTargetAgentBinding) { b.AgentID = "other-agent" }},
		{"missing-identity", func(b *moduleapi.RuntimeTargetAgentBinding) { b.IdentityID = "" }},
		{"missing-generation", func(b *moduleapi.RuntimeTargetAgentBinding) { b.Generation = 0 }},
		{"missing-expiry", func(b *moduleapi.RuntimeTargetAgentBinding) { b.ExpiresAt = time.Time{} }},
		{"expired", func(b *moduleapi.RuntimeTargetAgentBinding) { b.ExpiresAt = time.Now().Add(-time.Minute) }},
		{"revoked", func(b *moduleapi.RuntimeTargetAgentBinding) { now := time.Now(); b.RevokedAt = &now }},
		{"inactive", func(b *moduleapi.RuntimeTargetAgentBinding) { b.Status = moduleapi.RuntimeTargetAgentStatusRevoked }},
	} {
		t.Run(test.name, func(t *testing.T) { assertExecutionCertificateRejected(t, test.change) })
	}
}

func assertExecutionCertificateRejected(t *testing.T, change func(*moduleapi.RuntimeTargetAgentBinding)) {
	t.Helper()
	certificate := testAgentCertificate(t, "spiffe://graft/runtime-target/7/runtime-agent/agent-7")
	binding := executionCertificateBinding(t, certificate)
	change(&binding)
	gateway := &recordingRuntimeAgentGateway{}
	engine := gin.New()
	engine.Use(RequireAgentMTLSIdentity())
	server := &AgentServer{engine: engine}
	if err := server.ConfigureExecutionRoutes(gateway, recordingAgentBindingReader{binding: binding}); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"claim", "lease-1/renew", "lease-1/material", "lease-1/logs", "lease-1/results", "lease-1/receipts"} {
		request := httptest.NewRequest(http.MethodPost, "/agent/v1/execution-leases/"+suffix, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{certificate}}}
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s status=%d", suffix, recorder.Code)
		}
	}
	if gateway.claims != 0 || gateway.inspects != 0 || gateway.materials != 0 {
		t.Fatalf("rejected certificate reached gateway: %#v", gateway)
	}
}

func TestLongPollAgentExecutionRejectsBindingChangeBeforeRetry(t *testing.T) {
	for _, name := range []string{"rotation", "revocation"} {
		t.Run(name, func(t *testing.T) {
			identity := AgentMTLSIdentity{AgentIdentity: moduleapi.AgentIdentity{IdentityID: "identity-7", TargetID: 7, AgentID: "agent-7", Generation: 2, CertificateSerial: "serial-7", PublicKeyFingerprint: "fingerprint-7"}}
			binding := moduleapi.RuntimeTargetAgentBinding{IdentityID: "identity-7", TargetID: 7, AgentID: "agent-7", Generation: 2, CertificateSerial: "serial-7", PublicKeyFingerprint: "fingerprint-7", Status: moduleapi.RuntimeTargetAgentStatusActive, ExpiresAt: time.Now().Add(time.Hour)}
			gateway := &recordingRuntimeAgentGateway{misses: 1}
			_, err := longPollAgentExecution(context.Background(), gateway, moduleapi.ExternalExecutionClaimRequest{}, func() bool {
				if gateway.claims > 0 {
					if name == "rotation" {
						binding.Generation++
						binding.CertificateSerial = "rotated"
					} else {
						now := time.Now()
						binding.RevokedAt = &now
					}
				}
				return agentIdentityMatchesBinding(identity, binding)
			})
			if !errors.Is(err, errAgentExecutionIdentityRejected) || gateway.claims != 1 {
				t.Fatalf("err=%v claims=%d", err, gateway.claims)
			}
		})
	}
}
