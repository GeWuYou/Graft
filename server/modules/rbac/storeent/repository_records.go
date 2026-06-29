package storeent

import (
	"context"
	"fmt"
	"strings"
	"time"

	rbacstore "graft/server/modules/rbac/store"
)

func (r *repository) queryRoleByID(ctx context.Context, id int64) (rbacstore.Role, error) {
	return scanRole(r.db.QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE id = $1 AND deleted_at = 0`,
		id,
	))
}

func (r *repository) queryRoleByIDIncludingDisabled(ctx context.Context, id int64) (rbacstore.Role, error) {
	return scanRole(r.db.QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE id = $1`,
		id,
	))
}

func (r *repository) findRoleByName(ctx context.Context, name string) (rbacstore.Role, error) {
	return scanRole(r.db.QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE name = $1 AND deleted_at = 0`,
		strings.TrimSpace(name),
	))
}

func (r *repository) createRoleRecord(ctx context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	now := time.Now().UTC()
	return scanRole(r.db.QueryRowContext(
		ctx,
		`INSERT INTO roles (name, display, description, builtin, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
		VALUES ($1, $2, $3, $4, $5, 0, $6, 0, 0, 0)
		RETURNING id, name, display, description, builtin, deleted_at, created_at, updated_at,
			0 AS permission_count,
			0 AS user_count`,
		strings.TrimSpace(input.Name),
		input.Display,
		nullableString(input.Description),
		input.Builtin,
		now,
		now,
	))
}

func (r *repository) setRoleBuiltin(ctx context.Context, id uint64, builtin bool, errorContext string) (rbacstore.Role, error) {
	dbID, err := toDBID(id)
	if err != nil {
		return rbacstore.Role{}, err
	}

	record, err := scanRole(r.db.QueryRowContext(
		ctx,
		`UPDATE roles
		SET builtin = $2, updated_at = $3, updated_by = 0
		WHERE id = $1
		RETURNING id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count`,
		dbID,
		builtin,
		time.Now().UTC(),
	))
	if err != nil {
		return rbacstore.Role{}, fmt.Errorf("%s: %w", errorContext, err)
	}
	return record, nil
}

func (r *repository) findPermissionByCode(ctx context.Context, code string) (rbacstore.Permission, error) {
	return scanPermission(r.db.QueryRowContext(
		ctx,
		`SELECT id, code, display, display_key, description, description_key, category, created_at, updated_at, 0 AS role_binding_count
		FROM permissions
		WHERE code = $1 AND deleted_at = 0`,
		strings.TrimSpace(code),
	))
}

func (r *repository) queryPermissionByID(ctx context.Context, id int64) (rbacstore.Permission, error) {
	return scanPermission(r.db.QueryRowContext(
		ctx,
		`SELECT id, code, display, display_key, description, description_key, category, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.permission_id = permissions.id) AS role_binding_count
		FROM permissions
		WHERE id = $1 AND deleted_at = 0`,
		id,
	))
}

func (r *repository) createPermissionRecord(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	now := time.Now().UTC()
	return scanPermission(r.db.QueryRowContext(
		ctx,
		`INSERT INTO permissions (code, display, display_key, description, description_key, category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8, 0, 0, 0)
		RETURNING id, code, display, display_key, description, description_key, category, created_at, updated_at, 0 AS role_binding_count`,
		strings.TrimSpace(input.Code),
		input.Display,
		nullableString(input.DisplayKey),
		nullableString(input.Description),
		nullableString(input.DescriptionKey),
		input.Category,
		now,
		now,
	))
}

func (r *repository) reconcilePermissionMetadata(
	ctx context.Context,
	record rbacstore.Permission,
	input rbacstore.EnsurePermissionInput,
) (rbacstore.Permission, error) {
	permissionID, err := toDBID(record.ID)
	if err != nil {
		return rbacstore.Permission{}, err
	}
	metadata := permissionMetadataFromInput(record, input)

	if permissionMetadataEqual(record, metadata) {
		return record, nil
	}

	if err := r.updatePermissionMetadata(ctx, permissionID, record.Code, metadata); err != nil {
		return rbacstore.Permission{}, err
	}
	updated, err := r.findPermissionByCode(ctx, record.Code)
	if err != nil {
		return rbacstore.Permission{}, fmt.Errorf("reload reconciled permission %s: %w", record.Code, err)
	}
	return updated, nil
}

type permissionMetadata struct {
	display        string
	displayKey     *string
	description    *string
	descriptionKey *string
	category       string
}

func permissionMetadataFromInput(record rbacstore.Permission, input rbacstore.EnsurePermissionInput) permissionMetadata {
	display := strings.TrimSpace(input.Display)
	category := strings.TrimSpace(input.Category)
	if display == "" {
		display = record.Display
	}
	if category == "" {
		category = record.Category
	}
	return permissionMetadata{
		display:        display,
		displayKey:     input.DisplayKey,
		description:    input.Description,
		descriptionKey: input.DescriptionKey,
		category:       category,
	}
}

func permissionMetadataEqual(record rbacstore.Permission, metadata permissionMetadata) bool {
	return record.Display == metadata.display &&
		stringPtrEqual(record.DisplayKey, metadata.displayKey) &&
		stringPtrEqual(record.Description, metadata.description) &&
		stringPtrEqual(record.DescriptionKey, metadata.descriptionKey) &&
		record.Category == metadata.category
}

func (r *repository) updatePermissionMetadata(
	ctx context.Context,
	permissionID int64,
	code string,
	metadata permissionMetadata,
) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE permissions
		SET display = $1, display_key = $2, description = $3, description_key = $4, category = $5, updated_at = $6, updated_by = 0
		WHERE id = $7 AND deleted_at = 0`,
		metadata.display,
		nullableString(metadata.displayKey),
		nullableString(metadata.description),
		nullableString(metadata.descriptionKey),
		metadata.category,
		now,
		permissionID,
	)
	if err != nil {
		return fmt.Errorf("reconcile permission %s metadata: %w", code, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read reconciled permission %s rows affected: %w", code, err)
	}
	if affected == 0 {
		return rbacstore.ErrPermissionNotFound
	}
	return nil
}
