package container

import (
	"net/netip"
	"strings"
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

func validDockerNetworkIPAM(value *DockerNetworkIPAMConfig) bool {
	subnet, err := netip.ParsePrefix(strings.TrimSpace(value.Subnet))
	if err != nil || !subnet.Addr().Is4() {
		return false
	}
	if gatewayText := strings.TrimSpace(value.Gateway); gatewayText != "" {
		gateway, err := netip.ParseAddr(gatewayText)
		if err != nil || !gateway.Is4() || !subnet.Contains(gateway) {
			return false
		}
	}
	return true
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
