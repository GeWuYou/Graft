package rbac

import (
	"context"
	"testing"

	rbacstore "graft/server/plugins/rbac/store"
)

type accessServiceTestRepository struct {
	roles       []rbacstore.Role
	permissions []rbacstore.Permission
}

func (r accessServiceTestRepository) EnsureRole(context.Context, rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r accessServiceTestRepository) EnsurePermission(context.Context, rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	return rbacstore.Permission{}, nil
}

func (r accessServiceTestRepository) CreateRole(context.Context, rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r accessServiceTestRepository) UpdateRole(context.Context, rbacstore.UpdateRoleInput) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r accessServiceTestRepository) SetRoleStatus(context.Context, rbacstore.SetRoleStatusInput) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r accessServiceTestRepository) SoftDeleteRole(context.Context, rbacstore.SoftDeleteRoleInput) error {
	return nil
}

func (r accessServiceTestRepository) AssignPermissionsToRole(context.Context, rbacstore.AssignPermissionsToRoleInput) error {
	return nil
}

func (r accessServiceTestRepository) ReplacePermissionsForRole(context.Context, rbacstore.ReplacePermissionsForRoleInput) error {
	return nil
}

func (r accessServiceTestRepository) AddPermissionsToRole(context.Context, rbacstore.AddPermissionsToRoleInput) error {
	return nil
}

func (r accessServiceTestRepository) RemovePermissionsFromRole(context.Context, rbacstore.RemovePermissionsFromRoleInput) error {
	return nil
}

func (r accessServiceTestRepository) AssignRoleToUser(context.Context, rbacstore.AssignRoleToUserInput) error {
	return nil
}

func (r accessServiceTestRepository) ReplaceRolesForUser(context.Context, rbacstore.ReplaceRolesForUserInput) error {
	return nil
}

func (r accessServiceTestRepository) AddRolesToUser(context.Context, rbacstore.AddRolesToUserInput) error {
	return nil
}

func (r accessServiceTestRepository) RemoveRolesFromUser(context.Context, rbacstore.RemoveRolesFromUserInput) error {
	return nil
}

func (r accessServiceTestRepository) GetRoleByID(context.Context, uint64) (rbacstore.Role, error) {
	return rbacstore.Role{}, nil
}

func (r accessServiceTestRepository) GetPermissionByID(context.Context, uint64) (rbacstore.Permission, error) {
	return rbacstore.Permission{}, nil
}

func (r accessServiceTestRepository) ListRolesByUserID(context.Context, uint64) ([]rbacstore.Role, error) {
	return r.roles, nil
}

func (r accessServiceTestRepository) ListRolesByUserIDs(context.Context, []uint64) (map[uint64][]rbacstore.Role, error) {
	return map[uint64][]rbacstore.Role{
		7: r.roles,
	}, nil
}

func (r accessServiceTestRepository) ListRoles(context.Context, rbacstore.RoleFilter) ([]rbacstore.Role, error) {
	return nil, nil
}

func (r accessServiceTestRepository) ListPermissionsByUserID(context.Context, uint64) ([]rbacstore.Permission, error) {
	return r.permissions, nil
}

func (r accessServiceTestRepository) ListPermissions(context.Context, rbacstore.PermissionFilter) ([]rbacstore.Permission, error) {
	return nil, nil
}

func (r accessServiceTestRepository) ListRolePermissionBindings(context.Context, uint64) ([]rbacstore.RolePermissionBinding, error) {
	return nil, nil
}

func TestAccessServiceListsStableRoleNamesAndPermissionCodes(t *testing.T) {
	service := accessService{
		rbac: accessServiceTestRepository{
			roles: []rbacstore.Role{
				{Name: "  editor "},
				{Name: ""},
				{Name: "admin"},
				{Name: "editor"},
				{Name: "viewer"},
				{Name: "  "},
				{Name: "admin"},
			},
			permissions: []rbacstore.Permission{
				{Code: "  audit.write "},
				{Code: ""},
				{Code: "audit.read"},
				{Code: "audit.write"},
				{Code: "user.read"},
				{Code: "  "},
			},
		},
	}

	roles, err := service.ListRoleNamesByUserID(context.Background(), 7)
	if err != nil {
		t.Fatalf("list role names: %v", err)
	}
	if len(roles) != 3 || roles[0] != "admin" || roles[1] != "editor" || roles[2] != "viewer" {
		t.Fatalf("unexpected role names: %#v", roles)
	}

	codes, err := service.ListPermissionCodesByUserID(context.Background(), 7)
	if err != nil {
		t.Fatalf("list permission codes: %v", err)
	}
	if len(codes) != 3 || codes[0] != "audit.read" || codes[1] != "audit.write" || codes[2] != "user.read" {
		t.Fatalf("unexpected permission codes: %#v", codes)
	}
}
