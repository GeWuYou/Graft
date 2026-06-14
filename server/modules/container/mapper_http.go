// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"time"

	containergen "graft/server/internal/contract/openapi/generated"
)

func toContainerListResponse(runtime RuntimeInfo, items []Summary) containergen.ContainerListResponse {
	mapped := make([]containergen.ContainerSummary, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, toSummary(item))
	}
	return containergen.ContainerListResponse{
		Items:   mapped,
		Runtime: toRuntimeInfo(runtime),
	}
}

func toSummary(item Summary) containergen.ContainerSummary {
	return containergen.ContainerSummary{
		Id:            item.ID,
		Names:         item.Names,
		Image:         item.Image,
		ImageId:       optionalString(item.ImageID),
		Labels:        optionalStringMap(item.Labels),
		Ports:         toPorts(item.Ports),
		RestartPolicy: optionalString(item.RestartPolicy),
		Runtime:       item.Runtime,
		CreatedAt:     mustTime(item.CreatedAt),
		StartedAt:     optionalTime(item.StartedAt),
		State:         containergen.ContainerSummaryState(item.State),
		Status:        item.Status,
	}
}

func toDetail(detail Detail) containergen.ContainerDetail {
	return containergen.ContainerDetail{
		Command:          optionalString(detail.Command),
		CreatedAt:        mustTime(detail.CreatedAt),
		Entrypoint:       optionalStringSlice(detail.Entrypoint),
		Id:               detail.ID,
		Image:            detail.Image,
		ImageId:          optionalString(detail.ImageID),
		InspectUpdatedAt: optionalTime(detail.InspectUpdatedAt),
		Labels:           optionalStringMap(detail.Labels),
		Mounts:           toMounts(detail.Mounts),
		Names:            detail.Names,
		Networks:         toNetworks(detail.Networks),
		Ports:            toPorts(detail.Ports),
		RestartPolicy:    optionalString(detail.RestartPolicy),
		Runtime:          detail.Runtime,
		RuntimeInfo:      toRuntimeInfo(detail.RuntimeInfo),
		StartedAt:        optionalTime(detail.StartedAt),
		State:            containergen.ContainerDetailState(detail.State),
		Status:           detail.Status,
		WorkingDir:       optionalString(detail.WorkingDir),
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
		Name:         optionalString(result.Name),
		Result:       containergen.ContainerActionResponseResult(result.Result),
		Runtime:      result.Runtime,
		StatusAfter:  result.StatusAfter,
		StatusBefore: optionalString(result.StatusBefore),
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
		return time.Unix(0, 0).UTC()
	}
	return parsed
}
