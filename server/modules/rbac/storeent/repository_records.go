package storeent

import (
	"context"
	"fmt"
	"strings"
	"time"

	rbacstore "graft/server/modules/rbac/store"
)

func (r *repository) queryRoleByID(ctx context.Context, id int64) (rbacstore.Role, error) {
	return scanRole(r.executor(ctx).QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, type, builtin_key, editable, disabled_at, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE id = $1 AND deleted_at = 0`,
		id,
	))
}

func (r *repository) queryRoleByIDIncludingDisabled(ctx context.Context, id int64) (rbacstore.Role, error) {
	return scanRole(r.executor(ctx).QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, type, builtin_key, editable, disabled_at, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE id = $1 AND deleted_at = 0`,
		id,
	))
}

func (r *repository) findRoleByName(ctx context.Context, name string) (rbacstore.Role, error) {
	return scanRole(r.executor(ctx).QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, type, builtin_key, editable, disabled_at, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE name = $1 AND deleted_at = 0`,
		strings.TrimSpace(name),
	))
}

func (r *repository) createRoleRecord(ctx context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	now := time.Now().UTC()
	return scanRole(r.executor(ctx).QueryRowContext(
		ctx,
		`INSERT INTO roles (name, display, description, builtin, type, builtin_key, editable, created_at, created_by, updated_at, updated_by, disabled_at, deleted_at, deleted_by)
		VALUES ($1, $2, $3, $4, CASE WHEN $4 THEN 'system' ELSE 'custom' END, NULL, NOT $4, $5, 0, $6, 0, 0, 0, 0)
		RETURNING id, name, display, description, builtin, type, builtin_key, editable, disabled_at, deleted_at, created_at, updated_at,
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

	record, err := scanRole(r.executor(ctx).QueryRowContext(
		ctx,
		`UPDATE roles
		SET builtin = $2, type = CASE WHEN $2 THEN 'system' ELSE 'custom' END, editable = NOT $2, updated_at = $3, updated_by = 0
		WHERE id = $1
		RETURNING id, name, display, description, builtin, type, builtin_key, editable, disabled_at, deleted_at, created_at, updated_at,
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
	return scanPermission(r.executor(ctx).QueryRowContext(
		ctx,
		`SELECT id, code, display, display_key, description, description_key, module, resource, action, risk_level, created_at, updated_at, 0 AS role_binding_count
		FROM permissions
		WHERE code = $1 AND deleted_at = 0`,
		strings.TrimSpace(code),
	))
}

func (r *repository) queryPermissionByID(ctx context.Context, id int64) (rbacstore.Permission, error) {
	return scanPermission(r.executor(ctx).QueryRowContext(
		ctx,
		`SELECT id, code, display, display_key, description, description_key, module, resource, action, risk_level, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.permission_id = permissions.id) AS role_binding_count
		FROM permissions
		WHERE id = $1 AND deleted_at = 0`,
		id,
	))
}

func (r *repository) createPermissionRecord(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	now := time.Now().UTC()
	return scanPermission(r.executor(ctx).QueryRowContext(
		ctx,
		`INSERT INTO permissions (code, display, display_key, description, description_key, module, resource, action, risk_level, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, $11, 0, 0, 0)
		RETURNING id, code, display, display_key, description, description_key, module, resource, action, risk_level, created_at, updated_at, 0 AS role_binding_count`,
		strings.TrimSpace(input.Code),
		input.Display,
		nullableString(input.DisplayKey),
		nullableString(input.Description),
		nullableString(input.DescriptionKey),
		input.Module,
		input.Resource,
		input.Action,
		input.RiskLevel,
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
	module         string
	resource       string
	action         string
	riskLevel      string
}

func permissionMetadataFromInput(record rbacstore.Permission, input rbacstore.EnsurePermissionInput) permissionMetadata {
	display := strings.TrimSpace(input.Display)
	module := strings.TrimSpace(input.Module)
	if display == "" {
		display = record.Display
	}
	if module == "" {
		module = record.Module
	}
	return permissionMetadata{
		display:        display,
		displayKey:     input.DisplayKey,
		description:    input.Description,
		descriptionKey: input.DescriptionKey,
		module:         module,
		resource:       strings.TrimSpace(input.Resource),
		action:         strings.TrimSpace(input.Action),
		riskLevel:      strings.TrimSpace(input.RiskLevel),
	}
}

func permissionMetadataEqual(record rbacstore.Permission, metadata permissionMetadata) bool {
	return record.Display == metadata.display &&
		stringPtrEqual(record.DisplayKey, metadata.displayKey) &&
		stringPtrEqual(record.Description, metadata.description) &&
		stringPtrEqual(record.DescriptionKey, metadata.descriptionKey) &&
		record.Module == metadata.module &&
		record.Resource == metadata.resource &&
		record.Action == metadata.action &&
		record.RiskLevel == metadata.riskLevel
}

func (r *repository) updatePermissionMetadata(
	ctx context.Context,
	permissionID int64,
	code string,
	metadata permissionMetadata,
) error {
	now := time.Now().UTC()
	result, err := r.executor(ctx).ExecContext(
		ctx,
		`UPDATE permissions
			SET display = $1, display_key = $2, description = $3, description_key = $4, module = $5, resource = $6, action = $7, risk_level = $8, updated_at = $9, updated_by = 0
			WHERE id = $10 AND deleted_at = 0`,
		metadata.display,
		nullableString(metadata.displayKey),
		nullableString(metadata.description),
		nullableString(metadata.descriptionKey),
		metadata.module,
		metadata.resource,
		metadata.action,
		metadata.riskLevel,
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
