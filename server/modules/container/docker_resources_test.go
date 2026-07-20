package container

import (
	"context"
	"testing"

	dockertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	mobyclient "github.com/moby/moby/client"
)

func TestListDockerVolumesFiltersSortsAndPages(t *testing.T) {
	t.Parallel()

	used := int64(2)
	items := []DockerVolume{
		{Name: "zeta", Driver: "local", Scope: "local", ReferenceCount: &used},
		{Name: "alpha", Driver: "local", Scope: "local", ReferenceCount: &used},
		{Name: "beta", Driver: "nfs", Scope: "global"},
	}
	result := listDockerVolumes(items, DockerVolumeListQuery{Driver: "local", Usage: "used", Limit: 1, Offset: 1})

	if result.Total != 2 || result.Limit != 1 || result.Offset != 1 {
		t.Fatalf("unexpected page metadata: %#v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "zeta" {
		t.Fatalf("expected sorted second matching volume, got %#v", result.Items)
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

func TestDockerVolumeUsageUnknownDoesNotMatchUnused(t *testing.T) {
	if dockerVolumeUsageMatches(nil, "unused") {
		t.Fatal("unknown reference count must not match unused")
	}
	zero := int64(0)
	if !dockerVolumeUsageMatches(&zero, "unused") {
		t.Fatal("zero reference count must match unused")
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

func TestListDockerVolumesSummaryUsesFullSnapshotBeforeFilters(t *testing.T) {
	t.Parallel()

	used, unused, size := int64(1), int64(0), int64(1024)
	result := listDockerVolumes([]DockerVolume{
		{Name: "used", ReferenceCount: &used, SizeBytes: &size},
		{Name: "unused", ReferenceCount: &unused, SizeBytes: &size},
		{Name: "unknown", SizeBytes: nil},
	}, DockerVolumeListQuery{Usage: "used", Limit: 20})

	if result.Total != 1 || result.Summary.Total != 3 || result.Summary.InUse != 1 || result.Summary.Unused != 1 || result.Summary.ReferenceUnknown != 1 {
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
	volume              volume.Volume
	volumeDiskUsageCall int
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
	return nil, nil
}

func (c *volumeDetailDockerClient) VolumeDiskUsage(context.Context) ([]volume.Volume, error) {
	c.volumeDiskUsageCall++
	return nil, nil
}

func (c *volumeDetailDockerClient) VolumeInspect(context.Context, string) (volume.Volume, error) {
	return c.volume, nil
}

func (c *volumeDetailDockerClient) VolumeRemove(context.Context, string, bool) error {
	return nil
}

func TestReadDockerVolumeDoesNotCalculateGlobalUsage(t *testing.T) {
	t.Parallel()

	refCount, size := int64(2), int64(4096)
	client := &volumeDetailDockerClient{volume: volume.Volume{
		Name:      "data",
		Driver:    "local",
		UsageData: &volume.UsageData{RefCount: refCount, Size: size},
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
}
