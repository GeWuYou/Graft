package storeent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	rbacstore "graft/server/modules/rbac/store"
)

func (r *repository) AssignRoleToUser(ctx context.Context, input rbacstore.AssignRoleToUserInput) error {
	userID, err := toDBID(input.UserID)
	if err != nil {
		return err
	}
	roleID, err := toDBID(input.RoleID)
	if err != nil {
		return err
	}
	if err := r.ensureAssignableRoles(ctx, []int64{roleID}); err != nil {
		return err
	}

	if err := insertUserRole(ctx, userID, roleID, execQuerier{db: r.db}); err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return fmt.Errorf("assign role %d to user %d: %w", input.RoleID, input.UserID, err)
	}
	return nil
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
				return countEnabledRolesByIDs(ctx, tx, ids)
			},
			deleteStale: func(ctx context.Context, tx *sql.Tx, targetID int64, ids []int64) error {
				return deleteStableUserRoles(ctx, tx, targetID, ids)
			},
			bindingExists: func(ctx context.Context, tx *sql.Tx, targetID int64, relationID int64) (bool, error) {
				return recordExists(ctx, tx, "SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2", targetID, relationID)
			},
			createBinding: func(ctx context.Context, tx *sql.Tx, targetID int64, relationID int64) error {
				return insertUserRole(ctx, targetID, relationID, execQuerier{tx: tx})
			},
		},
	)
}

func (r *repository) ReplaceRolesForUsersAtomically(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	tx, committed, err := r.beginBatchUserRoleMutationTx(ctx, "start replace user roles batch tx")
	if err != nil {
		return err
	}
	defer rollbackUncommitted(tx, &committed)

	for _, userID := range input.UserIDs {
		if err := r.replaceRolesForUserTx(ctx, tx, rbacstore.ReplaceRolesForUserInput{
			UserID:  userID,
			RoleIDs: input.RoleIDs,
		}); err != nil {
			return err
		}
	}

	return commitBatchUserRoleMutationTx(tx, &committed, "commit replace user roles batch")
}

func (r *repository) AddRolesToUser(ctx context.Context, input rbacstore.AddRolesToUserInput) error {
	roleIDs, err := toUniqueDBIDs(input.RoleIDs)
	if err != nil {
		return err
	}
	if err := r.ensureAssignableRoles(ctx, roleIDs); err != nil {
		return err
	}

	userID, err := toDBID(input.UserID)
	if err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if err := insertUserRole(ctx, userID, roleID, execQuerier{db: r.db}); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return fmt.Errorf("add role %d to user %d: %w", roleID, input.UserID, err)
		}
	}

	return nil
}

func (r *repository) AddRolesToUsersAtomically(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	tx, committed, err := r.beginBatchUserRoleMutationTx(ctx, "start add user roles batch tx")
	if err != nil {
		return err
	}
	defer rollbackUncommitted(tx, &committed)

	roleIDs, err := toUniqueDBIDs(input.RoleIDs)
	if err != nil {
		return err
	}
	if err := ensureAssignableRolesTx(ctx, tx, roleIDs); err != nil {
		return err
	}

	for _, userID := range input.UserIDs {
		if err := addRolesToUserTx(ctx, tx, rbacstore.AddRolesToUserInput{
			UserID:  userID,
			RoleIDs: input.RoleIDs,
		}); err != nil {
			return err
		}
	}

	return commitBatchUserRoleMutationTx(tx, &committed, "commit add user roles batch")
}

func (r *repository) RemoveRolesFromUser(ctx context.Context, input rbacstore.RemoveRolesFromUserInput) error {
	userID, err := toDBID(input.UserID)
	if err != nil {
		return err
	}
	roleIDs, err := toUniqueDBIDs(input.RoleIDs)
	if err != nil {
		return err
	}
	if len(roleIDs) == 0 {
		return nil
	}

	query, args := buildDeleteBindingsQuery("DELETE FROM user_roles WHERE user_id = ?", userID, "role_id", roleIDs)
	_, execErr := r.db.ExecContext(ctx, query, args...)
	if execErr != nil {
		return fmt.Errorf("remove roles from user %d: %w", input.UserID, execErr)
	}
	return nil
}

func (r *repository) RemoveRolesFromUsersAtomically(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	tx, committed, err := r.beginBatchUserRoleMutationTx(ctx, "start remove user roles batch tx")
	if err != nil {
		return err
	}
	defer rollbackUncommitted(tx, &committed)

	for _, userID := range input.UserIDs {
		if err := removeRolesFromUserTx(ctx, tx, rbacstore.RemoveRolesFromUserInput{
			UserID:  userID,
			RoleIDs: input.RoleIDs,
		}); err != nil {
			return err
		}
	}

	return commitBatchUserRoleMutationTx(tx, &committed, "commit remove user roles batch")
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
		`SELECT r.id, r.name, r.display, r.description, r.builtin, r.deleted_at, r.created_at, r.updated_at,
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

	dbIDs, err := toUniqueDBIDs(userIDs)
	if err != nil {
		return nil, err
	}

	query, args := buildDollarInQuery(
		`SELECT ur.user_id, r.id, r.name, r.display, r.description, r.builtin, r.deleted_at, r.created_at, r.updated_at,
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

// insertUserRole 将用户和角色的绑定记录写入 user_roles，并设置创建时间为 UTC。
// @param userID 用户 ID。
// @param roleID 角色 ID。
// @param target 执行数据库操作的对象。
// @returns 执行插入时返回的错误；成功时为 nil。
func insertUserRole(ctx context.Context, userID int64, roleID int64, target execQuerier) error {
	_, err := target.ExecContext(
		ctx,
		`INSERT INTO user_roles (user_id, role_id, created_at)
		VALUES ($1, $2, $3)`,
		userID,
		roleID,
		time.Now().UTC(),
	)
	return err
}
