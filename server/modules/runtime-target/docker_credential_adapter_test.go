package runtimetarget

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

type recordingCredentialProvider struct {
	session      moduleapi.EphemeralCredentialSession
	injected     moduleapi.CredentialInjectionTarget
	request      moduleapi.CredentialRequest
	assessment   moduleapi.CredentialEligibilityRequest
	assessments  int
	eligibility  moduleapi.CredentialEligibility
	revoked      int
	revokeErr    error
	prepareErrAt int
	prepares     int
}

func (p *recordingCredentialProvider) Assess(_ context.Context, request moduleapi.CredentialEligibilityRequest) (moduleapi.CredentialEligibility, error) {
	p.assessments++
	p.assessment = request
	if p.eligibility.Status != "" {
		return p.eligibility, nil
	}
	return moduleapi.CredentialEligibility{Status: moduleapi.CredentialEligibilityEligible}, nil
}

func (p *recordingCredentialProvider) Prepare(_ context.Context, request moduleapi.CredentialRequest) (moduleapi.EphemeralCredentialSession, error) {
	p.prepares++
	p.request = request
	if p.prepareErrAt == p.prepares {
		return moduleapi.EphemeralCredentialSession{}, errors.New("prepare failed")
	}
	return p.session, nil
}

func (p *recordingCredentialProvider) Inject(_ context.Context, _ moduleapi.EphemeralCredentialSession, target moduleapi.CredentialInjectionTarget) error {
	p.injected = target
	return nil
}

func (p *recordingCredentialProvider) Revoke(context.Context, moduleapi.EphemeralCredentialSession) error {
	p.revoked++
	return p.revokeErr
}

type failingCredentialPublicationClient struct{}

func (failingCredentialPublicationClient) PublishImageOnTarget(context.Context, int64, moduleapi.DockerImageBuildResult, moduleapi.RegistryPublicationBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	return moduleapi.DockerImageBuildResult{}, errors.New("provider failed")
}

func (failingCredentialPublicationClient) PublishOCIManifestOnTarget(context.Context, int64, moduleapi.OCIManifestPublicationInput, moduleapi.RegistryPublicationBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIManifestPublicationResult, error) {
	return moduleapi.OCIManifestPublicationResult{}, errors.New("provider failed")
}

func (failingCredentialPublicationClient) CopyOCIArtifactOnTarget(context.Context, int64, moduleapi.OCIArtifactCopyInput, moduleapi.RegistryArtifactCopyBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error) {
	return moduleapi.OCIArtifactCopyResult{}, errors.New("provider failed")
}

func (failingCredentialPublicationClient) VerifyOCIRegistryOnTarget(context.Context, moduleapi.OCIRegistryVerificationRequest) (moduleapi.OCIRegistryVerificationResult, error) {
	return moduleapi.OCIRegistryVerificationResult{}, errors.New("provider failed")
}

type recordingOCIRegistryVerificationClient struct {
	failingCredentialPublicationClient
	request moduleapi.OCIRegistryVerificationRequest
	result  moduleapi.OCIRegistryVerificationResult
	config  string
}

func (c *recordingOCIRegistryVerificationClient) VerifyOCIRegistryOnTarget(ctx context.Context, request moduleapi.OCIRegistryVerificationRequest) (moduleapi.OCIRegistryVerificationResult, error) {
	c.request = request
	c.config, _ = ctx.Value(dockerCredentialConfigContextKey{}).(string)
	return c.result, nil
}

func testPublicationBinding() moduleapi.RegistryPublicationBinding {
	// #nosec G101 -- 测试中的不透明凭据引用不包含认证材料。
	const testCredentialRef = "credential:test"
	return moduleapi.RegistryPublicationBinding{
		Destination:   moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:test", RepositoryRef: "team/api", Reference: "v1"},
		Endpoint:      "https://registry.example",
		CredentialRef: testCredentialRef,
		AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral},
	}
}

func TestCredentialAdapterRevokesAndCleansAfterProviderFailure(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	binding := testPublicationBinding()
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image", Repository: binding.Destination.RepositoryRef, Tag: binding.Destination.Reference}, binding, nil)
	if err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("PublishImage() error = %v", err)
	}
	if provider.revoked != 1 {
		t.Fatalf("credential revoke count = %d, want 1", provider.revoked)
	}
	if provider.injected.ConfigDir == "" {
		t.Fatal("credential injection target was not provided")
	}
	if provider.request.CredentialRef != binding.CredentialRef || provider.request.Endpoint != binding.Endpoint || provider.request.RepositoryRef != binding.Destination.RepositoryRef || provider.request.Operation != "push" {
		t.Fatalf("credential request = %#v", provider.request)
	}
	if _, statErr := os.Stat(provider.injected.ConfigDir); !os.IsNotExist(statErr) {
		t.Fatalf("isolated credential directory still exists: stat error = %v", statErr)
	}
}

func TestCredentialAdapterVerifiesScopedOCIRegistryAuthenticationWithoutRepositoryAuthorizationClaim(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}}
	client := &recordingOCIRegistryVerificationClient{result: moduleapi.OCIRegistryVerificationResult{Reachable: true, ProtocolCompatible: true, AuthenticationChallenged: true, AuthenticationSucceeded: true, ProviderScopeConforms: true}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: client}
	request := testOCIRegistryVerificationRequest()

	result, err := adapter.VerifyOCIRegistry(context.Background(), request)
	if err != nil || result != client.result {
		t.Fatalf("VerifyOCIRegistry() = %#v, %v", result, err)
	}
	assertOCIRegistryVerificationLifecycle(t, provider, client, request)
}

func TestCredentialAdapterStopsBeforePrepareWhenProviderScopeDoesNotConform(t *testing.T) {
	provider := &recordingCredentialProvider{eligibility: moduleapi.CredentialEligibility{Status: moduleapi.CredentialEligibilityIneligible}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	result, err := adapter.VerifyOCIRegistry(context.Background(), testOCIRegistryVerificationRequest())
	if err != nil || result.ProviderScopeConforms || provider.prepares != 0 || provider.revoked != 0 {
		t.Fatalf("VerifyOCIRegistry() = %#v, %v; provider=%#v", result, err, provider)
	}
}

func testOCIRegistryVerificationRequest() moduleapi.OCIRegistryVerificationRequest {
	// #nosec G101 -- opaque test reference, not credential material.
	const testCredentialRef = "credential:test"
	return moduleapi.OCIRegistryVerificationRequest{RuntimeTargetID: 7, CredentialRef: testCredentialRef, Endpoint: "https://registry.example", RepositoryRef: "team/api", Operation: "push"}
}

func assertOCIRegistryVerificationLifecycle(t *testing.T, provider *recordingCredentialProvider, client *recordingOCIRegistryVerificationClient, request moduleapi.OCIRegistryVerificationRequest) {
	t.Helper()
	if provider.assessments != 1 {
		t.Fatalf("credential assessment count = %d", provider.assessments)
	}
	if provider.assessment.CredentialRef != request.CredentialRef || provider.assessment.Endpoint != request.Endpoint || provider.assessment.RepositoryRef != request.RepositoryRef || provider.assessment.Operation != request.Operation {
		t.Fatalf("credential assessment = %#v", provider.assessment)
	}
	if provider.request.CredentialRef != request.CredentialRef || provider.request.Endpoint != request.Endpoint || provider.request.RepositoryRef != request.RepositoryRef || provider.request.Operation != request.Operation {
		t.Fatalf("credential preparation = %#v", provider.request)
	}
	if client.request != request || client.config == "" || provider.revoked != 1 {
		t.Fatalf("verification client request=%#v config=%q revoked=%d", client.request, client.config, provider.revoked)
	}
	if _, statErr := os.Stat(client.config); !os.IsNotExist(statErr) {
		t.Fatalf("isolated credential directory still exists: stat error = %v", statErr)
	}
}

func TestCredentialAdapterRejectsPublicationBindingMismatchBeforeCredentialPreparation(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	binding := testPublicationBinding()
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image", Repository: binding.Destination.RepositoryRef, Tag: "other"}, binding, nil)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("PublishImage() error = %v", err)
	}
	if provider.prepares != 0 || provider.injected.ConfigDir != "" || provider.revoked != 0 {
		t.Fatalf("credential provider was reached for mismatched binding: %#v", provider)
	}
}

func TestCredentialAdapterReturnsNeedsAttentionWhenCredentialCleanupCannotBeVerified(t *testing.T) {
	provider := &recordingCredentialProvider{
		session:   moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)},
		revokeErr: errors.New("session:session-1 revoke failed"),
	}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	binding := testPublicationBinding()
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image", Repository: binding.Destination.RepositoryRef, Tag: binding.Destination.Reference}, binding, nil)
	var failure *moduleapi.ExecutionFailure
	if !errors.As(err, &failure) {
		t.Fatalf("PublishImage() error = %v, want structured failure", err)
	}
	if failure.Code != credentialCleanupUnverifiedCode || failure.Class != moduleapi.ExecutionFailureClassInternal || failure.Disposition != moduleapi.RecoveryDispositionNeedsAttention {
		t.Fatalf("cleanup failure = %#v", failure)
	}
	if strings.Contains(err.Error(), "session-1") || strings.Contains(err.Error(), provider.injected.ConfigDir) {
		t.Fatalf("cleanup failure leaked evidence: %v", err)
	}
}

func TestCredentialAdapterRejectsExpiredSessionWithoutInjection(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "expired", ExpiresAt: time.Now().UTC().Add(-time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	binding := testPublicationBinding()
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image", Repository: binding.Destination.RepositoryRef, Tag: binding.Destination.Reference}, binding, nil)
	if err == nil || !strings.Contains(err.Error(), "session is invalid") {
		t.Fatalf("PublishImage() error = %v", err)
	}
	if provider.revoked != 1 {
		t.Fatalf("expired session revoke count = %d, want 1", provider.revoked)
	}
}

func TestCredentialAdapterFailsClosedWhenEarlyRevokeCannotBeVerified(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "expired", ExpiresAt: time.Now().UTC().Add(-time.Minute)}, revokeErr: errors.New("revoke failed")}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	binding := testPublicationBinding()
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image", Repository: binding.Destination.RepositoryRef, Tag: binding.Destination.Reference}, binding, nil)
	var failure *moduleapi.ExecutionFailure
	if !errors.As(err, &failure) || failure.Code != credentialCleanupUnverifiedCode {
		t.Fatalf("early cleanup error = %v, want %s", err, credentialCleanupUnverifiedCode)
	}
}

func TestCredentialAdapterCopyFailsClosedWhenSourceRevokeCannotBeVerified(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}, prepareErrAt: 2, revokeErr: errors.New("revoke failed")}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	input := moduleapi.OCIArtifactCopyInput{Source: moduleapi.ArtifactPublicationSource{RepositoryRef: "team/source"}, Destination: moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:destination", RepositoryRef: "team/destination", Reference: "promoted"}}
	binding := moduleapi.RegistryArtifactCopyBinding{SourceEndpoint: "https://source.example", SourceCredentialRef: "ref:source", SourceAuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}, Destination: moduleapi.RegistryPublicationBinding{Destination: input.Destination, Endpoint: "https://destination.example", CredentialRef: "ref:destination", AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}}
	_, err := adapter.CopyOCIArtifact(context.Background(), 1, input, binding, nil)
	var failure *moduleapi.ExecutionFailure
	if !errors.As(err, &failure) || failure.Code != credentialCleanupUnverifiedCode {
		t.Fatalf("early copy cleanup error = %v, want %s", err, credentialCleanupUnverifiedCode)
	}
}

func TestCredentialAdapterRejectsLegacyCredentialStore(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	binding := testPublicationBinding()
	binding.AuthExecution.Mode = moduleapi.RegistryAuthExecutionDockerStore
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image", Repository: binding.Destination.RepositoryRef, Tag: binding.Destination.Reference}, binding, nil)
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("PublishImage() error = %v", err)
	}
	if provider.revoked != 0 || provider.injected.ConfigDir != "" {
		t.Fatalf("legacy credential mode reached provider: %#v", provider)
	}
}

func TestCredentialAdapterRevokesBothCopySessionsAfterProviderFailure(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	input := moduleapi.OCIArtifactCopyInput{Source: moduleapi.ArtifactPublicationSource{RepositoryRef: "team/source"}, Destination: moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:destination", RepositoryRef: "team/destination", Reference: "promoted"}}
	binding := moduleapi.RegistryArtifactCopyBinding{SourceEndpoint: "https://source.example", SourceCredentialRef: "ref:source", SourceAuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}, Destination: moduleapi.RegistryPublicationBinding{Destination: input.Destination, Endpoint: "https://destination.example", CredentialRef: "ref:destination", AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}}
	_, err := adapter.CopyOCIArtifact(context.Background(), 1, input, binding, nil)
	if err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("CopyOCIArtifact() error = %v", err)
	}
	if provider.revoked != 2 {
		t.Fatalf("credential revoke count = %d, want 2", provider.revoked)
	}
	if _, statErr := os.Stat(provider.injected.ConfigDir); !os.IsNotExist(statErr) {
		t.Fatalf("isolated credential directory still exists: stat error = %v", statErr)
	}
}
