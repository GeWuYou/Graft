package httpx

import (
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
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
	request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{testAgentCertificate(t, "spiffe://graft/runtime-target/7/builder-agent/builder-7/generation/3")}}}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected verified request to pass, got %d", response.Code)
	}
}

func TestRequireAgentMTLSIdentityRejectsUnverifiedOrNoncanonicalIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
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
			engine := gin.New()
			engine.Use(RequireAgentMTLSIdentity())
			engine.GET("/agent", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodGet, "/agent", nil)
			if testCase.verified {
				request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{testCase.certificate}}}
			}
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expected rejected request, got %d", response.Code)
			}
		})
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
