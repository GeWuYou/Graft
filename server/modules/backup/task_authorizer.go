package backup

import (
	"context"
	"errors"

	"graft/server/internal/moduleapi"
	backupcontract "graft/server/modules/backup/contract"
)

type backupTaskOwnerAuthorizer struct{ authorizer moduleapi.Authorizer }

func (backupTaskOwnerAuthorizer) OwnerType() string { return backupTaskOwnerType }

func (a backupTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.authorizer == nil || owner.Type != backupTaskOwnerType || owner.ID != backupTaskOwnerID {
		return moduleapi.ErrUnauthenticated
	}
	permission := backupcontract.BackupReadPermission
	if action == moduleapi.TaskOwnerActionCancel || action == moduleapi.TaskOwnerActionRetry {
		permission = backupcontract.BackupCreatePermission
	}
	if action != moduleapi.TaskOwnerActionView && action != moduleapi.TaskOwnerActionCancel && action != moduleapi.TaskOwnerActionRetry {
		return errors.New("backup task owner action is unsupported")
	}
	return a.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permission)
}
