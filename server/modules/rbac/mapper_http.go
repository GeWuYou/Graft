package rbac

import (
	"fmt"
	"math"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	rbacstore "graft/server/modules/rbac/store"
)

func toRoleListResponse(result rbacstore.RoleListResult, limit, offset int) (generated.RoleListResponse, error) {
	items := make([]generated.RoleListItem, 0, len(result.Items))
	for _, role := range result.Items {
		item, err := toRoleListItem(role)
		if err != nil {
			return generated.RoleListResponse{}, err
		}
		items = append(items, item)
	}

	return generated.RoleListResponse{Items: items, Total: int64(result.Total), Limit: limit, Offset: offset}, nil
}

func toRoleListItem(role rbacstore.Role) (generated.RoleListItem, error) {
	id, err := mustConvertGeneratedID(role.ID, "rbac role id")
	if err != nil {
		return generated.RoleListItem{}, err
	}

	return generated.RoleListItem{
		Id:              id,
		Name:            role.Name,
		Display:         role.Display,
		Description:     role.Description,
		Builtin:         role.Builtin,
		System:          role.Type == "system",
		Type:            generated.RoleListItemType(role.Type),
		BuiltinKey:      role.BuiltinKey,
		Editable:        role.Editable,
		Status:          generated.RoleListItemStatus(role.Status),
		UpdatedAt:       role.UpdatedAt.UTC().Format(time.RFC3339),
		PermissionCount: role.PermissionCount,
		UserCount:       role.UserCount,
	}, nil
}

func toRoleDetailResponse(role rbacstore.Role) (generated.RoleDetailResponse, error) {
	id, err := mustConvertGeneratedID(role.ID, "rbac role id")
	if err != nil {
		return generated.RoleDetailResponse{}, err
	}

	return generated.RoleDetailResponse{
		Id:              id,
		Name:            role.Name,
		Display:         role.Display,
		Description:     role.Description,
		Builtin:         role.Builtin,
		Status:          generated.RoleDetailResponseStatus(role.Status),
		CreatedAt:       role.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       role.UpdatedAt.UTC().Format(time.RFC3339),
		PermissionCount: role.PermissionCount,
		UserCount:       role.UserCount,
	}, nil
}

func toRolePermissionBindingResponse(bindings []rbacstore.RolePermissionBinding) (generated.RolePermissionBindingResponse, error) {
	permissionIDs := make([]int64, 0, len(bindings))
	items := make([]generated.RolePermissionBindingItem, 0, len(bindings))
	for _, item := range bindings {
		permissionID, err := mustConvertGeneratedID(item.PermissionID, "rbac permission id")
		if err != nil {
			return generated.RolePermissionBindingResponse{}, err
		}
		permissionIDs = append(permissionIDs, permissionID)
		items = append(items, generated.RolePermissionBindingItem{
			PermissionId: permissionID,
			Scope:        generated.RolePermissionBindingItemScope(item.Scope),
		})
	}

	return generated.RolePermissionBindingResponse{PermissionIds: permissionIDs, Bindings: items}, nil
}

func toUserRoleBindingResponse(roleIDs []uint64) (generated.UserRoleBindingResponse, error) {
	converted := make([]int64, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		convertedRoleID, err := mustConvertGeneratedID(roleID, "rbac role id")
		if err != nil {
			return generated.UserRoleBindingResponse{}, err
		}
		converted = append(converted, convertedRoleID)
	}

	return generated.UserRoleBindingResponse{RoleIds: converted}, nil
}

func toPermissionListResponse(result rbacstore.PermissionListResult, limit, offset int) (generated.PermissionListResponse, error) {
	items := make([]generated.PermissionListItem, 0, len(result.Items))
	for _, item := range result.Items {
		converted, err := toPermissionListItem(item)
		if err != nil {
			return generated.PermissionListResponse{}, err
		}
		items = append(items, converted)
	}

	return generated.PermissionListResponse{Items: items, Total: int64(result.Total), Limit: limit, Offset: offset}, nil
}

func toPermissionListItem(item rbacstore.Permission) (generated.PermissionListItem, error) {
	id, err := mustConvertGeneratedID(item.ID, "rbac permission id")
	if err != nil {
		return generated.PermissionListItem{}, err
	}

	return generated.PermissionListItem{
		Id:               id,
		Code:             item.Code,
		Display:          item.Display,
		DisplayKey:       item.DisplayKey,
		Description:      item.Description,
		DescriptionKey:   item.DescriptionKey,
		Module:           item.Module,
		Resource:         item.Resource,
		Action:           item.Action,
		RiskLevel:        generated.PermissionListItemRiskLevel(item.RiskLevel),
		CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.UTC().Format(time.RFC3339),
		RoleBindingCount: item.RoleBindingCount,
	}, nil
}

func mustConvertGeneratedID(id uint64, label string) (int64, error) {
	if id > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds int64: %d", label, id)
	}
	return int64(id), nil
}
