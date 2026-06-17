// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"time"

	containergen "graft/server/internal/contract/openapi/generated"
)

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

func toDetail(detail Detail) containergen.ContainerDetail {
	return containergen.ContainerDetail{
		CanRemove:         optionalBool(detail.CanRemove),
		CanRestart:        optionalBool(detail.CanRestart),
		CanStart:          optionalBool(detail.CanStart),
		CanStop:           optionalBool(detail.CanStop),
		Command:           optionalStringSlice(detail.Command),
		ComposeProject:    optionalString(detail.ComposeProject),
		ComposeService:    optionalString(detail.ComposeService),
		CreatedAt:         mustTime(detail.CreatedAt),
		Entrypoint:        optionalStringSlice(detail.Entrypoint),
		Environment:       optionalEnvironment(detail.Environment),
		EnvironmentPolicy: optionalEnvironmentPolicy(detail.EnvironmentPolicy),
		Health:            optionalDetailHealth(detail.Health),
		Healthcheck:       optionalHealthcheck(detail.Healthcheck),
		Id:                detail.ID,
		Image:             detail.Image,
		ImageId:           optionalString(detail.ImageID),
		InspectUpdatedAt:  optionalTime(detail.InspectUpdatedAt),
		Labels:            optionalStringMap(detail.Labels),
		LastExitCode:      detail.LastExitCode,
		Mounts:            toMounts(detail.Mounts),
		Name:              detail.Name,
		Names:             detail.Names,
		NetworkSummary:    optionalString(detail.NetworkSummary),
		Networks:          toNetworks(detail.Networks),
		OomKilled:         detail.OOMKilled,
		Ports:             toPorts(detail.Ports),
		PrimaryIp:         optionalString(detail.PrimaryIP),
		Resource:          toResourceSummary(detail.Resource),
		RestartCount:      detail.RestartCount,
		RestartPolicy:     optionalString(detail.RestartPolicy),
		Runtime:           detail.Runtime,
		RuntimeInfo:       toRuntimeInfo(detail.RuntimeInfo),
		ShortId:           detail.ShortID,
		StartedAt:         optionalTime(detail.StartedAt),
		State:             containergen.ContainerDetailState(detail.State),
		Status:            detail.Status,
		WorkingDir:        optionalString(detail.WorkingDir),
	}
}

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

func optionalEnvironment(environment []EnvironmentVariable) *[]containergen.ContainerEnvironmentEntry {
	if len(environment) == 0 {
		return nil
	}
	mapped := make([]containergen.ContainerEnvironmentEntry, 0, len(environment))
	for _, item := range environment {
		mapped = append(mapped, containergen.ContainerEnvironmentEntry{
			Key:       item.Key,
			Masked:    item.Masked,
			Sensitive: item.Sensitive,
			Source:    containergen.ContainerEnvironmentEntrySource(item.Source),
			Value:     optionalString(item.Value),
		})
	}
	return &mapped
}

func optionalEnvironmentPolicy(policy string) containergen.ContainerDetailEnvironmentPolicy {
	normalized := normalizeEnvironmentPolicy(policy)
	value := containergen.ContainerDetailEnvironmentPolicy(normalized.String())
	return value
}

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

func toResourceSummary(resource ResourceSummary) *containergen.ContainerResourceSummary {
	unavailableReason := optionalString(resource.UnavailableReason)
	statsErrorKey := optionalString(resource.StatsErrorKey)
	statsErrorMessage := optionalString(resource.StatsErrorMessage)
	return &containergen.ContainerResourceSummary{
		Available:                  resource.Available,
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

func toMounts(mounts []Mount) []containergen.ContainerMount {
	mapped := make([]containergen.ContainerMount, 0, len(mounts))
	for _, mount := range mounts {
		mapped = append(mapped, containergen.ContainerMount{
			Destination: mount.Destination,
			Mode:        mount.Mode,
			Name:        optionalString(mount.Name),
			ReadOnly:    mount.ReadOnly,
			Source:      optionalString(mount.Source),
			Type:        mount.Type,
		})
	}
	return mapped
}

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

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

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
