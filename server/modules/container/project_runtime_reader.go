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
	offset := 0
	for {
		items, listErr := runtime.List(ctx, ListQuery{
			Limit:           maxContainerListLimit,
			Offset:          offset,
			Orchestrator:    containerOrchestratorCompose,
			SourceScopeKind: composeProjectScopeKind,
			SourceScope:     canonicalProjectName,
		})
		if listErr != nil {
			return summary, listErr
		}
		appendProjectMembers(&summary, items, canonicalProjectName)
		if len(items) < maxContainerListLimit {
			break
		}
		offset += len(items)
	}
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

// appendProjectMembers 将匹配指定项目的运行时摘要转换为成员列表，并更新运行与停止数量。
// 仅会追加与 canonicalProjectName 对应的条目；状态为 running 的成员计入运行数，其余成员计入停止数。
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

// toProjectMember 将运行时摘要项转换为指定项目的成员信息。
// 仅当 item 的 ComposeProject 去除空格后与 canonicalProjectName 忽略大小写相等时，才返回转换后的成员信息。
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
