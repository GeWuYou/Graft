package rbac

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	rbacstore "graft/server/plugins/rbac/store"
)

type replaceStableIDsHandlerConfig struct {
	invalidField string
	readIDs      func(ginCtx *gin.Context) ([]uint64, error)
	write        func(ctx context.Context, targetID uint64, ids []uint64) error
}

func normalizeCreateRoleInput(request createRoleRequest) (rbacstore.CreateRoleInput, bool) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return rbacstore.CreateRoleInput{}, false
	}

	return rbacstore.CreateRoleInput{
		Name:        name,
		Display:     strings.TrimSpace(request.Display),
		Description: normalizeOptionalString(request.Description),
		Builtin:     false,
	}, true
}

func normalizeUpdateRoleInput(roleID uint64, request updateRoleRequest) (rbacstore.UpdateRoleInput, bool) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return rbacstore.UpdateRoleInput{}, false
	}

	return rbacstore.UpdateRoleInput{
		ID:          roleID,
		Name:        name,
		Display:     strings.TrimSpace(request.Display),
		Description: normalizeOptionalString(request.Description),
	}, true
}

func readRolePermissionIDs(ginCtx *gin.Context) ([]uint64, error) {
	var request replaceRolePermissionsRequest
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	return optionalStableIDs(request.PermissionIDs), nil
}

func readUserRoleIDs(ginCtx *gin.Context) ([]uint64, error) {
	var request replaceUserRolesRequest
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	return optionalStableIDs(request.RoleIDs), nil
}

func optionalStableIDs(ids *[]uint64) []uint64 {
	if ids == nil {
		return nil
	}
	return *ids
}

func normalizeOptionalString(input *string) *string {
	if input == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func hasInvalidStableIDs(ids []uint64) bool {
	for _, id := range ids {
		if id == 0 {
			return true
		}
	}

	return false
}

func parseManagementID(input string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(input), 10, 64)
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("id must be greater than zero")
	}

	return id, nil
}
