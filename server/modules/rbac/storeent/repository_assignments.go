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
	if _, err := toDBID(targetID); err != nil {
		return err
	}
	if _, err := toUniqueDBIDs(relationIDs); err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", config.startContext, err)
	}
	committed := false
	defer rollbackUncommitted(tx, &committed)

	if err := replaceStableAssignmentsTx(ctx, tx, targetID, relationIDs, config); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(config.commitFormat+": %w", targetID, err)
	}
	committed = true
	return nil
}

func replaceStableAssignmentsTx(
	ctx context.Context,
	tx *sql.Tx,
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
	return nil
}

func rollbackUncommitted(tx *sql.Tx, committed *bool) {
	if !*committed {
		_ = tx.Rollback()
	}
}

func (r *repository) beginBatchUserRoleMutationTx(ctx context.Context, contextText string) (*sql.Tx, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", contextText, err)
	}
	return tx, false, nil
}

func commitBatchUserRoleMutationTx(tx *sql.Tx, committed *bool, contextText string) error {
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", contextText, err)
	}
	*committed = true
	return nil
}

func (r *repository) replaceRolesForUserTx(ctx context.Context, tx *sql.Tx, input rbacstore.ReplaceRolesForUserInput) error {
	return replaceStableAssignmentsTx(
		ctx,
		tx,
		input.UserID,
		input.RoleIDs,
		replaceAssignmentConfig{
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

func addRolesToUserTx(ctx context.Context, tx *sql.Tx, input rbacstore.AddRolesToUserInput) error {
	roleIDs, err := toUniqueDBIDs(input.RoleIDs)
	if err != nil {
		return err
	}
	userID, err := toDBID(input.UserID)
	if err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		_, execErr := tx.ExecContext(
			ctx,
			`INSERT INTO user_roles (user_id, role_id, created_at)
			VALUES ($1, $2, $3)`,
			userID,
			roleID,
			time.Now().UTC(),
		)
		if execErr == nil || isUniqueViolation(execErr) {
			continue
		}
		return fmt.Errorf("add role %d to user %d: %w", roleID, input.UserID, execErr)
	}
	return nil
}

func removeRolesFromUserTx(ctx context.Context, tx *sql.Tx, input rbacstore.RemoveRolesFromUserInput) error {
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
	if _, execErr := tx.ExecContext(ctx, query, args...); execErr != nil {
		return fmt.Errorf("remove roles from user %d: %w", input.UserID, execErr)
	}
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
