package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

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
	Close() error
}

type dockerComposeRunnerLauncher struct{ client dockerRunnerClient }

type dockerRunnerClient interface {
	ImagePull(context.Context, string, mobyclient.ImagePullOptions) (mobyclient.ImagePullResponse, error)
	ContainerCreate(context.Context, mobyclient.ContainerCreateOptions) (mobyclient.ContainerCreateResult, error)
	ContainerStart(context.Context, string, mobyclient.ContainerStartOptions) (mobyclient.ContainerStartResult, error)
	ContainerInspect(context.Context, string, mobyclient.ContainerInspectOptions) (mobyclient.ContainerInspectResult, error)
	ContainerRemove(context.Context, string, mobyclient.ContainerRemoveOptions) (mobyclient.ContainerRemoveResult, error)
	Close() error
}

// Close 释放 Docker API client 持有的连接。
func (l *dockerComposeRunnerLauncher) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

// NewDockerComposeRunnerLauncher 创建只可执行官方 Compose runner 的 Docker socket launcher。
func NewDockerComposeRunnerLauncher() (ComposeRunnerLauncher, error) {
	client, err := mobyclient.New(mobyclient.WithHost("unix:///var/run/docker.sock"))
	if err != nil {
		return nil, fmt.Errorf("create compose runner docker client: %w", err)
	}
	return &dockerComposeRunnerLauncher{client: client}, nil
}

//nolint:cyclop // 启动流程中的每条失败分支对应一个独立的资源与错误边界。
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
	options := mobyclient.ContainerCreateOptions{Config: &configuration, HostConfig: &host, NetworkingConfig: &network.NetworkingConfig{}}
	options.Name = composeRunnerContainerName(input.OperationID)
	created, err := l.client.ContainerCreate(ctx, options)
	if err != nil {
		return fmt.Errorf("create compose runner: %w", err)
	}
	if _, err := l.client.ContainerStart(ctx, created.ID, mobyclient.ContainerStartOptions{}); err != nil {
		if cleanupErr := l.removeUnstartedRunner(ctx, created.ID); cleanupErr != nil {
			return fmt.Errorf("start compose runner: %w; clean up unstarted runner: %v", err, cleanupErr)
		}
		return fmt.Errorf("start compose runner: %w", err)
	}
	return nil
}

func (l *dockerComposeRunnerLauncher) removeUnstartedRunner(ctx context.Context, id string) error {
	inspected, err := l.client.ContainerInspect(ctx, id, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return err
	}
	if inspected.Container.State == nil {
		return errors.New("inspect compose runner returned no state")
	}
	if inspected.Container.State.Running {
		return nil
	}
	_, err = l.client.ContainerRemove(ctx, id, mobyclient.ContainerRemoveOptions{})
	return err
}

func composeRunnerContainerName(operationID string) string { return "graft-update-" + operationID }

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
	groups := []string{}
	if stat, err := os.Stat(socket); err == nil {
		if details, ok := stat.Sys().(*syscall.Stat_t); ok {
			groups = append(groups, strconv.FormatUint(uint64(details.Gid), 10))
		}
	}
	return containertypes.Config{Image: input.Preflight.RunnerReference, User: "65532:65532", Env: []string{"GRAFT_UPDATE_RUNNER_INPUT=" + inputPath}, Labels: map[string]string{
		"io.graft.update.operation": input.OperationID,
		"io.graft.update.protocol":  "compose-runner/v1",
	}}, containertypes.HostConfig{AutoRemove: true, Binds: []string{root + ":" + root + ":rw", socket + ":" + socket + ":rw"}, GroupAdd: groups, NetworkMode: "none", ReadonlyRootfs: true, CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges:true"}}
}
