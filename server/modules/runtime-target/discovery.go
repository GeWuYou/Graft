package runtimetarget

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	mobyclient "github.com/moby/moby/client"
	store "graft/server/modules/runtime-target/store"
)

const localDockerEndpoint = "unix:///var/run/docker.sock"
const localDockerProbeTimeout = 2 * time.Second

// discoverLocalDocker 探测本地 Docker Unix 套接字的可用性，并将结果写入仓库。
// 若主机不存在 Docker 套接字且仓库中没有既有的本地 Docker 记录，则跳过写入。
func discoverLocalDocker(parent context.Context, repository *store.SQLRepository) error {
	if parent == nil {
		parent = context.Background()
	}
	_, statErr := os.Stat("/var/run/docker.sock")
	if errors.Is(statErr, os.ErrNotExist) {
		// 主机从未安装 Docker 时不创建目标；一旦发现过则保留身份，
		// 使后续故障仍可见并可恢复。
		if _, err := repository.FindSystemLocalDocker(parent); errors.Is(err, store.ErrNotFound) {
			return nil
		} else if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(parent, localDockerProbeTimeout)
	defer cancel()
	dialer := net.Dialer{}
	connection, err := dialer.DialContext(ctx, "unix", "/var/run/docker.sock")
	if connection != nil {
		_ = connection.Close()
	}
	if err == nil {
		err = pingLocalDocker(ctx)
	}
	return repository.UpsertLocalDocker(ctx, store.LocalDockerProbe{Endpoint: localDockerEndpoint, Available: err == nil, Error: probeError(err), CheckedAt: time.Now().UTC()})
}

func pingLocalDocker(ctx context.Context) error {
	client, err := mobyclient.New(mobyclient.WithHost(localDockerEndpoint))
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	_, err = client.Ping(ctx, mobyclient.PingOptions{NegotiateAPIVersion: true})
	return err
}

// refreshTarget 在返回匹配目标前刷新本地 Docker 可用性记录。
func refreshTarget(ctx context.Context, repository *store.SQLRepository, id uint64) (store.Target, error) {
	target, err := repository.Get(ctx, id)
	if err != nil {
		return store.Target{}, err
	}
	if target.Provider != "docker" || target.ConnectionKind != "unix_socket" || target.EndpointLabel != localDockerEndpoint {
		return target, nil
	}
	if err := discoverLocalDocker(ctx, repository); err != nil {
		return store.Target{}, err
	}
	return repository.Get(ctx, id)
}

// probeError 将 Docker 探测错误转换为 Unix socket 不可用时持久化的消息；
// 探测成功时返回空字符串。
func probeError(err error) string {
	if err == nil {
		return ""
	}
	return "Docker Unix socket is unavailable"
}
