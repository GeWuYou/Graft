package runtimetarget

import (
	"context"
	"strings"
	"time"

	mobyclient "github.com/moby/moby/client"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

const composeProjectNameCheckTimeout = 5 * time.Second

const composeProjectLabel = "com.docker.compose.project"

type dockerComposeProjectNameProbe interface {
	Occupied(context.Context, store.Target, string) (bool, error)
}

type dockerComposeProjectNameProbeImpl struct{}

// CheckComposeProjectName reports whether a Compose name is already represented
// by runtime resources on the selected target. Provider-specific endpoint access
// remains inside the Runtime Target module.
func (r runtimeTargetReader) CheckComposeProjectName(ctx context.Context, id int64, name string) (moduleapi.ComposeProjectNameAvailability, error) {
	target, state, ready := r.composeProjectNameTarget(ctx, id, name)
	if !ready {
		return moduleapi.ComposeProjectNameAvailability{State: state}, nil
	}
	if target.Provider != "docker" {
		return moduleapi.ComposeProjectNameAvailability{State: moduleapi.ComposeProjectNameStateUnavailable}, nil
	}
	probe := r.composeProjectNameProbe
	if probe == nil {
		probe = dockerComposeProjectNameProbeImpl{}
	}
	occupied, err := probe.Occupied(ctx, target, strings.TrimSpace(name))
	if err != nil {
		return moduleapi.ComposeProjectNameAvailability{State: moduleapi.ComposeProjectNameStateError}, nil
	}
	if occupied {
		return moduleapi.ComposeProjectNameAvailability{State: moduleapi.ComposeProjectNameStateOccupied}, nil
	}
	return moduleapi.ComposeProjectNameAvailability{State: moduleapi.ComposeProjectNameStateAvailable}, nil
}

func (r runtimeTargetReader) composeProjectNameTarget(ctx context.Context, id int64, name string) (store.Target, moduleapi.ComposeProjectNameState, bool) {
	if r.repository == nil || id < 1 || strings.TrimSpace(name) == "" {
		return store.Target{}, moduleapi.ComposeProjectNameStateError, false
	}
	target, err := r.repository.Get(ctx, uint64(id))
	if err != nil {
		return store.Target{}, moduleapi.ComposeProjectNameStateUnavailable, false
	}
	if _, ok := composeTargetSummary(target); !ok || !target.Availability {
		return store.Target{}, moduleapi.ComposeProjectNameStateUnavailable, false
	}
	return target, "", true
}

// Occupied uses the Docker Compose project label with All=true, which includes
// stopped Compose containers as well as running ones.
func (dockerComposeProjectNameProbeImpl) Occupied(ctx context.Context, target store.Target, name string) (bool, error) {
	checkCtx, cancel := context.WithTimeout(ctx, composeProjectNameCheckTimeout)
	defer cancel()
	client, err := mobyclient.New(mobyclient.WithHost(target.EndpointLabel))
	if err != nil {
		return false, err
	}
	defer closeDockerClient(client)
	items, err := client.ContainerList(checkCtx, mobyclient.ContainerListOptions{
		All:     true,
		Filters: make(mobyclient.Filters).Add("label", composeProjectLabel+"="+name),
	})
	if err != nil {
		return false, err
	}
	return len(items.Items) > 0, nil
}
