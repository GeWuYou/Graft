package container

import (
	"context"
	"encoding/json"
	"testing"

	"graft/server/internal/moduleapi"
)

type dockerMutationResourceRuntime struct {
	fakeRuntime
	image   DockerImage
	network DockerNetwork
	volume  DockerVolume
}

func (r dockerMutationResourceRuntime) ListDockerImages(context.Context) (DockerImageListResult, error) {
	return DockerImageListResult{Items: []DockerImage{r.image}}, nil
}
func (r dockerMutationResourceRuntime) ReadDockerImage(context.Context, string) (DockerImage, error) {
	return r.image, nil
}
func (dockerMutationResourceRuntime) ListDockerNetworks(context.Context) ([]DockerNetwork, error) {
	return nil, nil
}
func (r dockerMutationResourceRuntime) ReadDockerNetwork(context.Context, string) (DockerNetwork, error) {
	return r.network, nil
}
func (dockerMutationResourceRuntime) ListDockerVolumes(context.Context) ([]DockerVolume, error) {
	return nil, nil
}
func (r dockerMutationResourceRuntime) ReadDockerVolume(context.Context, string) (DockerVolume, error) {
	return r.volume, nil
}

func TestDockerMutationsFreezeProviderNeutralExternalStages(t *testing.T) {
	t.Parallel()
	tasks := &containerTaskRuntimeStub{receipt: moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}}
	runtime := dockerMutationResourceRuntime{
		image:   DockerImage{ID: "sha256:image", RepositoryTags: []string{"example/app:stable"}},
		network: DockerNetwork{ID: "network-1", Name: "private"},
		volume:  DockerVolume{Name: "data"},
	}
	service, err := newRouteTestService(containerServiceOptions{runtime: runtime, enabled: true, dangerousActionsEnabled: true, tasks: tasks})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cases := []struct {
		operation string
		submit    func() (moduleapi.TaskReceipt, error)
	}{
		{containerImageTagOperation, func() (moduleapi.TaskReceipt, error) {
			return service.SubmitDockerImageTag(context.Background(), "sha256:image", "example/app:stable", 7, "tag")
		}},
		{containerImageUntagOperation, func() (moduleapi.TaskReceipt, error) {
			return service.SubmitDockerImageUntag(context.Background(), "sha256:image", "example/app:stable", 7, "untag")
		}},
		{containerImageRemoveOperation, func() (moduleapi.TaskReceipt, error) {
			return service.SubmitDockerImageRemove(context.Background(), "sha256:image", true, 7, "image-remove")
		}},
		{containerNetworkCreateOperation, func() (moduleapi.TaskReceipt, error) {
			return service.SubmitDockerNetworkCreate(context.Background(), DockerNetworkCreateCommand{Name: "private", Driver: "bridge"}, 7, "network-create")
		}},
		{containerNetworkRemoveOperation, func() (moduleapi.TaskReceipt, error) {
			return service.SubmitDockerNetworkRemove(context.Background(), "network-1", "private", 7, "network-remove")
		}},
		{containerVolumeRemoveOperation, func() (moduleapi.TaskReceipt, error) {
			return service.SubmitDockerVolumeRemove(context.Background(), "data", false, 7, "volume-remove")
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.operation, func(t *testing.T) {
			before := len(tasks.submissions)
			if _, err := testCase.submit(); err != nil {
				t.Fatalf("submit mutation: %v", err)
			}
			stage := tasks.submissions[before].Plan.Stages[0]
			assertContainerExternalExecution(t, stage, testCase.operation)
			assertProviderNeutralContainerPayload(t, stage.Input)
		})
	}
}

func TestDockerMutationBatchUsesOrderedFailFastStages(t *testing.T) {
	t.Parallel()
	tasks := &containerTaskRuntimeStub{}
	service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, dangerousActionsEnabled: true, tasks: tasks})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if _, err := service.SubmitDockerImageBatchRemove(context.Background(), []string{"sha256:first", "sha256:second"}, false, 7, "batch-remove"); err != nil {
		t.Fatalf("submit batch mutation: %v", err)
	}
	stages := tasks.submissions[0].Plan.Stages
	if len(stages) != 2 || stages[0].Key != containerImageRemoveOperation+"-1" || stages[1].Key != containerImageRemoveOperation+"-2" {
		t.Fatalf("expected ordered per-item stages, got %#v", stages)
	}
	for _, stage := range stages {
		assertContainerExternalExecution(t, stage, containerImageRemoveOperation)
		assertProviderNeutralContainerPayload(t, stage.Input)
	}
}

func assertProviderNeutralContainerPayload(t *testing.T, input json.RawMessage) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		t.Fatalf("decode container payload: %v", err)
	}
	for _, forbidden := range []string{"operation", "version", "endpoint", "socket", "certificate", "credential", "path", "argv", "command"} {
		if _, exists := fields[forbidden]; exists {
			t.Fatalf("forbidden field %q persisted in %s", forbidden, input)
		}
	}
}
