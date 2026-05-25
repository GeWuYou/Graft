package rbac

import (
	"math"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	rbacstore "graft/server/plugins/rbac/store"
)

func toRoleListResponse(roles []rbacstore.Role) generated.RoleListResponse {
	items := make([]generated.RoleListItem, 0, len(roles))
	for _, role := range roles {
		items = append(items, toRoleListItem(role))
	}

	return generated.RoleListResponse{Items: items}
}

func toRoleListItem(role rbacstore.Role) generated.RoleListItem {
	return generated.RoleListItem{
		Id:              mustConvertGeneratedID(role.ID, "rbac role id"),
		Name:            role.Name,
		Display:         role.Display,
		Description:     role.Description,
		Builtin:         role.Builtin,
		UpdatedAt:       role.UpdatedAt.UTC().Format(time.RFC3339),
		PermissionCount: role.PermissionCount,
		UserCount:       role.UserCount,
	}
}

func toRolePermissionBindingResponse(bindings []rbacstore.RolePermissionBinding) generated.RolePermissionBindingResponse {
	permissionIDs := make([]int64, 0, len(bindings))
	for _, item := range bindings {
		permissionIDs = append(permissionIDs, mustConvertGeneratedID(item.PermissionID, "rbac permission id"))
	}

	return generated.RolePermissionBindingResponse{PermissionIds: permissionIDs}
}

func toUserRoleBindingResponse(roleIDs []uint64) generated.UserRoleBindingResponse {
	converted := make([]int64, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		converted = append(converted, mustConvertGeneratedID(roleID, "rbac role id"))
	}

	return generated.UserRoleBindingResponse{RoleIds: converted}
}

func toPermissionListResponse(permissions []rbacstore.Permission) generated.PermissionListResponse {
	items := make([]generated.PermissionListItem, 0, len(permissions))
	for _, item := range permissions {
		items = append(items, toPermissionListItem(item))
	}

	return generated.PermissionListResponse{Items: items}
}

func toPermissionListItem(item rbacstore.Permission) generated.PermissionListItem {
	return generated.PermissionListItem{
		Id:               mustConvertGeneratedID(item.ID, "rbac permission id"),
		Code:             item.Code,
		Display:          item.Display,
		Description:      item.Description,
		Category:         item.Category,
		CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.UTC().Format(time.RFC3339),
		RoleBindingCount: item.RoleBindingCount,
	}
}

func mustConvertGeneratedID(id uint64, label string) int64 {
	if id > math.MaxInt64 {
		panic(label + " exceeds int64")
	}
	return int64(id)
}
