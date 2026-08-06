package build

import (
	"context"
	"errors"

	"graft/server/internal/moduleapi"
	buildcontract "graft/server/modules/build/contract"
)

// buildTaskOwnerAuthorizer 将 Task Runtime 的通用操作映射到 Build 已声明的权限，
// 防止 Promotion 任务绕过所属模块的读取、取消和人工重试边界。
type buildTaskOwnerAuthorizer struct {
	ownerType  string
	authorizer moduleapi.Authorizer
}

func (a buildTaskOwnerAuthorizer) OwnerType() string { return a.ownerType }

func (a buildTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.authorizer == nil || owner.Type != a.ownerType {
		return moduleapi.ErrUnauthenticated
	}
	permission := buildcontract.BuildReadPermission
	switch action {
	case moduleapi.TaskOwnerActionCancel:
		permission = buildcontract.BuildCancelPermission
	case moduleapi.TaskOwnerActionRetry:
		permission = buildcontract.BuildRetryPermission
	case moduleapi.TaskOwnerActionView:
	default:
		return errors.New("build task owner action is unsupported")
	}
	return a.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permission)
}
