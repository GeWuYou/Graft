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
const dockerVolumeUsageTimeout = 3 * time.Second

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

// DockerContainerReference 是 Docker 资源关系中容器的安全展示投影。
type DockerContainerReference struct {
	ID   string
	Name string
}

// DockerVolumeContainerReference 是引用数据卷的容器安全展示投影。
type DockerVolumeContainerReference = DockerContainerReference

// DockerNetworkContainerReference 是连接网络的容器安全展示投影。
type DockerNetworkContainerReference = DockerContainerReference

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
	ID                  string
	Name                string
	Driver              string
	Scope               string
	CreatedAt           string
	Internal            bool
	Attachable          bool
	Ingress             bool
	ContainerCount      int
	ContainerReferences []DockerNetworkContainerReference
	IPAM                *DockerNetworkIPAMDetail
	Removable           bool
	Labels              map[string]string
	Context             DockerResourceContext
	RelationshipStatus  DockerResourceRelationshipStatus
}

// DockerResourceSource 表示由服务端从可信运行时事实归一化的 Docker 资源业务来源。
type DockerResourceSource string

const (
	dockerResourceSourceCompose       DockerResourceSource = "compose"
	dockerResourceSourceDockerDefault DockerResourceSource = "docker_default"
	dockerResourceSourceDocker        DockerResourceSource = "docker"
)

// DockerResourceRelationshipStatus 表示当前关系快照的可用性和使用状态。
type DockerResourceRelationshipStatus string

const (
	dockerResourceRelationshipStatusUsed      DockerResourceRelationshipStatus = "used"
	dockerResourceRelationshipStatusUnused    DockerResourceRelationshipStatus = "unused"
	dockerResourceRelationshipStatusUnknown   DockerResourceRelationshipStatus = "unknown"
	dockerResourceRelationshipStatusException DockerResourceRelationshipStatus = "exception"
)

// DockerResourceContext 是面向资源页首屏的业务上下文，不承载原始 inspect 或 Labels 数据。
type DockerResourceContext struct {
	Runtime         string
	RuntimeTarget   string
	Source          DockerResourceSource
	ComposeProject  string
	ComposeResource string
	ManagedBy       string
}

// DockerNetworkIPAMDetail 是网络详情可展示的 IPAM 配置投影，不包含 driver options 或原始 inspect 数据。
type DockerNetworkIPAMDetail struct {
	Driver string
	Config []DockerNetworkIPAMDetailConfig
}

// DockerNetworkIPAMDetailConfig 是单条网络 IPAM 子网和网关配置。
type DockerNetworkIPAMDetailConfig struct {
	Subnet  string
	Gateway string
}

// DockerNetworkListQuery 描述 Docker 网络列表的筛选和分页条件。
type DockerNetworkListQuery struct {
	Limit, Offset                 int
	Keyword, Driver, Scope, Usage string
	Source                        DockerResourceSource
	ComposeProject                string
}

// DockerNetworkListResult 是 Docker 网络列表的分页投影，摘要始终基于完整快照。
type DockerNetworkListResult struct {
	Items                []DockerNetwork
	Total, Limit, Offset int
	Summary              DockerNetworkListSummary
}

// DockerNetworkListSummary 描述完整 Docker 网络快照的使用统计。
type DockerNetworkListSummary struct{ Total, InUse, Unused int }

func listDockerNetworks(items []DockerNetwork, query DockerNetworkListQuery) DockerNetworkListResult {
	query.Offset, query.Limit = max(query.Offset, 0), max(query.Limit, 0)
	filtered := make([]DockerNetwork, 0, len(items))
	for _, item := range items {
		if !dockerNetworkMatches(item, query) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	start := min(query.Offset, len(filtered))
	end := len(filtered)
	if query.Limit <= end-start {
		end = start + query.Limit
	}
	return DockerNetworkListResult{Items: filtered[start:end], Total: len(filtered), Limit: query.Limit, Offset: query.Offset, Summary: summarizeDockerNetworks(items)}
}

func dockerNetworkMatches(item DockerNetwork, query DockerNetworkListQuery) bool {
	return dockerResourceAttributesMatch(
		dockerResourceAttributes{Name: item.Name, Driver: item.Driver, Scope: item.Scope, Context: item.Context},
		dockerResourceAttributeFilter{Keyword: query.Keyword, Driver: query.Driver, Scope: query.Scope, Source: query.Source, ComposeProject: query.ComposeProject},
	) && dockerRelationshipUsageMatches(dockerNetworkRelationshipStatus(item), query.Usage)
}

func summarizeDockerNetworks(items []DockerNetwork) DockerNetworkListSummary {
	summary := DockerNetworkListSummary{Total: len(items)}
	for _, item := range items {
		switch dockerNetworkRelationshipStatus(item) {
		case dockerResourceRelationshipStatusUsed:
			summary.InUse++
		case dockerResourceRelationshipStatusUnused:
			summary.Unused++
		}
	}
	return summary
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
	Context             DockerResourceContext
	RelationshipStatus  DockerResourceRelationshipStatus
}

// DockerVolumeListQuery 描述数据卷列表的受限筛选和分页条件。
type DockerVolumeListQuery struct {
	Limit, Offset                 int
	Keyword, Driver, Scope, Usage string
	Source                        DockerResourceSource
	ComposeProject                string
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
		switch dockerVolumeRelationshipStatus(item) {
		case dockerResourceRelationshipStatusUnknown, dockerResourceRelationshipStatusException:
			summary.ReferenceUnknown++
		case dockerResourceRelationshipStatusUsed:
			summary.InUse++
		case dockerResourceRelationshipStatusUnused:
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
	return dockerResourceAttributesMatch(
		dockerResourceAttributes{Name: item.Name, Driver: item.Driver, Scope: item.Scope, Context: item.Context},
		dockerResourceAttributeFilter{Keyword: keyword, Driver: query.Driver, Scope: query.Scope, Source: query.Source, ComposeProject: query.ComposeProject},
	) && dockerRelationshipUsageMatches(dockerVolumeRelationshipStatus(item), query.Usage)
}

type dockerResourceAttributes struct {
	Name, Driver, Scope string
	Context             DockerResourceContext
}

type dockerResourceAttributeFilter struct {
	Keyword, Driver, Scope, ComposeProject string
	Source                                 DockerResourceSource
}

func dockerResourceAttributesMatch(attributes dockerResourceAttributes, filter dockerResourceAttributeFilter) bool {
	return dockerResourceNameMatches(attributes.Name, filter.Keyword) &&
		dockerResourceValueMatches(attributes.Driver, filter.Driver) &&
		dockerResourceValueMatches(attributes.Scope, filter.Scope) &&
		dockerResourceSourceMatches(attributes.Context.Source, filter.Source) &&
		dockerResourceComposeProjectMatches(attributes.Context.ComposeProject, filter.ComposeProject)
}

func dockerResourceNameMatches(name, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	return keyword == "" || strings.Contains(strings.ToLower(name), keyword)
}

func dockerResourceValueMatches(value, filter string) bool {
	return filter == "" || value == filter
}

func dockerResourceSourceMatches(source, filter DockerResourceSource) bool {
	return filter == "" || source == filter
}

func dockerResourceComposeProjectMatches(project, filter string) bool {
	filter = strings.TrimSpace(filter)
	return filter == "" || strings.EqualFold(project, filter)
}

func dockerRelationshipUsageMatches(status DockerResourceRelationshipStatus, usage string) bool {
	switch usage {
	case "used":
		return status == dockerResourceRelationshipStatusUsed
	case "unused":
		return status == dockerResourceRelationshipStatusUnused
	default:
		return true
	}
}

func dockerRelationshipStatus(known bool, count int) DockerResourceRelationshipStatus {
	if !known {
		return dockerResourceRelationshipStatusException
	}
	if count > 0 {
		return dockerResourceRelationshipStatusUsed
	}
	return dockerResourceRelationshipStatusUnused
}

func dockerRelationshipStatusFromReferenceCount(referenceCount *int64) DockerResourceRelationshipStatus {
	if referenceCount == nil {
		return dockerResourceRelationshipStatusUnknown
	}
	return dockerRelationshipStatus(true, int(*referenceCount))
}

func dockerNetworkRelationshipStatus(item DockerNetwork) DockerResourceRelationshipStatus {
	if item.RelationshipStatus != "" {
		return item.RelationshipStatus
	}
	return dockerRelationshipStatus(true, item.ContainerCount)
}

func dockerVolumeRelationshipStatus(item DockerVolume) DockerResourceRelationshipStatus {
	if item.RelationshipStatus != "" {
		return item.RelationshipStatus
	}
	return dockerRelationshipStatusFromReferenceCount(item.ReferenceCount)
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
	containers, containersErr := client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true})
	references := dockerNetworkContainerReferences(containers)
	counts := dockerNetworkContainerReferenceCounts(references)
	for _, item := range items {
		projected := dockerNetwork(item.Network, counts[strings.TrimSpace(item.ID)])
		if containersErr == nil {
			projected.ContainerReferences = append([]DockerNetworkContainerReference(nil), references[strings.TrimSpace(item.ID)]...)
		}
		projected.RelationshipStatus = dockerRelationshipStatus(containersErr == nil, projected.ContainerCount)
		projected.Removable = !isDockerDefaultNetwork(projected.Name) && projected.RelationshipStatus == dockerResourceRelationshipStatusUnused
		result = append(result, projected)
	}
	return result, nil
}

func dockerNetworkContainerReferenceCounts(references map[string][]DockerNetworkContainerReference) map[string]int {
	counts := make(map[string]int, len(references))
	for networkID, items := range references {
		counts[networkID] = len(items)
	}
	return counts
}

func dockerNetworkContainerReferences(items []container.Summary) map[string][]DockerNetworkContainerReference {
	byNetwork := make(map[string]map[string]DockerNetworkContainerReference)
	for _, item := range items {
		reference, ok := dockerContainerReference(item)
		if !ok {
			continue
		}
		if item.NetworkSettings == nil {
			continue
		}
		for id, endpoint := range item.NetworkSettings.Networks {
			addDockerNetworkReference(byNetwork, dockerNetworkReferenceID(id, endpoint), reference)
		}
	}
	result := make(map[string][]DockerNetworkContainerReference, len(byNetwork))
	for networkID, references := range byNetwork {
		result[networkID] = sortedDockerContainerReferences(references)
	}
	return result
}

func dockerNetworkReferenceID(id string, endpoint *network.EndpointSettings) string {
	networkID := strings.TrimSpace(id)
	if endpoint != nil && strings.TrimSpace(endpoint.NetworkID) != "" {
		networkID = strings.TrimSpace(endpoint.NetworkID)
	}
	return networkID
}

func addDockerNetworkReference(byNetwork map[string]map[string]DockerNetworkContainerReference, networkID string, reference DockerNetworkContainerReference) {
	if networkID == "" {
		return
	}
	if byNetwork[networkID] == nil {
		byNetwork[networkID] = make(map[string]DockerNetworkContainerReference)
	}
	byNetwork[networkID][reference.ID] = reference
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
	projected := dockerNetwork(item.Network, len(item.Containers))
	projected.ContainerReferences = dockerNetworkInspectReferences(item.Containers)
	return projected, nil
}

func dockerNetworkInspectReferences(containers map[string]network.EndpointResource) []DockerNetworkContainerReference {
	references := make(map[string]DockerNetworkContainerReference, len(containers))
	for id, endpoint := range containers {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		references[id] = DockerNetworkContainerReference{ID: id, Name: strings.TrimPrefix(strings.TrimSpace(endpoint.Name), "/")}
	}
	return sortedDockerContainerReferences(references)
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
	usageCtx, usageCancel := context.WithTimeout(ctx, dockerVolumeUsageTimeout)
	usage, usageErr := client.VolumeDiskUsage(usageCtx)
	usageCancel()
	usageByName := make(map[string]volume.UsageData, len(usage))
	if usageErr == nil {
		for _, item := range usage {
			if item.UsageData != nil {
				usageByName[item.Name] = *item.UsageData
			}
		}
	}
	containers, containersErr := client.ContainerList(readCtx, mobyclient.ContainerListOptions{All: true})
	references := dockerVolumeReferences(containers)
	items := make([]DockerVolume, 0, len(result))
	for _, item := range result {
		projected := dockerVolume(item)
		if data, ok := usageByName[item.Name]; ok {
			projected.ReferenceCount, projected.SizeBytes = nullableUsage(data.RefCount), nullableUsage(data.Size)
		}
		applyDockerVolumeContainerReferences(&projected, references[projected.Name], containersErr)
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
	containers, err := client.ContainerList(readCtx, mobyclient.ContainerListOptions{
		All:     true,
		Filters: make(mobyclient.Filters).Add("volume", projected.Name),
	})
	applyDockerVolumeContainerReferences(&projected, dockerVolumeReferences(containers)[projected.Name], err)
	// 引用查询是详情的附加信息；Docker 查询失败时保留核心 volume 投影，并以 nil 表示引用未知。
	return projected, nil
}

func applyDockerVolumeContainerReferences(projected *DockerVolume, references []DockerVolumeContainerReference, err error) {
	if err != nil {
		if projected.ReferenceCount == nil {
			projected.RelationshipStatus = dockerResourceRelationshipStatusException
		}
		return
	}
	projected.ContainerReferences = append([]DockerVolumeContainerReference(nil), references...)
	if projected.ReferenceCount == nil {
		projected.RelationshipStatus = dockerRelationshipStatus(true, len(projected.ContainerReferences))
	}
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
		result[volumeName] = sortedDockerContainerReferences(references)
	}
	return result
}

func sortedDockerContainerReferences(references map[string]DockerContainerReference) []DockerContainerReference {
	items := make([]DockerContainerReference, 0, len(references))
	for _, reference := range references {
		items = append(items, reference)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func dockerContainerReference(item container.Summary) (DockerContainerReference, bool) {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return DockerContainerReference{}, false
	}
	name := ""
	if len(item.Names) > 0 {
		name = strings.TrimPrefix(strings.TrimSpace(item.Names[0]), "/")
	}
	return DockerContainerReference{ID: id, Name: name}, true
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
	name := strings.TrimSpace(item.Name)
	status := dockerRelationshipStatus(true, containerCount)
	return DockerNetwork{ID: strings.TrimSpace(item.ID), Name: name, Driver: strings.TrimSpace(item.Driver), Scope: strings.TrimSpace(item.Scope), CreatedAt: item.Created.UTC().Format(time.RFC3339), Internal: item.Internal, Attachable: item.Attachable, Ingress: item.Ingress, ContainerCount: containerCount, Removable: !isDockerDefaultNetwork(name) && status == dockerResourceRelationshipStatusUnused, Labels: cloneLabels(item.Labels), Context: dockerResourceContext(item.Labels, name, composeNetworkLabel, isDockerDefaultNetwork(name)), RelationshipStatus: status, IPAM: dockerNetworkDetailIPAM(item.IPAM)}
}

func dockerNetworkDetailIPAM(item network.IPAM) *DockerNetworkIPAMDetail {
	config := make([]DockerNetworkIPAMDetailConfig, 0, len(item.Config))
	for _, value := range item.Config {
		subnet, gateway := "", ""
		if value.Subnet.IsValid() {
			subnet = value.Subnet.String()
		}
		if value.Gateway.IsValid() {
			gateway = value.Gateway.String()
		}
		if subnet == "" && gateway == "" {
			continue
		}
		config = append(config, DockerNetworkIPAMDetailConfig{Subnet: subnet, Gateway: gateway})
	}
	if strings.TrimSpace(item.Driver) == "" && len(config) == 0 {
		return nil
	}
	return &DockerNetworkIPAMDetail{Driver: strings.TrimSpace(item.Driver), Config: config}
}

// dockerVolume converts a Docker volume into a normalized DockerVolume projection, preserving usage metrics when available.
func dockerVolume(item volume.Volume) DockerVolume {
	var referenceCount, sizeBytes *int64
	if item.UsageData != nil {
		referenceCount = nullableUsage(item.UsageData.RefCount)
		sizeBytes = nullableUsage(item.UsageData.Size)
	}
	name := strings.TrimSpace(item.Name)
	return DockerVolume{Name: name, Driver: strings.TrimSpace(item.Driver), Scope: strings.TrimSpace(item.Scope), CreatedAt: strings.TrimSpace(item.CreatedAt), Labels: cloneLabels(item.Labels), ReferenceCount: referenceCount, SizeBytes: sizeBytes, Context: dockerResourceContext(item.Labels, name, composeVolumeLabel, false), RelationshipStatus: dockerRelationshipStatusFromReferenceCount(referenceCount)}
}

func dockerResourceContext(labels map[string]string, resourceName, composeResourceLabel string, defaultDockerResource bool) DockerResourceContext {
	resourceContext := DockerResourceContext{Runtime: runtimeNameDocker, Source: dockerResourceSourceDocker}
	if defaultDockerResource {
		resourceContext.Source = dockerResourceSourceDockerDefault
		return resourceContext
	}
	project := strings.TrimSpace(labels[composeProjectLabel])
	if project == "" {
		return resourceContext
	}
	resourceContext.Source = dockerResourceSourceCompose
	resourceContext.ComposeProject = project
	resourceContext.ComposeResource = firstNonEmpty(strings.TrimSpace(labels[composeResourceLabel]), resourceName)
	return resourceContext
}

var _ DockerResourceReader = (*DockerRuntime)(nil)
