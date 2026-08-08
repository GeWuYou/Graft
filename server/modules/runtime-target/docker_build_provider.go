package runtimetarget

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

const providerBuildLogBuffer = 64 * 1024

// dockerTargetProvider 是 Runtime Target 所有的 Docker provider boundary。
// 它从私有仓储读取连接事实，并把受控请求编译为 Docker CLI 调用；Build 不接触 endpoint 或 credential。
type dockerTargetProvider struct {
	repository *store.SQLRepository
}

var (
	providerCommandRunner = runProviderCommand
	providerOutputRunner  = runProviderOutput
)

type dockerCredentialConfigContextKey struct{}

// ProviderID 返回 Runtime Target provider 的稳定内部标识，不包含连接信息。
func (p dockerTargetProvider) ProviderID() string { return "docker-target" }

func (p dockerTargetProvider) connection(ctx context.Context, targetID int64) (store.DockerTargetConnection, error) {
	if p.repository == nil || targetID < 1 {
		return store.DockerTargetConnection{}, errors.New("runtime target provider is unavailable")
	}
	return p.repository.GetDockerTargetConnection(ctx, uint64(targetID)) //nolint:gosec // targetID is positive and bounded by the public contract.
}

// ConformProviderExecution 在任何 Snapshot 交付前重新验证 Docker provider 的可执行边界。
// 证明只描述 capability 和生命周期支持，不返回连接、凭据或物化路径事实。
func (p dockerTargetProvider) ConformProviderExecution(ctx context.Context, request moduleapi.ProviderExecutionConformanceRequest) (moduleapi.ProviderExecutionConformanceResult, error) {
	if !validProviderConformanceRequest(request) {
		return moduleapi.ProviderExecutionConformanceResult{}, errors.New("provider conformance input is invalid")
	}
	if !isDockerBuildDriver(request.DriverRef) {
		return moduleapi.ProviderExecutionConformanceResult{}, errors.New("docker provider does not support the selected driver")
	}
	connection, err := p.connection(ctx, request.TargetID)
	if err != nil {
		return moduleapi.ProviderExecutionConformanceResult{}, err
	}
	wantMode := dockerSnapshotDeliveryMode(connection.ConnectionKind)
	if request.DeliveryMode != wantMode {
		return moduleapi.ProviderExecutionConformanceResult{}, errors.New("provider conformance delivery mode does not match target")
	}
	return moduleapi.ProviderExecutionConformanceResult{
		ProviderID:            p.ProviderID(),
		ConformanceVersion:    "v1",
		Executable:            true,
		SnapshotDeliveryProof: true,
		DriverExecutionProof:  true,
		PublicationProof:      true,
		CancellationProof:     true,
		CleanupProof:          true,
	}, nil
}

func validProviderConformanceRequest(request moduleapi.ProviderExecutionConformanceRequest) bool {
	return request.TargetID > 0 && strings.TrimSpace(request.DriverRef) != "" && strings.TrimSpace(request.Platform) != "" && strings.TrimSpace(request.SnapshotID) != "" && strings.TrimSpace(request.ContentDigest) != "" && strings.TrimSpace(request.DeliveryMode) != ""
}

func isDockerBuildDriver(driverRef string) bool {
	return driverRef == "docker-engine" || driverRef == "docker-buildx"
}

func dockerSnapshotDeliveryMode(connectionKind string) string {
	if connectionKind == "unix_socket" {
		return moduleapi.SnapshotDeliveryModeTargetLocal
	}
	return moduleapi.SnapshotDeliveryModeProviderTransfer
}

// DeliverWorkspaceSnapshot 校验快照物化根和目标连接类型，并返回不携带连接细节的交付证明。
func (p dockerTargetProvider) DeliverWorkspaceSnapshot(ctx context.Context, request moduleapi.WorkspaceSnapshotDeliveryRequest) (moduleapi.WorkspaceSnapshotDeliveryResult, error) {
	if request.TargetID < 1 || strings.TrimSpace(request.SnapshotID) == "" || strings.TrimSpace(request.ContentDigest) == "" {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, errors.New("workspace snapshot delivery input is invalid")
	}
	root, err := managedSnapshotRootForReference(request.MaterializationRef)
	if err != nil {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, err
	}
	connection, err := p.connection(ctx, request.TargetID)
	if err != nil {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, err
	}
	wantMode := moduleapi.SnapshotDeliveryModeTargetLocal
	if connection.ConnectionKind != "unix_socket" {
		wantMode = moduleapi.SnapshotDeliveryModeProviderTransfer
	}
	if request.DeliveryMode != wantMode {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, errors.New("workspace snapshot delivery mode does not match target provider")
	}
	if wantMode == moduleapi.SnapshotDeliveryModeTargetLocal && root == "" {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, errors.New("workspace snapshot materialization is unavailable")
	}
	return moduleapi.WorkspaceSnapshotDeliveryResult{TargetID: request.TargetID, SnapshotID: request.SnapshotID, ContentDigest: request.ContentDigest, DeliveryMode: request.DeliveryMode}, nil
}

// BuildImageOnTarget 使用 Runtime Target 私有连接在选定 Docker 目标上执行受控构建。
//
//nolint:cyclop,gocyclo // Provider boundary keeps target validation, command construction and immutable result inspection together.
func (p dockerTargetProvider) BuildImageOnTarget(ctx context.Context, targetID int64, input moduleapi.DockerImageBuildInput, sink moduleapi.DockerImageBuildLogSink) (result moduleapi.DockerImageBuildResult, err error) {
	connection, err := p.connection(ctx, targetID)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	agentID, err := p.trackDockerBuilderExecution(ctx, targetID)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	defer func() {
		if finishErr := p.finishDockerBuilderExecution(context.WithoutCancel(ctx), targetID, agentID); finishErr != nil && err == nil {
			err = fmt.Errorf("finish Docker builder execution: %w", finishErr)
		}
	}()
	paths, err := normalizeProviderBuildInput(input)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	iidFile, err := os.CreateTemp(paths.root, ".graft-build-iid-*")
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("create Docker build iid file: %w", err)
	}
	iidPath := iidFile.Name()
	if err := iidFile.Close(); err != nil {
		_ = os.Remove(iidPath)
		return moduleapi.DockerImageBuildResult{}, err
	}
	defer func() { _ = os.Remove(iidPath) }()
	args := []string{"--host", connection.Endpoint, "build", "--progress=plain", "--iidfile", iidPath, "--file", paths.dockerfilePath, "--tag", input.ImageRepository + ":" + input.ImageTag}
	if strings.TrimSpace(input.Platform) != "" {
		args = append(args, "--platform", input.Platform)
	}
	for _, buildArg := range input.BuildArgs {
		if strings.TrimSpace(buildArg.Name) == "" || strings.ContainsAny(buildArg.Name+buildArg.Value, "\x00\r\n") {
			return moduleapi.DockerImageBuildResult{}, errors.New("docker build argument is invalid")
		}
		args = append(args, "--build-arg", buildArg.Name+"="+buildArg.Value)
	}
	args = append(args, paths.contextPath)
	if err := runProviderCommand(ctx, sink, args...); err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	imageIDBytes, err := os.ReadFile(iidPath) // #nosec G304 -- the file was created in the managed workspace by this call.
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("read Docker build image identity: %w", err)
	}
	imageID := strings.TrimSpace(string(imageIDBytes))
	if imageID == "" {
		return moduleapi.DockerImageBuildResult{}, errors.New("docker build image identity is unavailable")
	}
	inspect, err := runProviderOutput(ctx, "--host", connection.Endpoint, "image", "inspect", "--format", "{{json .}}", imageID)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	var image struct {
		Size         int64    `json:"Size"`
		Os           string   `json:"Os"`
		Architecture string   `json:"Architecture"`
		Variant      string   `json:"Variant"`
		RepoDigests  []string `json:"RepoDigests"`
	}
	if err := json.Unmarshal(inspect, &image); err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("decode Docker image inspection: %w", err)
	}
	return moduleapi.DockerImageBuildResult{ImageID: imageID, Digest: providerImageDigest(image.RepoDigests, input.ImageRepository), Repository: input.ImageRepository, Tag: input.ImageTag, SizeBytes: image.Size, OS: image.Os, Architecture: image.Architecture, Variant: image.Variant}, nil
}

func (p dockerTargetProvider) trackDockerBuilderExecution(ctx context.Context, targetID int64) (string, error) {
	agent, err := p.repository.GetActiveDockerBuilderTelemetryAgent(ctx, targetID)
	if errors.Is(err, store.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve Docker builder agent: %w", err)
	}
	if err := p.repository.QueueBuilderAgentBuild(ctx, targetID, agent.AgentID); err != nil {
		return "", fmt.Errorf("queue Docker builder execution: %w", err)
	}
	if err := p.repository.StartBuilderAgentBuild(ctx, targetID, agent.AgentID); err != nil {
		_ = p.repository.CancelQueuedBuilderAgentBuild(context.WithoutCancel(ctx), targetID, agent.AgentID)
		return "", fmt.Errorf("start Docker builder execution: %w", err)
	}
	return agent.AgentID, nil
}

func (p dockerTargetProvider) finishDockerBuilderExecution(ctx context.Context, targetID int64, agentID string) error {
	if strings.TrimSpace(agentID) == "" {
		return nil
	}
	return p.repository.FinishBuilderAgentBuild(ctx, targetID, agentID)
}

// PublishImageOnTarget 在选定 Docker 目标上发布已构建镜像，并返回目标产生的摘要事实。
//
//nolint:cyclop // 发布边界必须集中复验隔离凭据、目标与摘要证据。
func (p dockerTargetProvider) PublishImageOnTarget(ctx context.Context, targetID int64, result moduleapi.DockerImageBuildResult, binding moduleapi.RegistryPublicationBinding, sink moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	if !hasIsolatedCredentialConfig(ctx) {
		return result, errors.New("isolated Docker credential context is required")
	}
	connection, err := p.connection(ctx, targetID)
	if err != nil {
		return result, err
	}
	if result.ImageID == "" || binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return result, errors.New("docker publication input is invalid")
	}
	ref, repository, err := providerPublicationReference(binding)
	if err != nil {
		return result, err
	}
	if err := runProviderCommand(ctx, sink, "--host", connection.Endpoint, "tag", result.ImageID, ref); err != nil {
		return result, err
	}
	if err := runProviderCommand(ctx, sink, "--host", connection.Endpoint, "push", ref); err != nil {
		return result, err
	}
	digest, err := providerPublishedDigest(ctx, connection.Endpoint, ref, repository)
	if err != nil || digest == "" {
		if err == nil {
			err = errors.New("published Docker image digest is unavailable")
		}
		return result, err
	}
	result.Repository = repository
	result.Tag = binding.Destination.Reference
	result.Digest = digest
	return result, nil
}

// PublishOCIManifestOnTarget 在选定 Docker Buildx provider 上合并并读取多平台 Manifest 摘要。
//
//nolint:cyclop // Provider boundary keeps input, source, publication and digest validation atomic.
func (p dockerTargetProvider) PublishOCIManifestOnTarget(ctx context.Context, targetID int64, input moduleapi.OCIManifestPublicationInput, binding moduleapi.RegistryPublicationBinding, sink moduleapi.DockerImageBuildLogSink) (moduleapi.OCIManifestPublicationResult, error) {
	if !hasIsolatedCredentialConfig(ctx) {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("isolated Docker credential context is required")
	}
	connection, err := p.connection(ctx, targetID)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	if len(input.PlatformArtifacts) < 2 || binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("OCI manifest publication input is invalid")
	}
	ref, _, err := providerPublicationReference(binding)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	sources := make([]string, 0, len(input.PlatformArtifacts))
	for _, artifact := range input.PlatformArtifacts {
		if strings.TrimSpace(artifact.Digest) == "" || !strings.HasPrefix(artifact.Digest, "sha256:") {
			return moduleapi.OCIManifestPublicationResult{}, errors.New("OCI manifest platform digest is invalid")
		}
		sources = append(sources, ref+"@"+artifact.Digest)
	}
	args := append([]string{"--host", connection.Endpoint, "buildx", "imagetools", "create", "--tag", ref}, sources...)
	if err := runProviderCommand(ctx, sink, args...); err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	raw, err := runProviderOutput(ctx, "--host", connection.Endpoint, "buildx", "imagetools", "inspect", "--raw", ref)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	var descriptor struct {
		MediaType string `json:"mediaType"`
	}
	if err := json.Unmarshal(raw, &descriptor); err != nil || descriptor.MediaType == "" {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("published OCI manifest media type is unavailable")
	}
	digestOutput, err := runProviderOutput(ctx, "--host", connection.Endpoint, "buildx", "imagetools", "inspect", "--format", "{{.Manifest.Digest}}", ref)
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(digestOutput)), "sha256:") {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("published OCI manifest digest is invalid")
	}
	return moduleapi.OCIManifestPublicationResult{Digest: strings.TrimSpace(string(digestOutput)), MediaType: descriptor.MediaType, SizeBytes: int64(len(raw))}, nil
}

// CopyOCIArtifactOnTarget copies one immutable OCI artifact through the selected
// Docker target. Registry bindings remain private to this execution boundary.
//
//nolint:cyclop,gocyclo // Provider 必须逐项校验私有 binding、不可变来源与复制后的 Registry 证明。
func (p dockerTargetProvider) CopyOCIArtifactOnTarget(ctx context.Context, targetID int64, input moduleapi.OCIArtifactCopyInput, binding moduleapi.RegistryArtifactCopyBinding, sink moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error) {
	if !hasIsolatedCredentialConfig(ctx) {
		return moduleapi.OCIArtifactCopyResult{}, errors.New("isolated Docker credential context is required")
	}
	if targetID < 1 || !validOCIArtifactCopyInput(input) || !validOCIArtifactCopyBinding(input, binding) {
		return moduleapi.OCIArtifactCopyResult{}, errors.New("OCI artifact copy input is invalid")
	}
	connection, err := p.connection(ctx, targetID)
	if err != nil {
		return moduleapi.OCIArtifactCopyResult{}, err
	}
	sourceRef, err := providerRegistryDigestReference(binding.SourceEndpoint, input.Source.RepositoryRef, input.Source.Digest)
	if err != nil {
		return moduleapi.OCIArtifactCopyResult{}, err
	}
	destinationRef, _, err := providerPublicationReference(binding.Destination)
	if err != nil {
		return moduleapi.OCIArtifactCopyResult{}, err
	}
	args := []string{"--host", connection.Endpoint, "buildx", "imagetools", "create", "--tag", destinationRef, sourceRef}
	if err := providerCommandRunner(ctx, sink, args...); err != nil {
		return moduleapi.OCIArtifactCopyResult{}, err
	}
	raw, err := providerOutputRunner(ctx, "--host", connection.Endpoint, "buildx", "imagetools", "inspect", "--raw", destinationRef)
	if err != nil {
		return moduleapi.OCIArtifactCopyResult{}, err
	}
	var descriptor struct {
		MediaType string `json:"mediaType"`
	}
	if err := json.Unmarshal(raw, &descriptor); err != nil || strings.TrimSpace(descriptor.MediaType) == "" {
		return moduleapi.OCIArtifactCopyResult{}, errors.New("copied OCI artifact media type is unavailable")
	}
	mediaType := strings.TrimSpace(descriptor.MediaType)
	if mediaType != strings.TrimSpace(input.Source.MediaType) {
		return moduleapi.OCIArtifactCopyResult{}, errors.New("copied OCI artifact media type does not match source")
	}
	digestOutput, err := providerOutputRunner(ctx, "--host", connection.Endpoint, "buildx", "imagetools", "inspect", "--format", "{{.Manifest.Digest}}", destinationRef)
	if err != nil {
		return moduleapi.OCIArtifactCopyResult{}, err
	}
	digest := strings.TrimSpace(string(digestOutput))
	if digest != strings.TrimSpace(input.Source.Digest) {
		return moduleapi.OCIArtifactCopyResult{}, errors.New("copied OCI artifact digest does not match source")
	}
	if len(raw) == 0 {
		return moduleapi.OCIArtifactCopyResult{}, errors.New("copied OCI artifact size is unavailable")
	}
	return moduleapi.OCIArtifactCopyResult{Digest: digest, MediaType: mediaType, SizeBytes: int64(len(raw))}, nil
}

//nolint:cyclop // 执行外部复制前必须拒绝每个会改变来源或目的地身份的输入缺口。
func validOCIArtifactCopyInput(input moduleapi.OCIArtifactCopyInput) bool {
	source := input.Source
	destination := input.Destination
	digest := strings.TrimSpace(source.Digest)
	return source.DestinationKind == "oci_registry" && destination.Kind == "oci_registry" &&
		strings.TrimSpace(source.ArtifactID) != "" && strings.TrimSpace(source.PublicationID) != "" &&
		strings.TrimSpace(source.ConnectionRef) != "" && strings.TrimSpace(source.RepositoryRef) != "" &&
		strings.TrimSpace(source.MediaType) != "" &&
		strings.HasPrefix(digest, "sha256:") && len(strings.TrimPrefix(digest, "sha256:")) == 64 &&
		!strings.ContainsAny(source.ArtifactID+source.PublicationID+source.ConnectionRef+source.RepositoryRef+digest, "\x00\r\n") &&
		strings.TrimSpace(destination.ConnectionRef) != "" && strings.TrimSpace(destination.RepositoryRef) != "" && strings.TrimSpace(destination.Reference) != "" &&
		!strings.ContainsAny(destination.ConnectionRef+destination.RepositoryRef+destination.Reference, "\x00\r\n")
}

func validOCIArtifactCopyBinding(input moduleapi.OCIArtifactCopyInput, binding moduleapi.RegistryArtifactCopyBinding) bool {
	destination := binding.Destination
	return strings.TrimSpace(binding.SourceEndpoint) != "" && strings.TrimSpace(binding.SourceCredentialRef) != "" && binding.SourceAuthExecution.Mode == moduleapi.RegistryAuthExecutionEphemeral &&
		strings.TrimSpace(destination.Endpoint) != "" && strings.TrimSpace(destination.CredentialRef) != "" && destination.AuthExecution.Mode == moduleapi.RegistryAuthExecutionEphemeral &&
		destination.Destination == input.Destination
}

func providerRegistryDigestReference(endpoint, repository, digest string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("registry source endpoint is invalid")
	}
	repository = strings.Trim(strings.TrimSpace(repository), "/")
	digest = strings.TrimSpace(digest)
	if repository == "" || !strings.HasPrefix(digest, "sha256:") || len(strings.TrimPrefix(digest, "sha256:")) != 64 || strings.ContainsAny(repository+digest, "\x00\r\n") {
		return "", errors.New("registry source reference is invalid")
	}
	base := parsed.Host
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		base += "/" + path
	}
	return base + "/" + repository + "@" + digest, nil
}

func managedSnapshotRoot(path string) (string, error) {
	root, err := filepath.Abs(filepath.Clean(strings.TrimSpace(path)))
	if err != nil || root == "." || !filepath.IsAbs(root) {
		return "", errors.New("workspace snapshot materialization path is invalid")
	}
	managed := filepath.Join(os.TempDir(), "graft-build-snapshots")
	relative, err := filepath.Rel(managed, root)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", errors.New("workspace snapshot materialization is outside the managed Build root")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", errors.New("workspace snapshot materialization is unavailable")
	}
	return root, nil
}

type providerBuildPaths struct {
	root           string
	contextPath    string
	dockerfilePath string
}

func normalizeProviderBuildInput(input moduleapi.DockerImageBuildInput) (providerBuildPaths, error) {
	root, err := managedSnapshotRootForInput(input)
	if err != nil {
		return providerBuildPaths{}, err
	}
	contextPath, err := safeProviderRelativePath(input.ContextPath, true)
	if err != nil {
		return providerBuildPaths{}, err
	}
	dockerfilePath, err := safeProviderRelativePath(input.DockerfilePath, false)
	if err != nil {
		return providerBuildPaths{}, err
	}
	if strings.TrimSpace(input.ImageRepository) == "" || strings.TrimSpace(input.ImageTag) == "" || strings.ContainsAny(input.ImageRepository+input.ImageTag, "\x00\r\n") {
		return providerBuildPaths{}, errors.New("docker build image reference is invalid")
	}
	return providerBuildPaths{root: root, contextPath: filepath.Join(root, contextPath), dockerfilePath: filepath.Join(root, dockerfilePath)}, nil
}

func managedSnapshotRootForInput(input moduleapi.DockerImageBuildInput) (string, error) {
	if strings.TrimSpace(input.MaterializationRef) != "" {
		return managedSnapshotRootForReference(input.MaterializationRef)
	}
	return managedSnapshotRoot(input.WorkspaceRoot)
}

func managedSnapshotRootForReference(reference string) (string, error) {
	const prefix = "build-snapshot:"
	name := strings.TrimPrefix(strings.TrimSpace(reference), prefix)
	if name == "" || name == reference || name != filepath.Base(name) || !strings.HasPrefix(name, "snapshot-") {
		return "", errors.New("workspace snapshot materialization reference is invalid")
	}
	return managedSnapshotRoot(filepath.Join(os.TempDir(), "graft-build-snapshots", name))
}

func safeProviderRelativePath(value string, allowDot bool) (string, error) {
	value = filepath.Clean(strings.TrimSpace(value))
	if value == "" || (!allowDot && value == ".") || filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, ".."+string(os.PathSeparator)) || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("docker build path is invalid")
	}
	return value, nil
}

func providerPublicationReference(binding moduleapi.RegistryPublicationBinding) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(binding.Endpoint))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", "", errors.New("registry publication endpoint is invalid")
	}
	repository := strings.Trim(strings.TrimSpace(binding.Destination.RepositoryRef), "/")
	reference := strings.TrimSpace(binding.Destination.Reference)
	if repository == "" || reference == "" || strings.ContainsAny(repository+reference, "\x00\r\n") {
		return "", "", errors.New("registry publication reference is invalid")
	}
	base := parsed.Host
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		base += "/" + path
	}
	return base + "/" + repository + ":" + reference, repository, nil
}

func providerPublishedDigest(ctx context.Context, endpoint, ref, repository string) (string, error) {
	raw, err := runProviderOutput(ctx, "--host", endpoint, "image", "inspect", "--format", "{{json .RepoDigests}}", ref)
	if err != nil {
		return "", err
	}
	var digests []string
	if err := json.Unmarshal(raw, &digests); err != nil {
		return "", err
	}
	return providerImageDigest(digests, repository), nil
}

func providerImageDigest(digests []string, repository string) string {
	prefix := strings.TrimSpace(repository) + "@"
	for _, digest := range digests {
		if strings.HasPrefix(strings.TrimSpace(digest), prefix) {
			return strings.TrimSpace(digest)
		}
	}
	if len(digests) > 0 {
		return strings.TrimSpace(digests[0])
	}
	return ""
}

func runProviderOutput(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "docker", args...) // #nosec G204 -- args are assembled only from validated provider facts.
	applyIsolatedDockerEnvironment(ctx, command)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("docker provider command failed: %w", err)
	}
	return output, nil
}

func runProviderCommand(ctx context.Context, sink moduleapi.DockerImageBuildLogSink, args ...string) error {
	command := exec.CommandContext(ctx, "docker", args...) // #nosec G204 -- args are assembled only from validated provider facts.
	applyIsolatedDockerEnvironment(ctx, command)
	logs := newProviderLogSink(ctx, sink)
	command.Stdout = logs.writer("stdout")
	command.Stderr = logs.writer("stderr")
	if err := command.Run(); err != nil {
		_ = logs.flush()
		return fmt.Errorf("docker provider command failed: %w", err)
	}
	if err := logs.flush(); err != nil {
		return err
	}
	return logs.err()
}

// isolatedDockerEnvironment 删除继承的 Docker 认证变量，仅为本次受控操作传入隔离配置目录。
func isolatedDockerEnvironment(configDir string) []string {
	inherited := os.Environ()
	environment := make([]string, 0, len(inherited)+1)
	for _, entry := range inherited {
		key, _, _ := strings.Cut(entry, "=")
		if key == "DOCKER_CONFIG" || key == "DOCKER_AUTH_CONFIG" {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, "DOCKER_CONFIG="+configDir)
}

func applyIsolatedDockerEnvironment(ctx context.Context, command *exec.Cmd) {
	if configDir, ok := ctx.Value(dockerCredentialConfigContextKey{}).(string); ok && configDir != "" {
		command.Env = isolatedDockerEnvironment(configDir)
	}
}

func hasIsolatedCredentialConfig(ctx context.Context) bool {
	configDir, ok := ctx.Value(dockerCredentialConfigContextKey{}).(string)
	return ok && strings.TrimSpace(configDir) != ""
}

type providerLogSink struct {
	ctx     context.Context
	sink    moduleapi.DockerImageBuildLogSink
	mu      sync.Mutex
	writers map[string]*providerLogWriter
	sinkErr error
}

type providerLogWriter struct {
	owner  *providerLogSink
	stream string
	buffer strings.Builder
}

func newProviderLogSink(ctx context.Context, sink moduleapi.DockerImageBuildLogSink) *providerLogSink {
	return &providerLogSink{ctx: ctx, sink: sink, writers: make(map[string]*providerLogWriter)}
}
func (s *providerLogSink) writer(stream string) io.Writer {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := &providerLogWriter{owner: s, stream: stream}
	s.writers[stream] = w
	return w
}
func (s *providerLogSink) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.writers {
		if err := w.flushLocked(); err != nil {
			return err
		}
	}
	return nil
}
func (s *providerLogSink) err() error { s.mu.Lock(); defer s.mu.Unlock(); return s.sinkErr }

func (w *providerLogWriter) Write(chunk []byte) (int, error) {
	originalLength := len(chunk)
	w.owner.mu.Lock()
	defer w.owner.mu.Unlock()
	if w.owner.sinkErr != nil {
		return 0, w.owner.sinkErr
	}
	if w.owner.sink == nil {
		return len(chunk), nil
	}
	for len(chunk) > 0 {
		line, rest, found := bytes.Cut(chunk, []byte{'\n'})
		_, _ = w.buffer.Write(line)
		if w.buffer.Len() > providerBuildLogBuffer {
			return 0, errors.New("docker provider log line exceeds limit")
		}
		if found {
			if err := w.flushLocked(); err != nil {
				return 0, err
			}
			chunk = rest
		} else {
			break
		}
	}
	return originalLength, nil
}
func (w *providerLogWriter) flushLocked() error {
	if w.buffer.Len() == 0 || w.owner.sink == nil {
		return nil
	}
	line := strings.TrimSuffix(w.buffer.String(), "\r")
	w.buffer.Reset()
	if err := w.owner.sink(w.owner.ctx, moduleapi.TaskLogEntry{Stream: w.stream, Level: "info", Line: line}); err != nil {
		w.owner.sinkErr = err
		return err
	}
	return nil
}
