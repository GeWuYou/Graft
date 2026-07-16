package project

import (
	"context"
	"testing"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type recordingProjectTaskAuthorizer struct{ permission string }

func (a *recordingProjectTaskAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	a.permission = permission
	return nil
}

func TestProjectTaskOwnerAuthorizerUsesPublicApplicationID(t *testing.T) {
	t.Parallel()
	applicationID := "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"
	service, err := NewService(applicationLookupRepository{aggregate: projectstore.ApplicationAggregate{
		Application: projectstore.Application{ApplicationRecordID: 42, ApplicationID: applicationID},
	}})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	authorizer := &recordingProjectTaskAuthorizer{}
	service.SetAuthorizer(authorizer)

	err = (projectTaskOwnerAuthorizer{service: service}).AuthorizeTaskOwner(
		context.Background(),
		&moduleapi.CurrentUser{ID: 7},
		moduleapi.TaskOwnerActionView,
		moduleapi.TaskOwner{Type: applicationTaskOwnerType, ID: applicationID},
	)
	if err != nil {
		t.Fatalf("authorize public task owner: %v", err)
	}
	if authorizer.permission != projectcontract.ApplicationViewPermission.String() {
		t.Fatalf("permission = %q", authorizer.permission)
	}

	err = (projectTaskOwnerAuthorizer{service: service}).AuthorizeTaskOwner(
		context.Background(),
		&moduleapi.CurrentUser{ID: 7},
		moduleapi.TaskOwnerActionView,
		moduleapi.TaskOwner{Type: applicationTaskOwnerType, ID: "42"},
	)
	if err == nil {
		t.Fatal("numeric project ID must not authorize a Task owner")
	}
}
