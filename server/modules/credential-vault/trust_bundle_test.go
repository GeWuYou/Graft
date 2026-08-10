package credentialvault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

func TestVaultPKIClientReadTrustBundleUsesCAPKIExpiry(t *testing.T) {
	certificatePEM, _, _ := newVaultConformanceCertificate(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/auth/approle/login":
			_, _ = writer.Write([]byte(`{"auth":{"client_token":"session-token"}}`))
		case "/v1/pki/cert/ca":
			if request.Header.Get("X-Vault-Token") != "session-token" {
				t.Errorf("CA request token = %q", request.Header.Get("X-Vault-Token"))
			}
			_, _ = writer.Write([]byte(`{"data":{"certificate":` + strconvQuote(certificatePEM) + `}}`))
		default:
			t.Errorf("unexpected request path %q", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	caPath := writeVaultTestCA(t, server)
	roleIDPath, secretIDPath := writeConformanceSecrets(t)
	client, err := NewVaultPKIClient(config.CredentialVaultConfig{Address: server.URL, CAFile: caPath, AuthMount: "approle", AuthRole: "agent", AuthRoleIDFile: roleIDPath, AuthSecretIDFile: secretIDPath, PKIMount: "pki", PKIRole: "agent", TrustBundleRef: "vault://pki/ca"}, &conformanceIssuanceStore{})
	if err != nil {
		t.Fatalf("create Vault client: %v", err)
	}
	bundle, err := client.ReadTrustBundle(context.Background(), moduleapi.TrustBundleRequest{})
	if err != nil {
		t.Fatalf("read trust bundle: %v", err)
	}
	if bundle.Reference != "vault://pki/ca" || bundle.Version != "vault-pki" || bundle.ExpiresAt.IsZero() {
		t.Fatalf("trust bundle = %#v", bundle)
	}
}
