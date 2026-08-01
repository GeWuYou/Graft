package project

import (
	"context"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
)

type projectTaskOwnerAuthorizer struct{ service *Service }

func (a projectTaskOwnerAuthorizer) OwnerType() string { return applicationTaskOwnerType }

func (a projectTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.service == nil || a.service.authorizer == nil {
		return errProjectActorAttribution
	}
	projectID, err := a.service.ResolveApplicationID(ctx, owner.ID)
	if err != nil {
		return err
	}
	permission := projectcontract.ApplicationViewPermission.String()
	if action == moduleapi.TaskOwnerActionCancel || action == moduleapi.TaskOwnerActionRetry {
		permission = projectcontract.ApplicationLifecyclePermission.String()
	}
	if err := a.service.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permission); err != nil {
		return err
	}
	if a.service.permissionScopes == nil {
		return nil
	}
	aggregate, err := a.service.getAggregate(ctx, projectID)
	if err != nil {
		return err
	}
	return a.service.ensureApplicationScope(ctx, aggregate, permission)
}
