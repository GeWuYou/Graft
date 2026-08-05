package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"graft/server/internal/moduleapi"
)

type containerImageBuilder struct{ service *service }

func (b containerImageBuilder) BuildImage(ctx context.Context, input moduleapi.DockerImageBuildInput, sink moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	if b.service == nil {
		return moduleapi.DockerImageBuildResult{}, errRuntimeDisabled
	}
	return b.service.BuildImage(ctx, input, sink)
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
	written := len(chunk)
	for len(chunk) > 0 {
		line, rest, found := bytes.Cut(chunk, []byte{'\n'})
		w.buffer.Write(line)
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
