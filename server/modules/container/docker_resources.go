package container

import (
	"cmp"
	"context"
	"errors"
	"slices"
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
	VolumeRemove(context.Context, string, bool) error
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

// DockerVolumeListQuery 描述数据卷列表的受限筛选和分页条件。
type DockerVolumeListQuery struct {
	Limit, Offset                 int
	Keyword, Driver, Scope, Usage string
}

// DockerVolumeListResult 是数据卷列表的分页投影。
type DockerVolumeListResult struct {
	Items                []DockerVolume
	Total, Limit, Offset int
}

// DockerResourceReader marks a runtime that can list Docker-native resources.
type DockerResourceReader interface {
	ListDockerImages(context.Context) ([]DockerImage, error)
	ReadDockerImage(context.Context, string) (DockerImage, error)
	ListDockerNetworks(context.Context) ([]DockerNetwork, error)
	ReadDockerNetwork(context.Context, string) (DockerNetwork, error)
	ListDockerVolumes(context.Context) ([]DockerVolume, error)
	ReadDockerVolume(context.Context, string) (DockerVolume, error)
	RemoveDockerVolume(context.Context, string, bool) error
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
		return DockerVolume{}, mapDockerVolumeError(err)
	}
	return dockerVolume(item), nil
}

// RemoveDockerVolume 将删除请求委托给 Docker，并避免暴露驱动私有信息。
func (r *DockerRuntime) RemoveDockerVolume(ctx context.Context, id string, force bool) error {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return errUnsupportedContainerRuntime
	}
	if err := client.VolumeRemove(ctx, id, force); err != nil {
		return mapDockerVolumeError(err)
	}
	return nil
}

func mapDockerVolumeError(err error) error {
	mapped := mapDockerError(err)
	if errors.Is(mapped, errContainerNotFound) {
		return errDockerVolumeNotFound
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "volume is in use") || strings.Contains(message, "volume is being used") || strings.Contains(message, "remove a volume that is in use") {
		return errDockerVolumeConflict
	}
	return mapped
}

// listDockerVolumes 按受限条件筛选净化后的数据卷，并以名称稳定排序后分页。
func listDockerVolumes(items []DockerVolume, query DockerVolumeListQuery) DockerVolumeListResult {
	filtered := make([]DockerVolume, 0, len(items))
	for _, item := range items {
		if !matchesDockerVolume(item, query) {
			continue
		}
		filtered = append(filtered, item)
	}
	slices.SortFunc(filtered, func(a, b DockerVolume) int { return cmp.Compare(a.Name, b.Name) })
	total := len(filtered)
	start := min(query.Offset, total)
	end := min(start+query.Limit, total)
	return DockerVolumeListResult{Items: filtered[start:end], Total: total, Limit: query.Limit, Offset: query.Offset}
}

func matchesDockerVolume(item DockerVolume, query DockerVolumeListQuery) bool {
	if query.Keyword != "" && !strings.Contains(strings.ToLower(item.Name), strings.ToLower(query.Keyword)) {
		return false
	}
	if query.Driver != "" && item.Driver != query.Driver {
		return false
	}
	if query.Scope != "" && item.Scope != query.Scope {
		return false
	}
	return matchesDockerVolumeUsage(item.ReferenceCount, query.Usage)
}

func matchesDockerVolumeUsage(referenceCount *int64, usage string) bool {
	switch usage {
	case "used":
		return referenceCount != nil && *referenceCount > 0
	case "unused":
		return referenceCount != nil && *referenceCount == 0
	default:
		return true
	}
}

// dockerImageSummary converts a Docker image summary into a sanitized DockerImage projection.
func dockerImageSummary(item image.Summary) DockerImage {
	return DockerImage{ID: strings.TrimSpace(item.ID), RepositoryTags: append([]string(nil), item.RepoTags...), RepositoryDigests: append([]string(nil), item.RepoDigests...), CreatedAt: time.Unix(item.Created, 0).UTC().Format(time.RFC3339), SizeBytes: item.Size, Containers: item.Containers, Labels: cloneLabels(item.Labels)}
}

// imageLabels 返回镜像检查结果中的标签；配置为空时返回 nil。
func imageLabels(item image.InspectResponse) map[string]string {
	if item.Config == nil {
		return nil
	}
	return item.Config.Labels
}

// dockerNetwork converts Docker network data into a normalized DockerNetwork value.
func dockerNetwork(item network.Network, containerCount int) DockerNetwork {
	return DockerNetwork{ID: strings.TrimSpace(item.ID), Name: strings.TrimSpace(item.Name), Driver: strings.TrimSpace(item.Driver), Scope: strings.TrimSpace(item.Scope), CreatedAt: item.Created.UTC().Format(time.RFC3339), Internal: item.Internal, Attachable: item.Attachable, Ingress: item.Ingress, ContainerCount: containerCount, Labels: cloneLabels(item.Labels)}
}

// dockerVolume converts a Docker volume into a normalized DockerVolume projection, preserving usage metrics when available.
func dockerVolume(item volume.Volume) DockerVolume {
	var referenceCount, sizeBytes *int64
	if item.UsageData != nil {
		referenceCount = dockerInt64Ptr(item.UsageData.RefCount)
		sizeBytes = dockerInt64Ptr(item.UsageData.Size)
	}
	return DockerVolume{Name: strings.TrimSpace(item.Name), Driver: strings.TrimSpace(item.Driver), Scope: strings.TrimSpace(item.Scope), CreatedAt: strings.TrimSpace(item.CreatedAt), Labels: cloneLabels(item.Labels), ReferenceCount: referenceCount, SizeBytes: sizeBytes}
}

// dockerInt64Ptr returns a pointer to the provided integer value.
func dockerInt64Ptr(value int64) *int64 { return &value }

var _ DockerResourceReader = (*DockerRuntime)(nil)
