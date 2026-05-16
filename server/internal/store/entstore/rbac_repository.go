package entstore

import (
	"context"
	"fmt"

	"graft/server/internal/ent"
	entpermission "graft/server/internal/ent/permission"
	entrole "graft/server/internal/ent/role"
	entrolepermission "graft/server/internal/ent/rolepermission"
	entuserrole "graft/server/internal/ent/userrole"
	"graft/server/internal/store"
)

type rbacRepository struct {
	client *ent.Client
}

// EnsureRole 幂等确保目标角色存在。
func (r *rbacRepository) EnsureRole(ctx context.Context, input store.EnsureRoleInput) (store.Role, error) {
	return ensureUniqueEntity(
		func() (*ent.Role, error) {
			return r.client.Role.Query().
				Where(entrole.NameEQ(input.Name)).
				Only(ctx)
		},
		func() (*ent.Role, error) {
			return r.client.Role.Create().
				SetName(input.Name).
				SetDisplay(input.Display).
				SetNillableDescription(input.Description).
				SetBuiltin(input.Builtin).
				Save(ctx)
		},
		toStoreRole,
		"query ensured role by name",
		"create ensured role",
		"re-query ensured role after conflict",
	)
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
