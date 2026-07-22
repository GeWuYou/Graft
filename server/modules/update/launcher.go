package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
)

const (
	runnerInputDirectory     = ".graft-update/inputs"
	runnerInputDirectoryMode = 0o700
	runnerInputFileMode      = 0o600
)

// ComposeRunnerLauncher 是 server 启动一次性 runner 的最小 Docker 边界。
// 它不暴露通用容器创建能力，调用方无法提供命令、环境或挂载。
type ComposeRunnerLauncher interface {
	Launch(context.Context, RunnerInput) error
}

type dockerComposeRunnerLauncher struct{ client dockerRunnerClient }

type dockerRunnerClient interface {
	ImagePull(context.Context, string, mobyclient.ImagePullOptions) (mobyclient.ImagePullResponse, error)
	ContainerCreate(context.Context, mobyclient.ContainerCreateOptions) (mobyclient.ContainerCreateResult, error)
	ContainerStart(context.Context, string, mobyclient.ContainerStartOptions) (mobyclient.ContainerStartResult, error)
}

// NewDockerComposeRunnerLauncher 创建只可执行官方 Compose runner 的 Docker socket launcher。
func NewDockerComposeRunnerLauncher() (ComposeRunnerLauncher, error) {
	client, err := mobyclient.New(mobyclient.WithHost("unix:///var/run/docker.sock"))
	if err != nil {
		return nil, fmt.Errorf("create compose runner docker client: %w", err)
	}
	return &dockerComposeRunnerLauncher{client: client}, nil
}

func (l *dockerComposeRunnerLauncher) Launch(ctx context.Context, input RunnerInput) error {
	if l == nil || l.client == nil {
		return errors.New("compose runner launcher is unavailable")
	}
	if err := ValidateRunnerInput(input); err != nil {
		return fmt.Errorf("validate compose runner launch: %w", err)
	}
	inputPath, err := persistRunnerInput(input)
	if err != nil {
		return err
	}
	pulled, err := l.client.ImagePull(ctx, input.Preflight.RunnerReference, mobyclient.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull digest-pinned compose runner: %w", err)
	}
	if _, err := io.Copy(io.Discard, pulled); err != nil {
		_ = pulled.Close()
		return fmt.Errorf("read compose runner pull result: %w", err)
	}
	if err := pulled.Close(); err != nil {
		return fmt.Errorf("close compose runner pull result: %w", err)
	}
	configuration, host := composeRunnerContainerConfig(input, inputPath)
	created, err := l.client.ContainerCreate(ctx, mobyclient.ContainerCreateOptions{Config: &configuration, HostConfig: &host, NetworkingConfig: &network.NetworkingConfig{}, Name: "graft-update-" + input.OperationID})
	if err != nil {
		return fmt.Errorf("create compose runner: %w", err)
	}
	if _, err := l.client.ContainerStart(ctx, created.ID, mobyclient.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("start compose runner: %w", err)
	}
	return nil
}

func persistRunnerInput(input RunnerInput) (string, error) {
	if !runnerOperationID.MatchString(input.OperationID) || !filepath.IsAbs(input.Preflight.ComposeRoot) {
		return "", errors.New("compose runner input path is invalid")
	}
	contents, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode compose runner input: %w", err)
	}
	path := filepath.Join(input.Preflight.ComposeRoot, runnerInputDirectory, input.OperationID+".json")
	if err := os.MkdirAll(filepath.Dir(path), runnerInputDirectoryMode); err != nil {
		return "", fmt.Errorf("create compose runner input directory: %w", err)
	}
	if err := os.WriteFile(path, append(contents, '\n'), runnerInputFileMode); err != nil {
		return "", fmt.Errorf("write compose runner input: %w", err)
	}
	return path, nil
}

func composeRunnerContainerConfig(input RunnerInput, inputPath string) (containertypes.Config, containertypes.HostConfig) {
	root := input.Preflight.ComposeRoot
	socket := input.Preflight.DockerSocket
	return containertypes.Config{Image: input.Preflight.RunnerReference, Env: []string{"GRAFT_UPDATE_RUNNER_INPUT=" + inputPath}, Labels: map[string]string{
		"io.graft.update.operation": input.OperationID,
		"io.graft.update.protocol":  "compose-runner/v1",
	}}, containertypes.HostConfig{AutoRemove: true, Binds: []string{root + ":" + root + ":rw", socket + ":" + socket + ":rw"}, NetworkMode: "none", ReadonlyRootfs: true, CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges:true"}}
}
