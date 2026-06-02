package storeent

import (
	"context"
	"database/sql"
	"errors"
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
			description TEXT NULL,
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

	now := time.Now().UTC()
	roleResult, err := db.ExecContext(context.Background(),
		`INSERT INTO roles (name, display, description, builtin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"editor", "编辑", nil, false, now, now,
	)
	if err != nil {
		t.Fatalf("seed role: %v", err)
	}
	roleID, err := roleResult.LastInsertId()
	if err != nil {
		t.Fatalf("read role id: %v", err)
	}
	userResult, err := db.ExecContext(context.Background(),
		`INSERT INTO users (username, display, created_at, updated_at)
		VALUES (?, ?, ?, ?)`,
		"alice", "Alice", now, now,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID, err := userResult.LastInsertId()
	if err != nil {
		t.Fatalf("read user id: %v", err)
	}

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

func TestRepositoryEnsurePermissionAndListPermissionsIncludeTimestamps(t *testing.T) {
	db := openTestDB(t)
	repo := &repository{db: db}

	record, err := repo.EnsurePermission(context.Background(), rbacstore.EnsurePermissionInput{
		Code:        "user.create",
		Display:     "Create Users",
		Description: stringPtr("Allows creating user management data."),
		Category:    "api",
	})
	if err != nil {
		t.Fatalf("ensure permission: %v", err)
	}
	if record.Code != "user.create" {
		t.Fatalf("expected ensured permission code user.create, got %#v", record)
	}
	if record.CreatedAt.IsZero() || record.UpdatedAt.IsZero() {
		t.Fatalf("expected ensured permission timestamps, got %#v", record)
	}

	permissions, err := repo.ListPermissions(context.Background(), rbacstore.PermissionFilter{})
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	if len(permissions) != 1 {
		t.Fatalf("expected one permission, got %d", len(permissions))
	}
	if permissions[0].CreatedAt.IsZero() || permissions[0].UpdatedAt.IsZero() {
		t.Fatalf("expected listed permission timestamps, got %#v", permissions[0])
	}
}

func stringPtr(value string) *string {
	return &value
}
