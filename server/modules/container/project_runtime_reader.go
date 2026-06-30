package container

import (
	"context"
	"sort"
	"strings"

	"graft/server/internal/moduleapi"
)

type containerProjectRuntimeReader struct {
	service *service
}

func (r containerProjectRuntimeReader) ListProjectMembers(
	ctx context.Context,
	hostScope string,
	canonicalProjectName string,
) (moduleapi.ContainerProjectRuntimeSummary, error) {
	if r.service == nil {
		return moduleapi.ContainerProjectRuntimeSummary{}, errRuntimeDisabled
	}
	summary := moduleapi.ContainerProjectRuntimeSummary{
		CanonicalProjectName: strings.TrimSpace(canonicalProjectName),
		Members:              []moduleapi.ContainerProjectMember{},
	}
	if strings.TrimSpace(hostScope) == "" || strings.TrimSpace(canonicalProjectName) == "" {
		return summary, nil
	}
	runtime, err := r.service.runtimeForRequest()
	if err != nil {
		return summary, err
	}
	items, err := runtime.List(ctx, ListQuery{
		Limit:           maxContainerListLimit,
		Offset:          0,
		Orchestrator:    containerOrchestratorCompose,
		SourceScopeKind: composeProjectScopeKind,
		SourceScope:     canonicalProjectName,
	})
	if err != nil {
		return summary, err
	}
	appendProjectMembers(&summary, items, canonicalProjectName)
	sort.Slice(summary.Members, func(i, j int) bool {
		if summary.Members[i].ServiceName == summary.Members[j].ServiceName {
			if summary.Members[i].ContainerName == summary.Members[j].ContainerName {
				return summary.Members[i].ContainerID < summary.Members[j].ContainerID
			}
			return summary.Members[i].ContainerName < summary.Members[j].ContainerName
		}
		return summary.Members[i].ServiceName < summary.Members[j].ServiceName
	})
	return summary, nil
}

func appendProjectMembers(
	summary *moduleapi.ContainerProjectRuntimeSummary,
	items []Summary,
	canonicalProjectName string,
) {
	for _, item := range items {
		member, ok := toProjectMember(item, canonicalProjectName)
		if !ok {
			continue
		}
		summary.Members = append(summary.Members, member)
		if member.CanonicalState == "running" {
			summary.RunningCount++
			continue
		}
		summary.StoppedCount++
	}
}

func toProjectMember(item Summary, canonicalProjectName string) (moduleapi.ContainerProjectMember, bool) {
	if !strings.EqualFold(strings.TrimSpace(item.ComposeProject), canonicalProjectName) {
		return moduleapi.ContainerProjectMember{}, false
	}
	return moduleapi.ContainerProjectMember{
		ContainerID:    strings.TrimSpace(item.ID),
		ContainerName:  strings.TrimSpace(item.Name),
		ServiceName:    strings.TrimSpace(item.ComposeService),
		CanonicalState: normalizeContainerState(item.State),
	}, true
}

var _ moduleapi.ContainerProjectRuntimeReader = containerProjectRuntimeReader{}
