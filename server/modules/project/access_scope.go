package project

import (
	"context"
	"errors"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// permissionScope 在常规 HTTP 权限校验后读取 RBAC 的有效资源范围。
func (s *Service) permissionScope(ctx context.Context, permission string) (moduleapi.PermissionScope, error) {
	if s == nil {
		return moduleapi.PermissionScopeNone, moduleapi.ErrPermissionDenied
	}
	// 模块装配强制提供该依赖；全范围回退仅保留无 HTTP 主体的隔离服务测试兼容性。
	if s.permissionScopes == nil {
		return moduleapi.PermissionScopeAll, nil
	}
	actorID, ok := currentProjectActorID(ctx)
	if !ok {
		return moduleapi.PermissionScopeNone, moduleapi.ErrPermissionDenied
	}
	return s.permissionScopes.ResolvePermissionScope(ctx, actorID, permission)
}

func currentProjectActorID(ctx context.Context) (uint64, bool) {
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.User == nil || auth.User.ID == 0 {
		return 0, false
	}
	return auth.User.ID, true
}

func (s *Service) ensureApplicationScope(ctx context.Context, aggregate projectstore.ApplicationAggregate, permission string) error {
	scope, err := s.permissionScope(ctx, permission)
	if err != nil || scope == moduleapi.PermissionScopeNone {
		if err != nil {
			return err
		}
		return moduleapi.ErrPermissionDenied
	}
	if scope == moduleapi.PermissionScopeAll {
		return nil
	}
	actorID, ok := currentProjectActorID(ctx)
	if !ok || aggregate.Application.CreatedBy == nil || *aggregate.Application.CreatedBy != actorID {
		return moduleapi.ErrPermissionDenied
	}
	return nil
}

func (s *Service) listComposeTargetsForScope(ctx context.Context, scope moduleapi.PermissionScope) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	if scope == moduleapi.PermissionScopeAll {
		return s.listComposeTargets(ctx)
	}
	if scope != moduleapi.PermissionScopeOwned || s.runtimeTargetAssignments == nil {
		return []moduleapi.ComposeRuntimeTargetSummary{}, nil
	}
	actorID, ok := currentProjectActorID(ctx)
	if !ok {
		return nil, moduleapi.ErrPermissionDenied
	}
	return s.runtimeTargetAssignments.ListAssignedComposeTargets(ctx, actorID)
}

func (s *Service) ensureComposeTargetUse(ctx context.Context, targetID uint64) error {
	scope, err := s.permissionScope(ctx, projectcontract.ApplicationCreatePermission.String())
	if err != nil || scope == moduleapi.PermissionScopeNone {
		if err != nil {
			return err
		}
		return moduleapi.ErrPermissionDenied
	}
	if scope == moduleapi.PermissionScopeAll {
		return nil
	}
	if s.runtimeTargetAssignments == nil {
		return errors.New("runtime target deployment assignment reader is unavailable")
	}
	actorID, ok := currentProjectActorID(ctx)
	if !ok {
		return moduleapi.ErrPermissionDenied
	}
	allowed, err := s.runtimeTargetAssignments.CanUseComposeTarget(ctx, actorID, targetID)
	if err != nil {
		return err
	}
	if !allowed {
		return moduleapi.ErrPermissionDenied
	}
	return nil
}
