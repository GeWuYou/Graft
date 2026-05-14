package entstore

import (
	"context"
	"fmt"

	"graft/server/internal/ent"
	entpermission "graft/server/internal/ent/permission"
	entrolepermission "graft/server/internal/ent/rolepermission"
	entuserrole "graft/server/internal/ent/userrole"
	"graft/server/internal/store"
)

type rbacRepository struct {
	client *ent.Client
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
		roles = append(roles, store.Role{
			ID:          uint64(record.ID),
			Name:        record.Name,
			Display:     record.Display,
			Description: record.Description,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
		})
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
		permissions = append(permissions, store.Permission{
			ID:          uint64(record.ID),
			Code:        record.Code,
			Display:     record.Display,
			Description: record.Description,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
		})
	}

	return permissions, nil
}
