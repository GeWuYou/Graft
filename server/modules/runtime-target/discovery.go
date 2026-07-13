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
		// Do not add a target on hosts that have never had Docker. Once discovered,
		// retain the identity so a later outage remains visible and recoverable.
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

// refreshTarget refreshes the local Docker availability record before returning a matching target.
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

// probeError converts a Docker probe error into the message stored for an unavailable Unix socket.
// It returns an empty string when the probe succeeds.
func probeError(err error) string {
	if err == nil {
		return ""
	}
	return "Docker Unix socket is unavailable"
}
