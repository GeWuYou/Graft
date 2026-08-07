package credential

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

const testRegistrySecret = "registry-secret-value"

func TestFileProviderScopesAndRevokesCredentialSessions(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	provider := newTestFileProvider(t, now)
	session, err := provider.Prepare(context.Background(), moduleapi.CredentialRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push", ExpiresAt: now.Add(5 * time.Minute)})
	if err != nil {
		t.Fatalf("prepare credential: %v", err)
	}
	if session.ID == "" || !session.ExpiresAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("unexpected session: %#v", session)
	}

	directory := t.TempDir()
	if err := os.Chmod(directory, credentialConfigDirMode); err != nil {
		t.Fatalf("secure credential directory: %v", err)
	}
	if err := provider.Inject(context.Background(), session, moduleapi.CredentialInjectionTarget{ConfigDir: directory, Endpoint: "https://registry.example", RepositoryRef: "team/api"}); err != nil {
		t.Fatalf("inject credential: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(directory, "config.json")) // #nosec G304 -- test reads its own fixed temporary config path.
	if err != nil {
		t.Fatalf("read injected config: %v", err)
	}
	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(contents, &config); err != nil || config.Auths["https://registry.example"].Auth == "" {
		t.Fatalf("injected Docker config is invalid: %v", err)
	}
	if err := provider.Revoke(context.Background(), session); err != nil {
		t.Fatalf("revoke credential: %v", err)
	}
	if err := provider.Inject(context.Background(), session, moduleapi.CredentialInjectionTarget{ConfigDir: t.TempDir(), Endpoint: "https://registry.example", RepositoryRef: "team/api"}); err == nil || !strings.Contains(err.Error(), "session is invalid") {
		t.Fatalf("inject revoked session error = %v", err)
	}
}

func TestFileProviderMergesExistingDockerConfigAuths(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	provider := newTestFileProvider(t, now)
	directory := t.TempDir()
	if err := os.Chmod(directory, credentialConfigDirMode); err != nil {
		t.Fatalf("secure credential directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "config.json"), []byte(`{"auths":{"https://existing.example":{"auth":"existing"}}}`), credentialConfigFileMode); err != nil {
		t.Fatalf("write existing config: %v", err)
	}
	session, err := provider.Prepare(context.Background(), moduleapi.CredentialRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push", ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("prepare credential: %v", err)
	}
	if err := provider.Inject(context.Background(), session, moduleapi.CredentialInjectionTarget{ConfigDir: directory, Endpoint: "https://registry.example", RepositoryRef: "team/api"}); err != nil {
		t.Fatalf("inject credential: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(directory, "config.json")) // #nosec G304 -- The test reads only the config file created in its t.TempDir.
	if err != nil {
		t.Fatalf("read merged config: %v", err)
	}
	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(contents, &config); err != nil {
		t.Fatalf("decode merged config: %v", err)
	}
	if config.Auths["https://existing.example"].Auth != "existing" || config.Auths["https://registry.example"].Auth == "" {
		t.Fatalf("merged auths = %#v", config.Auths)
	}
}

func TestFileProviderRejectsScopeMismatchWithoutSecretDisclosure(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	provider := newTestFileProvider(t, now)
	_, err := provider.Prepare(context.Background(), moduleapi.CredentialRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "outside/api", Operation: "push", ExpiresAt: now.Add(5 * time.Minute)})
	if err == nil || !strings.Contains(err.Error(), "scope is unavailable") || strings.Contains(err.Error(), testRegistrySecret) {
		t.Fatalf("scope mismatch error = %v", err)
	}
}

func TestFileProviderRejectsExpiredSourceAndUnsafeTarget(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	provider := newTestFileProvider(t, now)
	provider.now = func() time.Time { return now.Add(31 * time.Minute) }
	_, err := provider.Prepare(context.Background(), moduleapi.CredentialRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push", ExpiresAt: now.Add(36 * time.Minute)})
	if err == nil || !strings.Contains(err.Error(), "credential is expired") || strings.Contains(err.Error(), testRegistrySecret) {
		t.Fatalf("expired source error = %v", err)
	}

	provider = newTestFileProvider(t, now)
	session, err := provider.Prepare(context.Background(), moduleapi.CredentialRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push", ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("prepare credential: %v", err)
	}
	unsafeDirectory := filepath.Join(t.TempDir(), "unsafe")
	if err := os.Mkdir(unsafeDirectory, 0o750); err != nil {
		t.Fatalf("create unsafe directory: %v", err)
	}
	if err := provider.Inject(context.Background(), session, moduleapi.CredentialInjectionTarget{ConfigDir: unsafeDirectory, Endpoint: "https://registry.example", RepositoryRef: "team/api"}); err == nil || !strings.Contains(err.Error(), "injection target is invalid") {
		t.Fatalf("unsafe target error = %v", err)
	}
}

func TestNewFileProviderRejectsMissingOrInvalidSource(t *testing.T) {
	if _, err := NewFileProvider(filepath.Join(t.TempDir(), "missing.json")); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing source error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"credentials":[]}`), 0o600); err != nil {
		t.Fatalf("write invalid source: %v", err)
	}
	if _, err := NewFileProvider(path); err == nil {
		t.Fatal("invalid source unexpectedly accepted")
	}
}

func newTestFileProvider(t *testing.T, now time.Time) *FileProvider {
	t.Helper()
	path := filepath.Join(t.TempDir(), "credentials.json")
	contents := `{"version":1,"credentials":[{"credential_ref":"registry:release","endpoint":"https://registry.example","repositories":["team/*"],"operations":["push","pull","manifest-push"],"username":"build","password":"` + testRegistrySecret + `","expires_at":"2026-08-07T12:30:00Z"}]}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write credential source: %v", err)
	}
	provider, err := NewFileProvider(path)
	if err != nil {
		t.Fatalf("new file provider: %v", err)
	}
	provider.now = func() time.Time { return now }
	return provider
}
