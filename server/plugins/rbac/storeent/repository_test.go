package storeent

import (
	"context"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/ent/enttest"
	rbacstore "graft/server/plugins/rbac/store"
)

func TestRepositoryRejectsInvalidID(t *testing.T) {
	repo := &repository{}

	if _, err := repo.GetRoleByID(context.Background(), 0); !errors.Is(err, rbacstore.ErrInvalidID) {
		t.Fatalf("expected GetRoleByID to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.UpdateRole(context.Background(), rbacstore.UpdateRoleInput{ID: 0}); !errors.Is(err, rbacstore.ErrInvalidID) {
		t.Fatalf("expected UpdateRole to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.ListRolesByUserID(context.Background(), 0); !errors.Is(err, rbacstore.ErrInvalidID) {
		t.Fatalf("expected ListRolesByUserID to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.ListRolePermissionBindings(context.Background(), 0); !errors.Is(err, rbacstore.ErrInvalidID) {
		t.Fatalf("expected ListRolePermissionBindings to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.ListPermissionsByUserID(context.Background(), 0); !errors.Is(err, rbacstore.ErrInvalidID) {
		t.Fatalf("expected ListPermissionsByUserID to return ErrInvalidID, got %v", err)
	}
	if err := repo.ReplacePermissionsForRole(context.Background(), rbacstore.ReplacePermissionsForRoleInput{RoleID: 0}); !errors.Is(err, rbacstore.ErrInvalidID) {
		t.Fatalf("expected ReplacePermissionsForRole to return ErrInvalidID, got %v", err)
	}
	if err := repo.ReplaceRolesForUser(context.Background(), rbacstore.ReplaceRolesForUserInput{UserID: 0}); !errors.Is(err, rbacstore.ErrInvalidID) {
		t.Fatalf("expected ReplaceRolesForUser to return ErrInvalidID, got %v", err)
	}
}

func TestRepositoryUserRoleWriteOperations(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-plugin-user-role-write-ops?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := &repository{client: client}

	role, err := client.Role.Create().
		SetName("editor").
		SetDisplay("编辑").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed role: %v", err)
	}
	user, err := client.User.Create().
		SetUsername("alice").
		SetDisplay("Alice").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if err := repo.AssignRoleToUser(context.Background(), rbacstore.AssignRoleToUserInput{
		UserID: toStoreID(user.ID),
		RoleID: toStoreID(role.ID),
	}); err != nil {
		t.Fatalf("assign role to user: %v", err)
	}

	bindings, err := client.UserRole.Query().All(context.Background())
	if err != nil {
		t.Fatalf("query user roles: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected one user-role binding, got %#v", bindings)
	}
}
