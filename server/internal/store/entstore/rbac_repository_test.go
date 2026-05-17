package entstore

import (
	"context"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/ent/enttest"
	entrole "graft/server/internal/ent/role"
	entrolepermission "graft/server/internal/ent/rolepermission"
	entuserrole "graft/server/internal/ent/userrole"
	"graft/server/internal/store"
)

// TestRBACRepositoryListRolesAndPermissions 验证新增只读查询会按稳定顺序返回角色/权限快照，
// 并保留 builtin/category 等管理字段映射。
func TestRBACRepositoryListRolesAndPermissions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-list-snapshots?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	firstRole, err := client.Role.Create().
		SetName("admin").
		SetDisplay("管理员").
		SetBuiltin(true).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed first role: %v", err)
	}
	secondRole, err := client.Role.Create().
		SetName("auditor").
		SetDisplay("审计员").
		SetBuiltin(false).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed second role: %v", err)
	}

	firstPermission, err := client.Permission.Create().
		SetCode("role.read").
		SetDisplay("查看角色").
		SetCategory("menu").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed first permission: %v", err)
	}
	secondPermission, err := client.Permission.Create().
		SetCode("permission.read").
		SetDisplay("查看权限").
		SetCategory("api").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed second permission: %v", err)
	}

	repo := &rbacRepository{client: client}

	roles, err := repo.ListRoles(context.Background())
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %#v", roles)
	}
	if roles[0].ID != toStoreID(firstRole.ID) || !roles[0].Builtin {
		t.Fatalf("unexpected first role snapshot: %#v", roles[0])
	}
	if roles[1].ID != toStoreID(secondRole.ID) || roles[1].Builtin {
		t.Fatalf("unexpected second role snapshot: %#v", roles[1])
	}

	permissions, err := repo.ListPermissions(context.Background())
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	if len(permissions) != 2 {
		t.Fatalf("expected 2 permissions, got %#v", permissions)
	}
	if permissions[0].ID != toStoreID(firstPermission.ID) || permissions[0].Category != "menu" {
		t.Fatalf("unexpected first permission snapshot: %#v", permissions[0])
	}
	if permissions[1].ID != toStoreID(secondPermission.ID) || permissions[1].Category != "api" {
		t.Fatalf("unexpected second permission snapshot: %#v", permissions[1])
	}
}

// TestEnsureRoleUpgradesBuiltinState 验证 EnsureRole 会把既有默认角色向 builtin=true 的真值收敛。
func TestEnsureRoleUpgradesBuiltinState(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-ensure-builtin?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	if _, err := client.Role.Create().
		SetName("admin").
		SetDisplay("管理员").
		SetBuiltin(false).
		Save(context.Background()); err != nil {
		t.Fatalf("seed role: %v", err)
	}

	repo := &rbacRepository{client: client}
	role, err := repo.EnsureRole(context.Background(), store.EnsureRoleInput{
		Name:    "admin",
		Display: "管理员",
		Builtin: true,
	})
	if err != nil {
		t.Fatalf("ensure role: %v", err)
	}
	if !role.Builtin {
		t.Fatalf("expected ensured role to become builtin, got %#v", role)
	}

	record, err := client.Role.Query().Where(entrole.NameEQ("admin")).Only(context.Background())
	if err != nil {
		t.Fatalf("reload role: %v", err)
	}
	if !record.Builtin {
		t.Fatalf("expected persisted role builtin=true, got %#v", record)
	}
}

// TestEnsurePermissionPersistsCategory 验证 EnsurePermission 创建路径会持久化调用方提供的分类真值。
func TestEnsurePermissionPersistsCategory(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-ensure-permission-category?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := &rbacRepository{client: client}
	permissionRecord, err := repo.EnsurePermission(context.Background(), store.EnsurePermissionInput{
		Code:     "role.read",
		Display:  "查看角色",
		Category: "api",
	})
	if err != nil {
		t.Fatalf("ensure permission: %v", err)
	}
	if permissionRecord.Category != "api" {
		t.Fatalf("expected ensured permission category api, got %#v", permissionRecord)
	}

	record, err := client.Permission.Query().Only(context.Background())
	if err != nil {
		t.Fatalf("reload permission: %v", err)
	}
	if record.Category != "api" {
		t.Fatalf("expected persisted permission category api, got %#v", record)
	}
}

// TestRBACRepositoryRoleWriteOperations 验证角色写接口与权限覆盖式绑定会保留稳定仓储语义。
func TestRBACRepositoryRoleWriteOperations(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-write-ops?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := &rbacRepository{client: client}

	updatedRole := createAndUpdateRoleForTest(t, repo)
	roleEntID, err := toEntID(updatedRole.ID)
	if err != nil {
		t.Fatalf("convert updated role id: %v", err)
	}

	firstPermission, err := client.Permission.Create().
		SetCode("user.read").
		SetDisplay("查看用户").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed first permission: %v", err)
	}
	secondPermission, err := client.Permission.Create().
		SetCode("user.update").
		SetDisplay("编辑用户").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed second permission: %v", err)
	}
	if _, err := client.RolePermission.Create().
		SetRoleID(roleEntID).
		SetPermissionID(firstPermission.ID).
		Save(context.Background()); err != nil {
		t.Fatalf("seed role permission: %v", err)
	}

	if err := repo.ReplacePermissionsForRole(context.Background(), store.ReplacePermissionsForRoleInput{
		RoleID:        updatedRole.ID,
		PermissionIDs: []uint64{toStoreID(secondPermission.ID)},
	}); err != nil {
		t.Fatalf("replace role permissions: %v", err)
	}

	rolePermissions, err := client.RolePermission.Query().
		Where(entrolepermission.RoleIDEQ(roleEntID)).
		All(context.Background())
	if err != nil {
		t.Fatalf("query replaced role permissions: %v", err)
	}
	if len(rolePermissions) != 1 || rolePermissions[0].PermissionID != secondPermission.ID {
		t.Fatalf("expected stale permissions to be replaced, got %#v", rolePermissions)
	}
}

// TestRBACRepositoryUserRoleWriteOperations 验证用户角色覆盖式绑定会保留稳定仓储语义。
func TestRBACRepositoryUserRoleWriteOperations(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-user-role-write-ops?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := &rbacRepository{client: client}

	updatedRole := createAndUpdateRoleForTest(t, repo)
	roleEntID, err := toEntID(updatedRole.ID)
	if err != nil {
		t.Fatalf("convert updated role id: %v", err)
	}

	user, err := client.User.Create().
		SetUsername("alice").
		SetDisplay("Alice").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	otherRole, err := client.Role.Create().
		SetName("auditor").
		SetDisplay("审计员").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed other role: %v", err)
	}
	if _, err := client.UserRole.Create().
		SetUserID(user.ID).
		SetRoleID(roleEntID).
		Save(context.Background()); err != nil {
		t.Fatalf("seed user role: %v", err)
	}

	if err := repo.ReplaceRolesForUser(context.Background(), store.ReplaceRolesForUserInput{
		UserID:  toStoreID(user.ID),
		RoleIDs: []uint64{toStoreID(otherRole.ID)},
	}); err != nil {
		t.Fatalf("replace user roles: %v", err)
	}

	userRoles, err := client.UserRole.Query().
		Where(entuserrole.UserIDEQ(user.ID)).
		All(context.Background())
	if err != nil {
		t.Fatalf("query replaced user roles: %v", err)
	}
	if len(userRoles) != 1 || userRoles[0].RoleID != otherRole.ID {
		t.Fatalf("expected stale user roles to be replaced, got %#v", userRoles)
	}
}

func createAndUpdateRoleForTest(t *testing.T, repo *rbacRepository) store.Role {
	t.Helper()

	role, err := repo.CreateRole(context.Background(), store.CreateRoleInput{
		Name:    "editor",
		Display: "编辑",
		Builtin: false,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	updatedRole, err := repo.UpdateRole(context.Background(), store.UpdateRoleInput{
		ID:      role.ID,
		Name:    "editor-plus",
		Display: "高级编辑",
	})
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if updatedRole.Name != "editor-plus" || updatedRole.Display != "高级编辑" {
		t.Fatalf("unexpected updated role: %#v", updatedRole)
	}

	return updatedRole
}

// TestListRolePermissionBindings 验证角色权限绑定读取会返回稳定排序的权限 ID 快照。
func TestListRolePermissionBindings(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-role-permission-bindings?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	role, err := client.Role.Create().
		SetName("editor").
		SetDisplay("编辑").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed role: %v", err)
	}

	firstPermission, err := client.Permission.Create().
		SetCode("user.read").
		SetDisplay("查看用户").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed first permission: %v", err)
	}
	secondPermission, err := client.Permission.Create().
		SetCode("user.update").
		SetDisplay("编辑用户").
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed second permission: %v", err)
	}

	if _, err := client.RolePermission.Create().
		SetRoleID(role.ID).
		SetPermissionID(secondPermission.ID).
		Save(context.Background()); err != nil {
		t.Fatalf("seed second role permission: %v", err)
	}
	if _, err := client.RolePermission.Create().
		SetRoleID(role.ID).
		SetPermissionID(firstPermission.ID).
		Save(context.Background()); err != nil {
		t.Fatalf("seed first role permission: %v", err)
	}

	repo := &rbacRepository{client: client}
	bindings, err := repo.ListRolePermissionBindings(context.Background(), toStoreID(role.ID))
	if err != nil {
		t.Fatalf("list role permission bindings: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %#v", bindings)
	}
	if bindings[0].PermissionID != toStoreID(firstPermission.ID) || bindings[1].PermissionID != toStoreID(secondPermission.ID) {
		t.Fatalf("expected bindings sorted by permission id, got %#v", bindings)
	}
}

// TestListRolePermissionBindingsReturnsRoleNotFound 验证角色权限绑定读取会保留稳定未命中语义。
func TestListRolePermissionBindingsReturnsRoleNotFound(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rbac-role-permission-bindings-missing?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	repo := &rbacRepository{client: client}

	_, err := repo.ListRolePermissionBindings(context.Background(), 42)
	if !errors.Is(err, store.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}
