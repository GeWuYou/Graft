package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	rbacstore "graft/server/modules/rbac/store"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:rbac-module-storeent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	schema := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			display TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			display TEXT NOT NULL,
			description TEXT NULL,
			builtin BOOLEAN NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			created_by INTEGER NOT NULL DEFAULT 0,
			updated_by INTEGER NOT NULL DEFAULT 0,
			deleted_at INTEGER NOT NULL DEFAULT 0,
			deleted_by INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			display TEXT NOT NULL,
			display_key TEXT NULL,
			description TEXT NULL,
			description_key TEXT NULL,
			category TEXT NOT NULL DEFAULT 'api',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			created_by INTEGER NOT NULL DEFAULT 0,
			updated_by INTEGER NOT NULL DEFAULT 0,
			deleted_at INTEGER NOT NULL DEFAULT 0,
			deleted_by INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE user_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			role_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			UNIQUE(user_id, role_id)
		);`,
		`CREATE TABLE role_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			permission_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			UNIQUE(role_id, permission_id)
		);`,
	}
	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}

	return db
}

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
	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, "editor", 0)
	userID := seedUser(t, db, "alice")

	if err := repo.AssignRoleToUser(context.Background(), rbacstore.AssignRoleToUserInput{
		UserID: toStoreID(userID),
		RoleID: toStoreID(roleID),
	}); err != nil {
		t.Fatalf("assign role to user: %v", err)
	}

	rows, err := db.QueryContext(context.Background(), `SELECT user_id, role_id FROM user_roles`)
	if err != nil {
		t.Fatalf("query user roles: %v", err)
	}

	count := 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate user roles: %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close user roles rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one user-role binding, got %d", count)
	}
}

func TestRepositoryAssignRoleToUserRejectsDisabledRole(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, "disabled-editor", time.Now().UTC().Unix())
	userID := seedUser(t, db, "disabled-user")

	err := repo.AssignRoleToUser(context.Background(), rbacstore.AssignRoleToUserInput{
		UserID: toStoreID(userID),
		RoleID: toStoreID(roleID),
	})
	if !errors.Is(err, rbacstore.ErrRoleDisabledAssignmentForbidden) {
		t.Fatalf("expected ErrRoleDisabledAssignmentForbidden, got %v", err)
	}
	if count := countRows(t, db, "SELECT COUNT(*) FROM user_roles"); count != 0 {
		t.Fatalf("expected no user-role bindings after rejected disabled role assignment, got %d", count)
	}
}

func TestRepositoryAssignPermissionsToRoleIsAtomic(t *testing.T) {
	assertRolePermissionMutationIsAtomic(
		t,
		"permission-assign-role",
		"permission.assign",
		func(ctx context.Context, repo *repository, roleID uint64, permissionIDs []uint64) error {
			return repo.AssignPermissionsToRole(ctx, rbacstore.AssignPermissionsToRoleInput{
				RoleID:        roleID,
				PermissionIDs: permissionIDs,
			})
		},
		"AssignPermissionsToRole",
	)
}

func TestRepositoryAddPermissionsToRoleIsAtomic(t *testing.T) {
	assertRolePermissionMutationIsAtomic(
		t,
		"permission-add-role",
		"permission.add",
		func(ctx context.Context, repo *repository, roleID uint64, permissionIDs []uint64) error {
			return repo.AddPermissionsToRole(ctx, rbacstore.AddPermissionsToRoleInput{
				RoleID:        roleID,
				PermissionIDs: permissionIDs,
			})
		},
		"AddPermissionsToRole",
	)
}

func assertRolePermissionMutationIsAtomic(
	t *testing.T,
	roleName string,
	permissionPrefix string,
	mutate func(context.Context, *repository, uint64, []uint64) error,
	action string,
) {
	t.Helper()

	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, roleName, 0)
	firstPermissionID := seedPermission(t, db, permissionPrefix+".first")
	secondPermissionID := seedPermission(t, db, permissionPrefix+".second")
	installRolePermissionAbortTrigger(t, db, secondPermissionID)

	err := mutate(context.Background(), repo, toStoreID(roleID), []uint64{toStoreID(firstPermissionID), toStoreID(secondPermissionID)})
	if err == nil || !strings.Contains(err.Error(), "blocked permission insert") {
		t.Fatalf("expected trigger failure from %s, got %v", action, err)
	}
	if count := countRows(t, db, "SELECT COUNT(*) FROM role_permissions"); count != 0 {
		t.Fatalf("expected atomic rollback for %s, got %d bindings", action, count)
	}
}

func TestRepositoryDisabledRolePermissionBindingsKeepCleanupPathsConsistent(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, "disabled-binding-role", time.Now().UTC().Unix())
	permissionID := seedPermission(t, db, "permission.disabled.binding")
	seedRolePermissionBinding(t, db, roleID, permissionID)

	addErr := repo.AddPermissionsToRole(context.Background(), rbacstore.AddPermissionsToRoleInput{
		RoleID:        toStoreID(roleID),
		PermissionIDs: []uint64{toStoreID(permissionID)},
	})
	if !errors.Is(addErr, rbacstore.ErrRoleDisabledAssignmentForbidden) {
		t.Fatalf("expected AddPermissionsToRole to reject disabled role mutations, got %v", addErr)
	}

	replaceErr := repo.ReplacePermissionsForRole(context.Background(), rbacstore.ReplacePermissionsForRoleInput{
		RoleID:        toStoreID(roleID),
		PermissionIDs: []uint64{toStoreID(permissionID)},
	})
	if !errors.Is(replaceErr, rbacstore.ErrRoleDisabledAssignmentForbidden) {
		t.Fatalf("expected ReplacePermissionsForRole to reject disabled role non-empty mutations, got %v", replaceErr)
	}

	bindings, listErr := repo.ListRolePermissionBindings(context.Background(), toStoreID(roleID))
	if listErr != nil {
		t.Fatalf("expected ListRolePermissionBindings to keep disabled role snapshot readable, got %v", listErr)
	}
	if len(bindings) != 1 || bindings[0].PermissionID != toStoreID(permissionID) {
		t.Fatalf("unexpected bindings for disabled role: %#v", bindings)
	}

	if err := repo.RemovePermissionsFromRole(context.Background(), rbacstore.RemovePermissionsFromRoleInput{
		RoleID:        toStoreID(roleID),
		PermissionIDs: []uint64{toStoreID(permissionID)},
	}); err != nil {
		t.Fatalf("expected RemovePermissionsFromRole to allow disabled role cleanup, got %v", err)
	}

	if err := repo.ReplacePermissionsForRole(context.Background(), rbacstore.ReplacePermissionsForRoleInput{
		RoleID:        toStoreID(roleID),
		PermissionIDs: nil,
	}); err != nil {
		t.Fatalf("expected ReplacePermissionsForRole to allow disabled role cleanup to empty set, got %v", err)
	}

	if count := countRows(t, db, "SELECT COUNT(*) FROM role_permissions"); count != 0 {
		t.Fatalf("expected disabled role cleanup to remove bindings, got %d", count)
	}
}

func TestRepositoryEnsurePermissionAndListPermissionsIncludeTimestamps(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	record, err := repo.EnsurePermission(context.Background(), rbacstore.EnsurePermissionInput{
		Code:           "user.create",
		Display:        "Create Users",
		DisplayKey:     stringPtr("rbac.permissionCatalog.userCreate.display"),
		Description:    stringPtr("Allows creating user management data."),
		DescriptionKey: stringPtr("rbac.permissionCatalog.userCreate.description"),
		Category:       "api",
	})
	if err != nil {
		t.Fatalf("ensure permission: %v", err)
	}
	assertEnsuredPermissionRecord(t, record)

	permissions, err := repo.ListPermissions(context.Background(), rbacstore.PermissionFilter{})
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	assertListedPermissionRecord(t, permissions)

	updated, err := repo.EnsurePermission(context.Background(), rbacstore.EnsurePermissionInput{
		Code:           "user.create",
		Display:        "Create Users Localized Fallback",
		DisplayKey:     stringPtr("rbac.permissionCatalog.userCreate.display"),
		Description:    stringPtr("Updated description fallback."),
		DescriptionKey: stringPtr("rbac.permissionCatalog.userCreate.description"),
		Category:       "api",
	})
	if err != nil {
		t.Fatalf("reconcile permission metadata: %v", err)
	}
	assertReconciledPermissionRecord(t, updated)
}

func assertEnsuredPermissionRecord(t *testing.T, record rbacstore.Permission) {
	t.Helper()

	if record.Code != "user.create" {
		t.Fatalf("expected ensured permission code user.create, got %#v", record)
	}
	if record.CreatedAt.IsZero() || record.UpdatedAt.IsZero() {
		t.Fatalf("expected ensured permission timestamps, got %#v", record)
	}
	assertPermissionKeys(t, record, "ensured")
}

func assertListedPermissionRecord(t *testing.T, permissions []rbacstore.Permission) {
	t.Helper()

	if len(permissions) != 1 {
		t.Fatalf("expected one permission, got %d", len(permissions))
	}
	if permissions[0].CreatedAt.IsZero() || permissions[0].UpdatedAt.IsZero() {
		t.Fatalf("expected listed permission timestamps, got %#v", permissions[0])
	}
	assertPermissionKeys(t, permissions[0], "listed")
}

func assertReconciledPermissionRecord(t *testing.T, updated rbacstore.Permission) {
	t.Helper()

	if updated.Display != "Create Users Localized Fallback" ||
		updated.Description == nil ||
		*updated.Description != "Updated description fallback." {
		t.Fatalf("expected reconciled permission metadata, got %#v", updated)
	}
}

func assertPermissionKeys(t *testing.T, record rbacstore.Permission, phase string) {
	t.Helper()

	if record.DisplayKey == nil || *record.DisplayKey != "rbac.permissionCatalog.userCreate.display" {
		t.Fatalf("expected %s permission display key, got %#v", phase, record)
	}
	if record.DescriptionKey == nil || *record.DescriptionKey != "rbac.permissionCatalog.userCreate.description" {
		t.Fatalf("expected %s permission description key, got %#v", phase, record)
	}
}

func stringPtr(value string) *string {
	return &value
}

func seedRole(t *testing.T, db *sql.DB, name string, deletedAt int64) int64 {
	t.Helper()

	now := time.Now().UTC()
	result, err := db.ExecContext(context.Background(),
		`INSERT INTO roles (name, display, description, builtin, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name, name, nil, false, now, now, deletedAt,
	)
	if err != nil {
		t.Fatalf("seed role %s: %v", name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read role id for %s: %v", name, err)
	}
	return id
}

func seedUser(t *testing.T, db *sql.DB, username string) int64 {
	t.Helper()

	now := time.Now().UTC()
	result, err := db.ExecContext(context.Background(),
		`INSERT INTO users (username, display, created_at, updated_at)
		VALUES (?, ?, ?, ?)`,
		username, username, now, now,
	)
	if err != nil {
		t.Fatalf("seed user %s: %v", username, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read user id for %s: %v", username, err)
	}
	return id
}

func seedPermission(t *testing.T, db *sql.DB, code string) int64 {
	t.Helper()

	now := time.Now().UTC()
	result, err := db.ExecContext(context.Background(),
		`INSERT INTO permissions (code, display, display_key, description, description_key, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		code, code, nil, nil, nil, "api", now, now,
	)
	if err != nil {
		t.Fatalf("seed permission %s: %v", code, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read permission id for %s: %v", code, err)
	}
	return id
}

func seedRolePermissionBinding(t *testing.T, db *sql.DB, roleID int64, permissionID int64) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)`,
		roleID,
		permissionID,
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("seed role permission binding role=%d permission=%d: %v", roleID, permissionID, err)
	}
}

func installRolePermissionAbortTrigger(t *testing.T, db *sql.DB, blockedPermissionID int64) {
	t.Helper()

	statement := fmt.Sprintf(`CREATE TRIGGER abort_role_permission_insert
		BEFORE INSERT ON role_permissions
		FOR EACH ROW
		WHEN NEW.permission_id = %d
		BEGIN
			SELECT RAISE(ABORT, 'blocked permission insert');
		END;`, blockedPermissionID)
	if _, err := db.Exec(statement); err != nil {
		t.Fatalf("install role permission trigger: %v", err)
	}
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()

	var count int
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows with %q: %v", query, err)
	}
	return count
}
