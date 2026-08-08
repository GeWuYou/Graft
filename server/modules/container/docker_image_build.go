package container

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
	"slices"
	"strings"
	"sync"

	"graft/server/internal/moduleapi"
)

const maxDockerBuildLogBuffer = 64 * 1024

type containerImageBuilder struct{ service *service }

// DeliverWorkspaceSnapshot 验证 Local Docker provider 只能消费 Build-owned、target-local Snapshot。
// provider-transfer 预留给真正拥有远程传输实现的 provider，当前必须 fail-closed。
func (b containerImageBuilder) DeliverWorkspaceSnapshot(ctx context.Context, request moduleapi.WorkspaceSnapshotDeliveryRequest) (moduleapi.WorkspaceSnapshotDeliveryResult, error) {
	if !validSnapshotDeliveryRequest(b.service, request) {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, errors.New("workspace snapshot delivery input is invalid")
	}
	if request.DeliveryMode != moduleapi.SnapshotDeliveryModeTargetLocal {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, errors.New("workspace snapshot delivery mode is unsupported by Docker provider")
	}
	if b.service.buildTargets == nil {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, errors.New("build runtime target capability reader is unavailable")
	}
	target, err := b.service.buildTargets.ReadBuildTarget(ctx, request.TargetID)
	if err != nil {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, fmt.Errorf("read snapshot delivery runtime target: %w", err)
	}
	if !supportsSnapshotDeliveryTarget(target, request.TargetID) {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, errors.New("snapshot delivery runtime target is unsupported")
	}
	if err := validateManagedSnapshotReference(request); err != nil {
		return moduleapi.WorkspaceSnapshotDeliveryResult{}, err
	}
	return moduleapi.WorkspaceSnapshotDeliveryResult{TargetID: request.TargetID, SnapshotID: request.SnapshotID, ContentDigest: request.ContentDigest, DeliveryMode: request.DeliveryMode}, nil
}

func validSnapshotDeliveryRequest(service *service, request moduleapi.WorkspaceSnapshotDeliveryRequest) bool {
	return service != nil && request.TargetID > 0 && strings.TrimSpace(request.SnapshotID) != "" && strings.TrimSpace(request.ContentDigest) != "" && strings.TrimSpace(request.MaterializationRef) != ""
}

func validateManagedSnapshotReference(request moduleapi.WorkspaceSnapshotDeliveryRequest) error {
	name, snapshotID, contentDigest, err := moduleapi.ParseWorkspaceSnapshotMaterializationReference(request.MaterializationRef)
	if err != nil {
		return err
	}
	if snapshotID != request.SnapshotID || contentDigest != request.ContentDigest {
		return errors.New("workspace snapshot materialization reference does not match snapshot")
	}
	return validateManagedSnapshotRoot(filepath.Join(os.TempDir(), "graft-build-snapshots", name))
}

func supportsSnapshotDeliveryTarget(target moduleapi.BuildRuntimeTargetSummary, targetID int64) bool {
	return target.ID == targetID && target.Provider == runtimeNameDocker && target.Available && slices.Contains(target.SnapshotDeliveryModes, moduleapi.SnapshotDeliveryModeTargetLocal) && slices.Contains(target.WorkspaceLocalities, "build-snapshot")
}

func validateManagedSnapshotRoot(path string) error {
	managedRoot := filepath.Join(os.TempDir(), "graft-build-snapshots")
	root, err := filepath.Abs(path)
	if err != nil {
		return errors.New("workspace snapshot materialization path is invalid")
	}
	relative, err := filepath.Rel(managedRoot, root)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("workspace snapshot materialization is outside the managed Build root")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return errors.New("workspace snapshot materialization is unavailable")
	}
	return nil
}

func (b containerImageBuilder) BuildImage(ctx context.Context, input moduleapi.DockerImageBuildInput, sink moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	if b.service == nil {
		return moduleapi.DockerImageBuildResult{}, errRuntimeDisabled
	}
	return b.service.BuildImage(ctx, input, sink)
}

// BuildImageOnTarget 在进入受控 Docker command path 前重新校验所选 target；
// endpoint 与 credential 仍由 Runtime Target/Container module 私有持有。
func (b containerImageBuilder) BuildImageOnTarget(ctx context.Context, targetID int64, input moduleapi.DockerImageBuildInput, sink moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	if b.service == nil || targetID < 1 {
		return moduleapi.DockerImageBuildResult{}, errors.New("build runtime target is unavailable")
	}
	if b.service.buildTargets == nil {
		return moduleapi.DockerImageBuildResult{}, errors.New("build runtime target capability reader is unavailable")
	}
	target, err := b.service.buildTargets.ReadBuildTarget(ctx, targetID)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("read build runtime target: %w", err)
	}
	if target.ID != targetID || target.Provider != runtimeNameDocker || !target.Available || (!supportsDockerEngineBuild(target.SupportedDrivers) && !supportsDockerBuildx(target.SupportedDrivers)) {
		return moduleapi.DockerImageBuildResult{}, errors.New("build runtime target provider is unsupported")
	}
	return b.BuildImage(ctx, input, sink)
}

// PublishImageOnTarget 通过所选 Docker target tag 并 push 已完成镜像；Registry
// endpoint 与 credential reference 只来自 Infrastructure-owned execution binding，
// 不来自 Build task input。
//
//nolint:cyclop // Publication 在同一 audited boundary 内维护 target revalidation、controlled tag/push 与 digest settlement。
func (b containerImageBuilder) PublishImageOnTarget(ctx context.Context, targetID int64, result moduleapi.DockerImageBuildResult, binding moduleapi.RegistryPublicationBinding, sink moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	if b.service == nil || targetID < 1 || strings.TrimSpace(result.ImageID) == "" {
		return result, errors.New("build publication input is invalid")
	}
	if binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionDockerStore {
		return result, errors.New("registry credential execution mode is unsupported")
	}
	if b.service.buildTargets == nil {
		return result, errors.New("build runtime target capability reader is unavailable")
	}
	target, err := b.service.buildTargets.ReadBuildTarget(ctx, targetID)
	if err != nil {
		return result, fmt.Errorf("read build runtime target: %w", err)
	}
	if target.ID != targetID || target.Provider != runtimeNameDocker || !target.Available || !supportsDockerEngineBuild(target.SupportedDrivers) {
		return result, errors.New("build runtime target provider is unsupported")
	}
	ref, err := publicationReference(binding)
	if err != nil {
		return result, err
	}
	if err := runDockerPublicationCommand(ctx, sink, "tag", result.ImageID, ref); err != nil {
		return result, err
	}
	if err := runDockerPublicationCommand(ctx, sink, "push", ref); err != nil {
		return result, err
	}
	image, err := b.service.DockerImage(ctx, result.ImageID)
	if err != nil {
		return result, fmt.Errorf("inspect published image: %w", err)
	}
	repository := strings.TrimSuffix(ref, ":"+binding.Destination.Reference)
	result.Digest = dockerImageBuildDigest(image.RepositoryDigests, repository)
	result.Repository = repository
	result.Tag = binding.Destination.Reference
	return result, nil
}

// PublishOCIManifestOnTarget 由明确声明 docker-buildx 的 Runtime Target 合并已发布的平台 digest，并读取
// registry 返回的最终 Manifest 摘要。Build 不参与 Docker CLI 参数组装。
//
//nolint:gocyclo,gocognit,cyclop // 外部发布边界必须在同一调用内完成 Target、来源、Manifest 与摘要的防御性校验。
func (b containerImageBuilder) PublishOCIManifestOnTarget(ctx context.Context, targetID int64, input moduleapi.OCIManifestPublicationInput, binding moduleapi.RegistryPublicationBinding, sink moduleapi.DockerImageBuildLogSink) (moduleapi.OCIManifestPublicationResult, error) {
	if b.service == nil || targetID < 1 || len(input.PlatformArtifacts) < 2 || binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionDockerStore || b.service.buildTargets == nil {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("OCI manifest publication input is invalid")
	}
	target, err := b.service.buildTargets.ReadBuildTarget(ctx, targetID)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, fmt.Errorf("read manifest build runtime target: %w", err)
	}
	if target.ID != targetID || target.Provider != runtimeNameDocker || !target.Available || !supportsDockerBuildx(target.SupportedDrivers) {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("manifest build runtime target provider is unsupported")
	}
	ref, err := publicationReference(binding)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	sources, err := manifestSourceReferences(ref, input.PlatformArtifacts)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	manifestArgs := append([]string{"buildx", "imagetools", "create", "--tag", ref}, sources...)
	if err := runDockerPublicationCommand(ctx, sink, manifestArgs[0], manifestArgs[1:]...); err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	raw, err := runDockerCommandOutput(ctx, "buildx", "imagetools", "inspect", "--raw", ref)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	var descriptor struct {
		MediaType string `json:"mediaType"`
	}
	if err := json.Unmarshal(raw, &descriptor); err != nil || strings.TrimSpace(descriptor.MediaType) == "" {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("published OCI manifest media type is unavailable")
	}
	digestOutput, err := runDockerCommandOutput(ctx, "buildx", "imagetools", "inspect", "--format", "{{.Manifest.Digest}}", ref)
	if err != nil {
		return moduleapi.OCIManifestPublicationResult{}, err
	}
	digest := strings.TrimSpace(string(digestOutput))
	if !isManifestDigest(digest) {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("published OCI manifest digest is invalid")
	}
	return moduleapi.OCIManifestPublicationResult{Digest: digest, MediaType: descriptor.MediaType, SizeBytes: int64(len(raw))}, nil
}

func publicationReference(binding moduleapi.RegistryPublicationBinding) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(binding.Endpoint))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("registry publication endpoint is invalid")
	}
	repository := strings.Trim(strings.TrimSpace(binding.Destination.RepositoryRef), "/")
	reference := strings.TrimSpace(binding.Destination.Reference)
	if repository == "" || reference == "" || strings.ContainsAny(repository+reference, "\x00\r\n") {
		return "", errors.New("registry publication reference is invalid")
	}
	pathPrefix := strings.Trim(parsed.Path, "/")
	base := parsed.Host
	if pathPrefix != "" {
		base += "/" + pathPrefix
	}
	if base == "" {
		return "", errors.New("registry publication endpoint is invalid")
	}
	return base + "/" + repository + ":" + reference, nil
}

func runDockerPublicationCommand(ctx context.Context, sink moduleapi.DockerImageBuildLogSink, action string, args ...string) error {
	commandArgs := append([]string{action}, args...)
	command := exec.CommandContext(ctx, "docker", commandArgs...) // #nosec G204 -- action and refs are validated by the provider-owned binding path.
	logs := newDockerBuildLogSink(ctx, sink)
	command.Stdout = logs.writer("stdout")
	command.Stderr = logs.writer("stderr")
	if err := command.Run(); err != nil {
		_ = logs.flush()
		return fmt.Errorf("docker %s: %w", action, err)
	}
	if err := logs.flush(); err != nil {
		return err
	}
	if err := logs.error(); err != nil {
		return err
	}
	return nil
}

func runDockerCommandOutput(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "docker", args...) // #nosec G204 -- 所有 Buildx 子命令与参数均由 Container provider 组装并校验。
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("docker %s: %w", strings.Join(args[:3], " "), err)
	}
	return output, nil
}

func supportsDockerEngineBuild(drivers []string) bool {
	for _, driver := range drivers {
		if driver == "docker-engine" {
			return true
		}
	}
	return false
}

func supportsDockerBuildx(drivers []string) bool {
	for _, driver := range drivers {
		if driver == "docker-buildx" {
			return true
		}
	}
	return false
}

func manifestSourceReferences(publicationRef string, artifacts []moduleapi.PlatformArtifact) ([]string, error) {
	tagSeparator := strings.LastIndex(publicationRef, ":")
	pathSeparator := strings.LastIndex(publicationRef, "/")
	if tagSeparator <= pathSeparator {
		return nil, errors.New("OCI manifest publication reference is invalid")
	}
	base := publicationRef[:tagSeparator]
	seenPlatforms := make(map[string]struct{}, len(artifacts))
	sources := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Platform == "" || !isManifestDigest(artifact.Digest) {
			return nil, errors.New("OCI manifest platform artifact is invalid")
		}
		if _, exists := seenPlatforms[artifact.Platform]; exists {
			return nil, errors.New("OCI manifest contains duplicate platform")
		}
		seenPlatforms[artifact.Platform] = struct{}{}
		sources = append(sources, base+"@"+artifact.Digest)
	}
	return sources, nil
}

func isManifestDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	for _, character := range value[len("sha256:"):] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

// BuildImage 在已授权应用工作区内执行受限的本地 Dockerfile 构建，不接受外部 daemon 或自由 CLI 参数。
//
//nolint:cyclop // Docker 进程、日志和 iid 文件的生命周期需要在同一受控执行边界内收束。
func (s *service) BuildImage(ctx context.Context, input moduleapi.DockerImageBuildInput, sink moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	if s == nil || !s.enabled {
		return moduleapi.DockerImageBuildResult{}, errRuntimeDisabled
	}
	root, contextPath, dockerfilePath, err := normalizeDockerBuildInput(input)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	iidFile, err := os.CreateTemp(root, ".graft-build-iid-*")
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("create Docker build iid file: %w", err)
	}
	iidPath := iidFile.Name()
	if err := iidFile.Close(); err != nil {
		_ = os.Remove(iidPath)
		return moduleapi.DockerImageBuildResult{}, err
	}
	defer func() { _ = os.Remove(iidPath) }()
	args := []string{"build", "--progress=plain", "--iidfile", iidPath, "--file", dockerfilePath, "--tag", input.ImageRepository + ":" + input.ImageTag}
	if strings.TrimSpace(input.Platform) != "" {
		args = append(args, "--platform", input.Platform)
	}
	for _, arg := range input.BuildArgs {
		args = append(args, "--build-arg", arg.Name+"="+arg.Value)
	}
	args = append(args, contextPath)
	command := exec.CommandContext(ctx, "docker", args...) // #nosec G204 -- Build validates all paths and Build owns the fixed Docker argument grammar.
	command.Dir = root
	logs := newDockerBuildLogSink(ctx, sink)
	command.Stdout = logs.writer("stdout")
	command.Stderr = logs.writer("stderr")
	err = command.Run()
	if flushErr := logs.flush(); flushErr != nil {
		return moduleapi.DockerImageBuildResult{}, flushErr
	}
	if sinkErr := logs.error(); sinkErr != nil {
		return moduleapi.DockerImageBuildResult{}, sinkErr
	}
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("docker build: %w", err)
	}
	imageID, err := os.ReadFile(iidPath) // #nosec G304 -- 文件由同一调用创建，且从受控 workspace 中删除。
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("read Docker build image identity: %w", err)
	}
	builtImageID := strings.TrimSpace(string(imageID))
	image, err := s.DockerImage(ctx, builtImageID)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("inspect Docker build image: %w", err)
	}
	return moduleapi.DockerImageBuildResult{
		ImageID:      builtImageID,
		Digest:       dockerImageBuildDigest(image.RepositoryDigests, input.ImageRepository),
		Repository:   input.ImageRepository,
		Tag:          input.ImageTag,
		SizeBytes:    image.SizeBytes,
		OS:           image.OperatingSystem,
		Architecture: image.Architecture,
		Variant:      image.Variant,
	}, nil
}

type dockerBuildLogSink struct {
	ctx     context.Context
	sink    moduleapi.DockerImageBuildLogSink
	mu      sync.Mutex
	writers map[string]*dockerBuildLogWriter
	sinkErr error
}

type dockerBuildLogWriter struct {
	owner  *dockerBuildLogSink
	stream string
	buffer strings.Builder
}

func newDockerBuildLogSink(ctx context.Context, sink moduleapi.DockerImageBuildLogSink) *dockerBuildLogSink {
	return &dockerBuildLogSink{ctx: ctx, sink: sink, writers: make(map[string]*dockerBuildLogWriter)}
}

func (s *dockerBuildLogSink) writer(stream string) io.Writer {
	s.mu.Lock()
	defer s.mu.Unlock()
	writer := &dockerBuildLogWriter{owner: s, stream: stream}
	s.writers[stream] = writer
	return writer
}

func (s *dockerBuildLogSink) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, writer := range s.writers {
		if err := writer.flushLocked(); err != nil {
			return err
		}
	}
	return nil
}

func (s *dockerBuildLogSink) error() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sinkErr
}

func (w *dockerBuildLogWriter) Write(chunk []byte) (int, error) {
	w.owner.mu.Lock()
	defer w.owner.mu.Unlock()
	if w.owner.sinkErr != nil {
		return 0, w.owner.sinkErr
	}
	if w.owner.sink == nil {
		return len(chunk), nil
	}
	written := len(chunk)
	for len(chunk) > 0 {
		line, rest, found := bytes.Cut(chunk, []byte{'\n'})
		if _, err := w.buffer.Write(line); err != nil {
			return 0, err
		}
		if w.buffer.Len() > maxDockerBuildLogBuffer {
			return 0, errors.New("docker build log line exceeds limit")
		}
		if !found {
			break
		}
		if err := w.flushLocked(); err != nil {
			return 0, err
		}
		chunk = rest
	}
	return written, nil
}

func (w *dockerBuildLogWriter) flushLocked() error {
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

func dockerImageBuildDigest(digests []string, repository string) string {
	prefix := strings.TrimSpace(repository) + "@"
	for _, digest := range digests {
		if strings.HasPrefix(strings.TrimSpace(digest), prefix) {
			return strings.TrimSpace(digest)
		}
	}
	if len(digests) == 0 {
		return ""
	}
	return strings.TrimSpace(digests[0])
}

//nolint:revive // 三个规范化路径/身份结果保持调用方不会误混用它们。
func normalizeDockerBuildInput(input moduleapi.DockerImageBuildInput) (string, string, string, error) {
	root := filepath.Clean(strings.TrimSpace(input.WorkspaceRoot))
	if root == "." || !filepath.IsAbs(root) || strings.TrimSpace(input.ImageRepository) == "" || strings.TrimSpace(input.ImageTag) == "" {
		return "", "", "", errors.New("invalid Docker build input")
	}
	contextPath, err := safeDockerBuildRelativePath(input.ContextPath)
	if err != nil {
		return "", "", "", err
	}
	dockerfilePath, err := safeDockerBuildRelativePath(input.DockerfilePath)
	if err != nil {
		return "", "", "", err
	}
	for _, arg := range input.BuildArgs {
		if strings.TrimSpace(arg.Name) == "" || strings.ContainsAny(arg.Name, "=\x00\r\n") {
			return "", "", "", errors.New("invalid Docker build argument")
		}
	}
	return root, contextPath, dockerfilePath, nil
}

func safeDockerBuildRelativePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("invalid Docker build path")
	}
	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("docker build path escapes workspace")
	}
	return clean, nil
}
