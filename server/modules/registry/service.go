package registry

import (
	"context"
	"errors"
	"strings"

	"graft/server/internal/moduleapi"
	registrystore "graft/server/modules/registry/store"
)

const ociRegistryDestinationKind = "oci_registry"

// Service 解析已授权的、提供方中立的产物目的地；提供方端点和凭据使用刻意不进入提交边界。
type Service struct {
	repository registrystore.DestinationRepository
	users      moduleapi.UserCandidateReader
	execution  moduleapi.RuntimeOCIRegistryVerifier
	targets    moduleapi.RuntimeTargetBuildAssignmentReader
}

// NewService 创建 Registry 面向跨模块调用的有界服务。
func NewService(repository registrystore.DestinationRepository) *Service {
	return &Service{repository: repository}
}

// bindUserCandidateReader 在依赖模块的 Register 阶段完成后注入候选用户查询能力。
func (s *Service) bindUserCandidateReader(users moduleapi.UserCandidateReader) {
	if s != nil {
		s.users = users
	}
}

// bindRuntimeExecutionAdapter 注入 Runtime Target 拥有的私有 Registry 认证验证执行边界。
func (s *Service) bindRuntimeExecutionAdapter(adapter moduleapi.RuntimeOCIRegistryVerifier) {
	if s != nil {
		s.execution = adapter
	}
}

// bindRuntimeTargetBuildAssignments 注入 Runtime Target 拥有的目标执行授权复核边界。
func (s *Service) bindRuntimeTargetBuildAssignments(targets moduleapi.RuntimeTargetBuildAssignmentReader) {
	if s != nil {
		s.targets = targets
	}
}

// ResolveArtifactDestination 校验调用方的 OCI 发布目的地，并返回可冻结的非秘密引用。
//
//nolint:cyclop // Provider, reference and actor checks are deliberately kept at this trust boundary.
func (s *Service) ResolveArtifactDestination(ctx context.Context, actorID uint64, destination moduleapi.BuildDestination) (moduleapi.AuthorizedArtifactDestination, error) {
	if s == nil || s.repository == nil || actorID == 0 {
		return moduleapi.AuthorizedArtifactDestination{}, errors.New("registry destination resolver is unavailable")
	}
	destination = normalizeDestination(destination)
	if destination.Kind != ociRegistryDestinationKind || destination.ConnectionRef == "" || destination.RepositoryRef == "" || destination.Reference == "" {
		return moduleapi.AuthorizedArtifactDestination{}, errors.New("invalid artifact destination")
	}
	if strings.ContainsAny(destination.ConnectionRef+destination.RepositoryRef+destination.Reference, "\x00\r\n") {
		return moduleapi.AuthorizedArtifactDestination{}, errors.New("invalid artifact destination")
	}
	repository, err := s.repository.ResolveAuthorizedRepository(ctx, actorID, destination.ConnectionRef, destination.RepositoryRef)
	if err != nil {
		return moduleapi.AuthorizedArtifactDestination{}, err
	}
	if !repository.ConnectionAvailable || !repository.AllowPush {
		return moduleapi.AuthorizedArtifactDestination{}, errors.New("artifact destination is unavailable for publication")
	}
	return moduleapi.AuthorizedArtifactDestination{Kind: ociRegistryDestinationKind, ConnectionRef: repository.ConnectionRef, RepositoryRef: repository.RepositoryRef, Reference: destination.Reference}, nil
}

// ResolvePublicationBinding resolves provider details only for the execution adapter.
func (s *Service) ResolvePublicationBinding(ctx context.Context, destination moduleapi.AuthorizedArtifactDestination) (moduleapi.RegistryPublicationBinding, error) {
	if s == nil || s.repository == nil || destination.Kind != ociRegistryDestinationKind {
		return moduleapi.RegistryPublicationBinding{}, errors.New("registry publication resolver is unavailable")
	}
	repository, err := s.repository.ResolveRepositoryBinding(ctx, destination.ConnectionRef, destination.RepositoryRef)
	if err != nil {
		return moduleapi.RegistryPublicationBinding{}, err
	}
	if !repository.ConnectionAvailable || !repository.AllowPush {
		return moduleapi.RegistryPublicationBinding{}, errors.New("registry publication binding is unavailable")
	}
	return moduleapi.RegistryPublicationBinding{
		Destination:   destination,
		Endpoint:      repository.Endpoint,
		CredentialRef: repository.CredentialRef,
		AuthExecution: moduleapi.RegistryAuthExecution{
			Mode: moduleapi.RegistryAuthExecutionEphemeral,
		},
	}, nil
}

// AuthorizeArtifactCopy 校验 Build-owned source identity，并只在操作者拥有 Registry-owned
// pull 和 push 权限时授权新目的地；返回值不含 endpoint 或 credential，可安全冻结。
func (s *Service) AuthorizeArtifactCopy(ctx context.Context, actorID uint64, source moduleapi.ArtifactPublicationSource, destination moduleapi.BuildDestination) (moduleapi.AuthorizedArtifactCopy, error) {
	if s == nil || s.repository == nil || actorID == 0 || !validArtifactCopySource(source) {
		return moduleapi.AuthorizedArtifactCopy{}, errors.New("registry artifact copy authorization is unavailable")
	}
	sourceRepository, err := s.repository.ResolveAuthorizedCopySource(ctx, actorID, source.ConnectionRef, source.RepositoryRef)
	if err != nil {
		return moduleapi.AuthorizedArtifactCopy{}, err
	}
	if !sourceRepository.ConnectionAvailable || !sourceRepository.AllowPull {
		return moduleapi.AuthorizedArtifactCopy{}, errors.New("artifact copy source is unavailable")
	}
	resolvedDestination, err := s.ResolveArtifactDestination(ctx, actorID, destination)
	if err != nil {
		return moduleapi.AuthorizedArtifactCopy{}, err
	}
	return moduleapi.AuthorizedArtifactCopy{Source: source, Destination: resolvedDestination}, nil
}

// ResolveArtifactCopyBinding 只向执行 digest-preserving copy 的 Runtime Target provider
// 返回两个 Registry-owned 私有 binding。
func (s *Service) ResolveArtifactCopyBinding(ctx context.Context, copy moduleapi.AuthorizedArtifactCopy) (moduleapi.RegistryArtifactCopyBinding, error) {
	if s == nil || s.repository == nil || !validArtifactCopySource(copy.Source) || copy.Destination.Kind != ociRegistryDestinationKind {
		return moduleapi.RegistryArtifactCopyBinding{}, errors.New("registry artifact copy resolver is unavailable")
	}
	source, err := s.repository.ResolveRepositoryBinding(ctx, copy.Source.ConnectionRef, copy.Source.RepositoryRef)
	if err != nil {
		return moduleapi.RegistryArtifactCopyBinding{}, err
	}
	if !source.ConnectionAvailable || !source.AllowPull {
		return moduleapi.RegistryArtifactCopyBinding{}, errors.New("registry artifact copy source binding is unavailable")
	}
	destination, err := s.ResolvePublicationBinding(ctx, copy.Destination)
	if err != nil {
		return moduleapi.RegistryArtifactCopyBinding{}, err
	}
	return moduleapi.RegistryArtifactCopyBinding{
		SourceEndpoint:      source.Endpoint,
		SourceCredentialRef: source.CredentialRef,
		SourceAuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral},
		Destination:         destination,
	}, nil
}

func validArtifactCopySource(source moduleapi.ArtifactPublicationSource) bool {
	if source.DestinationKind != ociRegistryDestinationKind || strings.TrimSpace(source.ArtifactID) == "" || strings.TrimSpace(source.PublicationID) == "" || strings.TrimSpace(source.ConnectionRef) == "" || strings.TrimSpace(source.RepositoryRef) == "" {
		return false
	}
	digest := strings.TrimSpace(source.Digest)
	return strings.HasPrefix(digest, "sha256:") && len(strings.TrimPrefix(digest, "sha256:")) == 64 && !strings.ContainsAny(source.ArtifactID+source.PublicationID+source.ConnectionRef+source.RepositoryRef+digest, "\x00\r\n")
}

func normalizeDestination(destination moduleapi.BuildDestination) moduleapi.BuildDestination {
	destination.Kind = strings.TrimSpace(destination.Kind)
	destination.ConnectionRef = strings.TrimSpace(destination.ConnectionRef)
	destination.RepositoryRef = strings.TrimSpace(destination.RepositoryRef)
	destination.Reference = strings.TrimSpace(destination.Reference)
	return destination
}
