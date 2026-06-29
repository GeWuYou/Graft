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

func (r *repository) EnsurePermission(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	record, err := r.findPermissionByCode(ctx, input.Code)
	if err == nil {
		return r.reconcilePermissionMetadata(ctx, record, input)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return rbacstore.Permission{}, fmt.Errorf("query ensured permission by code: %w", err)
	}

	record, err = r.createPermissionRecord(ctx, input)
	if err == nil {
		return record, nil
	}
	if !isUniqueViolation(err) {
		return rbacstore.Permission{}, fmt.Errorf("create ensured permission: %w", err)
	}

	record, err = r.findPermissionByCode(ctx, input.Code)
	if err != nil {
		return rbacstore.Permission{}, fmt.Errorf("re-query ensured permission after conflict: %w", err)
	}
	return record, nil
}

func (r *repository) AssignPermissionsToRole(ctx context.Context, input rbacstore.AssignPermissionsToRoleInput) error {
	roleID, err := toDBID(input.RoleID)
	if err != nil {
		return err
	}

	for _, permissionIDValue := range input.PermissionIDs {
		permissionID, err := toDBID(permissionIDValue)
		if err != nil {
			return err
		}

		if err := insertRolePermission(ctx, roleID, permissionID, execQuerier{db: r.db}); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return fmt.Errorf("assign permission %d to role %d: %w", permissionIDValue, input.RoleID, err)
		}
	}

	return nil
}

func (r *repository) ReplacePermissionsForRole(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error {
	return r.replaceStableAssignments(
		ctx,
		input.RoleID,
		input.PermissionIDs,
		replaceAssignmentConfig{
			startContext:         "start replace role permissions tx",
			commitFormat:         "commit replace role permissions for role %d",
			checkTargetContext:   "check role %d before replacing permissions",
			countRelationContext: "count permissions for role %d replacement",
			deleteStaleContext:   "delete stale permissions for role %d",
			checkBindingContext:  "check role permission replacement",
			createBindingContext: "replace permission %d for role %d",
			targetMissing:        rbacstore.ErrRoleNotFound,
			relationMissing:      rbacstore.ErrPermissionNotFound,
			checkTargetExists: func(ctx context.Context, tx *sql.Tx, targetID int64) (bool, error) {
				return recordExists(ctx, tx, "SELECT 1 FROM roles WHERE id = $1 AND deleted_at = 0", targetID)
			},
			countRelationRecords: func(ctx context.Context, tx *sql.Tx, ids []int64) (int, error) {
				return countRecordsByIDsWhere(ctx, tx, "permissions", "deleted_at = 0", ids)
			},
			deleteStale: func(ctx context.Context, tx *sql.Tx, targetID int64, ids []int64) error {
				return deleteStableRolePermissions(ctx, tx, targetID, ids)
			},
			bindingExists: func(ctx context.Context, tx *sql.Tx, targetID int64, relationID int64) (bool, error) {
				return recordExists(ctx, tx, "SELECT 1 FROM role_permissions WHERE role_id = $1 AND permission_id = $2", targetID, relationID)
			},
			createBinding: func(ctx context.Context, tx *sql.Tx, targetID int64, relationID int64) error {
				return insertRolePermission(ctx, targetID, relationID, execQuerier{tx: tx})
			},
		},
	)
}

func (r *repository) AddPermissionsToRole(ctx context.Context, input rbacstore.AddPermissionsToRoleInput) error {
	if _, err := r.GetRoleByID(ctx, input.RoleID); err != nil {
		return err
	}
	permissionIDs, err := toUniqueDBIDs(input.PermissionIDs)
	if err != nil {
		return err
	}
	if err := r.ensurePermissionsExist(ctx, permissionIDs); err != nil {
		return err
	}

	roleID, err := toDBID(input.RoleID)
	if err != nil {
		return err
	}
	for _, permissionID := range permissionIDs {
		if err := insertRolePermission(ctx, roleID, permissionID, execQuerier{db: r.db}); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return fmt.Errorf("add permission %d to role %d: %w", permissionID, input.RoleID, err)
		}
	}

	return nil
}

func (r *repository) RemovePermissionsFromRole(ctx context.Context, input rbacstore.RemovePermissionsFromRoleInput) error {
	if _, err := r.GetRoleByID(ctx, input.RoleID); err != nil {
		return err
	}
	roleID, err := toDBID(input.RoleID)
	if err != nil {
		return err
	}
	permissionIDs, err := toUniqueDBIDs(input.PermissionIDs)
	if err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}
	if err := r.ensurePermissionsExist(ctx, permissionIDs); err != nil {
		return err
	}

	query, args := buildDeleteBindingsQuery("DELETE FROM role_permissions WHERE role_id = ?", roleID, "permission_id", permissionIDs)
	_, execErr := r.db.ExecContext(ctx, query, args...)
	if execErr != nil {
		return fmt.Errorf("remove permissions from role %d: %w", input.RoleID, execErr)
	}
	return nil
}

func (r *repository) GetPermissionByID(ctx context.Context, permissionID uint64) (rbacstore.Permission, error) {
	id, err := toDBID(permissionID)
	if err != nil {
		return rbacstore.Permission{}, err
	}

	record, err := r.queryPermissionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rbacstore.Permission{}, rbacstore.ErrPermissionNotFound
		}
		return rbacstore.Permission{}, fmt.Errorf("get permission by id %d: %w", permissionID, err)
	}

	return record, nil
}

func (r *repository) ListPermissions(ctx context.Context, filter rbacstore.PermissionFilter) ([]rbacstore.Permission, error) {
	where := []string{"deleted_at = 0"}
	var args []any
	if category := strings.TrimSpace(filter.Category); category != "" {
		args = append(args, category)
		where = append(where, fmt.Sprintf("category = $%d", len(args)))
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		args = append(args, "%"+query+"%", "%"+query+"%", "%"+query+"%")
		codeIndex := len(args) - (permissionSearchFields - 1)
		displayIndex := len(args) - 1
		categoryIndex := len(args)
		where = append(where, fmt.Sprintf("(code ILIKE $%d OR display ILIKE $%d OR category ILIKE $%d)", codeIndex, displayIndex, categoryIndex))
	}
	return queryAndScanRows(
		ctx,
		r.db,
		"list permissions",
		fmt.Sprintf(`SELECT id, code, display, display_key, description, description_key, category, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.permission_id = permissions.id) AS role_binding_count
		FROM permissions
		WHERE %s
		ORDER BY id ASC`, strings.Join(where, " AND ")),
		scanPermissionRows,
		args...,
	)
}

func (r *repository) ListPermissionsByUserID(ctx context.Context, userID uint64) ([]rbacstore.Permission, error) {
	id, err := toDBID(userID)
	if err != nil {
		return nil, err
	}

	return queryAndScanRows(
		ctx,
		r.db,
		"list permissions by user id",
		`SELECT DISTINCT p.id, p.code, p.display, p.display_key, p.description, p.description_key, p.category, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.permission_id = p.id) AS role_binding_count
		FROM user_roles ur
		INNER JOIN roles r ON r.id = ur.role_id
		INNER JOIN role_permissions rp ON rp.role_id = ur.role_id
		INNER JOIN permissions p ON p.id = rp.permission_id
		WHERE ur.user_id = $1 AND r.deleted_at = 0 AND p.deleted_at = 0
		ORDER BY p.id ASC`,
		scanPermissionRows,
		id,
	)
}

func (r *repository) ListUserIDsByPermissionCode(ctx context.Context, permissionCode string) ([]uint64, error) {
	code := strings.TrimSpace(permissionCode)
	if code == "" {
		return nil, rbacstore.ErrPermissionNotFound
	}

	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT ur.user_id
		FROM user_roles ur
		INNER JOIN roles r ON r.id = ur.role_id
		INNER JOIN role_permissions rp ON rp.role_id = ur.role_id
		INNER JOIN permissions p ON p.id = rp.permission_id
		WHERE p.code = $1 AND r.deleted_at = 0 AND p.deleted_at = 0
		ORDER BY ur.user_id ASC`, code)
	if err != nil {
		return nil, fmt.Errorf("list user ids by permission code: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	userIDs := make([]uint64, 0)
	for rows.Next() {
		var raw int64
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan permission user id: %w", err)
		}
		userIDs = append(userIDs, toStoreID(raw))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permission user ids: %w", err)
	}
	return userIDs, nil
}

func (r *repository) ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]rbacstore.RolePermissionBinding, error) {
	id, err := toDBID(roleID)
	if err != nil {
		return nil, err
	}

	if _, err := r.queryRoleByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rbacstore.ErrRoleNotFound
		}
		return nil, fmt.Errorf("get role for permission bindings: %w", err)
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT permission_id
		FROM role_permissions
		WHERE role_id = $1
		ORDER BY permission_id ASC`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("list role permission bindings: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	bindings := make([]rbacstore.RolePermissionBinding, 0)
	for rows.Next() {
		var permissionID int64
		if err := rows.Scan(&permissionID); err != nil {
			return nil, fmt.Errorf("scan role permission binding: %w", err)
		}
		bindings = append(bindings, rbacstore.RolePermissionBinding{
			RoleID:       roleID,
			PermissionID: toStoreID(permissionID),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate role permission bindings: %w", err)
	}

	return bindings, nil
}

type execQuerier struct {
	db *sql.DB
	tx *sql.Tx
}

func (q execQuerier) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	switch {
	case q.tx != nil:
		return q.tx.ExecContext(ctx, query, args...)
	case q.db != nil:
		return q.db.ExecContext(ctx, query, args...)
	default:
		return nil, errors.New("sql exec target is unavailable")
	}
}

// insertRolePermission 向 role_permissions 中插入角色与权限绑定记录。
//
// 记录的 created_at 使用当前 UTC 时间。
func insertRolePermission(ctx context.Context, roleID int64, permissionID int64, target execQuerier) error {
	_, err := target.ExecContext(
		ctx,
		`INSERT INTO role_permissions (role_id, permission_id, created_at)
		VALUES ($1, $2, $3)`,
		roleID,
		permissionID,
		time.Now().UTC(),
	)
	return err
}
