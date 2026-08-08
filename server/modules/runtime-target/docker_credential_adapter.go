package runtimetarget

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const credentialSessionTTL = 5 * time.Minute

const dockerCredentialConfigFileMode os.FileMode = 0o600

const dockerCredentialConfigDirMode os.FileMode = 0o700

// #nosec G101 -- 这是公开的失败分类标识，不是凭据或认证材料。
const credentialCleanupUnverifiedCode = "credential_cleanup_unverified"

// dockerCredentialExecutionAdapter 将 Registry 的短期凭据会话限制在单次 Docker
// 操作的临时 DOCKER_CONFIG 中，并在所有终态撤销会话。
type dockerCredentialExecutionAdapter struct {
	provider moduleapi.CredentialProvider
	client   credentialPublicationClient
}

type credentialPublicationClient interface {
	PublishImageOnTarget(context.Context, int64, moduleapi.DockerImageBuildResult, moduleapi.RegistryPublicationBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error)
	PublishOCIManifestOnTarget(context.Context, int64, moduleapi.OCIManifestPublicationInput, moduleapi.RegistryPublicationBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIManifestPublicationResult, error)
	CopyOCIArtifactOnTarget(context.Context, int64, moduleapi.OCIArtifactCopyInput, moduleapi.RegistryArtifactCopyBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error)
}

func (a dockerCredentialExecutionAdapter) PublishImage(ctx context.Context, targetID int64, result moduleapi.DockerImageBuildResult, binding moduleapi.RegistryPublicationBinding, sink moduleapi.DockerImageBuildLogSink) (published moduleapi.DockerImageBuildResult, err error) {
	if a.provider == nil || binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral || strings.TrimSpace(binding.CredentialRef) == "" || strings.TrimSpace(binding.Endpoint) == "" {
		return result, errors.New("ephemeral registry credential provider is unavailable")
	}
	session, err := a.provider.Prepare(ctx, moduleapi.CredentialRequest{CredentialRef: binding.CredentialRef, Endpoint: binding.Endpoint, RepositoryRef: binding.Destination.RepositoryRef, Operation: "push", ExpiresAt: time.Now().UTC().Add(credentialSessionTTL)})
	if err != nil {
		return result, fmt.Errorf("prepare registry credential: %w", err)
	}
	configDir, cleanup, err := isolatedDockerConfig(session)
	if err != nil {
		_ = a.provider.Revoke(context.WithoutCancel(ctx), session)
		return result, err
	}
	defer func() {
		revokeErr := a.provider.Revoke(context.WithoutCancel(ctx), session)
		cleanupErr := cleanup()
		if revokeErr != nil || cleanupErr != nil {
			err = credentialCleanupFailure(err)
		}
	}()
	if err := a.provider.Inject(ctx, session, moduleapi.CredentialInjectionTarget{ConfigDir: configDir, Endpoint: binding.Endpoint, RepositoryRef: binding.Destination.RepositoryRef}); err != nil {
		return result, fmt.Errorf("inject registry credential: %w", err)
	}
	return a.client.PublishImageOnTarget(context.WithValue(ctx, dockerCredentialConfigContextKey{}, configDir), targetID, result, binding, sink)
}

func (a dockerCredentialExecutionAdapter) PublishManifest(ctx context.Context, targetID int64, input moduleapi.OCIManifestPublicationInput, binding moduleapi.RegistryPublicationBinding, sink moduleapi.DockerImageBuildLogSink) (published moduleapi.OCIManifestPublicationResult, err error) {
	if a.provider == nil || binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral || strings.TrimSpace(binding.CredentialRef) == "" || strings.TrimSpace(binding.Endpoint) == "" {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("ephemeral registry credential provider is unavailable")
	}
	session, err := a.provider.Prepare(ctx, moduleapi.CredentialRequest{CredentialRef: binding.CredentialRef, Endpoint: binding.Endpoint, RepositoryRef: binding.Destination.RepositoryRef, Operation: "manifest-push", ExpiresAt: time.Now().UTC().Add(credentialSessionTTL)})
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, fmt.Errorf("prepare registry credential: %w", err)
	}
	configDir, cleanup, err := isolatedDockerConfig(session)
	if err != nil {
		_ = a.provider.Revoke(context.WithoutCancel(ctx), session)
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	defer func() {
		revokeErr := a.provider.Revoke(context.WithoutCancel(ctx), session)
		cleanupErr := cleanup()
		if revokeErr != nil || cleanupErr != nil {
			err = credentialCleanupFailure(err)
		}
	}()
	if err := a.provider.Inject(ctx, session, moduleapi.CredentialInjectionTarget{ConfigDir: configDir, Endpoint: binding.Endpoint, RepositoryRef: binding.Destination.RepositoryRef}); err != nil {
		return moduleapi.OCIManifestPublicationResult{}, fmt.Errorf("inject registry credential: %w", err)
	}
	return a.client.PublishOCIManifestOnTarget(context.WithValue(ctx, dockerCredentialConfigContextKey{}, configDir), targetID, input, binding, sink)
}

// CopyOCIArtifact 为来源读取和目标写入分别申请最小范围的短期凭据，二者只存在于同一隔离 Docker 配置中。
//
//nolint:cyclop,gocyclo // 双端凭据、隔离注入和全部终态撤销必须保持在同一安全边界内。
func (a dockerCredentialExecutionAdapter) CopyOCIArtifact(ctx context.Context, targetID int64, input moduleapi.OCIArtifactCopyInput, binding moduleapi.RegistryArtifactCopyBinding, sink moduleapi.DockerImageBuildLogSink) (copied moduleapi.OCIArtifactCopyResult, err error) {
	if a.provider == nil || binding.SourceAuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral || binding.Destination.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral || strings.TrimSpace(binding.SourceCredentialRef) == "" || strings.TrimSpace(binding.SourceEndpoint) == "" || strings.TrimSpace(binding.Destination.CredentialRef) == "" || strings.TrimSpace(binding.Destination.Endpoint) == "" {
		return moduleapi.OCIArtifactCopyResult{}, errors.New("ephemeral registry credential provider is unavailable")
	}
	sourceSession, err := a.provider.Prepare(ctx, moduleapi.CredentialRequest{CredentialRef: binding.SourceCredentialRef, Endpoint: binding.SourceEndpoint, RepositoryRef: input.Source.RepositoryRef, Operation: "pull", ExpiresAt: time.Now().UTC().Add(credentialSessionTTL)})
	if err != nil {
		return moduleapi.OCIArtifactCopyResult{}, fmt.Errorf("prepare source registry credential: %w", err)
	}
	destinationSession, err := a.provider.Prepare(ctx, moduleapi.CredentialRequest{CredentialRef: binding.Destination.CredentialRef, Endpoint: binding.Destination.Endpoint, RepositoryRef: input.Destination.RepositoryRef, Operation: "push", ExpiresAt: time.Now().UTC().Add(credentialSessionTTL)})
	if err != nil {
		_ = a.provider.Revoke(context.WithoutCancel(ctx), sourceSession)
		return moduleapi.OCIArtifactCopyResult{}, fmt.Errorf("prepare destination registry credential: %w", err)
	}
	configDir, cleanup, err := isolatedDockerConfig(sourceSession)
	if err != nil {
		_ = a.provider.Revoke(context.WithoutCancel(ctx), destinationSession)
		_ = a.provider.Revoke(context.WithoutCancel(ctx), sourceSession)
		return moduleapi.OCIArtifactCopyResult{}, err
	}
	defer func() {
		cleanupFailed := false
		for _, session := range []moduleapi.EphemeralCredentialSession{destinationSession, sourceSession} {
			if revokeErr := a.provider.Revoke(context.WithoutCancel(ctx), session); revokeErr != nil {
				cleanupFailed = true
			}
		}
		if cleanupErr := cleanup(); cleanupErr != nil {
			cleanupFailed = true
		}
		if cleanupFailed {
			err = credentialCleanupFailure(err)
		}
	}()
	if err := a.provider.Inject(ctx, sourceSession, moduleapi.CredentialInjectionTarget{ConfigDir: configDir, Endpoint: binding.SourceEndpoint, RepositoryRef: input.Source.RepositoryRef}); err != nil {
		return moduleapi.OCIArtifactCopyResult{}, fmt.Errorf("inject source registry credential: %w", err)
	}
	if err := a.provider.Inject(ctx, destinationSession, moduleapi.CredentialInjectionTarget{ConfigDir: configDir, Endpoint: binding.Destination.Endpoint, RepositoryRef: input.Destination.RepositoryRef}); err != nil {
		return moduleapi.OCIArtifactCopyResult{}, fmt.Errorf("inject destination registry credential: %w", err)
	}
	return a.client.CopyOCIArtifactOnTarget(context.WithValue(ctx, dockerCredentialConfigContextKey{}, configDir), targetID, input, binding, sink)
}

// credentialCleanupFailure 不泄漏本地临时目录、会话或底层凭据细节；Task Runtime
// 据此禁止自动重试和凭据/Reservation 复用，等待人工核对。
func credentialCleanupFailure(_ error) error {
	return &moduleapi.ExecutionFailure{
		Code:        credentialCleanupUnverifiedCode,
		Class:       moduleapi.ExecutionFailureClassInternal,
		Disposition: moduleapi.RecoveryDispositionNeedsAttention,
		Cause:       errors.New("credential cleanup could not be verified"),
	}
}

func isolatedDockerConfig(session moduleapi.EphemeralCredentialSession) (string, func() error, error) {
	if strings.TrimSpace(session.ID) == "" || session.ExpiresAt.Before(time.Now().UTC()) {
		return "", nil, errors.New("ephemeral registry credential session is invalid")
	}
	dir, err := os.MkdirTemp("", "graft-build-credential-")
	if err != nil {
		return "", nil, fmt.Errorf("create isolated Docker credential context: %w", err)
	}
	cleanup := func() error { return os.RemoveAll(dir) }
	// #nosec G302 -- credential directories require owner execute while forbidding group/other access.
	if err := os.Chmod(dir, dockerCredentialConfigDirMode); err != nil {
		_ = cleanup()
		return "", nil, fmt.Errorf("secure isolated Docker credential context: %w", err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}\n"), dockerCredentialConfigFileMode); err != nil {
		_ = cleanup()
		return "", nil, fmt.Errorf("write isolated Docker credential context: %w", err)
	}
	return dir, cleanup, nil
}
