package project

import (
	"context"
	"strconv"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
)

type projectTaskOwnerAuthorizer struct{ service *Service }

func (a projectTaskOwnerAuthorizer) OwnerType() string { return projectTaskOwnerType }

func (a projectTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.service == nil || a.service.authorizer == nil {
		return errProjectActorAttribution
	}
	projectID, err := strconv.ParseUint(owner.ID, 10, 64)
	if err != nil || projectID == 0 {
		return errProjectInvalidArgument
	}
	if _, err := a.service.Get(ctx, projectID); err != nil {
		return err
	}
	permission := projectcontract.ProjectViewPermission.String()
	if action == moduleapi.TaskOwnerActionCancel || action == moduleapi.TaskOwnerActionRetry {
		permission = projectcontract.ProjectLifecyclePermission.String()
	}
	return a.service.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permission)
}
