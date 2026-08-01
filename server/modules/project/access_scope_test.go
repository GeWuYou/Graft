package project

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
	projectstore "graft/server/modules/project/store"
)

type fixedPermissionScopeResolver moduleapi.PermissionScope

func (r fixedPermissionScopeResolver) ResolvePermissionScope(context.Context, uint64, string) (moduleapi.PermissionScope, error) {
	return moduleapi.PermissionScope(r), nil
}

func TestEnsureApplicationScopeRestrictsOwnedPermissionToCreator(t *testing.T) {
	t.Parallel()
	creator := uint64(7)
	service := &Service{permissionScopes: fixedPermissionScopeResolver(moduleapi.PermissionScopeOwned)}
	aggregate := projectstore.ApplicationAggregate{Application: projectstore.Application{CreatedBy: &creator}}
	ownedContext := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: creator}})
	if err := service.ensureApplicationScope(ownedContext, aggregate, "application.lifecycle"); err != nil {
		t.Fatalf("owner access = %v", err)
	}
	foreignContext := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 8}})
	if err := service.ensureApplicationScope(foreignContext, aggregate, "application.lifecycle"); !errors.Is(err, moduleapi.ErrPermissionDenied) {
		t.Fatalf("foreign owned access = %v, want permission denied", err)
	}
}

func TestEnsureApplicationScopeAllowsAllPermission(t *testing.T) {
	t.Parallel()
	service := &Service{permissionScopes: fixedPermissionScopeResolver(moduleapi.PermissionScopeAll)}
	contextWithActor := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 8}})
	if err := service.ensureApplicationScope(contextWithActor, projectstore.ApplicationAggregate{}, "application.view"); err != nil {
		t.Fatalf("all scope access = %v", err)
	}
}
