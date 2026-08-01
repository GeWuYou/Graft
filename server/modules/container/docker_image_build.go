package container

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
)

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
	output, err := command.CombinedOutput()
	if sink != nil {
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			if sinkErr := sink(ctx, moduleapi.TaskLogEntry{Stream: "stdout", Level: "info", Line: scanner.Text()}); sinkErr != nil {
				return moduleapi.DockerImageBuildResult{}, sinkErr
			}
		}
	}
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("docker build: %w", err)
	}
	imageID, err := os.ReadFile(iidPath) // #nosec G304 -- 文件由同一调用创建，且从受控 workspace 中删除。
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, fmt.Errorf("read Docker build image identity: %w", err)
	}
	return moduleapi.DockerImageBuildResult{ImageID: strings.TrimSpace(string(imageID)), Repository: input.ImageRepository, Tag: input.ImageTag}, nil
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
