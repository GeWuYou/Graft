package container

import (
	"strings"
	"time"

	containergen "graft/server/internal/contract/openapi/generated"
)

// toContainerListResponse 将容器列表结果转换为 OpenAPI 容器列表响应。
//
// 该响应包含分页信息、运行时信息、列表摘要以及逐项转换后的容器概要。
func toContainerListResponse(result ListResult) containergen.ContainerListResponse {
	mapped := make([]containergen.ContainerSummary, 0, len(result.Items))
	for _, item := range result.Items {
		mapped = append(mapped, toSummary(item))
	}
	return containergen.ContainerListResponse{
		Items:   mapped,
		Limit:   result.Limit,
		Offset:  result.Offset,
		Runtime: toRuntimeInfo(result.Runtime),
		Summary: toListSummary(result.Summary),
		Total:   result.Total,
	}
}

// toContainerDashboardSummaryResponse 组装容器仪表盘摘要响应。
// Overview、Hotspots 和 Anomalies 分别映射为对应的响应结构。
func toContainerDashboardSummaryResponse(result dashboardSummaryResult) containerDashboardSummaryResponse {
	return containerDashboardSummaryResponse{
		Overview:  toContainerDashboardOverview(result.Overview),
		Hotspots:  toContainerDashboardHotspots(result.Hotspots),
		Anomalies: toContainerDashboardAnomalies(result.Anomalies),
	}
}

// toSummary 将 Summary 域对象转换为 ContainerSummary 响应。
// 它会复制容器标识、名称、镜像、状态和重启信息，并将健康状态、端口、网络、资源、编排信息以及可选字段映射为对应的 OpenAPI 类型。
func toSummary(item Summary) containergen.ContainerSummary {
	return containergen.ContainerSummary{
		CanRemove:      optionalBool(item.CanRemove),
		CanRestart:     optionalBool(item.CanRestart),
		CanStart:       optionalBool(item.CanStart),
		CanStop:        optionalBool(item.CanStop),
		ComposeProject: optionalString(item.ComposeProject),
		ComposeService: optionalString(item.ComposeService),
		Id:             item.ID,
		ShortId:        item.ShortID,
		Name:           item.Name,
		Names:          item.Names,
		Image:          item.Image,
		ImageId:        optionalString(item.ImageID),
		Labels:         optionalStringMap(item.Labels),
		Health:         optionalSummaryHealth(item.Health),
		Ports:          toPorts(item.Ports),
		PrimaryIp:      optionalString(item.PrimaryIP),
		Orchestrator:   toOrchestratorInfo(item.Orchestrator),
		Networks:       optionalNetworks(item.Networks),
		NetworkSummary: optionalString(item.NetworkSummary),
		Resource:       toResourceSummary(item.Resource),
		RestartCount:   item.RestartCount,
		RestartPolicy:  optionalString(item.RestartPolicy),
		Runtime:        item.Runtime,
		CreatedAt:      mustTime(item.CreatedAt),
		StartedAt:      optionalTime(item.StartedAt),
		State:          containergen.ContainerSummaryState(item.State),
		Status:         item.Status,
	}
}

// ToDetail converts the internal Detail domain model into an OpenAPI container detail response.
func toDetail(detail Detail) containergen.ContainerDetail {
	return containergen.ContainerDetail{
		CanRemove:                    optionalBool(detail.CanRemove),
		CanRestart:                   optionalBool(detail.CanRestart),
		CanStart:                     optionalBool(detail.CanStart),
		CanStop:                      optionalBool(detail.CanStop),
		Command:                      optionalStringSlice(detail.Command),
		ComposeProject:               optionalString(detail.ComposeProject),
		ComposeService:               optionalString(detail.ComposeService),
		CreatedAt:                    mustTime(detail.CreatedAt),
		Entrypoint:                   optionalStringSlice(detail.Entrypoint),
		Environment:                  optionalEnvironment(detail.Environment),
		EnvironmentMaskedCopyEnabled: detail.EnvironmentMaskedCopyEnabled,
		EnvironmentPolicy:            optionalEnvironmentPolicy(detail.EnvironmentPolicy),
		Health:                       optionalDetailHealth(detail.Health),
		Healthcheck:                  optionalHealthcheck(detail.Healthcheck),
		Id:                           detail.ID,
		Image:                        detail.Image,
		ImageId:                      optionalString(detail.ImageID),
		InspectUpdatedAt:             optionalTime(detail.InspectUpdatedAt),
		Labels:                       optionalStringMap(detail.Labels),
		LastExitCode:                 detail.LastExitCode,
		Mounts:                       toMounts(detail.Mounts),
		Name:                         detail.Name,
		Names:                        detail.Names,
		NetworkSummary:               optionalString(detail.NetworkSummary),
		Networks:                     toNetworks(detail.Networks),
		OomKilled:                    detail.OOMKilled,
		Orchestrator:                 toOrchestratorInfo(detail.Orchestrator),
		Ports:                        toPorts(detail.Ports),
		PrimaryIp:                    optionalString(detail.PrimaryIP),
		Resource:                     toResourceSummary(detail.Resource),
		RestartCount:                 detail.RestartCount,
		RestartPolicy:                optionalString(detail.RestartPolicy),
		Runtime:                      detail.Runtime,
		RuntimeInfo:                  toRuntimeInfo(detail.RuntimeInfo),
		ShortId:                      detail.ShortID,
		StartedAt:                    optionalTime(detail.StartedAt),
		State:                        containergen.ContainerDetailState(detail.State),
		Status:                       detail.Status,
		WorkingDir:                   optionalString(detail.WorkingDir),
	}
}

// optionalHealthcheck converts a healthcheck into its generated response type.
// It returns nil if the input is nil or the healthcheck is not configured.
func optionalHealthcheck(healthcheck *Healthcheck) *containergen.ContainerHealthcheck {
	if healthcheck == nil || !healthcheck.Configured {
		return nil
	}
	return &containergen.ContainerHealthcheck{
		CheckedAt:      optionalTime(healthcheck.CheckedAt),
		Command:        append([]string(nil), healthcheck.Command...),
		Configured:     healthcheck.Configured,
		ExitCode:       healthcheck.ExitCode,
		FailingStreak:  healthcheck.FailingStreak,
		FailureMessage: optionalString(healthcheck.FailureMessage),
		Output:         optionalString(healthcheck.Output),
		Status:         containergen.ContainerHealthcheckStatus(healthcheck.Status),
	}
}

// Returns a pointer to the converted entries, or nil if the input is empty.
func optionalEnvironment(environment []EnvironmentVariable) *[]containergen.ContainerEnvironmentEntry {
	if len(environment) == 0 {
		return nil
	}
	mapped := make([]containergen.ContainerEnvironmentEntry, 0, len(environment))
	for _, item := range environment {
		mapped = append(mapped, containergen.ContainerEnvironmentEntry{
			CopyValue:    optionalString(item.CopyValue),
			DisplayValue: optionalString(item.DisplayValue),
			Key:          item.Key,
			Masked:       item.Masked,
			Sensitive:    item.Sensitive,
			Source:       containergen.ContainerEnvironmentEntrySource(item.Source),
			Value:        optionalString(item.Value),
			ValueHidden:  optionalBool(item.ValueHidden),
			ValueMasked:  optionalBool(item.ValueMasked),
		})
	}
	return &mapped
}

func optionalEnvironmentPolicy(policy string) containergen.ContainerDetailEnvironmentPolicy {
	normalized := normalizeEnvironmentPolicy(policy)
	value := containergen.ContainerDetailEnvironmentPolicy(normalized.String())
	return value
}

// toListSummary 将容器列表汇总信息映射为 OpenAPI 列表汇总响应。
// 它复制健康状态、运行状态和总数统计字段。
func toListSummary(summary ListSummary) containergen.ContainerListSummary {
	return containergen.ContainerListSummary{
		Error:             summary.Error,
		HealthUnavailable: summary.HealthUnavailable,
		Healthy:           summary.Healthy,
		Running:           summary.Running,
		Stopped:           summary.Stopped,
		Total:             summary.Total,
		Unhealthy:         summary.Unhealthy,
	}
}

// toContainerDashboardOverview 将容器仪表盘概览数据转换为响应结构。
// 它保留运行中与异常容器数量以及 CPU 总占比，并将内存总量字段按可选值返回。
func toContainerDashboardOverview(overview containerDashboardOverview) containerDashboardOverviewResponse {
	return containerDashboardOverviewResponse{
		AbnormalContainers:    overview.AbnormalContainers,
		CPUTotalPercent:       overview.CPUTotalPercent,
		MemoryTotalLimitBytes: optionalInt64(overview.MemoryTotalLimitBytes),
		MemoryTotalPercent:    overview.MemoryTotalPercent,
		MemoryTotalUsageBytes: optionalInt64(overview.MemoryTotalUsageBytes),
		RunningContainers:     overview.RunningContainers,
	}
}

// toContainerDashboardHotspots 组装容器仪表盘的热点信息响应。
// 它将 CPU 热点和内存热点分别转换为对应的列表响应。
func toContainerDashboardHotspots(hotspots containerDashboardHotspots) containerDashboardHotspotsResponse {
	return containerDashboardHotspotsResponse{
		CPUTop:    toContainerDashboardTopItems(hotspots.CPUTop),
		MemoryTop: toContainerDashboardTopItems(hotspots.MemoryTop),
	}
}

// 每个条目会保留容器标识、镜像、状态、重启次数与资源信息，并将健康状态转换为可选字段。
func toContainerDashboardTopItems(items []containerDashboardTopItem) []containerDashboardTopItemResponse {
	mapped := make([]containerDashboardTopItemResponse, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, containerDashboardTopItemResponse{
			Health:       optionalString(item.Health),
			ID:           item.ID,
			Image:        item.Image,
			Name:         item.Name,
			Resource:     toResourceSummary(item.Resource),
			RestartCount: item.RestartCount,
			ShortID:      item.ShortID,
			State:        item.State,
		})
	}
	return mapped
}

// toContainerDashboardAnomalies 将容器仪表盘的异常项转换为响应列表。
func toContainerDashboardAnomalies(items []containerDashboardAnomalyItem) []containerDashboardAnomalyItemResponse {
	mapped := make([]containerDashboardAnomalyItemResponse, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, containerDashboardAnomalyItemResponse{
			Health:       optionalString(item.Health),
			ID:           item.ID,
			Image:        item.Image,
			Name:         item.Name,
			Resource:     toResourceSummary(item.Resource),
			RestartCount: item.RestartCount,
			ShortID:      item.ShortID,
			State:        item.State,
		})
	}
	return mapped
}

// toResourceSummary 将资源统计信息转换为容器资源摘要响应。
// 该响应包含 CPU、内存、网络、进程及限流统计，并保留采集时间、不可用原因和统计错误信息。
func toResourceSummary(resource ResourceSummary) *containergen.ContainerResourceSummary {
	collectedAt := optionalTime(resource.CollectedAt)
	unavailableReason := optionalString(resource.UnavailableReason)
	statsErrorKey := optionalString(resource.StatsErrorKey)
	statsErrorMessage := optionalString(resource.StatsErrorMessage)
	return &containergen.ContainerResourceSummary{
		Available:                  resource.Available,
		CollectedAt:                collectedAt,
		CpuPercent:                 resource.CPUPercent,
		CpuUsageInKernelmode:       resource.CPUUsageInKernelmode,
		CpuUsageInUsermode:         resource.CPUUsageInUsermode,
		MemoryActiveFile:           resource.MemoryActiveFile,
		MemoryCache:                resource.MemoryCache,
		MemoryInactiveFile:         resource.MemoryInactiveFile,
		MemoryLimitBytes:           resource.MemoryLimitBytes,
		MemoryPercent:              resource.MemoryPercent,
		MemoryPgfault:              resource.MemoryPgfault,
		MemoryPgmajfault:           resource.MemoryPgmajfault,
		MemoryRss:                  resource.MemoryRSS,
		MemoryUsageBytes:           resource.MemoryUsageBytes,
		OnlineCpus:                 resource.OnlineCPUs,
		PidsCurrent:                resource.PIDsCurrent,
		PidsLimit:                  resource.PIDsLimit,
		RxBytes:                    resource.RxBytes,
		RxDropped:                  resource.RxDropped,
		RxErrors:                   resource.RxErrors,
		RxPackets:                  resource.RxPackets,
		StatsAvailable:             resource.StatsAvailable,
		StatsErrorKey:              statsErrorKey,
		StatsErrorMessage:          statsErrorMessage,
		SystemCpuUsage:             resource.SystemCPUUsage,
		ThrottlingPeriods:          resource.ThrottlingPeriods,
		ThrottlingThrottledPeriods: resource.ThrottlingThrottledPeriods,
		ThrottlingThrottledTime:    resource.ThrottlingThrottledTime,
		TotalCpuUsage:              resource.TotalCPUUsage,
		TxBytes:                    resource.TxBytes,
		TxDropped:                  resource.TxDropped,
		TxErrors:                   resource.TxErrors,
		TxPackets:                  resource.TxPackets,
		UnavailableReason:          unavailableReason,
	}
}

// toLogs converts a Logs domain model into a ContainerLogResponse.
func toLogs(logs Logs) containergen.ContainerLogResponse {
	return containergen.ContainerLogResponse{
		Id:         logs.ID,
		Lines:      logs.Lines,
		Name:       optionalString(logs.Name),
		Runtime:    logs.Runtime,
		Since:      optionalString(logs.Since),
		Stderr:     logs.Stderr,
		Stdout:     logs.Stdout,
		Tail:       logs.Tail,
		Timestamps: logs.Timestamps,
		Truncated:  logs.Truncated,
	}
}

// toShellSession 将 ShellSession 域模型转换为 OpenAPI 响应类型 ContainerShellSessionResponse。
func toShellSession(session ShellSession) containergen.ContainerShellSessionResponse {
	return containergen.ContainerShellSessionResponse{
		Cols:         session.Cols,
		Command:      containergen.ContainerShellSessionResponseCommand(session.Command),
		ExpiresAt:    session.ExpiresAt,
		Rows:         session.Rows,
		SessionId:    session.SessionID,
		WebsocketUrl: session.WebSocketURL,
	}
}

type mountUsageListResponse struct {
	Items []mountUsageResponse `json:"items"`
}

type containerDashboardSummaryResponse struct {
	Overview  containerDashboardOverviewResponse      `json:"overview"`
	Hotspots  containerDashboardHotspotsResponse      `json:"hotspots"`
	Anomalies []containerDashboardAnomalyItemResponse `json:"anomalies"`
}

type containerDashboardOverviewResponse struct {
	RunningContainers     int      `json:"running_containers"`
	AbnormalContainers    int      `json:"abnormal_containers"`
	CPUTotalPercent       float64  `json:"cpu_total_percent"`
	MemoryTotalUsageBytes *int64   `json:"memory_total_usage_bytes,omitempty"`
	MemoryTotalLimitBytes *int64   `json:"memory_total_limit_bytes,omitempty"`
	MemoryTotalPercent    *float64 `json:"memory_total_percent,omitempty"`
}

type containerDashboardHotspotsResponse struct {
	CPUTop    []containerDashboardTopItemResponse `json:"cpu_top"`
	MemoryTop []containerDashboardTopItemResponse `json:"memory_top"`
}

type containerDashboardTopItemResponse struct {
	ID           string                                 `json:"id"`
	Name         string                                 `json:"name"`
	ShortID      string                                 `json:"short_id"`
	Image        string                                 `json:"image"`
	State        string                                 `json:"state"`
	Health       *string                                `json:"health,omitempty"`
	RestartCount *int                                   `json:"restart_count,omitempty"`
	Resource     *containergen.ContainerResourceSummary `json:"resource,omitempty"`
}

type containerDashboardAnomalyItemResponse struct {
	ID           string                                 `json:"id"`
	Name         string                                 `json:"name"`
	ShortID      string                                 `json:"short_id"`
	Image        string                                 `json:"image"`
	State        string                                 `json:"state"`
	Health       *string                                `json:"health,omitempty"`
	RestartCount *int                                   `json:"restart_count,omitempty"`
	Resource     *containergen.ContainerResourceSummary `json:"resource,omitempty"`
}

type mountUsageResponse struct {
	MountID     string  `json:"mount_id"`
	ContainerID string  `json:"container_id"`
	Type        string  `json:"type"`
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	SizeBytes   *int64  `json:"size_bytes,omitempty"`
	SizeDisplay *string `json:"size_display,omitempty"`
	Status      string  `json:"status"`
	MeasuredAt  *string `json:"measured_at,omitempty"`
	Message     *string `json:"message,omitempty"`
	SharedHint  *string `json:"shared_hint,omitempty"`
}

// toMountUsageList converts a slice of mount usage items into a response list.
func toMountUsageList(items []MountUsage) mountUsageListResponse {
	mapped := make([]mountUsageResponse, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, toMountUsage(item))
	}
	return mountUsageListResponse{Items: mapped}
}

// toMountUsage converts a MountUsage into a mountUsageResponse. The SizeBytes field is populated only when the usage status indicates the value has been measured.
func toMountUsage(usage MountUsage) mountUsageResponse {
	var sizeBytes *int64
	if usage.Status == containerMountUsageStatusMeasured {
		sizeBytes = &usage.SizeBytes
	}
	return mountUsageResponse{
		MountID:     usage.MountID,
		ContainerID: usage.ContainerID,
		Type:        usage.Type,
		Source:      usage.Source,
		Destination: usage.Destination,
		SizeBytes:   sizeBytes,
		SizeDisplay: optionalString(usage.SizeDisplay),
		Status:      usage.Status,
		MeasuredAt:  optionalString(usage.MeasuredAt),
		Message:     optionalString(usage.Message),
		SharedHint:  optionalString(usage.SharedHint),
	}
}

// toContainerAction converts an action result to its OpenAPI response representation.
func toContainerAction(result ActionResult) containergen.ContainerActionResponse {
	return containergen.ContainerActionResponse{
		Action:       containergen.ContainerActionResponseAction(result.Action),
		Id:           result.ID,
		Message:      optionalString(result.Message),
		MessageKey:   optionalString(result.MessageKey),
		Name:         optionalString(result.Name),
		Result:       containergen.ContainerActionResponseResult(result.Result),
		Runtime:      result.Runtime,
		StatusAfter:  result.StatusAfter,
		StatusBefore: optionalString(result.StatusBefore),
	}
}

func toContainerBatchAction(result BatchActionResult) containergen.ContainerBatchActionResponse {
	items := make([]containergen.ContainerBatchActionItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, containergen.ContainerBatchActionItem{
			Id:         item.ID,
			Name:       optionalString(item.Name),
			Action:     containergen.ContainerBatchActionItemAction(item.Action),
			Success:    item.Success,
			ErrorCode:  optionalString(item.ErrorCode),
			MessageKey: optionalString(item.MessageKey),
			Message:    optionalString(item.Message),
		})
	}
	return containergen.ContainerBatchActionResponse{
		Total:        result.Total,
		SuccessCount: result.SuccessCount,
		FailedCount:  result.FailedCount,
		RequestId:    optionalString(result.RequestID),
		Items:        items,
	}
}

// toRuntimeInfo 将运行时信息转换为容器运行时信息响应。
func toRuntimeInfo(info RuntimeInfo) containergen.ContainerRuntimeInfo {
	return containergen.ContainerRuntimeInfo{
		ApiVersion:        optionalString(info.APIVersion),
		Architecture:      optionalString(info.Architecture),
		ContainersRunning: optionalInt(info.ContainersRunning),
		ContainersTotal:   optionalInt(info.ContainersTotal),
		Endpoint:          info.Endpoint,
		OperatingSystem:   optionalString(info.OperatingSystem),
		Runtime:           info.Runtime,
		ServerVersion:     optionalString(info.ServerVersion),
		Status:            containergen.ContainerRuntimeInfoStatus(info.Status),
	}
}

// toOrchestratorInfo 将编排器信息转换为 OpenAPI 容器编排器响应类型。
func toOrchestratorInfo(info OrchestratorInfo) *containergen.ContainerOrchestratorInfo {
	info = normalizedOrchestratorInfo(info)
	return &containergen.ContainerOrchestratorInfo{
		ActionLevel:        containergen.ContainerOrchestratorInfoActionLevel(info.ActionLevel),
		BatchActionAllowed: info.BatchActionAllowed,
		Confidence:         containergen.ContainerOrchestratorInfoConfidence(info.Confidence),
		ConfigFiles:        optionalStringSlice(info.ConfigFiles),
		Container:          optionalString(info.Container),
		DisplayName:        optionalString(info.DisplayName),
		GroupDisplayName:   optionalString(info.GroupDisplayName),
		GroupScopeKind:     optionalOrchestratorGroupScopeKind(info.GroupScopeKind),
		GroupValue:         optionalString(info.GroupValue),
		Managed:            info.Managed,
		MemberDisplayName:  optionalString(info.MemberDisplayName),
		MemberScopeKind:    optionalOrchestratorMemberScopeKind(info.MemberScopeKind),
		MemberValue:        optionalString(info.MemberValue),
		Namespace:          optionalString(info.Namespace),
		Pod:                optionalString(info.Pod),
		Project:            optionalString(info.Project),
		RecommendedAction:  optionalString(info.RecommendedAction),
		Service:            optionalString(info.Service),
		Stack:              optionalString(info.Stack),
		Task:               optionalString(info.Task),
		Type:               containergen.ContainerOrchestratorInfoType(info.Type),
		Warnings:           append([]string(nil), info.Warnings...),
		WorkingDir:         optionalString(info.WorkingDir),
	}
}

// optionalOrchestratorGroupScopeKind 将字符串转换为编排器组作用域类型的可选值，去除空白后若为空则返回 nil，否则返回指向转换后枚举值的指针。
func optionalOrchestratorGroupScopeKind(value string) *containergen.ContainerOrchestratorInfoGroupScopeKind {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	mapped := containergen.ContainerOrchestratorInfoGroupScopeKind(value)
	return &mapped
}

// optionalOrchestratorMemberScopeKind converts the given string to a ContainerOrchestratorInfoMemberScopeKind enum value, returning a pointer to the enum or nil if the string is empty after trimming whitespace.
func optionalOrchestratorMemberScopeKind(value string) *containergen.ContainerOrchestratorInfoMemberScopeKind {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	mapped := containergen.ContainerOrchestratorInfoMemberScopeKind(value)
	return &mapped
}

// toPorts 将端口信息转换为容器端口响应。
func toPorts(ports []Port) []containergen.ContainerPort {
	mapped := make([]containergen.ContainerPort, 0, len(ports))
	for _, port := range ports {
		mapped = append(mapped, containergen.ContainerPort{
			Ip:          optionalString(port.IP),
			PrivatePort: port.PrivatePort,
			PublicPort:  port.PublicPort,
			Type:        containergen.ContainerPortType(port.Type),
		})
	}
	return mapped
}

// ToMounts maps a slice of internal Mount objects to OpenAPI-generated ContainerMount response objects.
func toMounts(mounts []Mount) []containergen.ContainerMount {
	mapped := make([]containergen.ContainerMount, 0, len(mounts))
	for _, mount := range mounts {
		mapped = append(mapped, containergen.ContainerMount{
			Destination: mount.Destination,
			Mode:        mount.Mode,
			MountId:     mount.ID,
			Name:        optionalString(mount.Name),
			ReadOnly:    mount.ReadOnly,
			Source:      optionalString(mount.Source),
			Type:        mount.Type,
			Usage:       toGeneratedMountUsage(mount.Usage),
		})
	}
	return mapped
}

// toGeneratedMountUsage converts a MountUsage into a ContainerMountUsage response. The SizeBytes field is populated only when the usage status indicates a measurement is available.
func toGeneratedMountUsage(usage *MountUsage) *containergen.ContainerMountUsage {
	if usage == nil {
		return nil
	}
	var sizeBytes *int64
	if usage.Status == containerMountUsageStatusMeasured {
		sizeBytes = &usage.SizeBytes
	}
	return &containergen.ContainerMountUsage{
		ContainerId: usage.ContainerID,
		Destination: usage.Destination,
		MeasuredAt:  optionalTime(usage.MeasuredAt),
		Message:     optionalString(usage.Message),
		MountId:     usage.MountID,
		SharedHint:  optionalString(usage.SharedHint),
		SizeBytes:   sizeBytes,
		SizeDisplay: optionalString(usage.SizeDisplay),
		Source:      usage.Source,
		Status:      containergen.ContainerMountUsageStatus(usage.Status),
		Type:        usage.Type,
	}
}

// toNetworks converts a slice of networks into generated container network response types.
func toNetworks(networks []Network) []containergen.ContainerNetwork {
	mapped := make([]containergen.ContainerNetwork, 0, len(networks))
	for _, network := range networks {
		mapped = append(mapped, containergen.ContainerNetwork{
			EndpointId: optionalString(network.EndpointID),
			Gateway:    optionalString(network.Gateway),
			IpAddress:  optionalString(network.IPAddress),
			MacAddress: optionalString(network.MacAddress),
			Name:       network.Name,
			NetworkId:  optionalString(network.NetworkID),
		})
	}
	return mapped
}

func optionalNetworks(networks []Network) *[]containergen.ContainerNetwork {
	if len(networks) == 0 {
		return nil
	}
	mapped := toNetworks(networks)
	return &mapped
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalStringSlice(values []string) *[]string {
	if len(values) == 0 {
		return nil
	}
	return &values
}

func optionalStringMap(values map[string]string) *map[string]string {
	if len(values) == 0 {
		return nil
	}
	return &values
}

// optionalInt 将非零整数转换为指针。
//
// @param value 要转换的整数值。
// @returns value 为 0 时返回 nil，否则返回指向该值的指针。
func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

// optionalInt64 返回该值的指针，包括 0。
func optionalInt64(value int64) *int64 {
	return &value
}

// optionalBool 返回指向给定布尔值的指针。
func optionalBool(value bool) *bool {
	return &value
}

func optionalSummaryHealth(value string) *containergen.ContainerSummaryHealth {
	if value == "" {
		return nil
	}
	health := containergen.ContainerSummaryHealth(value)
	return &health
}

func optionalDetailHealth(value string) *containergen.ContainerDetailHealth {
	if value == "" {
		return nil
	}
	health := containergen.ContainerDetailHealth(value)
	return &health
}

func optionalTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed := mustTime(value)
	return &parsed
}

func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
