package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	rbacstore "graft/server/modules/rbac/store"
)

//nolint:cyclop // 重复键重试流程需要保持显式，才能维持这个稳定 upsert 边界的可审计性。
func (r *repository) EnsureRole(ctx context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	record, err := r.findRoleByName(ctx, input.Name)
	if err == nil {
		if input.Builtin && !record.Builtin {
			record, err = r.setRoleBuiltin(ctx, record.ID, true, "upgrade ensured role builtin state")
			if err != nil {
				return rbacstore.Role{}, err
			}
		}
		return record, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return rbacstore.Role{}, fmt.Errorf("query ensured role by name: %w", err)
	}

	record, err = r.createRoleRecord(ctx, input)
	if err == nil {
		return record, nil
	}
	if !isUniqueViolation(err) {
		return rbacstore.Role{}, fmt.Errorf("create ensured role: %w", err)
	}

	record, err = r.findRoleByName(ctx, input.Name)
	if err != nil {
		return rbacstore.Role{}, fmt.Errorf("re-query ensured role after conflict: %w", err)
	}
	if input.Builtin && !record.Builtin {
		record, err = r.setRoleBuiltin(ctx, record.ID, true, "upgrade ensured role builtin state after conflict")
		if err != nil {
			return rbacstore.Role{}, err
		}
	}

	return record, nil
}

func (r *repository) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	record, err := r.createRoleRecord(ctx, rbacstore.EnsureRoleInput{
		Name:        input.Name,
		Display:     input.Display,
		Description: input.Description,
		Builtin:     input.Builtin,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return rbacstore.Role{}, rbacstore.ErrRoleNameConflict
		}
		return rbacstore.Role{}, fmt.Errorf("create role: %w", err)
	}
	return record, nil
}

func (r *repository) UpdateRole(ctx context.Context, input rbacstore.UpdateRoleInput) (rbacstore.Role, error) {
	roleID, err := toDBID(input.ID)
	if err != nil {
		return rbacstore.Role{}, err
	}

	record, err := r.queryRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		}
		return rbacstore.Role{}, fmt.Errorf("get role by id %d: %w", input.ID, err)
	}

	record.Name = input.Name
	record.Display = input.Display
	record.Description = input.Description
	record.UpdatedAt = time.Now().UTC()

	updated, err := r.updateRoleRecord(ctx, roleID, record)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		case isUniqueViolation(err):
			return rbacstore.Role{}, rbacstore.ErrRoleNameConflict
		default:
			return rbacstore.Role{}, fmt.Errorf("update role %d: %w", input.ID, err)
		}
	}

	return updated, nil
}

func (r *repository) SetRoleStatus(ctx context.Context, input rbacstore.SetRoleStatusInput) (rbacstore.Role, error) {
	roleID, err := toDBID(input.ID)
	if err != nil {
		return rbacstore.Role{}, err
	}

	switch input.Status {
	case rbacstore.RoleStatusEnabled:
		return r.enableRole(ctx, input.ID, roleID)
	case rbacstore.RoleStatusDisabled:
		return r.disableRole(ctx, input.ID, roleID)
	default:
		return rbacstore.Role{}, rbacstore.ErrInvalidID
	}
}

func (r *repository) SoftDeleteRole(ctx context.Context, input rbacstore.SoftDeleteRoleInput) error {
	roleID, err := toDBID(input.ID)
	if err != nil {
		return err
	}

	if err := r.ensureSoftDeletableRole(ctx, input.ID, roleID); err != nil {
		return err
	}

	result, execErr := r.db.ExecContext(
		ctx,
		`UPDATE roles
		SET deleted_at = COALESCE(NULLIF(deleted_at, 0), $2),
			deleted_by = 0,
			updated_at = $3,
			updated_by = 0
		WHERE id = $1`,
		roleID,
		time.Now().UTC().Unix(),
		time.Now().UTC(),
	)
	if execErr != nil {
		return fmt.Errorf("soft delete role %d: %w", input.ID, execErr)
	}
	affected, execErr := result.RowsAffected()
	if execErr != nil {
		return fmt.Errorf("read soft delete role %d rows affected: %w", input.ID, execErr)
	}
	if affected == 0 {
		return rbacstore.ErrRoleNotFound
	}

	return nil
}

func (r *repository) GetRoleByID(ctx context.Context, roleID uint64) (rbacstore.Role, error) {
	id, err := toDBID(roleID)
	if err != nil {
		return rbacstore.Role{}, err
	}

	record, err := r.queryRoleByIDIncludingDisabled(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		}
		return rbacstore.Role{}, fmt.Errorf("get role by id %d: %w", roleID, err)
	}

	return record, nil
}

func (r *repository) ListRoles(ctx context.Context, filter rbacstore.RoleFilter) ([]rbacstore.Role, error) {
	where := []string{"1=1"}
	var args []any
	switch strings.TrimSpace(filter.Status) {
	case "", rbacstore.RoleStatusEnabled:
		where = append(where, "deleted_at = 0")
	case rbacstore.RoleStatusDisabled:
		where = append(where, "deleted_at <> 0")
	default:
		return nil, rbacstore.ErrInvalidID
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		args = append(args, "%"+query+"%", "%"+query+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR display ILIKE $%d)", len(args)-1, len(args)))
	}
	if filter.Builtin != nil {
		args = append(args, *filter.Builtin)
		where = append(where, fmt.Sprintf("builtin = $%d", len(args)))
	}
	return queryAndScanRows(
		ctx,
		r.db,
		"list roles",
		fmt.Sprintf(`SELECT id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE %s
		ORDER BY id ASC`, strings.Join(where, " AND ")),
		scanRoleRows,
		args...,
	)
}

func (r *repository) enableRole(ctx context.Context, inputID uint64, roleID int64) (rbacstore.Role, error) {
	updatedAt := time.Now().UTC()
	record, err := scanRole(r.db.QueryRowContext(
		ctx,
		`UPDATE roles
		SET deleted_at = 0, deleted_by = 0, updated_at = $2, updated_by = 0
		WHERE id = $1 AND deleted_at <> 0
		RETURNING id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count`,
		roleID,
		updatedAt,
	))
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return rbacstore.Role{}, fmt.Errorf("enable role %d: %w", inputID, err)
	}

	record, err = r.loadRoleIncludingDisabled(ctx, inputID, roleID, "enable")
	if err != nil {
		return rbacstore.Role{}, err
	}
	return record, nil
}

func (r *repository) disableRole(ctx context.Context, inputID uint64, roleID int64) (rbacstore.Role, error) {
	record, err := r.loadRoleIncludingDisabled(ctx, inputID, roleID, "disable")
	if err != nil {
		return rbacstore.Role{}, err
	}
	if record.Builtin {
		return rbacstore.Role{}, rbacstore.ErrRoleBuiltinImmutable
	}

	deletedAt := time.Now().UTC().Unix()
	updatedAt := time.Now().UTC()
	record, err = scanRole(r.db.QueryRowContext(
		ctx,
		`UPDATE roles
		SET deleted_at = CASE WHEN deleted_at = 0 THEN $2 ELSE deleted_at END,
			deleted_by = CASE WHEN deleted_at = 0 THEN 0 ELSE deleted_by END,
			updated_at = $3,
			updated_by = 0
		WHERE id = $1
		RETURNING id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count`,
		roleID,
		deletedAt,
		updatedAt,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		}
		return rbacstore.Role{}, fmt.Errorf("disable role %d: %w", inputID, err)
	}
	return record, nil
}

func (r *repository) loadRoleIncludingDisabled(ctx context.Context, inputID uint64, roleID int64, action string) (rbacstore.Role, error) {
	record, err := r.queryRoleByIDIncludingDisabled(ctx, roleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		}
		return rbacstore.Role{}, fmt.Errorf("get role %d before %s: %w", inputID, action, err)
	}
	return record, nil
}

func (r *repository) ensureSoftDeletableRole(ctx context.Context, inputID uint64, roleID int64) error {
	role, err := r.loadRoleIncludingDisabled(ctx, inputID, roleID, "soft delete")
	if err != nil {
		return err
	}
	if role.Builtin {
		return rbacstore.ErrRoleBuiltinImmutable
	}
	if role.Status == rbacstore.RoleStatusEnabled {
		return rbacstore.ErrRoleEnabledDeletionForbidden
	}
	if role.PermissionCount > 0 || role.UserCount > 0 {
		return rbacstore.ErrRoleBindingsExist
	}
	return nil
}

func (r *repository) updateRoleRecord(ctx context.Context, roleID int64, record rbacstore.Role) (rbacstore.Role, error) {
	return scanRole(r.db.QueryRowContext(
		ctx,
		`UPDATE roles
		SET name = $2, display = $3, description = $4, updated_at = $5, updated_by = 0
		WHERE id = $1
		RETURNING id, name, display, description, builtin, deleted_at, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count`,
		roleID,
		record.Name,
		record.Display,
		nullableString(record.Description),
		record.UpdatedAt,
	))
}
