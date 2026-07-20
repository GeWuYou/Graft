package container

import (
	"context"
	"net/netip"
	"strings"

	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
)

// DockerNetworkCreateCommand 仅承载允许透传给 Docker 的网络创建字段。
type DockerNetworkCreateCommand struct {
	Name, Driver         string
	Internal, Attachable bool
	Labels               map[string]string
	IPAM                 *DockerNetworkIPAMConfig
}

// DockerNetworkIPAMConfig 描述受限的 IPv4 子网和网关配置。
type DockerNetworkIPAMConfig struct{ Subnet, Gateway string }

// DockerNetworkActionResult 是网络变更完成后返回并用于审计的最小结果。
type DockerNetworkActionResult struct{ ID, Name, Action string }

type dockerNetworkWriter interface {
	NetworkCreate(context.Context, string, mobyclient.NetworkCreateOptions) (mobyclient.NetworkCreateResult, error)
	NetworkRemove(context.Context, string) error
}

// CreateDockerNetwork 通过受限命令创建 Docker 网络，不透传任意 driver 参数。
func (r *DockerRuntime) CreateDockerNetwork(ctx context.Context, command DockerNetworkCreateCommand) (DockerNetworkActionResult, error) {
	if _, ok := r.client.(dockerResourceClient); !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	writer, ok := r.client.(dockerNetworkWriter)
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	command.Name, command.Driver = strings.TrimSpace(command.Name), strings.TrimSpace(command.Driver)
	if command.Name == "" || !isDockerNetworkDriver(command.Driver) {
		return DockerNetworkActionResult{}, errInvalidDockerNetworkRequest
	}
	options := mobyclient.NetworkCreateOptions{Driver: command.Driver, Internal: command.Internal, Attachable: command.Attachable, Labels: cloneLabels(command.Labels)}
	if command.IPAM != nil {
		options.IPAM = dockerNetworkIPAM(command.IPAM)
		if options.IPAM == nil {
			return DockerNetworkActionResult{}, errInvalidDockerNetworkRequest
		}
	}
	result, err := writer.NetworkCreate(ctx, command.Name, options)
	if err != nil {
		return DockerNetworkActionResult{}, mapDockerError(err)
	}
	return DockerNetworkActionResult{ID: strings.TrimSpace(result.ID), Name: command.Name, Action: "create"}, nil
}

func dockerNetworkIPAM(value *DockerNetworkIPAMConfig) *network.IPAM {
	subnet, err := netip.ParsePrefix(strings.TrimSpace(value.Subnet))
	if err != nil || !subnet.Addr().Is4() {
		return nil
	}
	config := network.IPAMConfig{Subnet: subnet}
	if gatewayText := strings.TrimSpace(value.Gateway); gatewayText != "" {
		gateway, err := netip.ParseAddr(gatewayText)
		if err != nil || !gateway.Is4() || !subnet.Contains(gateway) {
			return nil
		}
		config.Gateway = gateway
	}
	return &network.IPAM{Config: []network.IPAMConfig{config}}
}

// RemoveDockerNetwork 在删除前校验名称，并拒绝删除默认网络或仍被容器使用的网络。
func (r *DockerRuntime) RemoveDockerNetwork(ctx context.Context, id, confirmation string) (DockerNetworkActionResult, error) {
	client, ok := r.client.(dockerResourceClient)
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	writer, ok := r.client.(dockerNetworkWriter)
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	item, err := client.NetworkInspect(ctx, id, mobyclient.NetworkInspectOptions{})
	if err != nil {
		return DockerNetworkActionResult{}, mapDockerError(err)
	}
	detail := dockerNetwork(item.Network, len(item.Containers))
	result := DockerNetworkActionResult{ID: detail.ID, Name: detail.Name, Action: "remove"}
	if strings.TrimSpace(confirmation) != detail.Name {
		return result, errDockerNetworkConfirmMismatch
	}
	if isDockerDefaultNetwork(detail.Name) {
		return result, errDockerNetworkDefaultProtected
	}
	if detail.ContainerCount > 0 {
		return result, errDockerNetworkInUse
	}
	if err := writer.NetworkRemove(ctx, detail.ID); err != nil {
		return result, mapDockerError(err)
	}
	return result, nil
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

func (s *service) CreateDockerNetwork(ctx context.Context, command DockerNetworkCreateCommand) (DockerNetworkActionResult, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerNetworkActionResult{}, err
	}
	writer, ok := reader.(interface {
		CreateDockerNetwork(context.Context, DockerNetworkCreateCommand) (DockerNetworkActionResult, error)
	})
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	return writer.CreateDockerNetwork(ctx, command)
}
func (s *service) RemoveDockerNetwork(ctx context.Context, id, confirmation string) (DockerNetworkActionResult, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerNetworkActionResult{}, err
	}
	writer, ok := reader.(interface {
		RemoveDockerNetwork(context.Context, string, string) (DockerNetworkActionResult, error)
	})
	if !ok {
		return DockerNetworkActionResult{}, errUnsupportedContainerRuntime
	}
	return writer.RemoveDockerNetwork(ctx, id, confirmation)
}
