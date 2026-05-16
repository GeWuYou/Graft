package entstore

import (
	"context"
	"fmt"

	"graft/server/internal/ent"
	entpermission "graft/server/internal/ent/permission"
	entrole "graft/server/internal/ent/role"
	entrolepermission "graft/server/internal/ent/rolepermission"
	entuser "graft/server/internal/ent/user"
	entuserrole "graft/server/internal/ent/userrole"
	"graft/server/internal/store"
)

type rbacRepository struct {
	client *ent.Client
}

// EnsureRole 幂等确保目标角色存在。
func (r *rbacRepository) EnsureRole(ctx context.Context, input store.EnsureRoleInput) (store.Role, error) {
	record, err := r.client.Role.Query().
		Where(entrole.NameEQ(input.Name)).
		Only(ctx)
	if err == nil {
		if input.Builtin && !record.Builtin {
			updated, updateErr := r.client.Role.UpdateOneID(record.ID).
				SetBuiltin(true).
				Save(ctx)
			if updateErr != nil {
				return store.Role{}, fmt.Errorf("upgrade ensured role builtin state: %w", updateErr)
			}

			record = updated
		}

		return toStoreRole(record), nil
	}
	if !ent.IsNotFound(err) {
		return store.Role{}, fmt.Errorf("query ensured role by name: %w", err)
	}

	record, err = r.client.Role.Create().
		SetName(input.Name).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		SetBuiltin(input.Builtin).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			record, lookupErr := r.client.Role.Query().
				Where(entrole.NameEQ(input.Name)).
				Only(ctx)
			if lookupErr != nil {
				return store.Role{}, fmt.Errorf("re-query ensured role after conflict: %w", lookupErr)
			}

			if input.Builtin && !record.Builtin {
				updated, updateErr := r.client.Role.UpdateOneID(record.ID).
					SetBuiltin(true).
					Save(ctx)
				if updateErr != nil {
					return store.Role{}, fmt.Errorf("upgrade ensured role builtin state after conflict: %w", updateErr)
				}

				record = updated
			}

			return toStoreRole(record), nil
		}

		return store.Role{}, fmt.Errorf("create ensured role: %w", err)
	}

	return toStoreRole(record), nil
}

// EnsurePermission 幂等确保目标权限存在。
func (r *rbacRepository) EnsurePermission(ctx context.Context, input store.EnsurePermissionInput) (store.Permission, error) {
	return ensureUniqueEntity(
		func() (*ent.Permission, error) {
			return r.client.Permission.Query().
				Where(entpermission.CodeEQ(input.Code)).
				Only(ctx)
		},
		func() (*ent.Permission, error) {
			return r.client.Permission.Create().
				SetCode(input.Code).
				SetDisplay(input.Display).
				SetNillableDescription(input.Description).
				SetCategory(input.Category).
				Save(ctx)
		},
		toStorePermission,
		"query ensured permission by code",
		"create ensured permission",
		"re-query ensured permission after conflict",
	)
}

// CreateRole 显式创建一个角色。
func (r *rbacRepository) CreateRole(ctx context.Context, input store.CreateRoleInput) (store.Role, error) {
	record, err := r.client.Role.Create().
		SetName(input.Name).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		SetBuiltin(input.Builtin).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return store.Role{}, store.ErrRoleNameConflict
		}

		return store.Role{}, fmt.Errorf("create role: %w", err)
	}

	return toStoreRole(record), nil
}

// UpdateRole 按稳定 ID 更新一个角色。
func (r *rbacRepository) UpdateRole(ctx context.Context, input store.UpdateRoleInput) (store.Role, error) {
	roleID, err := toEntID(input.ID)
	if err != nil {
		return store.Role{}, err
	}

	record, err := r.client.Role.UpdateOneID(roleID).
		SetName(input.Name).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		Save(ctx)
	if err != nil {
		switch {
		case ent.IsNotFound(err):
			return store.Role{}, store.ErrRoleNotFound
		case ent.IsConstraintError(err):
			return store.Role{}, store.ErrRoleNameConflict
		default:
			return store.Role{}, fmt.Errorf("update role %d: %w", input.ID, err)
		}
	}

	return toStoreRole(record), nil
}

// AssignPermissionsToRole 幂等把一组权限绑定到角色。
func (r *rbacRepository) AssignPermissionsToRole(ctx context.Context, input store.AssignPermissionsToRoleInput) error {
	roleID, err := toEntID(input.RoleID)
	if err != nil {
		return err
	}

	for _, permissionID := range input.PermissionIDs {
		entPermissionID, err := toEntID(permissionID)
		if err != nil {
			return err
		}

		exists, err := r.client.RolePermission.Query().
			Where(
				entrolepermission.RoleIDEQ(roleID),
				entrolepermission.PermissionIDEQ(entPermissionID),
			).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("check role permission assignment: %w", err)
		}
		if exists {
			continue
		}

		if _, err := r.client.RolePermission.Create().
			SetRoleID(roleID).
			SetPermissionID(entPermissionID).
			Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				continue
			}

			return fmt.Errorf("assign permission %d to role %d: %w", permissionID, input.RoleID, err)
		}
	}

	return nil
}

// ReplacePermissionsForRole 把角色权限覆盖为目标集合。
func (r *rbacRepository) ReplacePermissionsForRole(ctx context.Context, input store.ReplacePermissionsForRoleInput) error {
	roleID, err := toEntID(input.RoleID)
	if err != nil {
		return err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start replace role permissions tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	exists, err := tx.Role.Query().Where(entrole.IDEQ(roleID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check role %d before replacing permissions: %w", input.RoleID, err)
	}
	if !exists {
		return store.ErrRoleNotFound
	}

	permissionIDs, err := toUniqueEntIDs(input.PermissionIDs)
	if err != nil {
		return err
	}
	if len(permissionIDs) > 0 {
		count, countErr := tx.Permission.Query().
			Where(entpermission.IDIn(permissionIDs...)).
			Count(ctx)
		if countErr != nil {
			return fmt.Errorf("count permissions for role %d replacement: %w", input.RoleID, countErr)
		}
		if count != len(permissionIDs) {
			return store.ErrPermissionNotFound
		}
	}

	deleteQuery := tx.RolePermission.Delete().Where(entrolepermission.RoleIDEQ(roleID))
	if len(permissionIDs) > 0 {
		deleteQuery = deleteQuery.Where(entrolepermission.Not(entrolepermission.PermissionIDIn(permissionIDs...)))
	}
	if _, err := deleteQuery.Exec(ctx); err != nil {
		return fmt.Errorf("delete stale permissions for role %d: %w", input.RoleID, err)
	}

	for _, permissionID := range permissionIDs {
		exists, err := tx.RolePermission.Query().
			Where(
				entrolepermission.RoleIDEQ(roleID),
				entrolepermission.PermissionIDEQ(permissionID),
			).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("check role permission replacement: %w", err)
		}
		if exists {
			continue
		}

		if _, err := tx.RolePermission.Create().
			SetRoleID(roleID).
			SetPermissionID(permissionID).
			Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				continue
			}

			return fmt.Errorf("replace permission %d for role %d: %w", permissionID, input.RoleID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace role permissions for role %d: %w", input.RoleID, err)
	}
	tx = nil
	return nil
}

// AssignRoleToUser 幂等把目标角色绑定到用户。
func (r *rbacRepository) AssignRoleToUser(ctx context.Context, input store.AssignRoleToUserInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}
	roleID, err := toEntID(input.RoleID)
	if err != nil {
		return err
	}

	exists, err := r.client.UserRole.Query().
		Where(
			entuserrole.UserIDEQ(userID),
			entuserrole.RoleIDEQ(roleID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check user role assignment: %w", err)
	}
	if exists {
		return nil
	}

	if _, err := r.client.UserRole.Create().
		SetUserID(userID).
		SetRoleID(roleID).
		Save(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return nil
		}

		return fmt.Errorf("assign role %d to user %d: %w", input.RoleID, input.UserID, err)
	}

	return nil
}

// ReplaceRolesForUser 把用户角色覆盖为目标集合。
func (r *rbacRepository) ReplaceRolesForUser(ctx context.Context, input store.ReplaceRolesForUserInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start replace user roles tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	userExists, err := tx.User.Query().Where(entuser.IDEQ(userID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check user %d before replacing roles: %w", input.UserID, err)
	}
	if !userExists {
		return store.ErrUserNotFound
	}

	roleIDs, err := toUniqueEntIDs(input.RoleIDs)
	if err != nil {
		return err
	}
	if len(roleIDs) > 0 {
		count, countErr := tx.Role.Query().
			Where(entrole.IDIn(roleIDs...)).
			Count(ctx)
		if countErr != nil {
			return fmt.Errorf("count roles for user %d replacement: %w", input.UserID, countErr)
		}
		if count != len(roleIDs) {
			return store.ErrRoleNotFound
		}
	}

	deleteQuery := tx.UserRole.Delete().Where(entuserrole.UserIDEQ(userID))
	if len(roleIDs) > 0 {
		deleteQuery = deleteQuery.Where(entuserrole.Not(entuserrole.RoleIDIn(roleIDs...)))
	}
	if _, err := deleteQuery.Exec(ctx); err != nil {
		return fmt.Errorf("delete stale roles for user %d: %w", input.UserID, err)
	}

	for _, roleID := range roleIDs {
		exists, err := tx.UserRole.Query().
			Where(
				entuserrole.UserIDEQ(userID),
				entuserrole.RoleIDEQ(roleID),
			).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("check user role replacement: %w", err)
		}
		if exists {
			continue
		}

		if _, err := tx.UserRole.Create().
			SetUserID(userID).
			SetRoleID(roleID).
			Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				continue
			}

			return fmt.Errorf("replace role %d for user %d: %w", roleID, input.UserID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace user roles for user %d: %w", input.UserID, err)
	}
	tx = nil
	return nil
}

// GetRoleByID 按稳定 ID 返回单个角色记录。
func (r *rbacRepository) GetRoleByID(ctx context.Context, roleID uint64) (store.Role, error) {
	id, err := toEntID(roleID)
	if err != nil {
		return store.Role{}, err
	}

	record, err := r.client.Role.Query().
		Where(entrole.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return store.Role{}, store.ErrRoleNotFound
		}

		return store.Role{}, fmt.Errorf("get role by id %d: %w", roleID, err)
	}

	return toStoreRole(record), nil
}

// ListRolesByUserID 返回指定用户当前绑定的全部角色。
func (r *rbacRepository) ListRolesByUserID(ctx context.Context, userID uint64) ([]store.Role, error) {
	id, err := toEntID(userID)
	if err != nil {
		return nil, err
	}

	records, err := r.client.UserRole.Query().
		Where(entuserrole.UserIDEQ(id)).
		QueryRole().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles by user id: %w", err)
	}

	roles := make([]store.Role, 0, len(records))
	for _, record := range records {
		roles = append(roles, toStoreRole(record))
	}

	return roles, nil
}

// ListRoles 返回当前稳定排序下的全部角色快照。
func (r *rbacRepository) ListRoles(ctx context.Context) ([]store.Role, error) {
	records, err := r.client.Role.Query().
		Order(ent.Asc(entrole.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	roles := make([]store.Role, 0, len(records))
	for _, record := range records {
		roles = append(roles, toStoreRole(record))
	}

	return roles, nil
}

// ListPermissionsByUserID 返回指定用户经由角色解析得到的全部权限点。
func (r *rbacRepository) ListPermissionsByUserID(ctx context.Context, userID uint64) ([]store.Permission, error) {
	id, err := toEntID(userID)
	if err != nil {
		return nil, err
	}

	roleRecords, err := r.client.UserRole.Query().
		Where(entuserrole.UserIDEQ(id)).
		QueryRole().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user roles for permissions: %w", err)
	}
	if len(roleRecords) == 0 {
		return []store.Permission{}, nil
	}

	roleIDs := make([]int, 0, len(roleRecords))
	for _, roleRecord := range roleRecords {
		roleIDs = append(roleIDs, roleRecord.ID)
	}

	records, err := r.client.Permission.Query().
		Where(entpermission.HasRolePermissionsWith(entrolepermission.RoleIDIn(roleIDs...))).
		Unique(true).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions by user id: %w", err)
	}

	permissions := make([]store.Permission, 0, len(records))
	for _, record := range records {
		permissions = append(permissions, toStorePermission(record))
	}

	return permissions, nil
}

// ListPermissions 返回当前稳定排序下的全部权限快照。
func (r *rbacRepository) ListPermissions(ctx context.Context) ([]store.Permission, error) {
	records, err := r.client.Permission.Query().
		Order(ent.Asc(entpermission.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}

	permissions := make([]store.Permission, 0, len(records))
	for _, record := range records {
		permissions = append(permissions, toStorePermission(record))
	}

	return permissions, nil
}

func toUniqueEntIDs(ids []uint64) ([]int, error) {
	if len(ids) == 0 {
		return []int{}, nil
	}

	converted := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		entID, err := toEntID(id)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[entID]; ok {
			continue
		}

		seen[entID] = struct{}{}
		converted = append(converted, entID)
	}

	return converted, nil
}

func toStoreRole(record *ent.Role) store.Role {
	return store.Role{
		ID:          toStoreID(record.ID),
		Name:        record.Name,
		Display:     record.Display,
		Description: record.Description,
		Builtin:     record.Builtin,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func toStorePermission(record *ent.Permission) store.Permission {
	return store.Permission{
		ID:          toStoreID(record.ID),
		Code:        record.Code,
		Display:     record.Display,
		Description: record.Description,
		Category:    record.Category,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func ensureUniqueEntity[Entity any, Result any](
	lookup func() (*Entity, error),
	create func() (*Entity, error),
	toResult func(*Entity) Result,
	queryErrMsg string,
	createErrMsg string,
	conflictErrMsg string,
) (Result, error) {
	record, err := lookup()
	if err == nil {
		return toResult(record), nil
	}
	if !ent.IsNotFound(err) {
		var zero Result
		return zero, fmt.Errorf("%s: %w", queryErrMsg, err)
	}

	record, err = create()
	if err != nil {
		if ent.IsConstraintError(err) {
			record, lookupErr := lookup()
			if lookupErr != nil {
				var zero Result
				return zero, fmt.Errorf("%s: %w", conflictErrMsg, lookupErr)
			}
			return toResult(record), nil
		}

		var zero Result
		return zero, fmt.Errorf("%s: %w", createErrMsg, err)
	}

	return toResult(record), nil
}
