package credentialvault

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/event"
	"graft/server/internal/moduleapi"
	runtimetargetcontract "graft/server/modules/runtime-target/contract"
)

//nolint:gocognit,gocyclo,cyclop // 单个场景固定验证 AppRole、PKI 签发和秘密不落盘这条不可拆分的安全边界。
func TestVaultPKIClientUsesDockerSecretsForAppRoleAndPersistsOnlySerial(t *testing.T) {
	certificatePEM, certificateSerial, csrDER := newVaultConformanceCertificate(t)
	roleIDPath := filepath.Join(t.TempDir(), "role-id")
	secretIDPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(roleIDPath, []byte("role-from-docker-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretIDPath, []byte("secret-from-docker-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := &conformanceIssuanceStore{}
	var loginBody map[string]string
	var issueBody map[string]string
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/auth/approle/login":
			if request.Header.Get("X-Vault-Token") != "" {
				t.Errorf("login request unexpectedly has Vault token")
			}
			if err := json.NewDecoder(request.Body).Decode(&loginBody); err != nil {
				t.Errorf("decode login body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"auth":{"client_token":"session-token"}}`))
		case "/v1/pki/issue/agent":
			if got := request.Header.Get("X-Vault-Token"); got != "session-token" {
				t.Errorf("issue token = %q, want session-token", got)
			}
			if err := json.NewDecoder(request.Body).Decode(&issueBody); err != nil {
				t.Errorf("decode issue body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"data":{"certificate":` + strconvQuote(certificatePEM) + `,"serial_number":"` + certificateSerial + `","expiration":4102444800}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewVaultPKIClient(config.CredentialVaultConfig{
		Address: server.URL, AuthMount: "approle", AuthRole: "agent", AuthRoleIDFile: roleIDPath,
		AuthSecretIDFile: secretIDPath, PKIMount: "pki", PKIRole: "agent", TrustBundleRef: "vault://pki/ca",
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := client.IssueCSR(context.Background(), moduleapi.AgentCertificateIssuanceRequest{IssuanceKey: "issue-1", SPIFFEURI: "spiffe://graft/runtime-target/7/builder-agent/agent-7/generation/1", CSRDER: csrDER})
	if err != nil {
		t.Fatalf("issue certificate: %v", err)
	}
	if issued.CertificateSerial != certificateSerial {
		t.Fatalf("certificate serial = %q, want %q", issued.CertificateSerial, certificateSerial)
	}
	if loginBody["role_id"] != "role-from-docker-secret" || loginBody["secret_id"] != "secret-from-docker-secret" {
		t.Fatalf("AppRole login body = %#v", loginBody)
	}
	if !strings.HasPrefix(issueBody["csr"], "-----BEGIN CERTIFICATE REQUEST-----") {
		t.Fatalf("issue CSR is not PEM encoded: %q", issueBody["csr"])
	}
	if issueBody["uri_sans"] != "spiffe://graft/runtime-target/7/builder-agent/agent-7/generation/1" {
		t.Fatalf("issue URI SAN = %q", issueBody["uri_sans"])
	}
	if store.state != (IssuanceState{IssuanceKey: "issue-1", Serial: certificateSerial}) {
		t.Fatalf("persisted state = %#v", store.state)
	}
	stateJSON, _ := json.Marshal(store.state)
	for _, secret := range []string{"role-from-docker-secret", "secret-from-docker-secret", "session-token", certificatePEM} {
		if strings.Contains(string(stateJSON), secret) {
			t.Fatalf("persisted state contains secret material %q: %s", secret, stateJSON)
		}
	}
	if requestCount != 2 {
		t.Fatalf("Vault request count = %d, want login plus issue", requestCount)
	}
}

func TestVaultPKIClientReconcileRehydratesPersistedSerialAfterRestart(t *testing.T) {
	certificatePEM, certificateSerial, _ := newVaultConformanceCertificate(t)
	store := &conformanceIssuanceStore{state: IssuanceState{IssuanceKey: "issue-1", Serial: certificateSerial}}
	roleIDPath, secretIDPath := writeConformanceSecrets(t)
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/auth/approle/login":
			_, _ = writer.Write([]byte(`{"auth":{"client_token":"restart-token"}}`))
		case "/v1/pki/cert/serial-42":
			if request.Header.Get("X-Vault-Token") != "restart-token" {
				t.Errorf("certificate read token = %q", request.Header.Get("X-Vault-Token"))
			}
			_, _ = writer.Write([]byte(`{"data":{"certificate":` + strconvQuote(certificatePEM) + `,"serial_number":"serial-42","expiration":4102444800}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client, err := NewVaultPKIClient(config.CredentialVaultConfig{Address: server.URL, AuthMount: "approle", AuthRole: "agent", AuthRoleIDFile: roleIDPath, AuthSecretIDFile: secretIDPath, PKIMount: "pki", PKIRole: "agent"}, store)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := client.ReconcileCSR(context.Background(), "issue-1")
	if err != nil {
		t.Fatalf("reconcile certificate: %v", err)
	}
	if issued.CertificateSerial != certificateSerial || len(issued.CertificateChainDER) != 1 {
		t.Fatalf("reconciled certificate = %#v", issued)
	}
	if strings.Join(paths, ",") != "/v1/auth/approle/login,/v1/pki/cert/serial-42" {
		t.Fatalf("restart reconciliation paths = %v", paths)
	}
}

func TestAgentCertificateRevocationHandlerConformance(t *testing.T) {
	tests := []struct {
		name        string
		version     uint16
		id          string
		idempotency string
		payload     string
		wantErr     bool
		wantKey     string
		wantCalls   int
	}{
		{name: "explicit stable key", version: 1, id: "event-id", idempotency: "stable-key", payload: `{"certificate_serial":"serial-1"}`, wantKey: "stable-key", wantCalls: 1},
		{name: "event id fallback", version: 1, id: "event-id", payload: `{"certificate_serial":"serial-1"}`, wantKey: "event-id", wantCalls: 1},
		{name: "invalid version", version: 2, id: "event-id", payload: `{"certificate_serial":"serial-1"}`, wantErr: true},
		{name: "invalid envelope payload", version: 1, id: "event-id", payload: `{`, wantErr: true},
		{name: "empty serial no-op", version: 1, id: "event-id", payload: `{"certificate_serial":" "}`, wantCalls: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issuer := &conformanceRevocationIssuer{}
			handler := agentCertificateRevocationHandler{issuer: issuer}
			err := handler.Handle(context.Background(), event.Event{ID: test.id, Type: runtimetargetcontract.AgentCertificateRevocationEventType, Version: test.version, Payload: []byte(test.payload), IdempotencyKey: test.idempotency})
			if (err != nil) != test.wantErr {
				t.Fatalf("handle error = %v, wantErr %v", err, test.wantErr)
			}
			if issuer.calls != test.wantCalls {
				t.Fatalf("issuer calls = %d, want %d", issuer.calls, test.wantCalls)
			}
			if test.wantKey != "" && issuer.revocation.IdempotencyKey != test.wantKey {
				t.Fatalf("idempotency key = %q, want %q", issuer.revocation.IdempotencyKey, test.wantKey)
			}
		})
	}
}

type conformanceIssuanceStore struct{ state IssuanceState }

func (s *conformanceIssuanceStore) Load(_ context.Context, issuanceKey string) (IssuanceState, error) {
	if s.state.IssuanceKey != issuanceKey {
		return IssuanceState{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	return s.state, nil
}

func (s *conformanceIssuanceStore) Save(_ context.Context, state IssuanceState) error {
	s.state = state
	return nil
}

type conformanceRevocationIssuer struct {
	calls      int
	revocation moduleapi.AgentCertificateRevocation
}

func (i *conformanceRevocationIssuer) IssueCSR(context.Context, moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{}, nil
}

func (i *conformanceRevocationIssuer) ReconcileCSR(context.Context, string) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{}, nil
}

func (i *conformanceRevocationIssuer) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{}, nil
}

func (i *conformanceRevocationIssuer) RevokeCertificate(_ context.Context, revocation moduleapi.AgentCertificateRevocation) error {
	i.calls++
	i.revocation = revocation
	return nil
}

func writeConformanceSecrets(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	roleIDPath, secretIDPath := filepath.Join(dir, "role-id"), filepath.Join(dir, "secret-id")
	if err := os.WriteFile(roleIDPath, []byte("role\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretIDPath, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return roleIDPath, secretIDPath
}

func newVaultConformanceCertificate(t *testing.T) (string, string, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{SerialNumber: new(big.Int).SetInt64(42), Subject: pkix.Name{CommonName: "agent"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "agent"}}, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), "serial-42", csrDER
}

func strconvQuote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
