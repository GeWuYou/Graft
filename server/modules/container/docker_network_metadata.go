package container

import "strings"

const (
	dockerNetworkSourceCompose DockerNetworkSourceKind = "compose"
	dockerNetworkSourceSwarm   DockerNetworkSourceKind = "swarm"
	dockerNetworkSourceDocker  DockerNetworkSourceKind = "docker"
	dockerNetworkSourceCustom  DockerNetworkSourceKind = "custom"
	dockerNetworkSourceUnknown DockerNetworkSourceKind = "unknown"
)

const (
	dockerComposeLabelPrefix = "com.docker.compose."
	dockerSwarmLabelPrefix   = "com.docker.swarm."
	dockerStackLabelPrefix   = "com.docker.stack."
	dockerIOLabelPrefix      = "io.docker."

	dockerComposeProjectLabel = "com.docker.compose.project"
	dockerComposeNetworkLabel = "com.docker.compose.network"
	dockerComposeVersionLabel = "com.docker.compose.version"
	dockerSwarmStackLabel     = "com.docker.stack.namespace"
)

// DockerNetworkSource 是 Docker 网络来源的稳定业务投影，而非原始 Label 的替代品。
type DockerNetworkSource struct {
	Kind           DockerNetworkSourceKind
	ComposeProject string
	ComposeNetwork string
	ComposeVersion string
	SwarmStack     string
}

// DockerNetworkSourceKind 限定网络来源分类，避免 HTTP 投影依赖无约束字符串。
type DockerNetworkSourceKind string

// DockerNetworkLabelGroups 将 Docker 官方运行时标签与用户自定义标签分开，供列表和详情使用。
type DockerNetworkLabelGroups struct {
	System map[string]string
	User   map[string]string
}

// classifyDockerNetworkMetadata 从 Docker 运行时标签推导来源并保留可核查的分组结果。
func classifyDockerNetworkMetadata(name string, ingress bool, labels map[string]string) (DockerNetworkSource, DockerNetworkLabelGroups) {
	groups, signals := classifyDockerNetworkLabels(labels, ingress)
	return dockerNetworkSourceFromSignals(name, labels, signals), groups
}

type dockerNetworkLabelSignals struct {
	compose bool
	swarm   bool
}

func classifyDockerNetworkLabels(labels map[string]string, ingress bool) (DockerNetworkLabelGroups, dockerNetworkLabelSignals) {
	groups := DockerNetworkLabelGroups{System: make(map[string]string), User: make(map[string]string)}
	signals := dockerNetworkLabelSignals{swarm: ingress}
	for key, value := range labels {
		if isDockerSystemLabel(key) {
			groups.System[key] = value
		} else {
			groups.User[key] = value
		}
		if strings.HasPrefix(key, dockerComposeLabelPrefix) {
			signals.compose = true
		}
		if strings.HasPrefix(key, dockerSwarmLabelPrefix) || strings.HasPrefix(key, dockerStackLabelPrefix) {
			signals.swarm = true
		}
	}
	return groups, signals
}

func dockerNetworkSourceFromSignals(name string, labels map[string]string, signals dockerNetworkLabelSignals) DockerNetworkSource {
	switch {
	case signals.compose && signals.swarm:
		return DockerNetworkSource{Kind: dockerNetworkSourceUnknown}
	case signals.compose:
		return DockerNetworkSource{Kind: dockerNetworkSourceCompose, ComposeProject: strings.TrimSpace(labels[dockerComposeProjectLabel]), ComposeNetwork: strings.TrimSpace(labels[dockerComposeNetworkLabel]), ComposeVersion: strings.TrimSpace(labels[dockerComposeVersionLabel])}
	case signals.swarm:
		return DockerNetworkSource{Kind: dockerNetworkSourceSwarm, SwarmStack: strings.TrimSpace(labels[dockerSwarmStackLabel])}
	case isDockerDefaultNetwork(strings.TrimSpace(name)):
		return DockerNetworkSource{Kind: dockerNetworkSourceDocker}
	default:
		return DockerNetworkSource{Kind: dockerNetworkSourceCustom}
	}
}

func isDockerSystemLabel(key string) bool {
	return strings.HasPrefix(key, dockerComposeLabelPrefix) || strings.HasPrefix(key, dockerSwarmLabelPrefix) || strings.HasPrefix(key, dockerStackLabelPrefix) || strings.HasPrefix(key, dockerIOLabelPrefix)
}
