package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	mobyclient "github.com/moby/moby/client"
)

type dockerClientAdapter struct {
	*mobyclient.Client
}

func (dockerClientAdapter) dockerSystemInfo() {}

type dockerClientSystemInfo struct {
	APIVersion        string
	ServerVersion     string
	OperatingSystem   string
	Architecture      string
	Containers        int
	ContainersRunning int
}

func (dockerClientSystemInfo) dockerSystemInfo() {}

func (d dockerClientAdapter) Info(ctx context.Context) (systemInfo, error) {
	info, err := d.Client.Info(ctx, mobyclient.InfoOptions{})
	if err != nil {
		return nil, err
	}
	return dockerClientSystemInfo{
		APIVersion:        d.ClientVersion(),
		ServerVersion:     info.Info.ServerVersion,
		OperatingSystem:   info.Info.OperatingSystem,
		Architecture:      info.Info.Architecture,
		Containers:        info.Info.Containers,
		ContainersRunning: info.Info.ContainersRunning,
	}, nil
}

func (d dockerClientAdapter) ContainerList(ctx context.Context, options mobyclient.ContainerListOptions) ([]container.Summary, error) {
	result, err := d.Client.ContainerList(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (d dockerClientAdapter) ImageList(ctx context.Context, options mobyclient.ImageListOptions) ([]image.Summary, error) {
	result, err := d.Client.ImageList(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (d dockerClientAdapter) ImageInspect(ctx context.Context, imageID string) (image.InspectResponse, error) {
	result, err := d.Client.ImageInspect(ctx, imageID)
	if err != nil {
		return image.InspectResponse{}, err
	}
	return result.InspectResponse, nil
}

func (d dockerClientAdapter) NetworkList(ctx context.Context, options mobyclient.NetworkListOptions) ([]network.Summary, error) {
	result, err := d.Client.NetworkList(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (d dockerClientAdapter) NetworkInspect(ctx context.Context, networkID string, options mobyclient.NetworkInspectOptions) (network.Inspect, error) {
	result, err := d.Client.NetworkInspect(ctx, networkID, options)
	if err != nil {
		return network.Inspect{}, err
	}
	return result.Network, nil
}

func (d dockerClientAdapter) VolumeList(ctx context.Context, options mobyclient.VolumeListOptions) ([]volume.Volume, error) {
	result, err := d.Client.VolumeList(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (d dockerClientAdapter) VolumeDiskUsage(ctx context.Context) ([]volume.Volume, error) {
	result, err := d.Client.DiskUsage(ctx, mobyclient.DiskUsageOptions{Volumes: true, Verbose: true})
	if err != nil {
		return nil, err
	}
	return result.Volumes.Items, nil
}

func (d dockerClientAdapter) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	result, err := d.Client.VolumeInspect(ctx, volumeID, mobyclient.VolumeInspectOptions{})
	if err != nil {
		return volume.Volume{}, err
	}
	return result.Volume, nil
}

func (d dockerClientAdapter) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	_, err := d.Client.VolumeRemove(ctx, volumeID, mobyclient.VolumeRemoveOptions{Force: force})
	return err
}

func (d dockerClientAdapter) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	result, err := d.Client.ContainerInspect(ctx, containerID, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return result.Container, nil
}

func (d dockerClientAdapter) ContainerLogs(ctx context.Context, containerID string, options mobyclient.ContainerLogsOptions) (io.ReadCloser, error) {
	return d.Client.ContainerLogs(ctx, containerID, options)
}

func (d dockerClientAdapter) ContainerStatsOneShot(ctx context.Context, containerID string) (mobyclient.ContainerStatsResult, error) {
	return d.ContainerStats(ctx, containerID, mobyclient.ContainerStatsOptions{
		Stream:                false,
		IncludePreviousSample: true,
	})
}

func (d dockerClientAdapter) ContainerExecCreate(ctx context.Context, containerID string, options mobyclient.ExecCreateOptions) (mobyclient.ExecCreateResult, error) {
	return d.ExecCreate(ctx, containerID, options)
}

func (d dockerClientAdapter) ContainerExecAttach(ctx context.Context, execID string, config mobyclient.ExecAttachOptions) (mobyclient.HijackedResponse, error) {
	result, err := d.ExecAttach(ctx, execID, config)
	if err != nil {
		return mobyclient.HijackedResponse{}, err
	}
	return result.HijackedResponse, nil
}

func (d dockerClientAdapter) ContainerExecResize(ctx context.Context, execID string, options mobyclient.ExecResizeOptions) error {
	_, err := d.ExecResize(ctx, execID, options)
	return err
}

func (d dockerClientAdapter) ContainerStart(ctx context.Context, containerID string, options mobyclient.ContainerStartOptions) error {
	_, err := d.Client.ContainerStart(ctx, containerID, options)
	return err
}

func (d dockerClientAdapter) ContainerStop(ctx context.Context, containerID string, options mobyclient.ContainerStopOptions) error {
	_, err := d.Client.ContainerStop(ctx, containerID, options)
	return err
}

func (d dockerClientAdapter) ContainerRestart(ctx context.Context, containerID string, options mobyclient.ContainerRestartOptions) error {
	_, err := d.Client.ContainerRestart(ctx, containerID, options)
	return err
}

func (d dockerClientAdapter) ContainerRemove(ctx context.Context, containerID string, options mobyclient.ContainerRemoveOptions) error {
	_, err := d.Client.ContainerRemove(ctx, containerID, options)
	return err
}

func mapDockerError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errContainerRuntimeTimeout
	}
	if cerrdefs.IsNotFound(err) {
		return errContainerNotFound
	}
	if errors.Is(err, os.ErrNotExist) {
		return errRuntimeSocketMissing
	}
	if errors.Is(err, os.ErrPermission) {
		return errRuntimePermissionDenied
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errContainerRuntimeTimeout
	}
	if mapped := mapSyscallDockerError(err); mapped != nil {
		return mapped
	}
	return mapDockerMessageError(err.Error())
}

func mapSyscallDockerError(err error) error {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return nil
	}
	switch errno {
	case syscall.EACCES, syscall.EPERM:
		return errRuntimePermissionDenied
	case syscall.ENOENT:
		return errRuntimeSocketMissing
	case syscall.ECONNREFUSED, syscall.ECONNRESET:
		return errRuntimeDaemonUnavailable
	default:
		return nil
	}
}

// mapDockerMessageError 根据错误消息中的关键片段映射容器运行时错误。
func mapDockerMessageError(message string) error {
	normalized := strings.ToLower(message)
	for _, rule := range dockerErrorMessageRules {
		if strings.Contains(normalized, rule.fragment) {
			return rule.err
		}
	}
	return errRuntimeDaemonUnavailable
}

var dockerErrorMessageRules = []struct {
	fragment string
	err      error
}{
	{fragment: "permission denied", err: errRuntimePermissionDenied},
	{fragment: "no such file", err: errRuntimeSocketMissing},
	{fragment: "cannot connect", err: errRuntimeDaemonUnavailable},
	{fragment: "connection refused", err: errRuntimeDaemonUnavailable},
	{fragment: "not found", err: errContainerNotFound},
	{fragment: "is already", err: errInvalidContainerState},
	{fragment: "not running", err: errInvalidContainerState},
}

func (r *DockerRuntime) String() string {
	return fmt.Sprintf("DockerRuntime(%s)", safeEndpointLabel(r.endpoint))
}

var _ Runtime = (*DockerRuntime)(nil)
