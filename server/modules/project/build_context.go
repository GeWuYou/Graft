package project

import (
	"context"
	"errors"
	"math"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
)

// ResolveApplicationBuildContext 在不暴露 Project 持久化模型的前提下返回 Build 所需的已授权工作区事实。
//
//nolint:cyclop // 授权、存活应用和可用 Docker target 必须在同一资源边界内逐项确认。
func (s *Service) ResolveApplicationBuildContext(ctx context.Context, applicationID uint64) (moduleapi.ApplicationBuildContext, error) {
	if applicationID == 0 || s == nil || s.authorizer == nil {
		return moduleapi.ApplicationBuildContext{}, errProjectInvalidArgument
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.User == nil {
		return moduleapi.ApplicationBuildContext{}, moduleapi.ErrUnauthenticated
	}
	if err := s.authorizer.Authorize(ctx, auth, projectcontract.ApplicationViewPermission.String()); err != nil {
		return moduleapi.ApplicationBuildContext{}, err
	}
	aggregate, err := s.getAggregate(ctx, applicationID)
	if err != nil {
		return moduleapi.ApplicationBuildContext{}, err
	}
	item := aggregate.Application
	if item.RuntimeTargetID == nil || item.WorkspacePath == "" {
		return moduleapi.ApplicationBuildContext{}, errors.New("application build context is unavailable")
	}
	if *item.RuntimeTargetID > math.MaxInt64 {
		return moduleapi.ApplicationBuildContext{}, errProjectInvalidArgument
	}
	targetID := int64(*item.RuntimeTargetID) // #nosec G115 -- 已在转换前限制为 PostgreSQL bigint 可表示范围。
	target, err := s.runtimeTargets.ReadComposeTarget(ctx, &targetID)
	if err != nil {
		return moduleapi.ApplicationBuildContext{}, err
	}
	if target.Provider != "docker" {
		return moduleapi.ApplicationBuildContext{}, errors.New("application runtime target does not support Docker builds")
	}
	return moduleapi.ApplicationBuildContext{
		ApplicationID: applicationID, DisplayName: item.DisplayName, WorkspaceRoot: item.WorkspacePath,
		RuntimeTargetID: *item.RuntimeTargetID, RuntimeTargetName: target.DisplayName, RuntimeProvider: target.Provider, CanBuild: true,
	}, nil
}
