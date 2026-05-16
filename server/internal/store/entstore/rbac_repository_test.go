package entstore

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/ent/enttest"
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
