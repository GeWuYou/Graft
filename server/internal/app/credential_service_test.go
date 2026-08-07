package app

import (
	"os"
	"path/filepath"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/moduleapi"
)

func TestRegisterCoreServicesRegistersConfiguredRegistryCredentialProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	contents := []byte(`{"version":1,"credentials":[{"credential_ref":"registry:release","endpoint":"https://registry.example","repositories":["team/*"],"operations":["push"],"username":"build","password":"test-secret","expires_at":"2030-01-01T00:00:00Z"}]}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write credential source: %v", err)
	}
	runtime := &Runtime{config: &config.Config{RegistryCredentials: config.RegistryCredentialSourceConfig{File: path}}, services: container.New()}
	if err := runtime.registerCoreServices(); err != nil {
		t.Fatalf("register core services: %v", err)
	}
	if _, err := runtime.services.Resolve((*moduleapi.CredentialProvider)(nil)); err != nil {
		t.Fatalf("resolve configured credential provider: %v", err)
	}
}

func TestRegisterCoreServicesLeavesCredentialProviderUnregisteredWithoutSource(t *testing.T) {
	runtime := &Runtime{config: &config.Config{}, services: container.New()}
	if err := runtime.registerCoreServices(); err != nil {
		t.Fatalf("register core services: %v", err)
	}
	if _, err := runtime.services.Resolve((*moduleapi.CredentialProvider)(nil)); err == nil {
		t.Fatal("credential provider unexpectedly registered without a configured source")
	}
}
