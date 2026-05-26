package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	rbacstore "graft/server/plugins/rbac/store"
)

type repository struct {
	db *sql.DB
}

// NewRepository 基于共享连接池构建 RBAC 插件的 SQL repository。
func NewRepository(db *sql.DB) (rbacstore.Repository, error) {
	if db == nil {
		return nil, errors.New("rbac repository requires a non-nil sql db")
	}

	return &repository{db: db}, nil
}

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

func (r *repository) EnsurePermission(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	record, err := r.findPermissionByCode(ctx, input.Code)
	if err == nil {
		return record, nil
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

func (r *repository) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	record, err := r.createRoleRecord(ctx, rbacstore.EnsureRoleInput(input))
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

	row := r.db.QueryRowContext(
		ctx,
		`UPDATE roles
		SET name = $2, display = $3, description = $4, updated_at = $5, updated_by = 0
		WHERE id = $1
		RETURNING id, name, display, description, builtin, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count`,
		roleID,
		record.Name,
		record.Display,
		nullableString(record.Description),
		record.UpdatedAt,
	)

	updated, err := scanRole(row)
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

		_, err = r.db.ExecContext(
			ctx,
			`INSERT INTO role_permissions (role_id, permission_id, created_at)
			VALUES ($1, $2, $3)`,
			roleID,
			permissionID,
			time.Now().UTC(),
		)
		if err == nil || isUniqueViolation(err) {
			continue
		}

		return fmt.Errorf("assign permission %d to role %d: %w", permissionIDValue, input.RoleID, err)
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
				return recordExists(
					ctx,
					tx,
					"SELECT 1 FROM role_permissions WHERE role_id = $1 AND permission_id = $2",
					targetID,
					relationID,
				)
			},
			createBinding: func(ctx context.Context, tx *sql.Tx, targetID int64, relationID int64) error {
				_, err := tx.ExecContext(
					ctx,
					`INSERT INTO role_permissions (role_id, permission_id, created_at)
					VALUES ($1, $2, $3)`,
					targetID,
					relationID,
					time.Now().UTC(),
				)
				return err
			},
		},
	)
}

func (r *repository) AssignRoleToUser(ctx context.Context, input rbacstore.AssignRoleToUserInput) error {
	userID, err := toDBID(input.UserID)
	if err != nil {
		return err
	}
	roleID, err := toDBID(input.RoleID)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO user_roles (user_id, role_id, created_at)
		VALUES ($1, $2, $3)`,
		userID,
		roleID,
		time.Now().UTC(),
	)
	if err == nil || isUniqueViolation(err) {
		return nil
	}

	return fmt.Errorf("assign role %d to user %d: %w", input.RoleID, input.UserID, err)
}

func (r *repository) ReplaceRolesForUser(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
	return r.replaceStableAssignments(
		ctx,
		input.UserID,
		input.RoleIDs,
		replaceAssignmentConfig{
			startContext:         "start replace user roles tx",
			commitFormat:         "commit replace user roles for user %d",
			checkTargetContext:   "check user %d before replacing roles",
			countRelationContext: "count roles for user %d replacement",
			deleteStaleContext:   "delete stale roles for user %d",
			checkBindingContext:  "check user role replacement",
			createBindingContext: "replace role %d for user %d",
			targetMissing:        nil,
			relationMissing:      rbacstore.ErrRoleNotFound,
			checkTargetExists: func(context.Context, *sql.Tx, int64) (bool, error) {
				return true, nil
			},
			countRelationRecords: func(ctx context.Context, tx *sql.Tx, ids []int64) (int, error) {
				return countRecordsByIDsWhere(ctx, tx, "roles", "deleted_at = 0", ids)
			},
			deleteStale: func(ctx context.Context, tx *sql.Tx, targetID int64, ids []int64) error {
				return deleteStableUserRoles(ctx, tx, targetID, ids)
			},
			bindingExists: func(ctx context.Context, tx *sql.Tx, targetID int64, relationID int64) (bool, error) {
				return recordExists(
					ctx,
					tx,
					"SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2",
					targetID,
					relationID,
				)
			},
			createBinding: func(ctx context.Context, tx *sql.Tx, targetID int64, relationID int64) error {
				_, err := tx.ExecContext(
					ctx,
					`INSERT INTO user_roles (user_id, role_id, created_at)
					VALUES ($1, $2, $3)`,
					targetID,
					relationID,
					time.Now().UTC(),
				)
				return err
			},
		},
	)
}

func (r *repository) GetRoleByID(ctx context.Context, roleID uint64) (rbacstore.Role, error) {
	id, err := toDBID(roleID)
	if err != nil {
		return rbacstore.Role{}, err
	}

	record, err := r.queryRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		}
		return rbacstore.Role{}, fmt.Errorf("get role by id %d: %w", roleID, err)
	}

	return record, nil
}

func (r *repository) ListRolesByUserID(ctx context.Context, userID uint64) ([]rbacstore.Role, error) {
	id, err := toDBID(userID)
	if err != nil {
		return nil, err
	}

	return queryAndScanRows(
		ctx,
		r.db,
		"list roles by user id",
		`SELECT r.id, r.name, r.display, r.description, r.builtin, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = r.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur2 WHERE ur2.role_id = r.id) AS user_count
		FROM user_roles ur
		INNER JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.deleted_at = 0
		ORDER BY r.id ASC`,
		scanRoleRows,
		id,
	)
}

func (r *repository) ListRolesByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64][]rbacstore.Role, error) {
	if len(userIDs) == 0 {
		return map[uint64][]rbacstore.Role{}, nil
	}

	dbIDs := make([]int64, 0, len(userIDs))
	for _, userID := range userIDs {
		id, err := toDBID(userID)
		if err != nil {
			return nil, err
		}
		dbIDs = append(dbIDs, id)
	}

	query, args := buildDollarInQuery(
		`SELECT ur.user_id, r.id, r.name, r.display, r.description, r.builtin, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = r.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur2 WHERE ur2.role_id = r.id) AS user_count
		FROM user_roles ur
		INNER JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id IN (?) AND r.deleted_at = 0
		ORDER BY ur.user_id ASC, r.id ASC`,
		dbIDs,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list roles by user ids: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rolesByUserID := make(map[uint64][]rbacstore.Role, len(userIDs))
	for _, userID := range userIDs {
		rolesByUserID[userID] = []rbacstore.Role{}
	}

	for rows.Next() {
		var userID int64
		role, scanErr := scanRoleWithUserID(rows, &userID)
		if scanErr != nil {
			return nil, fmt.Errorf("list roles by user ids: scan row: %w", scanErr)
		}

		targetUserID := toStoreID(userID)
		rolesByUserID[targetUserID] = append(rolesByUserID[targetUserID], role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list roles by user ids: iterate rows: %w", err)
	}

	return rolesByUserID, nil
}

func (r *repository) ListRoles(ctx context.Context) ([]rbacstore.Role, error) {
	return queryAndScanRows(
		ctx,
		r.db,
		"list roles",
		`SELECT id, name, display, description, builtin, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE deleted_at = 0
		ORDER BY id ASC`,
		scanRoleRows,
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
		`SELECT DISTINCT p.id, p.code, p.display, p.description, p.category, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.permission_id = p.id) AS role_binding_count
		FROM user_roles ur
		INNER JOIN role_permissions rp ON rp.role_id = ur.role_id
		INNER JOIN permissions p ON p.id = rp.permission_id
		WHERE ur.user_id = $1 AND p.deleted_at = 0
		ORDER BY p.id ASC`,
		scanPermissionRows,
		id,
	)
}

func (r *repository) ListPermissions(ctx context.Context) ([]rbacstore.Permission, error) {
	return queryAndScanRows(
		ctx,
		r.db,
		"list permissions",
		`SELECT id, code, display, description, category, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.permission_id = permissions.id) AS role_binding_count
		FROM permissions
		WHERE deleted_at = 0
		ORDER BY id ASC`,
		scanPermissionRows,
	)
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

func (r *repository) queryRoleByID(ctx context.Context, id int64) (rbacstore.Role, error) {
	return scanRole(r.db.QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, created_at, updated_at,
			(SELECT COUNT(*) FROM role_permissions rp WHERE rp.role_id = roles.id) AS permission_count,
			(SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = roles.id) AS user_count
		FROM roles
		WHERE id = $1 AND deleted_at = 0`,
		id,
	))
}

func (r *repository) findRoleByName(ctx context.Context, name string) (rbacstore.Role, error) {
	return scanRole(r.db.QueryRowContext(
		ctx,
		`SELECT id, name, display, description, builtin, created_at, updated_at,
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
		RETURNING id, name, display, description, builtin, created_at, updated_at,
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
		RETURNING id, name, display, description, builtin, created_at, updated_at,
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
		`SELECT id, code, display, description, category, created_at, updated_at, 0 AS role_binding_count
		FROM permissions
		WHERE code = $1 AND deleted_at = 0`,
		strings.TrimSpace(code),
	))
}

func (r *repository) createPermissionRecord(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	now := time.Now().UTC()
	return scanPermission(r.db.QueryRowContext(
		ctx,
		`INSERT INTO permissions (code, display, description, category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
		VALUES ($1, $2, $3, $4, $5, 0, $6, 0, 0, 0)
		RETURNING id, code, display, description, category, created_at, updated_at, 0 AS role_binding_count`,
		strings.TrimSpace(input.Code),
		input.Display,
		nullableString(input.Description),
		input.Category,
		now,
		now,
	))
}

type replaceAssignmentConfig struct {
	startContext         string
	commitFormat         string
	checkTargetContext   string
	countRelationContext string
	deleteStaleContext   string
	checkBindingContext  string
	createBindingContext string
	targetMissing        error
	relationMissing      error
	checkTargetExists    func(context.Context, *sql.Tx, int64) (bool, error)
	countRelationRecords func(context.Context, *sql.Tx, []int64) (int, error)
	deleteStale          func(context.Context, *sql.Tx, int64, []int64) error
	bindingExists        func(context.Context, *sql.Tx, int64, int64) (bool, error)
	createBinding        func(context.Context, *sql.Tx, int64, int64) error
}

//nolint:gocognit,gocyclo // 这里保持替换事务步骤显式且有序，便于审查稳定赋值语义。
func (r *repository) replaceStableAssignments(
	ctx context.Context,
	targetID uint64,
	relationIDs []uint64,
	config replaceAssignmentConfig,
) error {
	dbTargetID, err := toDBID(targetID)
	if err != nil {
		return err
	}
	dbRelationIDs, err := toUniqueDBIDs(relationIDs)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", config.startContext, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := ensureAssignmentTarget(ctx, tx, targetID, dbTargetID, config); err != nil {
		return err
	}
	if err := validateAssignmentRelations(ctx, tx, targetID, dbRelationIDs, config); err != nil {
		return err
	}
	if err := deleteAssignmentStaleRows(ctx, tx, targetID, dbTargetID, dbRelationIDs, config); err != nil {
		return err
	}
	if err := insertAssignmentRows(ctx, tx, targetID, dbTargetID, dbRelationIDs, config); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(config.commitFormat+": %w", targetID, err)
	}
	committed = true
	return nil
}

func ensureAssignmentTarget(
	ctx context.Context,
	tx *sql.Tx,
	targetID uint64,
	dbTargetID int64,
	config replaceAssignmentConfig,
) error {
	exists, err := config.checkTargetExists(ctx, tx, dbTargetID)
	if err != nil {
		return fmt.Errorf(config.checkTargetContext+": %w", targetID, err)
	}
	if !exists && config.targetMissing != nil {
		return config.targetMissing
	}
	return nil
}

func validateAssignmentRelations(
	ctx context.Context,
	tx *sql.Tx,
	targetID uint64,
	dbRelationIDs []int64,
	config replaceAssignmentConfig,
) error {
	if len(dbRelationIDs) == 0 {
		return nil
	}

	count, err := config.countRelationRecords(ctx, tx, dbRelationIDs)
	if err != nil {
		return fmt.Errorf(config.countRelationContext+": %w", targetID, err)
	}
	if count != len(dbRelationIDs) {
		return config.relationMissing
	}

	return nil
}

func deleteAssignmentStaleRows(
	ctx context.Context,
	tx *sql.Tx,
	targetID uint64,
	dbTargetID int64,
	dbRelationIDs []int64,
	config replaceAssignmentConfig,
) error {
	if err := config.deleteStale(ctx, tx, dbTargetID, dbRelationIDs); err != nil {
		return fmt.Errorf(config.deleteStaleContext+": %w", targetID, err)
	}
	return nil
}

func insertAssignmentRows(
	ctx context.Context,
	tx *sql.Tx,
	targetID uint64,
	dbTargetID int64,
	dbRelationIDs []int64,
	config replaceAssignmentConfig,
) error {
	for _, relationID := range dbRelationIDs {
		bindingExists, err := config.bindingExists(ctx, tx, dbTargetID, relationID)
		if err != nil {
			return fmt.Errorf("%s: %w", config.checkBindingContext, err)
		}
		if bindingExists {
			continue
		}

		if err := config.createBinding(ctx, tx, dbTargetID, relationID); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return fmt.Errorf(config.createBindingContext+": %w", relationID, targetID, err)
		}
	}

	return nil
}

//nolint:gosec // 查询形状只由固定 SQL 片段和占位符数量拼装。
func deleteStableRolePermissions(ctx context.Context, tx *sql.Tx, roleID int64, permissionIDs []int64) error {
	query := "DELETE FROM role_permissions WHERE role_id = ?"
	args := []any{roleID}
	if len(permissionIDs) > 0 {
		query += " AND permission_id NOT IN (" + placeholders(len(permissionIDs)) + ")"
		for _, id := range permissionIDs {
			args = append(args, id)
		}
	}
	query, args = rebindPositional(query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

//nolint:gosec // 查询形状只由固定 SQL 片段和占位符数量拼装。
func deleteStableUserRoles(ctx context.Context, tx *sql.Tx, userID int64, roleIDs []int64) error {
	query := "DELETE FROM user_roles WHERE user_id = ?"
	args := []any{userID}
	if len(roleIDs) > 0 {
		query += " AND role_id NOT IN (" + placeholders(len(roleIDs)) + ")"
		for _, id := range roleIDs {
			args = append(args, id)
		}
	}
	query, args = rebindPositional(query, args)
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

//nolint:gosec // 调用方只会传入本包拥有的固定表名和固定 where 片段。
func countRecordsByIDsWhere(ctx context.Context, tx *sql.Tx, table string, extraWhere string, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id IN (%s)", table, placeholders(len(ids)))
	if strings.TrimSpace(extraWhere) != "" {
		query = fmt.Sprintf("%s AND %s", query, extraWhere)
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	query, args = rebindPositional(query, args)

	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func recordExists(ctx context.Context, tx *sql.Tx, query string, args ...any) (bool, error) {
	var marker int
	err := tx.QueryRowContext(ctx, query, args...).Scan(&marker)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	default:
		return false, err
	}
}

type roleScanner interface {
	Scan(dest ...any) error
}

//nolint:dupl // role 与 permission 的行映射器需要有意保持镜像结构。
func scanRole(scanner roleScanner) (rbacstore.Role, error) {
	var (
		id              int64
		name            string
		display         string
		description     sql.NullString
		builtin         bool
		createdAt       time.Time
		updatedAt       time.Time
		permissionCount int
		userCount       int
	)
	if err := scanner.Scan(
		&id,
		&name,
		&display,
		&description,
		&builtin,
		&createdAt,
		&updatedAt,
		&permissionCount,
		&userCount,
	); err != nil {
		return rbacstore.Role{}, err
	}

	return rbacstore.Role{
		ID:              toStoreID(id),
		Name:            name,
		Display:         display,
		Description:     nullStringPtr(description),
		Builtin:         builtin,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		PermissionCount: permissionCount,
		UserCount:       userCount,
	}, nil
}

func scanRoleRows(rows *sql.Rows) ([]rbacstore.Role, error) {
	roles := make([]rbacstore.Role, 0)
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func scanRoleWithUserID(scanner interface {
	Scan(dest ...any) error
}, userID *int64) (rbacstore.Role, error) {
	var record rbacstore.Role
	var description sql.NullString

	if err := scanner.Scan(
		userID,
		&record.ID,
		&record.Name,
		&record.Display,
		&description,
		&record.Builtin,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.PermissionCount,
		&record.UserCount,
	); err != nil {
		return rbacstore.Role{}, err
	}

	record.Description = nullStringPtr(description)
	return record, nil
}

func buildDollarInQuery(base string, ids []int64) (string, []any) {
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for index, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
		args = append(args, id)
	}

	return strings.Replace(base, "(?)", "("+strings.Join(placeholders, ", ")+")", 1), args
}

type permissionScanner interface {
	Scan(dest ...any) error
}

//nolint:dupl // role 与 permission 的行映射器需要有意保持镜像结构。
func scanPermission(scanner permissionScanner) (rbacstore.Permission, error) {
	var (
		id               int64
		code             string
		display          string
		description      sql.NullString
		category         string
		createdAt        time.Time
		updatedAt        time.Time
		roleBindingCount int
	)
	if err := scanner.Scan(&id, &code, &display, &description, &category, &createdAt, &updatedAt, &roleBindingCount); err != nil {
		return rbacstore.Permission{}, err
	}

	return rbacstore.Permission{
		ID:               toStoreID(id),
		Code:             code,
		Display:          display,
		Description:      nullStringPtr(description),
		Category:         category,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		RoleBindingCount: roleBindingCount,
	}, nil
}

func scanPermissionRows(rows *sql.Rows) ([]rbacstore.Permission, error) {
	permissions := make([]rbacstore.Permission, 0)
	for rows.Next() {
		permission, err := scanPermission(rows)
		if err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}

func queryAndScanRows[T any](
	ctx context.Context,
	db *sql.DB,
	contextLabel string,
	query string,
	scan func(*sql.Rows) ([]T, error),
	args ...any,
) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", contextLabel, err)
	}

	items, err := scan(rows)
	closeErr := rows.Close()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", contextLabel, err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close %s rows: %w", contextLabel, closeErr)
	}
	return items, nil
}

func toDBID(id uint64) (int64, error) {
	if id == 0 || id > math.MaxInt64 {
		return 0, rbacstore.ErrInvalidID
	}
	return int64(id), nil
}

func toStoreID(id int64) uint64 {
	//nolint:gosec // 数据库 ID 来自受控 schema，并保持为正数。
	return uint64(id)
}

func toUniqueDBIDs(ids []uint64) ([]int64, error) {
	if len(ids) == 0 {
		return []int64{}, nil
	}

	converted := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		dbID, err := toDBID(id)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[dbID]; ok {
			continue
		}
		seen[dbID] = struct{}{}
		converted = append(converted, dbID)
	}
	slices.Sort(converted)
	return converted, nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func rebindPositional(query string, args []any) (string, []any) {
	for index := range args {
		query = strings.Replace(query, "?", fmt.Sprintf("$%d", index+1), 1)
	}
	return query, args
}

func isUniqueViolation(err error) bool {
	type postgresCodeCarrier interface {
		SQLState() string
	}
	var pgErr postgresCodeCarrier
	if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
		return true
	}

	if isSQLiteUniqueViolation(err) {
		return true
	}

	// pgx surfaces duplicate-key failures with SQLSTATE 23505 in the error text
	// when the concrete pgconn type is only available transitively.
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key") ||
		strings.Contains(strings.ToLower(err.Error()), "sqlstate 23505")
}
