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

	rbacstore "graft/server/modules/rbac/store"
)

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
		deletedAt       int64
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
		&deletedAt,
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
		Status:          roleStatusFromDeletedAt(deletedAt),
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
		new(int64),
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.PermissionCount,
		&record.UserCount,
	); err != nil {
		return rbacstore.Role{}, err
	}

	record.Description = nullStringPtr(description)
	record.Status = rbacstore.RoleStatusEnabled
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
		displayKey       sql.NullString
		description      sql.NullString
		descriptionKey   sql.NullString
		category         string
		createdAt        time.Time
		updatedAt        time.Time
		roleBindingCount int
	)
	if err := scanner.Scan(
		&id,
		&code,
		&display,
		&displayKey,
		&description,
		&descriptionKey,
		&category,
		&createdAt,
		&updatedAt,
		&roleBindingCount,
	); err != nil {
		return rbacstore.Permission{}, err
	}

	return rbacstore.Permission{
		ID:               toStoreID(id),
		Code:             code,
		Display:          display,
		DisplayKey:       nullStringPtr(displayKey),
		Description:      nullStringPtr(description),
		DescriptionKey:   nullStringPtr(descriptionKey),
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

func roleStatusFromDeletedAt(deletedAt int64) string {
	if deletedAt != 0 {
		return rbacstore.RoleStatusDisabled
	}
	return rbacstore.RoleStatusEnabled
}

func buildDeleteBindingsQuery(base string, targetID int64, column string, relationIDs []int64) (string, []any) {
	query := base
	args := []any{targetID}
	if len(relationIDs) > 0 {
		query += " AND " + column + " IN (" + placeholders(len(relationIDs)) + ")"
		for _, id := range relationIDs {
			args = append(args, id)
		}
	}
	return rebindPositional(query, args)
}

func countEnabledRolesByIDs(ctx context.Context, tx *sql.Tx, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	query := `SELECT COUNT(*) FROM roles WHERE id IN (` + placeholders(len(ids)) + `) AND deleted_at = 0`
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

func (r *repository) ensurePermissionsExist(ctx context.Context, permissionIDs []int64) error {
	count, err := countExistingRecords(ctx, r.db, "permissions", permissionIDs)
	if err != nil {
		return fmt.Errorf("count permissions: %w", err)
	}
	if count != len(permissionIDs) {
		return rbacstore.ErrPermissionNotFound
	}
	return nil
}

func (r *repository) ensureAssignableRoles(ctx context.Context, roleIDs []int64) error {
	rows, err := queryRoleAssignmentStates(ctx, r.db, roleIDs)
	if err != nil {
		return err
	}
	return validateAssignableRoles(rows, roleIDs)
}

func ensureAssignableRolesTx(ctx context.Context, tx *sql.Tx, roleIDs []int64) error {
	rows, err := queryRoleAssignmentStatesTx(ctx, tx, roleIDs)
	if err != nil {
		return err
	}
	return validateAssignableRoles(rows, roleIDs)
}

func validateAssignableRoles(rows []roleAssignmentState, roleIDs []int64) error {
	if len(rows) != len(roleIDs) {
		return rbacstore.ErrRoleNotFound
	}
	for _, item := range rows {
		if item.deletedAt != 0 {
			return rbacstore.ErrRoleDisabledAssignmentForbidden
		}
	}
	return nil
}

type roleAssignmentState struct {
	id        int64
	deletedAt int64
}

func queryRoleAssignmentStates(ctx context.Context, db *sql.DB, roleIDs []int64) ([]roleAssignmentState, error) {
	if len(roleIDs) == 0 {
		return []roleAssignmentState{}, nil
	}
	query, args := buildDollarInQuery(
		`SELECT id, deleted_at FROM roles WHERE id IN (?)`,
		roleIDs,
	)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query role assignment states: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanRoleAssignmentStates(rows)
}

func queryRoleAssignmentStatesTx(ctx context.Context, tx *sql.Tx, roleIDs []int64) ([]roleAssignmentState, error) {
	if len(roleIDs) == 0 {
		return []roleAssignmentState{}, nil
	}
	query, args := buildDollarInQuery(
		`SELECT id, deleted_at FROM roles WHERE id IN (?)`,
		roleIDs,
	)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query role assignment states: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanRoleAssignmentStates(rows)
}

func scanRoleAssignmentStates(rows *sql.Rows) ([]roleAssignmentState, error) {
	result := make([]roleAssignmentState, 0)
	for rows.Next() {
		var item roleAssignmentState
		if scanErr := rows.Scan(&item.id, &item.deletedAt); scanErr != nil {
			return nil, fmt.Errorf("scan role assignment states: %w", scanErr)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate role assignment states: %w", err)
	}
	return result, nil
}

func countExistingRecords(ctx context.Context, db *sql.DB, table string, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	baseQuery, err := countExistingRecordsQuery(table)
	if err != nil {
		return 0, err
	}
	query, args := buildDollarInQuery(baseQuery, ids)
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func countExistingRecordsQuery(table string) (string, error) {
	switch table {
	case "permissions":
		return `SELECT COUNT(*) FROM permissions WHERE id IN (?) AND deleted_at = 0`, nil
	case "users":
		return `SELECT COUNT(*) FROM users WHERE id IN (?) AND deleted_at = 0`, nil
	default:
		return "", fmt.Errorf("unsupported countExistingRecords table %q", table)
	}
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func stringPtrEqual(left *string, right *string) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
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
	return false
}
