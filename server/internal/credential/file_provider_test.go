package credential

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

const testRegistrySecret = "registry-secret-value"

func TestDockerAuthKeyUsesRegistryHost(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		endpoint string
		want     string
	}{
		{endpoint: "https://registry.example", want: "registry.example"},
		{endpoint: "https://registry.example:5443/team", want: "registry.example:5443"},
		{endpoint: "https://docker.io", want: "https://index.docker.io/v1/"},
		{endpoint: "https://index.docker.io/v1", want: "https://index.docker.io/v1/"},
	} {
		if got := dockerAuthKey(test.endpoint); got != test.want {
			t.Errorf("dockerAuthKey(%q) = %q, want %q", test.endpoint, got, test.want)
		}
	}
}

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
	if err := json.Unmarshal(contents, &config); err != nil || config.Auths["registry.example"].Auth == "" {
		t.Fatalf("injected Docker config is invalid: %v", err)
	}
	if err := provider.Revoke(context.Background(), session); err != nil {
		t.Fatalf("revoke credential: %v", err)
	}
	if err := provider.Inject(context.Background(), session, moduleapi.CredentialInjectionTarget{ConfigDir: t.TempDir(), Endpoint: "https://registry.example", RepositoryRef: "team/api"}); err == nil || !strings.Contains(err.Error(), "session is invalid") {
		t.Fatalf("inject revoked session error = %v", err)
	}
}

func TestFileProviderResolvesMaterialOnlyForActiveScopedSession(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	provider := newTestFileProvider(t, now)
	session, err := provider.Prepare(context.Background(), moduleapi.CredentialRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push", ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("prepare credential: %v", err)
	}
	target := moduleapi.CredentialInjectionTarget{Endpoint: "https://registry.example", RepositoryRef: "team/api"}
	material, err := provider.ResolveCredentialMaterial(context.Background(), session, target)
	if err != nil {
		t.Fatalf("resolve credential material: %v", err)
	}
	if material.Username != "build" || material.Secret != testRegistrySecret {
		t.Fatalf("unexpected credential material: %#v", material)
	}
	if _, err := provider.ResolveCredentialMaterial(context.Background(), session, moduleapi.CredentialInjectionTarget{Endpoint: target.Endpoint, RepositoryRef: "outside/api"}); err == nil || strings.Contains(err.Error(), testRegistrySecret) {
		t.Fatalf("scope mismatch error = %v", err)
	}
	if err := provider.Revoke(context.Background(), session); err != nil {
		t.Fatalf("revoke credential: %v", err)
	}
	if _, err := provider.ResolveCredentialMaterial(context.Background(), session, target); err == nil || strings.Contains(err.Error(), testRegistrySecret) {
		t.Fatalf("revoked material error = %v", err)
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
	if config.Auths["https://existing.example"].Auth != "existing" || config.Auths["registry.example"].Auth == "" {
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

func TestFileProviderAssessesKnownScopeWithoutSecretDisclosure(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	provider := newTestFileProvider(t, now)

	eligible, err := provider.Assess(context.Background(), moduleapi.CredentialEligibilityRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push"})
	if err != nil || eligible.Status != moduleapi.CredentialEligibilityEligible {
		t.Fatalf("eligible assessment = %#v, %v", eligible, err)
	}
	if text := fmt.Sprintf("%#v", eligible); strings.Contains(text, testRegistrySecret) || strings.Contains(text, "password") || strings.Contains(text, "expires") {
		t.Fatalf("eligibility leaked secret-derived data: %s", text)
	}

	ineligible, err := provider.Assess(context.Background(), moduleapi.CredentialEligibilityRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "outside/api", Operation: "push"})
	if err != nil || ineligible.Status != moduleapi.CredentialEligibilityIneligible {
		t.Fatalf("ineligible assessment = %#v, %v", ineligible, err)
	}
}

func TestFileProviderAssessmentReloadsSourceAndRejectsExpiredOrInvalidRequests(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	provider := newTestFileProvider(t, now)
	request := moduleapi.CredentialEligibilityRequest{CredentialRef: "registry:release", Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push"}
	if eligibility, err := provider.Assess(context.Background(), request); err != nil || eligibility.Status != moduleapi.CredentialEligibilityEligible {
		t.Fatalf("initial assessment = %#v, %v", eligibility, err)
	}
	contents := `{"version":1,"credentials":[{"credential_ref":"registry:release","endpoint":"https://registry.example","repositories":["team/*"],"operations":["push"],"username":"build","password":"` + testRegistrySecret + `","expires_at":"2026-08-07T11:59:00Z"}]}`
	if err := os.WriteFile(provider.path, []byte(contents), 0o600); err != nil {
		t.Fatalf("replace credential source: %v", err)
	}
	if eligibility, err := provider.Assess(context.Background(), request); err != nil || eligibility.Status != moduleapi.CredentialEligibilityIneligible {
		t.Fatalf("reloaded expired assessment = %#v, %v", eligibility, err)
	}
	if eligibility, err := provider.Assess(context.Background(), moduleapi.CredentialEligibilityRequest{CredentialRef: "registry:release", Endpoint: "http://registry.example", RepositoryRef: "team/api", Operation: "push"}); err != nil || eligibility.Status != moduleapi.CredentialEligibilityIneligible {
		t.Fatalf("invalid assessment = %#v, %v", eligibility, err)
	}
	if err := os.WriteFile(provider.path, []byte(`{"version":1,"credentials":[]}`), 0o600); err != nil {
		t.Fatalf("invalidate credential source: %v", err)
	}
	if _, err := provider.Assess(context.Background(), request); err == nil || strings.Contains(err.Error(), testRegistrySecret) {
		t.Fatalf("invalid source assessment error = %v", err)
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
