package httpx

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/config"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/moduleapi"
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

//nolint:gocyclo // TLS listener、真实客户端握手、重连与身份提取共同覆盖不可拆分的 mTLS conformance seam。
func TestAgentServerAcceptsCASignedClientCertificateOverTLS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	ca, caKey := createAgentMTLSTestCA(t, now)
	serverCertificate := createAgentMTLSTestCertificate(t, ca, caKey, now, agentMTLSTestLeaf{commonName: "graft-agent-server"})
	identityURI, err := url.Parse("spiffe://graft/runtime-target/7/builder-agent/builder-7/generation/3")
	if err != nil {
		t.Fatalf("parse client identity URI: %v", err)
	}
	clientCertificate := createAgentMTLSTestCertificate(t, ca, caKey, now, agentMTLSTestLeaf{identityURI: identityURI, client: true})
	directory := t.TempDir()
	serverCertPath, serverKeyPath, clientCAPath := writeAgentMTLSTestMaterial(t, directory, ca, serverCertificate)
	server, err := NewAgentServer(config.AgentTLSConfig{Enabled: true, CertificateFile: serverCertPath, KeyFile: serverKeyPath, ClientCAFile: clientCAPath}, nil)
	if err != nil {
		t.Fatalf("create Agent server: %v", err)
	}
	reader := &recordingAgentLedgerReader{snapshot: moduleapi.RuntimeTargetLedgerSnapshot{SnapshotID: "snapshot", SnapshotDigest: "digest", ExpiresAt: now.Add(time.Minute)}}
	if err := server.ConfigureLedgerRoutes(reader); err != nil {
		t.Fatalf("configure ledger routes: %v", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind listener: %v", err)
	}
	errors, err := server.StartListener(listener)
	if err != nil {
		t.Fatalf("start listener: %v", err)
	}
	t.Cleanup(func() {
		_ = server.Shutdown(context.Background())
		<-errors
	})
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{clientCertificate}, ServerName: "graft-agent-server"}}}
	response, err := client.Get("https://" + listener.Addr().String() + agentLedgerSnapshotPath)
	if err != nil {
		t.Fatalf("GET ledger snapshot over mTLS: %v", err)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" {
		_ = response.Body.Close()
		t.Fatalf("mTLS snapshot response = %d %#v", response.StatusCode, response.Header)
	}
	_ = response.Body.Close()
	// 第二次请求复用同一 mTLS 客户端，覆盖 Agent 重连后的证书验证与身份提取路径。
	reconnect, err := client.Get("https://" + listener.Addr().String() + agentLedgerSnapshotPath)
	if err != nil {
		t.Fatalf("reconnect ledger snapshot over mTLS: %v", err)
	}
	if reconnect.StatusCode != http.StatusOK || reconnect.Header.Get("Cache-Control") != "no-store" {
		_ = reconnect.Body.Close()
		t.Fatalf("mTLS reconnect response = %d %#v", reconnect.StatusCode, reconnect.Header)
	}
	_ = reconnect.Body.Close()
	if reader.issued.TargetID != 7 || reader.issued.AgentID != "builder-7" || reader.issued.Generation != 3 || reader.issued.CertificateSerial == "" || reader.issued.PublicKeyFingerprint == "" {
		t.Fatalf("verified identity = %#v", reader.issued)
	}
}

func createAgentMTLSTestCA(t *testing.T, now time.Time) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "graft-agent-test-ca"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	encoded, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(encoded)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}
	return certificate, key
}

type agentMTLSTestLeaf struct {
	commonName  string
	identityURI *url.URL
	client      bool
}

func createAgentMTLSTestCertificate(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, now time.Time, leaf agentMTLSTestLeaf) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(now.UnixNano()), Subject: pkix.Name{CommonName: leaf.commonName}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
	if leaf.client {
		template.URIs = []*url.URL{leaf.identityURI}
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		template.DNSNames = []string{leaf.commonName}
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	encoded, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{encoded, ca.Raw}, PrivateKey: key}
}

func writeAgentMTLSTestMaterial(t *testing.T, directory string, ca *x509.Certificate, server tls.Certificate) (string, string, string) {
	t.Helper()
	serverCertPath, serverKeyPath, clientCAPath := directory+"/server.pem", directory+"/server-key.pem", directory+"/client-ca.pem"
	key := server.PrivateKey.(*ecdsa.PrivateKey)
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal server key: %v", err)
	}
	if err := os.WriteFile(serverCertPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate[0]}), 0o600); err != nil {
		t.Fatalf("write server certificate: %v", err)
	}
	if err := os.WriteFile(serverKeyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatalf("write server key: %v", err)
	}
	if err := os.WriteFile(clientCAPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw}), 0o600); err != nil {
		t.Fatalf("write client CA: %v", err)
	}
	return serverCertPath, serverKeyPath, clientCAPath
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
