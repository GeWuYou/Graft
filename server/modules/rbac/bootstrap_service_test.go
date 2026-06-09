// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

type bootstrapServiceTestRepository struct {
	ensureRoleInput        rbacstore.EnsureRoleInput
	ensurePermissionInputs []rbacstore.EnsurePermissionInput
	assignPermissionsInput rbacstore.AssignPermissionsToRoleInput
	assignRoleInput        rbacstore.AssignRoleToUserInput

	roleToReturn         rbacstore.Role
	permissionToReturn   rbacstore.Permission
	ensureRoleErr        error
	ensurePermissionErr  error
	assignPermissionsErr error
	assignRoleErr        error
}

func (r *bootstrapServiceTestRepository) EnsureRole(_ context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	r.ensureRoleInput = input
	if r.ensureRoleErr != nil {
		return rbacstore.Role{}, r.ensureRoleErr
	}
	if r.roleToReturn.ID == 0 {
		r.roleToReturn = rbacstore.Role{ID: 9, Name: input.Name}
	}
	return r.roleToReturn, nil
}

func (r *bootstrapServiceTestRepository) EnsurePermission(_ context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	r.ensurePermissionInputs = append(r.ensurePermissionInputs, input)
	if r.ensurePermissionErr != nil {
		return rbacstore.Permission{}, r.ensurePermissionErr
	}
	if r.permissionToReturn.ID == 0 {
		r.permissionToReturn = rbacstore.Permission{ID: uint64(len(r.ensurePermissionInputs)), Code: input.Code}
	}
	return r.permissionToReturn, nil
}

func (r *bootstrapServiceTestRepository) AssignPermissionsToRole(_ context.Context, input rbacstore.AssignPermissionsToRoleInput) error {
	r.assignPermissionsInput = input
	return r.assignPermissionsErr
}

func (r *bootstrapServiceTestRepository) AssignRoleToUser(_ context.Context, input rbacstore.AssignRoleToUserInput) error {
	r.assignRoleInput = input
	return r.assignRoleErr
}

func (r *bootstrapServiceTestRepository) CreateRole(context.Context, rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r *bootstrapServiceTestRepository) UpdateRole(context.Context, rbacstore.UpdateRoleInput) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r *bootstrapServiceTestRepository) SetRoleStatus(context.Context, rbacstore.SetRoleStatusInput) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r *bootstrapServiceTestRepository) SoftDeleteRole(context.Context, rbacstore.SoftDeleteRoleInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) ReplacePermissionsForRole(context.Context, rbacstore.ReplacePermissionsForRoleInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) AddPermissionsToRole(context.Context, rbacstore.AddPermissionsToRoleInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) RemovePermissionsFromRole(context.Context, rbacstore.RemovePermissionsFromRoleInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) ReplaceRolesForUser(context.Context, rbacstore.ReplaceRolesForUserInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) AddRolesToUser(context.Context, rbacstore.AddRolesToUserInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) RemoveRolesFromUser(context.Context, rbacstore.RemoveRolesFromUserInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) ReplaceRolesForUsersAtomically(context.Context, rbacstore.BatchUserRoleMutationInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) AddRolesToUsersAtomically(context.Context, rbacstore.BatchUserRoleMutationInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) RemoveRolesFromUsersAtomically(context.Context, rbacstore.BatchUserRoleMutationInput) error {
	return nil
}

func (r *bootstrapServiceTestRepository) GetRoleByID(context.Context, uint64) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r *bootstrapServiceTestRepository) GetPermissionByID(context.Context, uint64) (rbacstore.Permission, error) {
	return rbacstore.Permission{}, nil
}

func (r *bootstrapServiceTestRepository) ListRolesByUserID(context.Context, uint64) ([]rbacstore.Role, error) {
	return nil, nil
}

func (r *bootstrapServiceTestRepository) ListRolesByUserIDs(context.Context, []uint64) (map[uint64][]rbacstore.Role, error) {
	return map[uint64][]rbacstore.Role{}, nil
}

func (r *bootstrapServiceTestRepository) ListRoles(context.Context, rbacstore.RoleFilter) ([]rbacstore.Role, error) {
	return nil, nil
}

func (r *bootstrapServiceTestRepository) ListPermissionsByUserID(context.Context, uint64) ([]rbacstore.Permission, error) {
	return nil, nil
}

func (r *bootstrapServiceTestRepository) ListPermissions(context.Context, rbacstore.PermissionFilter) ([]rbacstore.Permission, error) {
	return nil, nil
}

func (r *bootstrapServiceTestRepository) ListRolePermissionBindings(context.Context, uint64) ([]rbacstore.RolePermissionBinding, error) {
	return nil, nil
}

func TestBootstrapServiceEnsuresDefaultAdminAccess(t *testing.T) {
	repo := &bootstrapServiceTestRepository{}
	service := bootstrapService{rbac: repo}
	permissions := []moduleapi.PermissionSeed{
		{
			Code:           "user.read",
			Display:        "Read users",
			DisplayKey:     "rbac.permissionCatalog.userRead.display",
			Description:    "  ",
			DescriptionKey: "rbac.permissionCatalog.userRead.description",
			Category:       "api",
		},
		{
			Code:           "user.write",
			Display:        "Write users",
			DisplayKey:     "rbac.permissionCatalog.userCreate.display",
			Description:    "write users",
			DescriptionKey: "rbac.permissionCatalog.userCreate.description",
			Category:       "api",
		},
	}

	if err := service.EnsureDefaultAdminAccess(context.Background(), 7, permissions); err != nil {
		t.Fatalf("ensure default admin access: %v", err)
	}
	assertDefaultAdminBootstrap(t, repo)
}

func assertDefaultAdminBootstrap(t *testing.T, repo *bootstrapServiceTestRepository) {
	t.Helper()

	if repo.ensureRoleInput.Name != builtinAdminRoleName || repo.ensureRoleInput.Display != "管理员" || !repo.ensureRoleInput.Builtin {
		t.Fatalf("unexpected role seed: %#v", repo.ensureRoleInput)
	}
	if len(repo.ensurePermissionInputs) != 2 {
		t.Fatalf("expected 2 permission seeds, got %d", len(repo.ensurePermissionInputs))
	}
	if repo.ensurePermissionInputs[0].Description != nil {
		t.Fatalf("expected blank permission description to become nil, got %#v", repo.ensurePermissionInputs[0].Description)
	}
	assertPermissionInputKeys(t, repo.ensurePermissionInputs[0], "rbac.permissionCatalog.userRead.display", "rbac.permissionCatalog.userRead.description")
	if repo.ensurePermissionInputs[1].Description == nil || *repo.ensurePermissionInputs[1].Description != "write users" {
		t.Fatalf("expected non-blank permission description to be preserved, got %#v", repo.ensurePermissionInputs[1].Description)
	}
	if len(repo.assignPermissionsInput.PermissionIDs) != 2 || repo.assignPermissionsInput.RoleID != repo.roleToReturn.ID {
		t.Fatalf("unexpected permission assignment: %#v", repo.assignPermissionsInput)
	}
	if repo.assignRoleInput.UserID != 7 || repo.assignRoleInput.RoleID != repo.roleToReturn.ID {
		t.Fatalf("unexpected role assignment: %#v", repo.assignRoleInput)
	}
}

func assertPermissionInputKeys(
	t *testing.T,
	input rbacstore.EnsurePermissionInput,
	displayKey string,
	descriptionKey string,
) {
	t.Helper()

	if input.DisplayKey == nil || *input.DisplayKey != displayKey {
		t.Fatalf("expected permission display key %q, got %#v", displayKey, input)
	}
	if input.DescriptionKey == nil || *input.DescriptionKey != descriptionKey {
		t.Fatalf("expected permission description key %q, got %#v", descriptionKey, input)
	}
}

func TestBootstrapServiceWrapsRepositoryErrors(t *testing.T) {
	t.Run("ensure role", func(t *testing.T) {
		repo := &bootstrapServiceTestRepository{ensureRoleErr: errors.New("boom")}
		err := bootstrapService{rbac: repo}.EnsureDefaultAdminAccess(context.Background(), 7, nil)
		if err == nil || !errors.Is(err, repo.ensureRoleErr) || err.Error() != "ensure default admin role: boom" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ensure permission", func(t *testing.T) {
		repo := &bootstrapServiceTestRepository{ensurePermissionErr: errors.New("perm boom")}
		err := bootstrapService{rbac: repo}.EnsureDefaultAdminAccess(context.Background(), 7, []moduleapi.PermissionSeed{{Code: "user.read", Display: "Read users"}})
		if err == nil || !errors.Is(err, repo.ensurePermissionErr) || err.Error() != "ensure permission user.read: perm boom" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("assign permissions", func(t *testing.T) {
		repo := &bootstrapServiceTestRepository{assignPermissionsErr: errors.New("assign perms boom")}
		err := bootstrapService{rbac: repo}.EnsureDefaultAdminAccess(context.Background(), 7, []moduleapi.PermissionSeed{{Code: "user.read", Display: "Read users"}})
		want := "assign permissions to default admin role: assign perms boom"
		if err == nil || !errors.Is(err, repo.assignPermissionsErr) || err.Error() != want {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("assign role", func(t *testing.T) {
		repo := &bootstrapServiceTestRepository{assignRoleErr: fmt.Errorf("assign role boom")}
		err := bootstrapService{rbac: repo}.EnsureDefaultAdminAccess(context.Background(), 7, nil)
		want := "assign default admin role to user: assign role boom"
		if err == nil || err.Error() != want || !errors.Is(err, repo.assignRoleErr) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
