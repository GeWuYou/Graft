package container

import (
	"context"
	"testing"
	"time"

	dockertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	mobyclient "github.com/moby/moby/client"
)

func TestListDockerVolumesFiltersSortsAndPages(t *testing.T) {
	t.Parallel()

	used, small, large := int64(2), int64(10), int64(20)
	items := []DockerVolume{
		{Name: "zeta", Driver: "local", Scope: "local", ReferenceCount: &used, SizeBytes: &large},
		{Name: "alpha", Driver: "local", Scope: "local", ReferenceCount: &used, SizeBytes: &small},
		{Name: "beta", Driver: "nfs", Scope: "global"},
	}
	result := listDockerVolumes(items, DockerVolumeListQuery{Driver: "local", Usage: "used", Limit: 1, Offset: 1})

	if result.Total != 2 || result.Limit != 1 || result.Offset != 1 {
		t.Fatalf("unexpected page metadata: %#v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "alpha" {
		t.Fatalf("expected sorted second matching volume, got %#v", result.Items)
	}
}

func TestListDockerVolumesSortsBySizeWithNilAlwaysLast(t *testing.T) {
	t.Parallel()

	small, large := int64(10), int64(20)
	items := []DockerVolume{
		{Name: "unknown-b"},
		{Name: "large-b", SizeBytes: &large},
		{Name: "small", SizeBytes: &small},
		{Name: "large-a", SizeBytes: &large},
		{Name: "unknown-a"},
	}
	tests := []struct {
		name, order string
		want        []string
	}{
		{name: "default descending", want: []string{"large-a", "large-b", "small", "unknown-a", "unknown-b"}},
		{name: "explicit descending", order: "desc", want: []string{"large-a", "large-b", "small", "unknown-a", "unknown-b"}},
		{name: "explicit ascending", order: "asc", want: []string{"small", "large-a", "large-b", "unknown-a", "unknown-b"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := listDockerVolumes(items, DockerVolumeListQuery{SortBy: "size_bytes", SortOrder: test.order, Limit: len(items)})
			for index, name := range test.want {
				if result.Items[index].Name != name {
					t.Fatalf("unexpected order at %d: got %q, want %q (%#v)", index, result.Items[index].Name, name, result.Items)
				}
			}
		})
	}
}

func TestListDockerNetworksFiltersPagesAndSummarizesSnapshot(t *testing.T) {
	items := []DockerNetwork{
		{Name: "zeta", Driver: "bridge", Scope: "local", ContainerCount: 1, Removable: false},
		{Name: "alpha", Driver: "bridge", Scope: "local", ContainerCount: 0, Removable: true},
		{Name: "other", Driver: "overlay", Scope: "swarm", ContainerCount: 0, Removable: true},
	}
	result := listDockerNetworks(items, DockerNetworkListQuery{Driver: "bridge", Usage: "unused", Limit: 1})
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Name != "alpha" {
		t.Fatalf("unexpected network page %#v", result)
	}
	if result.Summary != (DockerNetworkListSummary{Total: 3, InUse: 1, Unused: 2}) {
		t.Fatalf("unexpected network summary %#v", result.Summary)
	}
}

func TestSummarizeDockerNetworksDoesNotCountUnknownAsUnused(t *testing.T) {
	t.Parallel()

	items := []DockerNetwork{
		{RelationshipStatus: dockerResourceRelationshipStatusUsed},
		{RelationshipStatus: dockerResourceRelationshipStatusUnused},
		{RelationshipStatus: dockerResourceRelationshipStatusUnknown},
		{RelationshipStatus: dockerResourceRelationshipStatusException},
	}

	if got := summarizeDockerNetworks(items); got != (DockerNetworkListSummary{Total: 4, InUse: 1, Unused: 1}) {
		t.Fatalf("unknown network relationship must not be counted as unused: %#v", got)
	}
}

func TestListDockerVolumesNormalizesNegativePagination(t *testing.T) {
	t.Parallel()

	items := []DockerVolume{{Name: "alpha"}, {Name: "beta"}}
	result := listDockerVolumes(items, DockerVolumeListQuery{Limit: -1, Offset: -3})

	if result.Limit != 0 || result.Offset != 0 {
		t.Fatalf("expected non-negative pagination metadata, got %#v", result)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty page for normalized zero limit, got %#v", result.Items)
	}
}

func TestListDockerVolumesAvoidsPaginationSliceOverflow(t *testing.T) {
	t.Parallel()

	result := listDockerVolumes([]DockerVolume{{Name: "alpha"}}, DockerVolumeListQuery{Limit: int(^uint(0) >> 1), Offset: int(^uint(0) >> 1)})
	if result.Offset != int(^uint(0)>>1) || len(result.Items) != 0 {
		t.Fatalf("expected bounded empty page, got %#v", result)
	}
}

func TestDockerRelationshipUsageMatchesServerOwnedStatus(t *testing.T) {
	if dockerRelationshipUsageMatches(dockerResourceRelationshipStatusUnknown, "unused") {
		t.Fatal("unknown relationship status must not match unused")
	}
	if dockerRelationshipUsageMatches(dockerResourceRelationshipStatusException, "used") {
		t.Fatal("exception relationship status must not match used")
	}
	if !dockerRelationshipUsageMatches(dockerResourceRelationshipStatusUnused, "unused") {
		t.Fatal("unused relationship status must match unused")
	}
	if !dockerRelationshipUsageMatches(dockerResourceRelationshipStatusUsed, "used") {
		t.Fatal("used relationship status must match used")
	}
}

func TestDockerResourceContextClassifiesComposeAndDefaultResources(t *testing.T) {
	t.Parallel()

	compose := dockerResourceContext(map[string]string{
		composeProjectLabel: "gitea",
		composeNetworkLabel: "backend",
	}, "gitea_backend", composeNetworkLabel, false)
	if compose.Runtime != runtimeNameDocker || compose.Source != dockerResourceSourceCompose || compose.ComposeProject != "gitea" || compose.ComposeResource != "backend" {
		t.Fatalf("unexpected compose context: %#v", compose)
	}

	defaultNetwork := dockerResourceContext(nil, "bridge", composeNetworkLabel, true)
	if defaultNetwork.Runtime != runtimeNameDocker || defaultNetwork.Source != dockerResourceSourceDockerDefault {
		t.Fatalf("unexpected default Docker context: %#v", defaultNetwork)
	}
}

func TestListDockerResourcesFiltersNormalizedContext(t *testing.T) {
	t.Parallel()

	networks := []DockerNetwork{
		{Name: "gitea_backend", Context: DockerResourceContext{Source: dockerResourceSourceCompose, ComposeProject: "gitea"}, RelationshipStatus: dockerResourceRelationshipStatusUsed},
		{Name: "grafana_default", Context: DockerResourceContext{Source: dockerResourceSourceCompose, ComposeProject: "grafana"}, RelationshipStatus: dockerResourceRelationshipStatusUnused},
		{Name: "bridge", Context: DockerResourceContext{Source: dockerResourceSourceDockerDefault}, RelationshipStatus: dockerResourceRelationshipStatusUsed},
	}
	filteredNetworks := listDockerNetworks(networks, DockerNetworkListQuery{Source: dockerResourceSourceCompose, ComposeProject: "GITEA", Limit: 20})
	if filteredNetworks.Total != 1 || filteredNetworks.Items[0].Name != "gitea_backend" {
		t.Fatalf("unexpected context-filtered networks: %#v", filteredNetworks)
	}

	volumes := []DockerVolume{
		{Name: "gitea_data", Context: DockerResourceContext{Source: dockerResourceSourceCompose, ComposeProject: "gitea"}, RelationshipStatus: dockerResourceRelationshipStatusUsed},
		{Name: "docker_data", Context: DockerResourceContext{Source: dockerResourceSourceDocker}, RelationshipStatus: dockerResourceRelationshipStatusUnused},
	}
	filteredVolumes := listDockerVolumes(volumes, DockerVolumeListQuery{Source: dockerResourceSourceCompose, ComposeProject: "gitea", Usage: "used", Limit: 20})
	if filteredVolumes.Total != 1 || filteredVolumes.Items[0].Name != "gitea_data" {
		t.Fatalf("unexpected context-filtered volumes: %#v", filteredVolumes)
	}
}

func TestDockerVolumeReferencesDeduplicateMountsAndIgnoreNonVolumes(t *testing.T) {
	t.Parallel()

	refs := dockerVolumeReferences([]dockertypes.Summary{
		{
			ID:    "container-b",
			Names: []string{"/worker"},
			Mounts: []dockertypes.MountPoint{
				{Type: mount.TypeVolume, Name: "data"},
				{Type: mount.TypeVolume, Name: "data"},
				{Type: mount.TypeBind, Name: "host-path"},
			},
		},
		{ID: "container-a", Names: []string{"/api"}, Mounts: []dockertypes.MountPoint{{Type: mount.TypeVolume, Name: "data"}}},
	})

	data := refs["data"]
	if len(data) != 2 || data[0].Name != "api" || data[1].Name != "worker" {
		t.Fatalf("unexpected volume references: %#v", data)
	}
	if _, ok := refs["host-path"]; ok {
		t.Fatal("bind mount must not be reported as a Docker volume reference")
	}
}

func TestDockerNetworkReferencesDeduplicateAndUseNetworkID(t *testing.T) {
	t.Parallel()

	references := dockerNetworkContainerReferences([]dockertypes.Summary{
		{
			ID:    "container-b",
			Names: []string{"/worker"},
			NetworkSettings: &dockertypes.NetworkSettingsSummary{Networks: map[string]*network.EndpointSettings{
				"backend": {NetworkID: "network-backend"},
			}},
		},
		{
			ID:    "container-a",
			Names: []string{"/api"},
			NetworkSettings: &dockertypes.NetworkSettingsSummary{Networks: map[string]*network.EndpointSettings{
				"backend": {NetworkID: "network-backend"},
				"other":   {NetworkID: "network-other"},
			}},
		},
	})

	backend := references["network-backend"]
	if len(backend) != 2 || backend[0].Name != "api" || backend[1].Name != "worker" {
		t.Fatalf("unexpected network references: %#v", backend)
	}
	if other := references["network-other"]; len(other) != 1 || other[0].ID != "container-a" {
		t.Fatalf("unexpected secondary network references: %#v", other)
	}
}

func TestDockerNetworkReferencesIgnoreContainersWithoutNetworkSettings(t *testing.T) {
	t.Parallel()

	references := dockerNetworkContainerReferences([]dockertypes.Summary{{ID: "container-a"}})
	if len(references) != 0 {
		t.Fatalf("expected nil network settings to produce no references, got %#v", references)
	}
}

func TestDockerNetworkInspectReferencesExcludeEndpointMetadata(t *testing.T) {
	t.Parallel()

	references := dockerNetworkInspectReferences(map[string]network.EndpointResource{
		"container-b": {Name: "/worker", EndpointID: "endpoint-b"},
		"container-a": {Name: "/api", EndpointID: "endpoint-a"},
	})
	if len(references) != 2 || references[0] != (DockerNetworkContainerReference{ID: "container-a", Name: "api"}) || references[1] != (DockerNetworkContainerReference{ID: "container-b", Name: "worker"}) {
		t.Fatalf("unexpected sanitized inspect references: %#v", references)
	}
}

func TestListDockerVolumesSummaryUsesFullSnapshotBeforeFilters(t *testing.T) {
	t.Parallel()

	used, unused, size := int64(1), int64(0), int64(1024)
	result := listDockerVolumes([]DockerVolume{
		{Name: "used", ReferenceCount: &used, SizeBytes: &size},
		{Name: "unused", ReferenceCount: &unused, SizeBytes: &size},
		{Name: "status-used", RelationshipStatus: dockerResourceRelationshipStatusUsed, SizeBytes: &size},
		{Name: "unknown", SizeBytes: nil},
	}, DockerVolumeListQuery{Usage: "used", Limit: 20})

	if result.Total != 2 || result.Summary.Total != 4 || result.Summary.InUse != 2 || result.Summary.Unused != 1 || result.Summary.ReferenceUnknown != 1 {
		t.Fatalf("unexpected filtered result or full summary: %#v", result)
	}
	if result.Summary.SizeBytes != nil {
		t.Fatalf("expected incomplete size summary to be unavailable, got %v", *result.Summary.SizeBytes)
	}
}

func TestMapDockerVolumeErrorPreservesConflict(t *testing.T) {
	t.Parallel()

	if got := mapDockerVolumeError(assertError("volume is in use")); got != errDockerVolumeConflict {
		t.Fatalf("expected volume conflict, got %v", got)
	}
}

type volumeTestError string

func (e volumeTestError) Error() string { return string(e) }

func assertError(message string) error { return volumeTestError(message) }

type volumeDetailDockerClient struct {
	countingDockerClient
	volume               volume.Volume
	volumes              []volume.Volume
	usage                []volume.Volume
	volumeDiskUsageCall  int
	volumeDiskUsageCtx   context.Context
	containerListOptions mobyclient.ContainerListOptions
	containerListErr     error
}

func (c *volumeDetailDockerClient) ImageList(context.Context, mobyclient.ImageListOptions) ([]image.Summary, error) {
	return nil, nil
}

func (c *volumeDetailDockerClient) ImageInspect(context.Context, string) (image.InspectResponse, error) {
	return image.InspectResponse{}, nil
}

func (c *volumeDetailDockerClient) NetworkList(context.Context, mobyclient.NetworkListOptions) ([]network.Summary, error) {
	return nil, nil
}

func (c *volumeDetailDockerClient) NetworkInspect(context.Context, string, mobyclient.NetworkInspectOptions) (network.Inspect, error) {
	return network.Inspect{}, nil
}

func (c *volumeDetailDockerClient) VolumeList(context.Context, mobyclient.VolumeListOptions) ([]volume.Volume, error) {
	return c.volumes, nil
}

func (c *volumeDetailDockerClient) VolumeDiskUsage(ctx context.Context) ([]volume.Volume, error) {
	c.volumeDiskUsageCall++
	c.volumeDiskUsageCtx = ctx
	return c.usage, nil
}

func (c *volumeDetailDockerClient) VolumeInspect(context.Context, string) (volume.Volume, error) {
	return c.volume, nil
}

func (c *volumeDetailDockerClient) VolumeRemove(context.Context, string, bool) error {
	return nil
}

func (c *volumeDetailDockerClient) ContainerList(_ context.Context, options mobyclient.ContainerListOptions) ([]dockertypes.Summary, error) {
	c.containerListOptions = options
	return nil, c.containerListErr
}

func TestListDockerVolumesUsesShortLivedUsageContext(t *testing.T) {
	t.Parallel()

	client := &volumeDetailDockerClient{}
	runtime := &DockerRuntime{client: client}

	if _, err := runtime.ListDockerVolumes(context.Background()); err != nil {
		t.Fatalf("list volumes: %v", err)
	}
	deadline, ok := client.volumeDiskUsageCtx.Deadline()
	if !ok {
		t.Fatal("expected volume usage context to have a deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > dockerVolumeUsageTimeout {
		t.Fatalf("expected usage deadline within %s, got %s", dockerVolumeUsageTimeout, remaining)
	}
	if client.volumeDiskUsageCtx.Err() != context.Canceled {
		t.Fatalf("expected usage context to be canceled after the call, got %v", client.volumeDiskUsageCtx.Err())
	}
}

func TestListDockerVolumesKeepsKnownUsageStatusWhenContainerReferencesFail(t *testing.T) {
	t.Parallel()

	client := &volumeDetailDockerClient{
		volumes: []volume.Volume{{Name: "used"}, {Name: "unused"}},
		usage: []volume.Volume{
			{Name: "used", UsageData: &volume.UsageData{RefCount: 1, Size: 4096}},
			{Name: "unused", UsageData: &volume.UsageData{RefCount: 0, Size: 1024}},
		},
		containerListErr: assertError("container list unavailable"),
	}
	runtime := &DockerRuntime{client: client}

	items, err := runtime.ListDockerVolumes(context.Background())
	if err != nil {
		t.Fatalf("list volumes: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two volumes, got %#v", items)
	}
	if items[0].RelationshipStatus != dockerResourceRelationshipStatusUsed {
		t.Fatalf("expected used volume to retain Docker usage status, got %#v", items[0])
	}
	if items[1].RelationshipStatus != dockerResourceRelationshipStatusUnused {
		t.Fatalf("expected unused volume to retain Docker usage status, got %#v", items[1])
	}
	if summary := summarizeDockerVolumes(items); summary.InUse != 1 || summary.Unused != 1 || summary.ReferenceUnknown != 0 {
		t.Fatalf("unexpected volume summary %#v", summary)
	}
}

func TestReadDockerVolumeDoesNotCalculateGlobalUsage(t *testing.T) {
	t.Parallel()

	refCount, size := int64(2), int64(4096)
	client := &volumeDetailDockerClient{volume: volume.Volume{
		Name:       "data",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/data/_data",
		UsageData:  &volume.UsageData{RefCount: refCount, Size: size},
	}}
	runtime := &DockerRuntime{client: client}

	result, err := runtime.ReadDockerVolume(context.Background(), "data")
	if err != nil {
		t.Fatalf("read volume: %v", err)
	}
	if client.volumeDiskUsageCall != 0 {
		t.Fatalf("expected detail read to avoid global usage calculation, got %d calls", client.volumeDiskUsageCall)
	}
	if result.ReferenceCount == nil || *result.ReferenceCount != refCount || result.SizeBytes == nil || *result.SizeBytes != size {
		t.Fatalf("expected inspect usage mapping to be preserved, got %#v", result)
	}
	if result.Mountpoint != "/var/lib/docker/volumes/data/_data" {
		t.Fatalf("expected inspect mountpoint mapping, got %q", result.Mountpoint)
	}
	if got := client.containerListOptions.Filters["volume"]["data"]; !got {
		t.Fatalf("expected container query to filter volume name, got %#v", client.containerListOptions)
	}
}

func TestReadDockerVolumeReturnsUnknownReferencesWhenContainerListFails(t *testing.T) {
	t.Parallel()

	refCount, size := int64(2), int64(4096)
	client := &volumeDetailDockerClient{
		volume: volume.Volume{
			Name:      "data",
			UsageData: &volume.UsageData{RefCount: refCount, Size: size},
		},
		containerListErr: assertError("container list unavailable"),
	}
	runtime := &DockerRuntime{client: client}

	result, err := runtime.ReadDockerVolume(context.Background(), "data")
	if err != nil {
		t.Fatalf("read volume: %v", err)
	}
	if result.Name != "data" || result.ReferenceCount == nil || *result.ReferenceCount != refCount || result.SizeBytes == nil || *result.SizeBytes != size {
		t.Fatalf("expected core volume projection to survive reference failure, got %#v", result)
	}
	if result.ContainerReferences != nil {
		t.Fatalf("expected references to remain unknown, got %#v", result.ContainerReferences)
	}
	if result.RelationshipStatus != dockerResourceRelationshipStatusUsed {
		t.Fatalf("expected known relationship status, got %#v", result.RelationshipStatus)
	}
}

func TestReadDockerVolumeKeepsKnownRefCountWhenContainerListDisagrees(t *testing.T) {
	t.Parallel()

	refCount := int64(1)
	client := &volumeDetailDockerClient{volume: volume.Volume{
		Name:      "data",
		UsageData: &volume.UsageData{RefCount: refCount, Size: 4096},
	}}
	runtime := &DockerRuntime{client: client}

	result, err := runtime.ReadDockerVolume(context.Background(), "data")
	if err != nil {
		t.Fatalf("read volume: %v", err)
	}
	if result.RelationshipStatus != dockerResourceRelationshipStatusUsed {
		t.Fatalf("expected known Docker refcount to preserve used status, got %#v", result)
	}
}
