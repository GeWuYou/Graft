package container

import (
	"context"
	"errors"
	"os"
	"strings"

	"graft/server/internal/moduleapi"
)

// CurrentContainer 返回当前 server 容器的复制后 Docker inspect 原始事实投影。
func (r containerProjectRuntimeReader) CurrentContainer(ctx context.Context) (moduleapi.DockerContainerFacts, error) {
	if r.service == nil {
		return moduleapi.DockerContainerFacts{}, errRuntimeDisabled
	}
	runtime, err := r.service.runtimeForRequestContext(ctx)
	if err != nil {
		return moduleapi.DockerContainerFacts{}, err
	}
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return moduleapi.DockerContainerFacts{}, errors.New("current server container identity is unavailable")
	}
	detail, err := runtime.Detail(ctx, Ref{Value: strings.TrimSpace(hostname)})
	if err != nil {
		return moduleapi.DockerContainerFacts{}, err
	}
	labels := make(map[string]string, len(detail.Labels))
	for key, value := range detail.Labels {
		labels[key] = value
	}
	mounts := make([]moduleapi.DockerMountFact, 0, len(detail.Mounts))
	for _, mount := range detail.Mounts {
		mounts = append(mounts, moduleapi.DockerMountFact{Type: mount.Type, Source: mount.Source, Destination: mount.Destination})
	}
	return moduleapi.DockerContainerFacts{ContainerID: strings.TrimSpace(hostname), Labels: labels, Mounts: mounts}, nil
}

var _ moduleapi.DockerFactsProvider = containerProjectRuntimeReader{}
