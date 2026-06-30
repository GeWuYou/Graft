package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
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
			disabled_at INTEGER NOT NULL DEFAULT 0,
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

	roleID := seedRole(t, db, seededRoleRecord{name: "editor"})
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

	roleID := seedRole(t, db, seededRoleRecord{name: "disabled-editor", disabledAt: time.Now().UTC().Unix()})
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

func TestRepositoryListRolesSeparatesDisabledFromSoftDeleted(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	enabledRoleID := seedRole(t, db, seededRoleRecord{name: "enabled"})
	disabledRoleID := seedRole(t, db, seededRoleRecord{name: "disabled", disabledAt: time.Now().UTC().Unix()})
	softDeletedRoleID := seedRole(t, db, seededRoleRecord{name: "soft-deleted", disabledAt: time.Now().UTC().Unix(), deletedAt: time.Now().UTC().Unix()})

	enabledRoles, err := repo.ListRoles(context.Background(), rbacstore.RoleFilter{})
	if err != nil {
		t.Fatalf("list enabled roles: %v", err)
	}
	assertRoleIDs(t, enabledRoles, []uint64{toStoreID(enabledRoleID)})
	if enabledRoles[0].Status != rbacstore.RoleStatusEnabled {
		t.Fatalf("expected enabled role status, got %#v", enabledRoles[0])
	}

	disabledRoles, err := repo.ListRoles(context.Background(), rbacstore.RoleFilter{Status: rbacstore.RoleStatusDisabled})
	if err != nil {
		t.Fatalf("list disabled roles: %v", err)
	}
	assertRoleIDs(t, disabledRoles, []uint64{toStoreID(disabledRoleID)})
	if disabledRoles[0].Status != rbacstore.RoleStatusDisabled {
		t.Fatalf("expected disabled role status, got %#v", disabledRoles[0])
	}

	if _, err := repo.GetRoleByID(context.Background(), toStoreID(softDeletedRoleID)); !errors.Is(err, rbacstore.ErrRoleNotFound) {
		t.Fatalf("expected soft-deleted role lookup to return ErrRoleNotFound, got %v", err)
	}
}

func TestRepositorySetRoleStatusDoesNotReviveSoftDeletedRole(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, seededRoleRecord{
		name:       "archived-editor",
		disabledAt: time.Now().UTC().Add(-time.Minute).Unix(),
		deletedAt:  time.Now().UTC().Unix(),
	})

	if _, err := repo.SetRoleStatus(context.Background(), rbacstore.SetRoleStatusInput{
		ID:     toStoreID(roleID),
		Status: rbacstore.RoleStatusEnabled,
	}); !errors.Is(err, rbacstore.ErrRoleNotFound) {
		t.Fatalf("expected enable soft-deleted role to return ErrRoleNotFound, got %v", err)
	}

	if _, err := repo.SetRoleStatus(context.Background(), rbacstore.SetRoleStatusInput{
		ID:     toStoreID(roleID),
		Status: rbacstore.RoleStatusDisabled,
	}); !errors.Is(err, rbacstore.ErrRoleNotFound) {
		t.Fatalf("expected disable soft-deleted role to return ErrRoleNotFound, got %v", err)
	}

	assertRoleLifecycleFields(t, db, roleID, lifecycleSnapshot{
		disabledAtNonZero: true,
		deletedAtNonZero:  true,
	})
}

func TestRepositorySetRoleStatusEnableRoleIsIdempotentForEnabledRole(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, seededRoleRecord{
		name: "already-enabled-role",
	})

	record, err := repo.SetRoleStatus(context.Background(), rbacstore.SetRoleStatusInput{
		ID:     toStoreID(roleID),
		Status: rbacstore.RoleStatusEnabled,
	})
	if err != nil {
		t.Fatalf("expected enable role to be idempotent for enabled role, got %v", err)
	}
	if record.Status != rbacstore.RoleStatusEnabled {
		t.Fatalf("expected enabled role status, got %#v", record)
	}

	assertRoleLifecycleFields(t, db, roleID, lifecycleSnapshot{
		disabledAtNonZero: false,
		deletedAtNonZero:  false,
	})
}

func TestRepositorySoftDeleteRoleRejectsDisabledRoleWithBindings(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, seededRoleRecord{
		name:       "disabled-bound-role",
		disabledAt: time.Now().UTC().Add(-time.Minute).Unix(),
	})
	permissionID := seedPermission(t, db, "rbac.delete.race")
	seedRolePermissionBinding(t, db, roleID, permissionID)

	err := repo.SoftDeleteRole(context.Background(), rbacstore.SoftDeleteRoleInput{
		ID: toStoreID(roleID),
	})
	if !errors.Is(err, rbacstore.ErrRoleBindingsExist) {
		t.Fatalf("expected disabled role bindings to block soft delete with ErrRoleBindingsExist, got %v", err)
	}

	assertRoleLifecycleFields(t, db, roleID, lifecycleSnapshot{
		disabledAtNonZero: true,
		deletedAtNonZero:  false,
	})
	if count := countRows(t, db, "SELECT COUNT(*) FROM role_permissions"); count != 1 {
		t.Fatalf("expected role binding to remain visible after blocked delete, got %d", count)
	}
}

func TestRepositorySoftDeleteRoleRejectsEnabledRole(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, seededRoleRecord{name: "enabled-delete-target"})

	err := repo.SoftDeleteRole(context.Background(), rbacstore.SoftDeleteRoleInput{
		ID: toStoreID(roleID),
	})
	if !errors.Is(err, rbacstore.ErrRoleEnabledDeletionForbidden) {
		t.Fatalf("expected enabled role delete to return ErrRoleEnabledDeletionForbidden, got %v", err)
	}

	assertRoleLifecycleFields(t, db, roleID, lifecycleSnapshot{
		disabledAtNonZero: false,
		deletedAtNonZero:  false,
	})
}

func TestRepositoryListRolesByUserIDAndPermissionsSkipDisabledAndSoftDeletedRoles(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	userID := seedUser(t, db, "role-reader")
	enabledRoleID := seedRole(t, db, seededRoleRecord{name: "enabled-reader"})
	disabledRoleID := seedRole(t, db, seededRoleRecord{name: "disabled-reader", disabledAt: time.Now().UTC().Unix()})
	softDeletedRoleID := seedRole(t, db, seededRoleRecord{name: "deleted-reader", disabledAt: time.Now().UTC().Unix(), deletedAt: time.Now().UTC().Add(time.Minute).Unix()})
	permissionID := seedPermission(t, db, "audit.read")

	seedUserRoleBinding(t, db, userID, enabledRoleID)
	seedUserRoleBinding(t, db, userID, disabledRoleID)
	seedUserRoleBinding(t, db, userID, softDeletedRoleID)
	seedRolePermissionBinding(t, db, enabledRoleID, permissionID)
	seedRolePermissionBinding(t, db, disabledRoleID, permissionID)
	seedRolePermissionBinding(t, db, softDeletedRoleID, permissionID)

	roles, err := repo.ListRolesByUserID(context.Background(), toStoreID(userID))
	if err != nil {
		t.Fatalf("list roles by user id: %v", err)
	}
	assertRoleIDs(t, roles, []uint64{toStoreID(enabledRoleID)})

	permissions, err := repo.ListPermissionsByUserID(context.Background(), toStoreID(userID))
	if err != nil {
		t.Fatalf("list permissions by user id: %v", err)
	}
	if len(permissions) != 1 || permissions[0].Code != "audit.read" {
		t.Fatalf("expected one permission from enabled role only, got %#v", permissions)
	}

	userIDs, err := repo.ListUserIDsByPermissionCode(context.Background(), "audit.read")
	if err != nil {
		t.Fatalf("list user ids by permission code: %v", err)
	}
	if len(userIDs) != 1 || userIDs[0] != toStoreID(userID) {
		t.Fatalf("expected enabled-role permission lookup to return only target user, got %#v", userIDs)
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

	roleID := seedRole(t, db, seededRoleRecord{name: roleName})
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

func TestRepositoryRemovePermissionsFromRoleAllowsDisabledRoleCleanup(t *testing.T) {
	fixture := setupDisabledRolePermissionBindingFixture(t, "disabled-binding-remove-role", "permission.disabled.binding.remove")

	addErr := fixture.repo.AddPermissionsToRole(context.Background(), rbacstore.AddPermissionsToRoleInput{
		RoleID:        fixture.roleID,
		PermissionIDs: []uint64{fixture.permissionID},
	})
	if !errors.Is(addErr, rbacstore.ErrRoleDisabledAssignmentForbidden) {
		t.Fatalf("expected AddPermissionsToRole to reject disabled role mutations, got %v", addErr)
	}

	assertRolePermissionBindingSnapshot(t, fixture.repo, fixture.roleID, fixture.permissionID)

	if err := fixture.repo.RemovePermissionsFromRole(context.Background(), rbacstore.RemovePermissionsFromRoleInput{
		RoleID:        fixture.roleID,
		PermissionIDs: []uint64{fixture.permissionID},
	}); err != nil {
		t.Fatalf("expected RemovePermissionsFromRole to allow disabled role cleanup, got %v", err)
	}

	if count := countRows(t, fixture.db, "SELECT COUNT(*) FROM role_permissions"); count != 0 {
		t.Fatalf("expected disabled role cleanup to remove bindings, got %d", count)
	}
}

func TestRepositoryReplacePermissionsForRoleAllowsDisabledRoleCleanupToEmptySet(t *testing.T) {
	fixture := setupDisabledRolePermissionBindingFixture(t, "disabled-binding-replace-role", "permission.disabled.binding.replace")

	replaceErr := fixture.repo.ReplacePermissionsForRole(context.Background(), rbacstore.ReplacePermissionsForRoleInput{
		RoleID:        fixture.roleID,
		PermissionIDs: []uint64{fixture.permissionID},
	})
	if !errors.Is(replaceErr, rbacstore.ErrRoleDisabledAssignmentForbidden) {
		t.Fatalf("expected ReplacePermissionsForRole to reject disabled role non-empty mutations, got %v", replaceErr)
	}

	assertRolePermissionBindingSnapshot(t, fixture.repo, fixture.roleID, fixture.permissionID)

	if err := fixture.repo.ReplacePermissionsForRole(context.Background(), rbacstore.ReplacePermissionsForRoleInput{
		RoleID:        fixture.roleID,
		PermissionIDs: nil,
	}); err != nil {
		t.Fatalf("expected ReplacePermissionsForRole to allow disabled role cleanup to empty set, got %v", err)
	}

	if count := countRows(t, fixture.db, "SELECT COUNT(*) FROM role_permissions"); count != 0 {
		t.Fatalf("expected disabled role cleanup to remove bindings, got %d", count)
	}
}

type disabledRolePermissionBindingFixture struct {
	db           *sql.DB
	repo         *repository
	roleID       uint64
	permissionID uint64
}

func setupDisabledRolePermissionBindingFixture(t *testing.T, roleName string, permissionCode string) disabledRolePermissionBindingFixture {
	t.Helper()

	db := openTestDB(t)
	repo := &repository{db: db}

	roleID := seedRole(t, db, seededRoleRecord{name: roleName, disabledAt: time.Now().UTC().Unix()})
	permissionID := seedPermission(t, db, permissionCode)
	seedRolePermissionBinding(t, db, roleID, permissionID)

	return disabledRolePermissionBindingFixture{
		db:           db,
		repo:         repo,
		roleID:       toStoreID(roleID),
		permissionID: toStoreID(permissionID),
	}
}

func assertRolePermissionBindingSnapshot(t *testing.T, repo *repository, roleID uint64, permissionID uint64) {
	t.Helper()

	bindings, err := repo.ListRolePermissionBindings(context.Background(), roleID)
	if err != nil {
		t.Fatalf("expected ListRolePermissionBindings to keep disabled role snapshot readable, got %v", err)
	}
	if len(bindings) != 1 || bindings[0].PermissionID != permissionID {
		t.Fatalf("unexpected bindings for disabled role: %#v", bindings)
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

type seededRoleRecord struct {
	name       string
	builtin    bool
	disabledAt int64
	deletedAt  int64
}

type lifecycleSnapshot struct {
	disabledAtNonZero bool
	deletedAtNonZero  bool
}

func seedRole(t *testing.T, db *sql.DB, record seededRoleRecord) int64 {
	t.Helper()

	now := time.Now().UTC()
	result, err := db.ExecContext(context.Background(),
		`INSERT INTO roles (name, display, description, builtin, created_at, updated_at, disabled_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		record.name, record.name, nil, record.builtin, now, now, record.disabledAt, record.deletedAt,
	)
	if err != nil {
		t.Fatalf("seed role %s: %v", record.name, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read role id for %s: %v", record.name, err)
	}
	return id
}

func seedUserRoleBinding(t *testing.T, db *sql.DB, userID int64, roleID int64) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO user_roles (user_id, role_id, created_at) VALUES (?, ?, ?)`,
		userID,
		roleID,
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("seed user role binding user=%d role=%d: %v", userID, roleID, err)
	}
}

func assertRoleIDs(t *testing.T, roles []rbacstore.Role, expected []uint64) {
	t.Helper()

	actual := make([]uint64, 0, len(roles))
	for _, role := range roles {
		actual = append(actual, role.ID)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected role order/content: got=%v want=%v", actual, expected)
	}
}

func assertRoleLifecycleFields(t *testing.T, db *sql.DB, roleID int64, expected lifecycleSnapshot) {
	t.Helper()

	var disabledAt int64
	var deletedAt int64
	if err := db.QueryRowContext(context.Background(), `SELECT disabled_at, deleted_at FROM roles WHERE id = ?`, roleID).Scan(&disabledAt, &deletedAt); err != nil {
		t.Fatalf("read role lifecycle fields: %v", err)
	}
	if (disabledAt != 0) != expected.disabledAtNonZero {
		t.Fatalf("unexpected disabled_at for role %d: %d", roleID, disabledAt)
	}
	if (deletedAt != 0) != expected.deletedAtNonZero {
		t.Fatalf("unexpected deleted_at for role %d: %d", roleID, deletedAt)
	}
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
