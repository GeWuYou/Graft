package rbac

import (
	"time"

	rbacstore "graft/server/plugins/rbac/store"
)

func toRoleListResponse(roles []rbacstore.Role) roleListResponse {
	items := make([]roleListItem, 0, len(roles))
	for _, role := range roles {
		items = append(items, roleListItem{
			ID:              role.ID,
			Name:            role.Name,
			Display:         role.Display,
			Description:     role.Description,
			Builtin:         role.Builtin,
			UpdatedAt:       role.UpdatedAt.UTC().Format(time.RFC3339),
			PermissionCount: role.PermissionCount,
			UserCount:       role.UserCount,
		})
	}

	return roleListResponse{Items: items}
}

func toRoleListItem(role rbacstore.Role) roleListItem {
	return roleListItem{
		ID:          role.ID,
		Name:        role.Name,
		Display:     role.Display,
		Description: role.Description,
		Builtin:     role.Builtin,
	}
}

func toRolePermissionBindingResponse(bindings []rbacstore.RolePermissionBinding) rolePermissionBindingResponse {
	permissionIDs := make([]uint64, 0, len(bindings))
	for _, item := range bindings {
		permissionIDs = append(permissionIDs, item.PermissionID)
	}

	return rolePermissionBindingResponse{PermissionIDs: permissionIDs}
}

func toUserRoleBindingResponse(roleIDs []uint64) userRoleBindingResponse {
	return userRoleBindingResponse{RoleIDs: roleIDs}
}

func toPermissionListResponse(permissions []rbacstore.Permission) permissionListResponse {
	items := make([]permissionListItem, 0, len(permissions))
	for _, item := range permissions {
		items = append(items, permissionListItem{
			ID:               item.ID,
			Code:             item.Code,
			Display:          item.Display,
			Description:      item.Description,
			Category:         item.Category,
			CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:        item.UpdatedAt.UTC().Format(time.RFC3339),
			RoleBindingCount: item.RoleBindingCount,
		})
	}

	return permissionListResponse{Items: items}
}
