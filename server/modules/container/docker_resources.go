package container

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	mobyclient "github.com/moby/moby/client"
)

const dockerImageReadTimeout = 10 * time.Second

// dockerResourceClient 与容器操作 runtime 接口分离，使只读 Docker 资源聚合可以独立替换和测试。
type dockerResourceClient interface {
	ImageList(context.Context, mobyclient.ImageListOptions) ([]image.Summary, error)
	ImageInspect(context.Context, string) (image.InspectResponse, error)
	ContainerList(context.Context, mobyclient.ContainerListOptions) ([]container.Summary, error)
	NetworkList(context.Context, mobyclient.NetworkListOptions) ([]network.Summary, error)
	NetworkInspect(context.Context, string, mobyclient.NetworkInspectOptions) (network.Inspect, error)
	VolumeList(context.Context, mobyclient.VolumeListOptions) ([]volume.Volume, error)
	VolumeInspect(context.Context, string) (volume.Volume, error)
	VolumeRemove(context.Context, string, bool) error
}

// DockerImage 是列表和详情读取共用的脱敏镜像投影，引用容器来自同一 runtime 快照。
type DockerImage struct {
	ID                  string
	RepositoryTags      []string
	RepositoryDigests   []string
	CreatedAt           string
	SizeBytes           int64
	Containers          int64
	ContainerReferences []DockerImageContainerReference
	Dangling            bool
	Labels              map[string]string
	Architecture        string
	OperatingSystem     string
}

// DockerImageContainerReference 是引用镜像的容器安全展示投影。
type DockerImageContainerReference struct {
	ID   string
	Name string
}

// DockerImageListResult 保存一次 Docker 镜像快照及其完整运行时统计。
// Items 由同一次 ImageList 调用产生，调用方可以在此快照上完成过滤和分页，避免重复访问 Docker daemon。
type DockerImageListResult struct {
	Items   []DockerImage
	Total   int
	Summary DockerImageListSummary
}

// DockerImageListSummary 描述完整 Docker runtime inventory，不受列表关键字过滤影响。
type DockerImageListSummary struct {
	Total     int
	SizeBytes int64
	InUse     int
	Dangling  int
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

func listDockerVolumes(items []DockerVolume, query DockerVolumeListQuery) DockerVolumeListResult {
	filtered := make([]DockerVolume, 0, len(items))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, item := range items {
		if keyword != "" && !strings.Contains(strings.ToLower(item.Name), keyword) {
			continue
		}
		if query.Driver != "" && item.Driver != query.Driver {
			continue
		}
		if query.Scope != "" && item.Scope != query.Scope {
			continue
		}
		if query.Usage == "used" && (item.ReferenceCount == nil || *item.ReferenceCount == 0) {
			continue
		}
		if query.Usage == "unused" && item.ReferenceCount != nil && *item.ReferenceCount > 0 {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	total := len(filtered)
	start := min(query.Offset, total)
	end := min(start+query.Limit, total)
	return DockerVolumeListResult{Items: filtered[start:end], Total: total, Limit: query.Limit, Offset: query.Offset}
}

// DockerResourceReader marks a runtime that can list Docker-native resources.
type DockerResourceReader interface {
	ListDockerImages(context.Context) (DockerImageListResult, error)
	ReadDockerImage(context.Context, string) (DockerImage, error)
	ListDockerNetworks(context.Context) ([]DockerNetwork, error)
	ReadDockerNetwork(context.Context, string) (DockerNetwork, error)
	ListDockerVolumes(context.Context) ([]DockerVolume, error)
	ReadDockerVolume(context.Context, string) (DockerVolume, error)
}

// ListDockerImages 从 Docker runtime 读取一次完整镜像快照，并在返回前计算库存摘要和稳定排序。
func (r *DockerRuntime) ListDockerImages(ctx context.Context) (DockerImageListResult, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerImageListResult{}, errUnsupportedContainerRuntime
	}
	readCtx, cancel := context.WithTimeout(ctx, dockerImageReadTimeout)
	defer cancel()
	items, err := client.ImageList(readCtx, mobyclient.ImageListOptions{All: true})
	if err != nil {
		return DockerImageListResult{}, mapDockerError(err)
	}
	containers, err := client.ContainerList(readCtx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return DockerImageListResult{}, mapDockerError(err)
	}
	references := dockerImageReferences(containers)
	result := make([]DockerImage, 0, len(items))
	var summary DockerImageListSummary
	for _, item := range items {
		image := dockerImageSummary(item)
		image.ContainerReferences = append([]DockerImageContainerReference(nil), references[image.ID]...)
		image.Containers = int64(len(image.ContainerReferences))
		image.Dangling = dockerImageIsDangling(image.RepositoryTags)
		result = append(result, image)
		summary.Total++
		summary.SizeBytes += maxInt64(item.Size, 0)
		if len(image.ContainerReferences) > 0 {
			summary.InUse++
		}
		if image.Dangling {
			summary.Dangling++
		}
	}
	sortDockerImages(result)
	return DockerImageListResult{Items: result, Summary: summary}, nil
}

func sortDockerImages(items []DockerImage) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := imageCreatedAt(items[i]), imageCreatedAt(items[j])
		if !left.Equal(right) {
			return left.After(right)
		}
		return items[i].ID < items[j].ID
	})
}

func imageCreatedAt(item DockerImage) time.Time {
	created, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return time.Time{}
	}
	return created
}

func dockerImageIsDangling(tags []string) bool {
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" && tag != "<none>:<none>" {
			return false
		}
	}
	return true
}

func maxInt64(value, minimum int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}

// ReadDockerImage 按镜像 ID 读取脱敏详情，并补充当前引用容器标签。
func (r *DockerRuntime) ReadDockerImage(ctx context.Context, id string) (DockerImage, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerImage{}, errUnsupportedContainerRuntime
	}
	readCtx, cancel := context.WithTimeout(ctx, dockerImageReadTimeout)
	defer cancel()
	item, err := client.ImageInspect(readCtx, id)
	if err != nil {
		return DockerImage{}, mapDockerError(err)
	}
	imageID := strings.TrimSpace(item.ID)
	containers, err := client.ContainerList(readCtx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return DockerImage{}, mapDockerError(err)
	}
	result := DockerImage{ID: imageID, RepositoryTags: append([]string(nil), item.RepoTags...), RepositoryDigests: append([]string(nil), item.RepoDigests...), CreatedAt: strings.TrimSpace(item.Created), SizeBytes: item.Size, Labels: cloneLabels(imageLabels(item)), Architecture: strings.TrimSpace(item.Architecture), OperatingSystem: strings.TrimSpace(item.Os), Dangling: dockerImageIsDangling(item.RepoTags)}
	result.ContainerReferences = dockerImageReferences(containers)[imageID]
	result.Containers = int64(len(result.ContainerReferences))
	return result, nil
}

func dockerImageReferences(items []container.Summary) map[string][]DockerImageContainerReference {
	result := make(map[string][]DockerImageContainerReference)
	for _, item := range items {
		imageID := strings.TrimSpace(item.ImageID)
		if imageID == "" {
			continue
		}
		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(strings.TrimSpace(item.Names[0]), "/")
		}
		result[imageID] = append(result[imageID], DockerImageContainerReference{ID: strings.TrimSpace(item.ID), Name: name})
	}
	return result
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
	if strings.Contains(strings.ToLower(err.Error()), "in use") {
		return errDockerVolumeConflict
	}
	return mapped
}

// dockerImageSummary 将 Docker 镜像概要转换为脱敏的 DockerImage 基础投影。
func dockerImageSummary(item image.Summary) DockerImage {
	return DockerImage{ID: strings.TrimSpace(item.ID), RepositoryTags: append([]string(nil), item.RepoTags...), RepositoryDigests: append([]string(nil), item.RepoDigests...), CreatedAt: time.Unix(item.Created, 0).UTC().Format(time.RFC3339), SizeBytes: item.Size, Containers: item.Containers, Dangling: dockerImageIsDangling(item.RepoTags), Labels: cloneLabels(item.Labels)}
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
