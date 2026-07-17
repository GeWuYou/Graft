package project

import (
	"context"
	"strings"

	projectstore "graft/server/modules/project/store"
)

const maxComposeContextReferenceCount = 100

// ComposeContextReferenceRequest 描述容器页批量解析 Compose Application 引用的输入。
type ComposeContextReferenceRequest struct {
	RuntimeTargetID    int64
	ComposeProjectName string
}

// ComposeContextReferenceResult 是供容器页渲染与详情导航使用的 Application 引用。
type ComposeContextReferenceResult struct {
	RuntimeTargetID    int64
	ComposeProjectName string
	ApplicationID      string
	DisplayName        string
}

// ResolveComposeContextReferences 以运行目标和 Compose 项目名的规范联合身份解析存活 Application。
func (s *Service) ResolveComposeContextReferences(
	ctx context.Context,
	requests []ComposeContextReferenceRequest,
) ([]ComposeContextReferenceResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return nil, err
	}
	if len(requests) == 0 || len(requests) > maxComposeContextReferenceCount {
		return nil, errProjectInvalidArgument
	}

	contexts, err := normalizeComposeContextReferenceRequests(requests)
	if err != nil {
		return nil, err
	}

	lookup, ok := repository.(projectstore.ComposeContextReferenceRepository)
	if !ok {
		return nil, errProjectServiceUnavailable
	}
	items, err := lookup.ResolveComposeContexts(ctx, contexts)
	if err != nil {
		return nil, mapStoreError(err)
	}
	result := make([]ComposeContextReferenceResult, 0, len(items))
	for _, item := range items {
		result = append(result, ComposeContextReferenceResult{
			RuntimeTargetID:    item.RuntimeTargetID,
			ComposeProjectName: item.ComposeProjectName,
			ApplicationID:      item.ApplicationID,
			DisplayName:        item.DisplayName,
		})
	}
	return result, nil
}

func normalizeComposeContextReferenceRequests(requests []ComposeContextReferenceRequest) ([]projectstore.ComposeContext, error) {
	contexts := make([]projectstore.ComposeContext, 0, len(requests))
	seen := make(map[projectstore.ComposeContext]struct{}, len(requests))
	for _, request := range requests {
		context := projectstore.ComposeContext{
			RuntimeTargetID:    request.RuntimeTargetID,
			ComposeProjectName: strings.TrimSpace(request.ComposeProjectName),
		}
		if context.RuntimeTargetID < 1 || context.ComposeProjectName == "" {
			return nil, errProjectInvalidArgument
		}
		if _, ok := seen[context]; ok {
			continue
		}
		seen[context] = struct{}{}
		contexts = append(contexts, context)
	}
	return contexts, nil
}
