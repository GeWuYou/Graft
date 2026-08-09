package httpx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/config"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
)

func TestNewAgentServerIsAbsentWhenDisabled(t *testing.T) {
	server, err := NewAgentServer(config.AgentTLSConfig{}, nil)
	if err != nil {
		t.Fatalf("create disabled Agent server: %v", err)
	}
	if server != nil {
		t.Fatal("disabled Agent TLS must not create a listener")
	}
}

func TestAgentServerStartListenerUsesBoundListenerLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind listener: %v", err)
	}
	server := &AgentServer{engine: gin.New(), tlsConfig: &tls.Config{MinVersion: tls.VersionTLS13}}
	errors, err := server.StartListener(listener)
	if err != nil {
		t.Fatalf("start bound listener: %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown bound listener: %v", err)
	}
	if err, ok := <-errors; ok || err != nil {
		t.Fatalf("listener result = %v, open=%t", err, ok)
	}
}

func TestRequireAgentMTLSIdentityExtractsVerifiedURISAN(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequireAgentMTLSIdentity())
	engine.GET("/agent", func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok {
			t.Fatal("expected mTLS identity context")
		}
		if identity.TargetID != 7 || identity.AgentID != "builder-7" || identity.Generation != 3 {
			t.Fatalf("unexpected identity %#v", identity)
		}
		if identity.CertificateSerial != "42" || identity.PublicKeyFingerprint == "" {
			t.Fatalf("unexpected certificate evidence %#v", identity)
		}
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/agent", nil)
	request.Header.Set("X-Agent-Identity", "spiffe://graft/runtime-target/999/builder-agent/forged/generation/9")
	certificate := testAgentCertificate(t, "spiffe://graft/runtime-target/7/builder-agent/builder-7/generation/3")
	request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{certificate}, {certificate}}}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected verified request to pass, got %d", response.Code)
	}
}

func TestRequireAgentMTLSIdentityRejectsUnverifiedOrNoncanonicalIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zapcore.WarnLevel)
	for _, testCase := range []struct {
		name        string
		certificate *x509.Certificate
		verified    bool
	}{
		{name: "unverified", certificate: testAgentCertificate(t, "spiffe://graft/runtime-target/7/builder-agent/builder-7/generation/3")},
		{name: "leading-zero-generation", certificate: testAgentCertificate(t, "spiffe://graft/runtime-target/7/builder-agent/builder-7/generation/03"), verified: true},
		{name: "wrong-authority", certificate: testAgentCertificate(t, "spiffe://other/runtime-target/7/builder-agent/builder-7/generation/3"), verified: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assertAgentMTLSIdentityRejected(t, zap.New(core), testCase.certificate, testCase.verified)
		})
	}
	if len(observed.All()) != 3 {
		t.Fatalf("expected one denial log per rejected request, got %d", len(observed.All()))
	}
	for _, entry := range observed.All() {
		if entry.Message != "agent mTLS identity rejected" {
			t.Fatalf("unexpected denial log message %q", entry.Message)
		}
		fields := entry.ContextMap()
		if fields["reason"] != "unverified_or_invalid_client_certificate" {
			t.Fatalf("unexpected denial reason %#v", fields)
		}
		if _, present := fields["certificate"]; present {
			t.Fatalf("certificate material must not appear in denial log %#v", fields)
		}
	}
}

func assertAgentMTLSIdentityRejected(t *testing.T, logger *zap.Logger, certificate *x509.Certificate, verified bool) {
	t.Helper()
	engine := gin.New()
	engine.Use(RequireAgentMTLSIdentity(logger))
	engine.GET("/agent", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/agent", nil)
	if verified {
		request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{certificate}}}
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	assertAgentMTLSRejectionResponse(t, response)
}

func assertAgentMTLSRejectionResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected rejected request, got %d", response.Code)
	}
	body := response.Body.String()
	var payload ErrorResponse
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode rejection envelope: %v", err)
	}
	if payload.Code != errorcode.AuthTokenInvalid.String() || payload.MessageKey != messagecontract.AuthTokenInvalid.String() {
		t.Fatalf("expected uniform unauthenticated envelope, got %#v", payload)
	}
	if strings.Contains(body, "spiffe://") || strings.Contains(body, "test-public-key") {
		t.Fatalf("certificate identity leaked in response: %s", body)
	}
}

func TestParseAgentCertificateIdentityRejectsAdditionalSANs(t *testing.T) {
	certificate := testAgentCertificate(t, "spiffe://graft/runtime-target/7/builder-agent/builder-7/generation/3")
	certificate.DNSNames = []string{"agent.example.test"}
	if _, err := parseAgentCertificateIdentity(certificate); err == nil {
		t.Fatal("expected extra DNS SAN to be rejected")
	}
}

func testAgentCertificate(t *testing.T, rawURI string) *x509.Certificate {
	t.Helper()
	identityURI, err := url.Parse(rawURI)
	if err != nil {
		t.Fatalf("parse test URI: %v", err)
	}
	return &x509.Certificate{URIs: []*url.URL{identityURI}, SerialNumber: big.NewInt(42), RawSubjectPublicKeyInfo: []byte("test-public-key")}
}
