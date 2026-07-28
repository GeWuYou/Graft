package update

import (
	"context"
	"errors"

	"graft/server/internal/moduleapi"
	updatecontract "graft/server/modules/update/contract"
)

const platformUpdateTaskOwnerType = "platform_update"

type platformUpdateTaskOwnerAuthorizer struct{ authorizer moduleapi.Authorizer }

func (platformUpdateTaskOwnerAuthorizer) OwnerType() string { return platformUpdateTaskOwnerType }

func (a platformUpdateTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.authorizer == nil {
		return moduleapi.ErrUnauthenticated
	}
	if owner.Type != platformUpdateTaskOwnerType {
		return errors.New("platform update task owner type is invalid")
	}
	if !runnerOperationID.MatchString(owner.ID) {
		return errors.New("platform update task operation identity is invalid")
	}

	permission := updatecontract.UpdateReadPermission.String()
	switch action {
	case moduleapi.TaskOwnerActionView:
	case moduleapi.TaskOwnerActionCancel, moduleapi.TaskOwnerActionRetry:
		permission = updatecontract.UpdateManagePermission.String()
	default:
		return errors.New("platform update task owner action is unsupported")
	}
	return a.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permission)
}
