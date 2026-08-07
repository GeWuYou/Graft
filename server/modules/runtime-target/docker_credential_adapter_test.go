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
	session  moduleapi.EphemeralCredentialSession
	injected moduleapi.CredentialInjectionTarget
	revoked  int
}

func (p *recordingCredentialProvider) Prepare(context.Context, moduleapi.CredentialRequest) (moduleapi.EphemeralCredentialSession, error) {
	return p.session, nil
}

func (p *recordingCredentialProvider) Inject(_ context.Context, _ moduleapi.EphemeralCredentialSession, target moduleapi.CredentialInjectionTarget) error {
	p.injected = target
	return nil
}

func (p *recordingCredentialProvider) Revoke(context.Context, moduleapi.EphemeralCredentialSession) error {
	p.revoked++
	return nil
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

func TestCredentialAdapterRevokesAndCleansAfterProviderFailure(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	credentialRef := "credential" + ":one"
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image"}, moduleapi.RegistryPublicationBinding{Endpoint: "https://registry.example", CredentialRef: credentialRef, AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}, nil)
	if err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("PublishImage() error = %v", err)
	}
	if provider.revoked != 1 {
		t.Fatalf("credential revoke count = %d, want 1", provider.revoked)
	}
	if provider.injected.ConfigDir == "" {
		t.Fatal("credential injection target was not provided")
	}
	if _, statErr := os.Stat(provider.injected.ConfigDir); !os.IsNotExist(statErr) {
		t.Fatalf("isolated credential directory still exists: stat error = %v", statErr)
	}
}

func TestCredentialAdapterRejectsExpiredSessionWithoutInjection(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "expired", ExpiresAt: time.Now().UTC().Add(-time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	credentialRef := "credential" + ":one"
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image"}, moduleapi.RegistryPublicationBinding{Endpoint: "https://registry.example", CredentialRef: credentialRef, AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}, nil)
	if err == nil || !strings.Contains(err.Error(), "session is invalid") {
		t.Fatalf("PublishImage() error = %v", err)
	}
	if provider.revoked != 1 {
		t.Fatalf("expired session revoke count = %d, want 1", provider.revoked)
	}
}

func TestCredentialAdapterRejectsLegacyCredentialStore(t *testing.T) {
	provider := &recordingCredentialProvider{session: moduleapi.EphemeralCredentialSession{ID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Minute)}}
	adapter := dockerCredentialExecutionAdapter{provider: provider, client: failingCredentialPublicationClient{}}
	_, err := adapter.PublishImage(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "image"}, moduleapi.RegistryPublicationBinding{Endpoint: "https://registry.example", CredentialRef: "ref:one", AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionDockerStore}}, nil)
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
	input := moduleapi.OCIArtifactCopyInput{Source: moduleapi.ArtifactPublicationSource{RepositoryRef: "team/source"}, Destination: moduleapi.AuthorizedArtifactDestination{RepositoryRef: "team/destination"}}
	binding := moduleapi.RegistryArtifactCopyBinding{SourceEndpoint: "https://source.example", SourceCredentialRef: "ref:source", SourceAuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}, Destination: moduleapi.RegistryPublicationBinding{Endpoint: "https://destination.example", CredentialRef: "ref:destination", AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}}
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
