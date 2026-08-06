package build

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
	buildcontract "graft/server/modules/build/contract"
)

type recordingBuildTaskAuthorizer struct{ permission string }

func (a *recordingBuildTaskAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	a.permission = permission
	return nil
}

func TestBuildTaskOwnerAuthorizerMapsPromotionLifecycleActions(t *testing.T) {
	authorizer := &recordingBuildTaskAuthorizer{}
	owner := moduleapi.TaskOwner{Type: artifactPromotionTaskOwnerType, ID: "publication_1"}
	for _, tc := range []struct {
		action moduleapi.TaskOwnerAction
		want   string
	}{
		{action: moduleapi.TaskOwnerActionView, want: buildcontract.BuildReadPermission},
		{action: moduleapi.TaskOwnerActionCancel, want: buildcontract.BuildCancelPermission},
		{action: moduleapi.TaskOwnerActionRetry, want: buildcontract.BuildRetryPermission},
	} {
		if err := (buildTaskOwnerAuthorizer{ownerType: artifactPromotionTaskOwnerType, authorizer: authorizer}).AuthorizeTaskOwner(context.Background(), &moduleapi.CurrentUser{ID: 7}, tc.action, owner); err != nil {
			t.Fatalf("authorize %q: %v", tc.action, err)
		}
		if authorizer.permission != tc.want {
			t.Fatalf("permission for %q = %q, want %q", tc.action, authorizer.permission, tc.want)
		}
	}
	if err := (buildTaskOwnerAuthorizer{ownerType: artifactPromotionTaskOwnerType, authorizer: authorizer}).AuthorizeTaskOwner(context.Background(), &moduleapi.CurrentUser{ID: 7}, moduleapi.TaskOwnerActionView, moduleapi.TaskOwner{Type: buildTaskOwnerType}); !errors.Is(err, moduleapi.ErrUnauthenticated) {
		t.Fatalf("unexpected owner authorization error: %v", err)
	}
}
