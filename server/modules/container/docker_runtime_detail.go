package container

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

// dockerSummary converts a Docker container summary to the module Summary type, normalizing metadata and detecting orchestrator information.
func dockerSummary(item container.Summary) Summary {
	names := cleanDockerNames(item.Names)
	networks := dockerSummaryNetworks(item)
	state := normalizeContainerState(string(item.State))
	labels := cloneLabels(item.Labels)
	orchestrator := dockerOrchestratorFromLabels(labels)
	return Summary{
		ID:             strings.TrimSpace(item.ID),
		ShortID:        shortRuntimeID(item.ID),
		Name:           firstNonEmpty(firstContainerName(names), shortRuntimeID(item.ID), strings.TrimSpace(item.ID)),
		Names:          names,
		Image:          strings.TrimSpace(item.Image),
		ImageID:        strings.TrimSpace(item.ImageID),
		Labels:         labels,
		Ports:          dockerPorts(item.Ports),
		PrimaryIP:      primaryNetworkIP(networks),
		Networks:       networks,
		NetworkSummary: networkSummary(networks),
		Resource: ResourceSummary{
			Available:         false,
			UnavailableReason: containerStatsNotCollectedReason,
			StatsAvailable:    false,
			StatsErrorKey:     containerStatsNotCollectedReason,
			StatsErrorMessage: "Container stats were not collected.",
		},
		Runtime:        runtimeNameDocker,
		CreatedAt:      time.Unix(item.Created, 0).UTC().Format(time.RFC3339),
		State:          state,
		Status:         strings.TrimSpace(item.Status),
		Health:         containerHealthUnavailable,
		ComposeProject: strings.TrimSpace(labels[composeProjectLabel]),
		ComposeService: strings.TrimSpace(labels[composeServiceLabel]),
		Orchestrator:   orchestrator,
		CanStart:       canStartState(state),
		CanStop:        canStopState(state),
		CanRestart:     canRestartState(state),
		CanRemove:      canRemoveState(state),
	}
}

// dockerDetail 从 Docker 容器检查输出和运行时元数据构建 Detail 结构体。
func dockerDetail(inspect container.InspectResponse, info RuntimeInfo) Detail {
	state, status, startedAt := dockerState(inspect)
	labels := dockerLabels(inspect)
	orchestrator := dockerOrchestratorFromLabels(labels)
	summary := Summary{
		ID:             strings.TrimSpace(inspect.ID),
		ShortID:        shortRuntimeID(inspect.ID),
		Names:          []string{strings.TrimPrefix(strings.TrimSpace(inspect.Name), "/")},
		Image:          dockerImage(inspect),
		ImageID:        strings.TrimSpace(inspect.Image),
		Labels:         labels,
		Ports:          dockerInspectPorts(inspect),
		Networks:       dockerNetworks(inspect),
		Resource:       unavailableResourceSummary(containerStatsNotCollectedReason),
		Runtime:        runtimeNameDocker,
		CreatedAt:      parseDockerTimeString(inspect.Created),
		StartedAt:      startedAt,
		State:          state,
		Status:         status,
		Health:         dockerHealth(inspect),
		ComposeProject: strings.TrimSpace(labels[composeProjectLabel]),
		ComposeService: strings.TrimSpace(labels[composeServiceLabel]),
		Orchestrator:   orchestrator,
		CanStart:       canStartState(state),
		CanStop:        canStopState(state),
		CanRestart:     canRestartState(state),
		CanRemove:      canRemoveState(state),
	}
	summary.Name = firstNonEmpty(firstContainerName(summary.Names), summary.ShortID, summary.ID)
	summary.PrimaryIP = primaryNetworkIP(summary.Networks)
	summary.NetworkSummary = networkSummary(summary.Networks)
	summary.RestartCount = intPtrAllowZero(inspect.RestartCount)
	detail := Detail{
		Summary:          summary,
		Healthcheck:      dockerHealthcheck(inspect),
		LastExitCode:     dockerLastExitCode(inspect),
		Mounts:           dockerMounts(inspect.Mounts),
		Networks:         dockerNetworks(inspect),
		OOMKilled:        dockerOOMKilled(inspect),
		RuntimeInfo:      info,
		InspectUpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if inspect.Config != nil {
		detail.Command = []string(inspect.Config.Cmd)
		detail.Entrypoint = []string(inspect.Config.Entrypoint)
		detail.Environment = dockerEnvironmentVariables(inspect.Config.Env)
		detail.WorkingDir = strings.TrimSpace(inspect.Config.WorkingDir)
	}
	if inspect.HostConfig != nil {
		detail.RestartPolicy = string(inspect.HostConfig.RestartPolicy.Name)
	}
	return detail
}

// dockerOrchestratorFromLabels 从容器标签检测编排器类型，并返回该编排器的元数据。
func dockerOrchestratorFromLabels(labels map[string]string) OrchestratorInfo {
	labels = cloneLabels(labels)
	if len(labels) == 0 {
		return OrchestratorInfo{
			Type:            containerOrchestratorStandalone,
			Managed:         false,
			Confidence:      orchestratorConfidenceHigh,
			GroupScopeKind:  "",
			MemberScopeKind: "",
		}
	}

	typeCount := 0
	info := OrchestratorInfo{
		Type:            containerOrchestratorStandalone,
		Managed:         false,
		Confidence:      orchestratorConfidenceHigh,
		GroupScopeKind:  "",
		MemberScopeKind: "",
	}

	if metadata, ok := kubernetesMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorKubernetes
		info.Managed = true
		info.GroupScopeKind = kubernetesNamespaceScopeKind
		info.GroupValue = metadata.Namespace
		info.GroupDisplayName = metadata.Namespace
		info.MemberScopeKind = kubernetesPodScopeKind
		info.MemberValue = metadata.Pod
		info.MemberDisplayName = metadata.Pod
		info.Namespace = metadata.Namespace
		info.Pod = metadata.Pod
		info.Container = metadata.Container
		info.DisplayName = firstNonEmpty(metadata.Namespace, metadata.Pod, "kubernetes")
		info.Confidence = orchestratorConfidenceHigh
	}
	if stack, task, ok := swarmMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorSwarm
		info.Managed = true
		info.GroupScopeKind = swarmStackScopeKind
		info.GroupValue = stack
		info.GroupDisplayName = stack
		info.MemberScopeKind = swarmTaskScopeKind
		info.MemberValue = task
		info.MemberDisplayName = task
		info.Stack = stack
		info.Task = task
		info.DisplayName = firstNonEmpty(stack, task, "swarm")
		info.Confidence = orchestratorConfidenceHigh
	}
	if project, service, ok := composeMetadata(labels); ok {
		typeCount++
		info.Type = containerOrchestratorCompose
		info.Managed = true
		info.GroupScopeKind = composeProjectScopeKind
		info.GroupValue = project
		info.GroupDisplayName = project
		info.MemberScopeKind = composeServiceScopeKind
		info.MemberValue = service
		info.MemberDisplayName = service
		info.Project = project
		info.Service = service
		info.DisplayName = firstNonEmpty(project, service, "compose")
		info.Confidence = orchestratorConfidenceHigh
	}
	if typeCount == 0 {
		return info
	}
	if typeCount > 1 {
		return OrchestratorInfo{
			Type:            containerOrchestratorUnknown,
			Managed:         true,
			Confidence:      orchestratorConfidenceLow,
			GroupScopeKind:  "",
			MemberScopeKind: "",
		}
	}
	return info
}

type kubernetesOrchestratorMetadata struct {
	Namespace string
	Pod       string
	Container string
}

// kubernetesMetadata extracts Kubernetes metadata from the provided labels, returning the extracted metadata and whether any Kubernetes metadata was found.
func kubernetesMetadata(labels map[string]string) (kubernetesOrchestratorMetadata, bool) {
	metadata := kubernetesOrchestratorMetadata{
		Namespace: strings.TrimSpace(labels["io.kubernetes.pod.namespace"]),
		Pod:       strings.TrimSpace(labels["io.kubernetes.pod.name"]),
		Container: strings.TrimSpace(labels["io.kubernetes.container.name"]),
	}
	ok := metadata.Namespace != "" || metadata.Pod != "" || metadata.Container != ""
	return metadata, ok
}

// swarmMetadata extracts Docker Swarm metadata from container labels.
func swarmMetadata(labels map[string]string) (stack string, task string, ok bool) {
	stack = strings.TrimSpace(labels["com.docker.stack.namespace"])
	task = strings.TrimSpace(labels["com.docker.swarm.task.name"])
	ok = stack != "" || task != ""
	return stack, task, ok
}

// composeMetadata 从容器标签中检测 Docker Compose 的项目和服务名称。
func composeMetadata(labels map[string]string) (project string, service string, ok bool) {
	project = strings.TrimSpace(labels[composeProjectLabel])
	service = strings.TrimSpace(labels[composeServiceLabel])
	ok = project != "" || service != ""
	return project, service, ok
}

// dockerEnvironmentVariables 将原始环境变量字符串列表解析为 EnvironmentVariable 对象。
func dockerEnvironmentVariables(values []string) []EnvironmentVariable {
	if len(values) == 0 {
		return nil
	}
	environment := make([]EnvironmentVariable, 0, len(values))
	for _, raw := range values {
		key, value, ok := strings.Cut(raw, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		environment = append(environment, EnvironmentVariable{
			Key:       key,
			Value:     value,
			Sensitive: isSensitiveEnvironmentKey(key),
			Source:    dockerEnvironmentSource,
		})
	}
	return environment
}

func dockerSummaryNetworks(item container.Summary) []Network {
	if item.NetworkSettings == nil || len(item.NetworkSettings.Networks) == 0 {
		return nil
	}
	networks := make([]Network, 0, len(item.NetworkSettings.Networks))
	for name, network := range item.NetworkSettings.Networks {
		if mapped, ok := dockerEndpointNetwork(name, network); ok {
			networks = append(networks, mapped)
		}
	}
	return networks
}

func primaryNetworkIP(networks []Network) string {
	for _, network := range networks {
		if strings.TrimSpace(network.IPAddress) != "" {
			return strings.TrimSpace(network.IPAddress)
		}
	}
	return ""
}

func networkSummary(networks []Network) string {
	names := make([]string, 0, len(networks))
	for _, network := range networks {
		if strings.TrimSpace(network.Name) != "" {
			names = append(names, strings.TrimSpace(network.Name))
		}
	}
	return strings.Join(names, ", ")
}

// dockerHealth derives the container's normalized health status from inspection data.
func dockerHealth(inspect container.InspectResponse) string {
	if inspect.State == nil || inspect.State.Health == nil {
		return containerHealthNone
	}
	switch inspect.State.Health.Status {
	case container.NoHealthcheck:
		return containerHealthNone
	case container.Starting:
		return containerHealthStarting
	case container.Healthy:
		return containerHealthHealthy
	case container.Unhealthy:
		return containerHealthUnhealthy
	default:
		return containerHealthUnavailable
	}
}

// health state is unavailable, status is set to unavailable.
func dockerHealthcheck(inspect container.InspectResponse) *Healthcheck {
	command := dockerHealthcheckCommand(inspect)
	if len(command) == 0 {
		return nil
	}
	result := &Healthcheck{
		Configured: true,
		Status:     dockerHealth(inspect),
		Command:    command,
	}
	if inspect.State == nil || inspect.State.Health == nil {
		result.Status = containerHealthUnavailable
		return result
	}

	health := inspect.State.Health
	result.FailingStreak = intPtrAllowZero(health.FailingStreak)
	if len(health.Log) == 0 {
		return result
	}
	last := health.Log[len(health.Log)-1]
	if last == nil {
		return result
	}
	result.ExitCode = intPtrAllowZero(last.ExitCode)
	result.Output = strings.TrimSpace(last.Output)
	if !last.End.IsZero() {
		result.CheckedAt = last.End.UTC().Format(time.RFC3339)
	} else if !last.Start.IsZero() {
		result.CheckedAt = last.Start.UTC().Format(time.RFC3339)
	}
	if last.ExitCode != 0 {
		result.FailureMessage = result.Output
	}
	return result
}

// dockerHealthcheckCommand extracts the healthcheck test command from a container's configuration.
func dockerHealthcheckCommand(inspect container.InspectResponse) []string {
	if inspect.Config == nil || inspect.Config.Healthcheck == nil {
		return nil
	}
	test := inspect.Config.Healthcheck.Test
	if len(test) == 0 {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(test[0]), "NONE") {
		return nil
	}
	command := make([]string, 0, len(test))
	for _, item := range test {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			command = append(command, trimmed)
		}
	}
	if len(command) == 0 {
		return nil
	}
	return command
}

// dockerLastExitCode returns a pointer to the container's exit code from the inspection state.
func dockerLastExitCode(inspect container.InspectResponse) *int {
	if inspect.State == nil {
		return nil
	}
	return intPtrAllowZero(inspect.State.ExitCode)
}

// dockerOOMKilled returns a pointer to the OOMKilled flag from the container's state.
func dockerOOMKilled(inspect container.InspectResponse) *bool {
	if inspect.State == nil {
		return nil
	}
	value := inspect.State.OOMKilled
	return &value
}

// ShortRuntimeID returns the runtime ID truncated to containerShortIDLength characters.
func shortRuntimeID(id string) string {
	value := strings.TrimSpace(id)
	if len(value) <= containerShortIDLength {
		return value
	}
	return value[:containerShortIDLength]
}

func canStartState(state string) bool {
	return state != "running" && state != "paused" && state != "removing"
}

func canStopState(state string) bool {
	return state == "running"
}

func canRestartState(state string) bool {
	return state != "removing" && state != "dead"
}

func canRemoveState(state string) bool {
	return state != "" && state != "unknown" && state != "removing"
}

func canRemoveWithoutForce(state string) bool {
	return canRemoveState(state) && state != "running" && state != "paused" && state != "restarting"
}

func dockerInfoToRuntimeInfo(info systemInfo, endpoint string) RuntimeInfo {
	value, ok := info.(dockerClientSystemInfo)
	if !ok {
		return RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled", Endpoint: endpoint}
	}
	return RuntimeInfo{
		Runtime:           runtimeNameDocker,
		Status:            "enabled",
		Endpoint:          endpoint,
		APIVersion:        value.APIVersion,
		ServerVersion:     value.ServerVersion,
		OperatingSystem:   value.OperatingSystem,
		Architecture:      value.Architecture,
		ContainersTotal:   value.Containers,
		ContainersRunning: value.ContainersRunning,
	}
}

// actionResultFromDetail 根据容器详情构造操作结果，并标记状态是否发生变化。
func actionResultFromDetail(detail Detail, ref Ref, action string, statusBefore string) ActionResult {
	statusAfter := detail.State
	result := actionResultCompleted
	if statusBefore != "" && statusBefore == statusAfter {
		result = actionResultUnchanged
	}
	return ActionResult{
		ID:           firstNonEmpty(detail.ID, ref.Value),
		Name:         firstContainerName(detail.Names),
		Image:        detail.Image,
		Action:       action,
		Result:       result,
		Runtime:      runtimeNameDocker,
		StatusBefore: statusBefore,
		StatusAfter:  statusAfter,
	}
}

func dockerState(inspect container.InspectResponse) (string, string, string) {
	if inspect.State == nil {
		return "unknown", "", ""
	}
	startedAt := parseDockerTimeString(inspect.State.StartedAt)
	return normalizeContainerState(string(inspect.State.Status)), strings.TrimSpace(string(inspect.State.Status)), startedAt
}

func dockerImage(inspect container.InspectResponse) string {
	if inspect.Config != nil && strings.TrimSpace(inspect.Config.Image) != "" {
		return strings.TrimSpace(inspect.Config.Image)
	}
	return strings.TrimSpace(inspect.Image)
}

func dockerLabels(inspect container.InspectResponse) map[string]string {
	if inspect.Config == nil {
		return nil
	}
	return cloneLabels(inspect.Config.Labels)
}

func dockerInspectPorts(inspect container.InspectResponse) []Port {
	if inspect.NetworkSettings == nil {
		return nil
	}
	ports := make([]Port, 0, len(inspect.NetworkSettings.Ports))
	for port, bindings := range inspect.NetworkSettings.Ports {
		privatePort, _ := strconv.Atoi(port.Port())
		if len(bindings) == 0 {
			ports = append(ports, Port{PrivatePort: privatePort, Type: string(port.Proto())})
			continue
		}
		for _, binding := range bindings {
			publicPort, _ := strconv.Atoi(binding.HostPort)
			ports = append(ports, Port{
				IP:          strings.TrimSpace(binding.HostIP.String()),
				PrivatePort: privatePort,
				PublicPort:  intPtr(publicPort),
				Type:        string(port.Proto()),
			})
		}
	}
	return ports
}

func dockerPorts(ports []container.PortSummary) []Port {
	mapped := make([]Port, 0, len(ports))
	for _, port := range ports {
		privatePort := int(port.PrivatePort)
		publicPort := int(port.PublicPort)
		item := Port{
			IP:          strings.TrimSpace(port.IP.String()),
			PrivatePort: privatePort,
			Type:        strings.TrimSpace(port.Type),
		}
		if publicPort > 0 {
			item.PublicPort = &publicPort
		}
		mapped = append(mapped, item)
	}
	return mapped
}

// dockerMounts converts Docker mount points to Mount structures.
func dockerMounts(mounts []container.MountPoint) []Mount {
	mapped := make([]Mount, 0, len(mounts))
	for _, mount := range mounts {
		item := Mount{
			Type:        string(mount.Type),
			Name:        strings.TrimSpace(mount.Name),
			Source:      strings.TrimSpace(mount.Source),
			Destination: strings.TrimSpace(mount.Destination),
			Mode:        strings.TrimSpace(mount.Mode),
			ReadOnly:    !mount.RW,
		}
		item.ID = stableMountID(item)
		mapped = append(mapped, item)
	}
	return mapped
}

// findMountByID locates a mount by its stable ID.
func findMountByID(mounts []Mount, mountID string) (Mount, bool) {
	mountID = strings.TrimSpace(mountID)
	for _, mount := range mounts {
		if mount.ID == mountID {
			return mount, true
		}
	}
	return Mount{}, false
}

// mountUsageSupported reports whether a mount supports usage measurement.
func mountUsageSupported(mount Mount) bool {
	switch strings.TrimSpace(strings.ToLower(mount.Type)) {
	case "bind", "volume":
		return strings.TrimSpace(mount.Source) != ""
	default:
		return false
	}
}

// mountUsageFromMount constructs mount usage information from mount details and measurement metadata.
func mountUsageFromMount(containerID string, mount Mount, status string, size int64, measuredAt string) MountUsage {
	usage := MountUsage{
		MountID:     mount.ID,
		ContainerID: containerID,
		Type:        strings.TrimSpace(mount.Type),
		Name:        strings.TrimSpace(mount.Name),
		Source:      strings.TrimSpace(mount.Source),
		Destination: strings.TrimSpace(mount.Destination),
		Status:      firstNonEmpty(strings.TrimSpace(status), containerMountUsageStatusNotMeasured),
		MeasuredAt:  strings.TrimSpace(measuredAt),
	}
	if strings.TrimSpace(mount.Name) != "" {
		usage.SharedHint = "named volume may be shared by multiple containers"
	}
	if usage.Status == containerMountUsageStatusMeasured {
		usage.SizeBytes = size
		usage.SizeDisplay = formatIECBytes(size)
	}
	if usage.Status == containerMountUsageStatusUnsupported {
		usage.Message = "Mount usage is not supported for this mount."
	}
	return usage
}

// mountUsageFromScanError maps mount scan errors into mount usage information.
func mountUsageFromScanError(containerID string, mount Mount, err error) MountUsage {
	status := containerMountUsageStatusError
	message := "Mount usage measurement failed."
	switch {
	case errors.Is(err, errRuntimePermissionDenied):
		status = containerMountUsageStatusPermissionDenied
		message = "Permission denied while measuring mount usage."
	case errors.Is(err, errContainerMountNotFound):
		status = containerMountUsageStatusNotFound
		message = "Mount source was not found while measuring usage."
	case errors.Is(err, errContainerRuntimeTimeout):
		status = containerMountUsageStatusTimeout
		message = "Mount usage measurement timed out."
	}
	usage := mountUsageFromMount(containerID, mount, status, 0, "")
	usage.Message = message
	return usage
}

// dockerNetworks converts networks from a Docker container inspection response into the module's Network format.
func dockerNetworks(inspect container.InspectResponse) []Network {
	if inspect.NetworkSettings == nil || len(inspect.NetworkSettings.Networks) == 0 {
		return nil
	}
	networks := make([]Network, 0, len(inspect.NetworkSettings.Networks))
	for name, network := range inspect.NetworkSettings.Networks {
		if mapped, ok := dockerEndpointNetwork(name, network); ok {
			networks = append(networks, mapped)
		}
	}
	return networks
}

func dockerEndpointNetwork(name string, network *network.EndpointSettings) (Network, bool) {
	if network == nil {
		return Network{}, false
	}
	return Network{
		Name:       strings.TrimSpace(name),
		NetworkID:  strings.TrimSpace(network.NetworkID),
		EndpointID: strings.TrimSpace(network.EndpointID),
		Gateway:    strings.TrimSpace(network.Gateway.String()),
		IPAddress:  strings.TrimSpace(network.IPAddress.String()),
		MacAddress: strings.TrimSpace(network.MacAddress.String()),
	}, true
}

func cleanDockerNames(names []string) []string {
	cleaned := make([]string, 0, len(names))
	for _, name := range names {
		if trimmed := strings.TrimPrefix(strings.TrimSpace(name), "/"); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func firstContainerName(names []string) string {
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}

func normalizeContainerState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "created", "running", "paused", "restarting", "removing", "exited", "dead":
		return strings.ToLower(strings.TrimSpace(state))
	default:
		return "unknown"
	}
}

func parseDockerTimeString(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "0001-") {
		return ""
	}
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return timestamp.UTC().Format(time.RFC3339)
}

func safeEndpointLabel(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return runtimeNameDocker
	}
	if parsed.Scheme == dockerSocketScheme {
		return "unix://" + parsed.Path
	}
	return parsed.Scheme
}

func intPtr(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func intPtrAllowZero(value int) *int {
	return &value
}
