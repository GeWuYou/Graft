package agent

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStableSPIFFEIdentityExcludesGeneration(t *testing.T) {
	if got := stableSPIFFE(7, "builder-1"); got != "spiffe://graft/runtime-target/7/builder-agent/builder-1" {
		t.Fatalf("identity = %s", got)
	}
}

func TestDefaultConfigFileUsesExplicitEnvironment(t *testing.T) {
	t.Setenv(configPathEnvironment, "/tmp/docker-builder-agent.json")
	if got := defaultConfigFile(); got != "/tmp/docker-builder-agent.json" {
		t.Fatalf("default config file = %q", got)
	}
}

func TestDefaultConfigFileFallsBackToContainerPath(t *testing.T) {
	t.Setenv(configPathEnvironment, "  ")
	if got := defaultConfigFile(); got != defaultConfigPath {
		t.Fatalf("default config file = %q", got)
	}
}

func TestConfigRejectsNonHTTPSListeners(t *testing.T) {
	for _, configuration := range []config{
		{BootstrapURL: "http://backend:8443", AgentURL: "https://backend:8444", TargetID: 7, AgentID: "builder-1", BootstrapCA: "/run/trust/ca.pem", TrustBundle: "/run/trust/ca.pem"},
		{BootstrapURL: "https://backend:8443", AgentURL: "http://backend:8444", TargetID: 7, AgentID: "builder-1", BootstrapCA: "/run/trust/ca.pem", TrustBundle: "/run/trust/ca.pem"},
	} {
		if err := configuration.applyDefaultsAndValidate(); err == nil {
			t.Fatal("config accepted an HTTP listener")
		}
	}
}

func TestRunUntilDeliveryStopsWhenDevelopmentContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runUntilDelivery(ctx, filepath.Join(t.TempDir(), "missing-agent.json"))
	if err != context.Canceled {
		t.Fatalf("run until delivery error = %v, want context canceled", err)
	}
}

func TestCSRContainsStableURI(t *testing.T) {
	request, err := createCSR(stableSPIFFE(7, "builder-1"))
	if err != nil {
		t.Fatal(err)
	}
	csr, err := x509.ParseCertificateRequest(request.csrDER)
	if err != nil {
		t.Fatal(err)
	}
	if len(csr.URIs) != 1 || csr.URIs[0].String() != stableSPIFFE(7, "builder-1") {
		t.Fatalf("CSR URI = %v", csr.URIs)
	}
}

func TestStateSeparatesKeyAndCertificate(t *testing.T) {
	dir := t.TempDir()
	state := persistedState{TargetID: 7, AgentID: "builder-1", Identity: stableSPIFFE(7, "builder-1"), Generation: 2, CertificatePEM: "cert"}
	if err := saveState(dir, state, []byte("key"), []byte("trust")); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Generation != 2 {
		t.Fatalf("generation=%d", loaded.Generation)
	}
	if strings.Contains(string(mustRead(t, filepath.Join(dir, "key.pem"))), "cert") {
		t.Fatal("key and cert were mixed")
	}
}

func TestLoadStateTreatsMissingMaterialAsUnenrolled(t *testing.T) {
	dir := t.TempDir()
	state := persistedState{TargetID: 7, AgentID: "builder-1", Identity: stableSPIFFE(7, "builder-1"), Generation: 2, CertificatePEM: "cert"}
	if err := saveState(dir, state, []byte("key"), []byte("trust")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "key.pem")); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CertificatePEM != "" {
		t.Fatalf("incomplete state remained enrolled: %#v", loaded)
	}
}

func TestNewMTLSClientUsesBackendDeliveredTrustBundle(t *testing.T) {
	dir := t.TempDir()
	caPEM, certPEM, keyPEM := testTLSMaterial(t)
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "trust-bundle.pem"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	currentTrust := filepath.Join(dir, "current-ca.pem")
	if err := os.WriteFile(currentTrust, caPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	client, err := newMTLSClient(config{StateDir: dir, TrustBundle: currentTrust, AgentURL: "https://127.0.0.1:8444"}, persistedState{CertificatePEM: string(certPEM)})
	if err != nil {
		t.Fatal(err)
	}
	pool := client.Transport.(*http.Transport).TLSClientConfig.RootCAs
	cert, err := x509.ParseCertificate(pemDecode(t, caPEM))
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // 测试只需确认 RootCAs 装载了配置文件中的唯一 CA。
	if roots := pool.Subjects(); len(roots) != 1 || string(roots[0]) != string(cert.RawSubject) {
		t.Fatalf("RootCAs did not use configured bundle: subjects=%d", len(roots))
	}
}

func testTLSMaterial(t *testing.T) (caPEM, certPEM, keyPEM []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(-time.Minute)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "graft.local"}, NotBefore: now, NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustMarshalKey(t, key)})
}

func mustMarshalKey(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

func pemDecode(t *testing.T, data []byte) []byte {
	t.Helper()
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("missing PEM block")
	}
	return block.Bytes
}

func TestStateTrustBundleMatchesConfiguredBundle(t *testing.T) {
	dir := t.TempDir()
	configured := filepath.Join(dir, "configured-ca.pem")
	stateDir := filepath.Join(dir, "state")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configured, []byte("ca-v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "trust-bundle.pem"), []byte("ca-v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !stateTrustBundleMatches(configured, stateDir) {
		t.Fatal("matching trust bundle was rejected")
	}
	if err := os.WriteFile(filepath.Join(stateDir, "trust-bundle.pem"), []byte("ca-v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if stateTrustBundleMatches(configured, stateDir) {
		t.Fatal("stale trust bundle was accepted")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	// #nosec G304 -- 测试只提供测试所有的临时状态目录中的路径。
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
