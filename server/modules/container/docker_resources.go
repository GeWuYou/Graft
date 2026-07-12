package container

import (
	"context"
	"strings"
	"time"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	mobyclient "github.com/moby/moby/client"
)

// dockerResourceClient is intentionally separate from dockerClient. It keeps
// existing container-only runtime test doubles independent of read-only
// Docker resource discovery.
type dockerResourceClient interface {
	ImageList(context.Context, mobyclient.ImageListOptions) ([]image.Summary, error)
	ImageInspect(context.Context, string) (image.InspectResponse, error)
	NetworkList(context.Context, mobyclient.NetworkListOptions) ([]network.Summary, error)
	NetworkInspect(context.Context, string, mobyclient.NetworkInspectOptions) (network.Inspect, error)
	VolumeList(context.Context, mobyclient.VolumeListOptions) ([]volume.Volume, error)
	VolumeInspect(context.Context, string) (volume.Volume, error)
}

// DockerImage is the sanitized image projection shared by list and detail reads.
type DockerImage struct {
	ID                string
	RepositoryTags    []string
	RepositoryDigests []string
	CreatedAt         string
	SizeBytes         int64
	Containers        int64
	Labels            map[string]string
	Architecture      string
	OperatingSystem   string
}

// DockerNetwork is the sanitized network projection shared by list and detail reads.
type DockerNetwork struct {
	ID             string
	Name           string
	Driver         string
	Scope          string
	CreatedAt      string
	Internal       bool
	Attachable     bool
	Ingress        bool
	ContainerCount int
	Labels         map[string]string
}

// DockerVolume is the sanitized volume projection shared by list and detail reads.
// Host mount paths and driver-specific status are intentionally omitted.
type DockerVolume struct {
	Name           string
	Driver         string
	Scope          string
	CreatedAt      string
	Labels         map[string]string
	ReferenceCount *int64
	SizeBytes      *int64
}

// DockerResourceReader marks a runtime that can list Docker-native resources.
type DockerResourceReader interface {
	ListDockerImages(context.Context) ([]DockerImage, error)
	ReadDockerImage(context.Context, string) (DockerImage, error)
	ListDockerNetworks(context.Context) ([]DockerNetwork, error)
	ReadDockerNetwork(context.Context, string) (DockerNetwork, error)
	ListDockerVolumes(context.Context) ([]DockerVolume, error)
	ReadDockerVolume(context.Context, string) (DockerVolume, error)
}

// ListDockerImages returns sanitized Docker images from the configured runtime.
func (r *DockerRuntime) ListDockerImages(ctx context.Context) ([]DockerImage, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return nil, errUnsupportedContainerRuntime
	}
	items, err := client.ImageList(ctx, mobyclient.ImageListOptions{All: true})
	if err != nil {
		return nil, mapDockerError(err)
	}
	result := make([]DockerImage, 0, len(items))
	for _, item := range items {
		result = append(result, dockerImageSummary(item))
	}
	return result, nil
}

// ReadDockerImage returns one sanitized Docker image by ID.
func (r *DockerRuntime) ReadDockerImage(ctx context.Context, id string) (DockerImage, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerImage{}, errUnsupportedContainerRuntime
	}
	item, err := client.ImageInspect(ctx, id)
	if err != nil {
		return DockerImage{}, mapDockerError(err)
	}
	return DockerImage{ID: strings.TrimSpace(item.ID), RepositoryTags: append([]string(nil), item.RepoTags...), RepositoryDigests: append([]string(nil), item.RepoDigests...), CreatedAt: strings.TrimSpace(item.Created), SizeBytes: item.Size, Labels: cloneLabels(imageLabels(item)), Architecture: strings.TrimSpace(item.Architecture), OperatingSystem: strings.TrimSpace(item.Os)}, nil
}

// ListDockerNetworks returns sanitized Docker networks from the configured runtime.
func (r *DockerRuntime) ListDockerNetworks(ctx context.Context) ([]DockerNetwork, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return nil, errUnsupportedContainerRuntime
	}
	items, err := client.NetworkList(ctx, mobyclient.NetworkListOptions{})
	if err != nil {
		return nil, mapDockerError(err)
	}
	result := make([]DockerNetwork, 0, len(items))
	for _, item := range items {
		result = append(result, dockerNetwork(item.Network, 0))
	}
	return result, nil
}

// ReadDockerNetwork returns one sanitized Docker network by ID.
func (r *DockerRuntime) ReadDockerNetwork(ctx context.Context, id string) (DockerNetwork, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerNetwork{}, errUnsupportedContainerRuntime
	}
	item, err := client.NetworkInspect(ctx, id, mobyclient.NetworkInspectOptions{})
	if err != nil {
		return DockerNetwork{}, mapDockerError(err)
	}
	return dockerNetwork(item.Network, len(item.Containers)), nil
}

// ListDockerVolumes returns sanitized Docker volumes from the configured runtime.
func (r *DockerRuntime) ListDockerVolumes(ctx context.Context) ([]DockerVolume, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return nil, errUnsupportedContainerRuntime
	}
	result, err := client.VolumeList(ctx, mobyclient.VolumeListOptions{})
	if err != nil {
		return nil, mapDockerError(err)
	}
	items := make([]DockerVolume, 0, len(result))
	for _, item := range result {
		items = append(items, dockerVolume(item))
	}
	return items, nil
}

// ReadDockerVolume returns one sanitized Docker volume by ID.
func (r *DockerRuntime) ReadDockerVolume(ctx context.Context, id string) (DockerVolume, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerVolume{}, errUnsupportedContainerRuntime
	}
	item, err := client.VolumeInspect(ctx, id)
	if err != nil {
		return DockerVolume{}, mapDockerError(err)
	}
	return dockerVolume(item), nil
}

func dockerImageSummary(item image.Summary) DockerImage {
	return DockerImage{ID: strings.TrimSpace(item.ID), RepositoryTags: append([]string(nil), item.RepoTags...), RepositoryDigests: append([]string(nil), item.RepoDigests...), CreatedAt: time.Unix(item.Created, 0).UTC().Format(time.RFC3339), SizeBytes: item.Size, Containers: item.Containers, Labels: cloneLabels(item.Labels)}
}

func imageLabels(item image.InspectResponse) map[string]string {
	if item.Config == nil {
		return nil
	}
	return item.Config.Labels
}

func dockerNetwork(item network.Network, containerCount int) DockerNetwork {
	return DockerNetwork{ID: strings.TrimSpace(item.ID), Name: strings.TrimSpace(item.Name), Driver: strings.TrimSpace(item.Driver), Scope: strings.TrimSpace(item.Scope), CreatedAt: item.Created.UTC().Format(time.RFC3339), Internal: item.Internal, Attachable: item.Attachable, Ingress: item.Ingress, ContainerCount: containerCount, Labels: cloneLabels(item.Labels)}
}

func dockerVolume(item volume.Volume) DockerVolume {
	var referenceCount, sizeBytes *int64
	if item.UsageData != nil {
		referenceCount = dockerInt64Ptr(item.UsageData.RefCount)
		sizeBytes = dockerInt64Ptr(item.UsageData.Size)
	}
	return DockerVolume{Name: strings.TrimSpace(item.Name), Driver: strings.TrimSpace(item.Driver), Scope: strings.TrimSpace(item.Scope), CreatedAt: strings.TrimSpace(item.CreatedAt), Labels: cloneLabels(item.Labels), ReferenceCount: referenceCount, SizeBytes: sizeBytes}
}

func dockerInt64Ptr(value int64) *int64 { return &value }

var _ DockerResourceReader = (*DockerRuntime)(nil)
