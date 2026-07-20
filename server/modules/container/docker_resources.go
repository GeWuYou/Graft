package container

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/mount"
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
	VolumeDiskUsage(context.Context) ([]volume.Volume, error)
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

// DockerVolumeContainerReference 是引用数据卷的容器安全展示投影。
type DockerVolumeContainerReference struct {
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
	Name                string
	Driver              string
	Scope               string
	CreatedAt           string
	Labels              map[string]string
	ReferenceCount      *int64
	SizeBytes           *int64
	ContainerReferences []DockerVolumeContainerReference
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
	Summary              DockerVolumeListSummary
}

// DockerVolumeListSummary 描述完整 Docker runtime 数据卷快照的统计结果。
type DockerVolumeListSummary struct {
	Total            int
	InUse            int
	Unused           int
	ReferenceUnknown int
	SizeBytes        *int64
}

func listDockerVolumes(items []DockerVolume, query DockerVolumeListQuery) DockerVolumeListResult {
	query.Offset = max(query.Offset, 0)
	query.Limit = max(query.Limit, 0)
	filtered := make([]DockerVolume, 0, len(items))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, item := range items {
		if dockerVolumeMatchesQuery(item, query, keyword) {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	total := len(filtered)
	start := min(query.Offset, total)
	end := total
	if query.Limit <= total-start {
		end = start + query.Limit
	}
	return DockerVolumeListResult{Items: filtered[start:end], Total: total, Limit: query.Limit, Offset: query.Offset, Summary: summarizeDockerVolumes(items)}
}

func summarizeDockerVolumes(items []DockerVolume) DockerVolumeListSummary {
	summary := DockerVolumeListSummary{Total: len(items)}
	var totalSize int64
	sizeAvailable := true
	for _, item := range items {
		switch {
		case item.ReferenceCount == nil:
			summary.ReferenceUnknown++
		case *item.ReferenceCount > 0:
			summary.InUse++
		default:
			summary.Unused++
		}
		if item.SizeBytes == nil {
			sizeAvailable = false
			continue
		}
		totalSize += *item.SizeBytes
	}
	if sizeAvailable {
		summary.SizeBytes = &totalSize
	}
	return summary
}

func dockerVolumeMatchesQuery(item DockerVolume, query DockerVolumeListQuery, keyword string) bool {
	if keyword != "" && !strings.Contains(strings.ToLower(item.Name), keyword) {
		return false
	}
	if query.Driver != "" && item.Driver != query.Driver {
		return false
	}
	if query.Scope != "" && item.Scope != query.Scope {
		return false
	}
	return dockerVolumeUsageMatches(item.ReferenceCount, query.Usage)
}

func dockerVolumeUsageMatches(referenceCount *int64, usage string) bool {
	switch usage {
	case "used":
		return referenceCount != nil && *referenceCount > 0
	case "unused":
		return referenceCount != nil && *referenceCount == 0
	default:
		return true
	}
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
	readCtx, cancel := context.WithTimeout(ctx, dockerImageReadTimeout)
	defer cancel()
	result, err := client.VolumeList(readCtx, mobyclient.VolumeListOptions{})
	if err != nil {
		return nil, mapDockerError(err)
	}
	usage, usageErr := client.VolumeDiskUsage(readCtx)
	usageByName := make(map[string]volume.UsageData, len(usage))
	if usageErr == nil {
		for _, item := range usage {
			if item.UsageData != nil {
				usageByName[item.Name] = *item.UsageData
			}
		}
	}
	containers, err := client.ContainerList(readCtx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return nil, mapDockerError(err)
	}
	references := dockerVolumeReferences(containers)
	items := make([]DockerVolume, 0, len(result))
	for _, item := range result {
		projected := dockerVolume(item)
		if data, ok := usageByName[item.Name]; ok {
			projected.ReferenceCount, projected.SizeBytes = nullableUsage(data.RefCount), nullableUsage(data.Size)
		}
		projected.ContainerReferences = append([]DockerVolumeContainerReference(nil), references[projected.Name]...)
		items = append(items, projected)
	}
	return items, nil
}

func nullableUsage(value int64) *int64 {
	if value < 0 {
		return nil
	}
	return &value
}

// ReadDockerVolume 读取单个脱敏 Docker 数据卷；详情路径不触发全量 system/df，仅映射 inspect 已提供的用量数据。
func (r *DockerRuntime) ReadDockerVolume(ctx context.Context, id string) (DockerVolume, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerVolume{}, errUnsupportedContainerRuntime
	}
	readCtx, cancel := context.WithTimeout(ctx, dockerImageReadTimeout)
	defer cancel()
	item, err := client.VolumeInspect(readCtx, id)
	if err != nil {
		return DockerVolume{}, mapDockerError(err)
	}
	projected := dockerVolume(item)
	if item.UsageData != nil {
		projected.ReferenceCount, projected.SizeBytes = nullableUsage(item.UsageData.RefCount), nullableUsage(item.UsageData.Size)
	}
	containers, err := client.ContainerList(readCtx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return DockerVolume{}, mapDockerError(err)
	}
	projected.ContainerReferences = append([]DockerVolumeContainerReference(nil), dockerVolumeReferences(containers)[projected.Name]...)
	return projected, nil
}

func dockerVolumeReferences(items []container.Summary) map[string][]DockerVolumeContainerReference {
	byVolume := make(map[string]map[string]DockerVolumeContainerReference)
	for _, item := range items {
		reference, ok := dockerContainerReference(item)
		if !ok {
			continue
		}
		for _, itemMount := range item.Mounts {
			volumeName, ok := dockerVolumeMountName(itemMount)
			if !ok {
				continue
			}
			if byVolume[volumeName] == nil {
				byVolume[volumeName] = make(map[string]DockerVolumeContainerReference)
			}
			byVolume[volumeName][reference.ID] = reference
		}
	}
	result := make(map[string][]DockerVolumeContainerReference, len(byVolume))
	for volumeName, references := range byVolume {
		items := make([]DockerVolumeContainerReference, 0, len(references))
		for _, reference := range references {
			items = append(items, reference)
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].Name != items[j].Name {
				return items[i].Name < items[j].Name
			}
			return items[i].ID < items[j].ID
		})
		result[volumeName] = items
	}
	return result
}

func dockerContainerReference(item container.Summary) (DockerVolumeContainerReference, bool) {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return DockerVolumeContainerReference{}, false
	}
	name := ""
	if len(item.Names) > 0 {
		name = strings.TrimPrefix(strings.TrimSpace(item.Names[0]), "/")
	}
	return DockerVolumeContainerReference{ID: id, Name: name}, true
}

func dockerVolumeMountName(itemMount container.MountPoint) (string, bool) {
	if itemMount.Type != mount.TypeVolume {
		return "", false
	}
	name := strings.TrimSpace(itemMount.Name)
	return name, name != ""
}

// RemoveDockerVolume 删除指定 Docker 数据卷；运行时错误会先映射为模块级错误。
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
		referenceCount = nullableUsage(item.UsageData.RefCount)
		sizeBytes = nullableUsage(item.UsageData.Size)
	}
	return DockerVolume{Name: strings.TrimSpace(item.Name), Driver: strings.TrimSpace(item.Driver), Scope: strings.TrimSpace(item.Scope), CreatedAt: strings.TrimSpace(item.CreatedAt), Labels: cloneLabels(item.Labels), ReferenceCount: referenceCount, SizeBytes: sizeBytes}
}

var _ DockerResourceReader = (*DockerRuntime)(nil)
