package container

import (
	"context"
	"errors"
	"net/netip"
	"sort"
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
	NetworkCreate(context.Context, string, mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error)
	NetworkRemove(context.Context, string) error
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

// DockerNetworkDetail 是网络检查的脱敏投影，包含 IPAM 和已连接容器端点。
type DockerNetworkDetail struct {
	DockerNetwork
	IPAM       DockerNetworkIPAM
	Containers []DockerNetworkContainerEndpoint
}

// DockerNetworkIPAM 描述 Docker 网络的 IP 地址分配配置。
type DockerNetworkIPAM struct {
	Driver string
	Config []DockerNetworkIPAMConfig
}

// DockerNetworkIPAMConfig 描述一个 IPv4 子网和可选网关。
type DockerNetworkIPAMConfig struct {
	Subnet  string
	Gateway string
}

// DockerNetworkContainerEndpoint 描述连接到网络的容器端点。
type DockerNetworkContainerEndpoint struct {
	ID         string
	Name       string
	EndpointID string
	IPv4       string
	IPv6       string
	MAC        string
}

// DockerNetworkCreateCommand 是创建网络所允许的显式字段，不透传任意 Docker driver options。
type DockerNetworkCreateCommand struct {
	Name       string
	Driver     string
	Internal   bool
	Attachable bool
	Labels     map[string]string
	IPAM       *DockerNetworkIPAMConfig
}

// DockerNetworkActionResult 是网络变更完成后可审计的最小结果。
type DockerNetworkActionResult struct {
	ID     string
	Name   string
	Action string
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
	ReadDockerNetwork(context.Context, string) (DockerNetworkDetail, error)
	CreateDockerNetwork(context.Context, DockerNetworkCreateCommand) (DockerNetworkActionResult, error)
	RemoveDockerNetwork(context.Context, string, string) (DockerNetworkActionResult, error)
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
func (r *DockerRuntime) ReadDockerNetwork(ctx context.Context, id string) (DockerNetworkDetail, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerNetworkDetail{}, errUnsupportedContainerRuntime
	}
	item, err := client.NetworkInspect(ctx, id, mobyclient.NetworkInspectOptions{})
	if err != nil {
		return DockerNetworkDetail{}, mapDockerNetworkError(err)
	}
	return dockerNetworkDetail(item), nil
}

// CreateDockerNetwork 通过受限模块命令创建网络，不透传任意 Docker driver options。
func (r *DockerRuntime) CreateDockerNetwork(ctx context.Context, command DockerNetworkCreateCommand) (DockerNetworkActionResult, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	if err := validateDockerNetworkCreateCommand(command); err != nil {
		return DockerNetworkActionResult{}, err
	}
	options := mobyclient.NetworkCreateOptions{Driver: command.Driver, Internal: command.Internal, Attachable: command.Attachable, Labels: cloneLabels(command.Labels)}
	if command.IPAM != nil {
		prefix, _ := netip.ParsePrefix(command.IPAM.Subnet)
		gateway, _ := netip.ParseAddr(command.IPAM.Gateway)
		options.IPAM = &network.IPAM{Config: []network.IPAMConfig{{Subnet: prefix, Gateway: gateway}}}
	}
	result, err := client.NetworkCreate(ctx, command.Name, options)
	if err != nil {
		return DockerNetworkActionResult{}, mapDockerNetworkError(err)
	}
	return DockerNetworkActionResult{ID: strings.TrimSpace(result.ID), Name: command.Name, Action: "create"}, nil
}

// RemoveDockerNetwork 在删除前校验名称确认和实时 Docker 检查结果，避免删除默认网络或仍被使用的网络。
func (r *DockerRuntime) RemoveDockerNetwork(ctx context.Context, id string, confirmation string) (DockerNetworkActionResult, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	item, err := client.NetworkInspect(ctx, id, mobyclient.NetworkInspectOptions{})
	if err != nil {
		return DockerNetworkActionResult{}, mapDockerNetworkError(err)
	}
	detail := dockerNetwork(item.Network, len(item.Containers))
	if strings.TrimSpace(confirmation) != detail.Name {
		return DockerNetworkActionResult{ID: detail.ID, Name: detail.Name, Action: "remove"}, errDockerNetworkConfirmMismatch
	}
	if isDockerDefaultNetwork(detail.Name) {
		return DockerNetworkActionResult{ID: detail.ID, Name: detail.Name, Action: "remove"}, errDockerNetworkDefaultProtected
	}
	if detail.ContainerCount > 0 {
		return DockerNetworkActionResult{ID: detail.ID, Name: detail.Name, Action: "remove"}, errDockerNetworkInUse
	}
	if err := client.NetworkRemove(ctx, detail.ID); err != nil {
		return DockerNetworkActionResult{ID: detail.ID, Name: detail.Name, Action: "remove"}, mapDockerNetworkError(err)
	}
	return DockerNetworkActionResult{ID: detail.ID, Name: detail.Name, Action: "remove"}, nil
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

func dockerNetworkDetail(item network.Inspect) DockerNetworkDetail {
	containers := make([]DockerNetworkContainerEndpoint, 0, len(item.Containers))
	for id, endpoint := range item.Containers {
		containers = append(containers, DockerNetworkContainerEndpoint{ID: strings.TrimSpace(id), Name: strings.TrimSpace(endpoint.Name), EndpointID: strings.TrimSpace(endpoint.EndpointID), IPv4: endpoint.IPv4Address.String(), IPv6: endpoint.IPv6Address.String(), MAC: endpoint.MacAddress.String()})
	}
	sort.Slice(containers, func(i, j int) bool { return containers[i].ID < containers[j].ID })
	configs := make([]DockerNetworkIPAMConfig, 0, len(item.IPAM.Config))
	for _, config := range item.IPAM.Config {
		configs = append(configs, DockerNetworkIPAMConfig{Subnet: config.Subnet.String(), Gateway: config.Gateway.String()})
	}
	return DockerNetworkDetail{DockerNetwork: dockerNetwork(item.Network, len(containers)), IPAM: DockerNetworkIPAM{Driver: strings.TrimSpace(item.IPAM.Driver), Config: configs}, Containers: containers}
}

func validateDockerNetworkCreateCommand(command DockerNetworkCreateCommand) error {
	if !validDockerNetworkIdentity(command) {
		return errInvalidDockerNetworkRequest
	}
	if !validDockerNetworkIPAM(command.IPAM) {
		return errInvalidDockerNetworkRequest
	}
	return nil
}

func validDockerNetworkIdentity(command DockerNetworkCreateCommand) bool {
	return strings.TrimSpace(command.Name) != "" && isDockerNetworkDriver(strings.TrimSpace(command.Driver))
}

func validDockerNetworkIPAM(config *DockerNetworkIPAMConfig) bool {
	if config == nil {
		return true
	}
	prefix, ok := dockerIPv4Prefix(config.Subnet)
	return ok && dockerGatewayInPrefix(config.Gateway, prefix)
}

func dockerIPv4Prefix(subnet string) (netip.Prefix, bool) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(subnet))
	return prefix, err == nil && prefix.Addr().Is4()
}

func dockerGatewayInPrefix(gateway string, prefix netip.Prefix) bool {
	gateway = strings.TrimSpace(gateway)
	if gateway == "" {
		return true
	}
	address, err := netip.ParseAddr(gateway)
	return err == nil && address.Is4() && prefix.Contains(address)
}

func isDockerNetworkDriver(driver string) bool {
	switch driver {
	case "bridge", "overlay", "macvlan", "ipvlan", "none":
		return true
	default:
		return false
	}
}

func isDockerDefaultNetwork(name string) bool {
	return name == "bridge" || name == "host" || name == "none"
}

func mapDockerNetworkError(err error) error {
	if errors.Is(mapDockerError(err), errContainerNotFound) {
		return errDockerNetworkNotFound
	}
	return mapDockerError(err)
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
