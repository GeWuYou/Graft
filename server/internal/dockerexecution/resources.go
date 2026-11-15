package dockerexecution

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	distributionref "github.com/distribution/reference"
	mobyclient "github.com/moby/moby/client"
)

// ResolveResources 同时冻结真实引擎身份和资源协调键，排队期间的别名变化不能改写操作目标。
func ResolveResources(ctx context.Context, environment Environment, request Request) (Request, []string, error) {
	var intent containerExecutionIntent
	if request.Capability != "container_execution" || strictDecode(request.Input, &intent) != nil || !validContainerIntent(request.OperationID, intent) {
		return request, nil, errors.New("container coordination input is invalid")
	}
	if request.OperationID == "container.image.pull.v1" {
		return request, []string{imageResourceKey(intent.ImageRef)}, nil
	}
	if request.OperationID == "container.network.create.v1" {
		return request, []string{"network:" + intent.Name}, nil
	}
	docker, err := mobyclient.New(mobyclient.WithHost(environment.DockerSocket))
	if err != nil {
		return request, nil, err
	}
	defer func() { _ = docker.Close() }()
	resolved, keys, err := resolveResourceIdentity(ctx, docker, intent)
	if err != nil {
		return request, nil, err
	}
	payload, err := json.Marshal(resolved)
	if err != nil {
		return request, nil, err
	}
	request.Input = payload
	return request, keys, nil
}

func resolveResourceIdentity(ctx context.Context, docker *mobyclient.Client, intent containerExecutionIntent) (containerExecutionIntent, []string, error) {
	switch {
	case intent.ContainerRef != "":
		return resolveContainerIdentity(ctx, docker, intent)
	case intent.ImageRef != "":
		return resolveImageIdentity(ctx, docker, intent)
	case intent.NetworkRef != "":
		return resolveNetworkIdentity(ctx, docker, intent)
	case intent.VolumeRef != "":
		return resolveVolumeIdentity(ctx, docker, intent)
	default:
		return intent, nil, errors.New("container coordination resource is unavailable")
	}
}

func resolveContainerIdentity(ctx context.Context, docker *mobyclient.Client, intent containerExecutionIntent) (containerExecutionIntent, []string, error) {
	inspected, err := docker.ContainerInspect(ctx, intent.ContainerRef, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return intent, nil, err
	}
	intent.ContainerRef = inspected.Container.ID
	keys := []string{"container:" + intent.ContainerRef}
	if inspected.Container.Config != nil {
		keys = withComposeResource(keys, inspected.Container.Config.Labels)
	}
	return intent, keys, nil
}

func resolveImageIdentity(ctx context.Context, docker *mobyclient.Client, intent containerExecutionIntent) (containerExecutionIntent, []string, error) {
	inspected, err := docker.ImageInspect(ctx, intent.ImageRef)
	if err != nil {
		return intent, nil, err
	}
	keys := []string{imageResourceKey(intent.ImageRef), "image:" + inspected.ID}
	intent.ImageRef = inspected.ID
	if intent.TargetRef != "" {
		keys = append(keys, imageResourceKey(intent.TargetRef))
	}
	if intent.TagRef != "" {
		keys = append(keys, imageResourceKey(intent.TagRef))
	}
	return intent, keys, nil
}

func resolveNetworkIdentity(ctx context.Context, docker *mobyclient.Client, intent containerExecutionIntent) (containerExecutionIntent, []string, error) {
	inspected, err := docker.NetworkInspect(ctx, intent.NetworkRef, mobyclient.NetworkInspectOptions{})
	if err != nil {
		return intent, nil, err
	}
	intent.NetworkRef = inspected.Network.ID
	keys := withComposeResource([]string{"network:" + inspected.Network.ID, "network:" + inspected.Network.Name}, inspected.Network.Labels)
	return intent, keys, nil
}

func resolveVolumeIdentity(ctx context.Context, docker *mobyclient.Client, intent containerExecutionIntent) (containerExecutionIntent, []string, error) {
	inspected, err := docker.VolumeInspect(ctx, intent.VolumeRef, mobyclient.VolumeInspectOptions{})
	if err != nil {
		return intent, nil, err
	}
	keys := withComposeResource([]string{"volume:" + inspected.Volume.Name}, inspected.Volume.Labels)
	return intent, keys, nil
}

func withComposeResource(keys []string, labels map[string]string) []string {
	if project := strings.TrimSpace(labels["com.docker.compose.project"]); project != "" {
		return append(keys, "compose:"+project)
	}
	return keys
}

func imageResourceKey(reference string) string {
	named, err := distributionref.ParseNormalizedNamed(reference)
	if err == nil {
		return "image:" + distributionref.TagNameOnly(named).String()
	}
	return "image:" + reference
}
