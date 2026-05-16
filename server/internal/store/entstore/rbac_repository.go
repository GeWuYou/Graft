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
	record, err := r.client.Role.Query().
		Where(entrole.NameEQ(input.Name)).
		Only(ctx)
	if err == nil {
		return toStoreRole(record), nil
	}
	if !ent.IsNotFound(err) {
		return store.Role{}, fmt.Errorf("query ensured role by name: %w", err)
	}

	record, err = r.client.Role.Create().
		SetName(input.Name).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			record, lookupErr := r.client.Role.Query().
				Where(entrole.NameEQ(input.Name)).
				Only(ctx)
			if lookupErr != nil {
				return store.Role{}, fmt.Errorf("re-query ensured role after conflict: %w", lookupErr)
			}
			return toStoreRole(record), nil
		}

		return store.Role{}, fmt.Errorf("create ensured role: %w", err)
	}

	return toStoreRole(record), nil
}

// EnsurePermission 幂等确保目标权限存在。
func (r *rbacRepository) EnsurePermission(ctx context.Context, input store.EnsurePermissionInput) (store.Permission, error) {
	record, err := r.client.Permission.Query().
		Where(entpermission.CodeEQ(input.Code)).
		Only(ctx)
	if err == nil {
		return toStorePermission(record), nil
	}
	if !ent.IsNotFound(err) {
		return store.Permission{}, fmt.Errorf("query ensured permission by code: %w", err)
	}

	record, err = r.client.Permission.Create().
		SetCode(input.Code).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			record, lookupErr := r.client.Permission.Query().
				Where(entpermission.CodeEQ(input.Code)).
				Only(ctx)
			if lookupErr != nil {
				return store.Permission{}, fmt.Errorf("re-query ensured permission after conflict: %w", lookupErr)
			}
			return toStorePermission(record), nil
		}

		return store.Permission{}, fmt.Errorf("create ensured permission: %w", err)
	}

	return toStorePermission(record), nil
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

func toStoreRole(record *ent.Role) store.Role {
	return store.Role{
		ID:          uint64(record.ID),
		Name:        record.Name,
		Display:     record.Display,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func toStorePermission(record *ent.Permission) store.Permission {
	return store.Permission{
		ID:          uint64(record.ID),
		Code:        record.Code,
		Display:     record.Display,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}
