package agent

import (
	"context"
	"crypto/x509"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	configuration := config{BootstrapURL: "http://backend:8443", AgentURL: "https://backend:8444", TargetID: 7, AgentID: "builder-1", BootstrapCA: "/run/trust/ca.pem", TrustBundle: "/run/trust/ca.pem"}
	if err := configuration.applyDefaultsAndValidate(); err == nil {
		t.Fatal("config accepted an HTTP bootstrap listener")
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
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	// #nosec G304 -- 测试只提供测试所有的临时状态目录中的路径。
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
