package runtimetarget

import (
	"context"
	"net"
	"time"

	store "graft/server/modules/runtime-target/store"
)

const localDockerEndpoint = "unix:///var/run/docker.sock"
const localDockerProbeTimeout = 2 * time.Second

// discoverLocalDocker 探测本地 Docker Unix 套接字的可用性，并将结果写入仓库。
// 返回写入探测结果时发生的错误。
func discoverLocalDocker(parent context.Context, repository *store.SQLRepository) error {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, localDockerProbeTimeout)
	defer cancel()
	dialer := net.Dialer{}
	connection, err := dialer.DialContext(ctx, "unix", "/var/run/docker.sock")
	available := err == nil
	if connection != nil {
		_ = connection.Close()
	}
	return repository.UpsertLocalDocker(ctx, store.LocalDockerProbe{Endpoint: localDockerEndpoint, Available: available, Error: probeError(err), CheckedAt: time.Now().UTC()})
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
