package rbac

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	openapicontract "graft/server/internal/contract/openapi"
	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	rbacstore "graft/server/plugins/rbac/store"
)

type replaceStableIDsHandlerConfig struct {
	invalidField string
	readIDs      func(ginCtx *gin.Context) ([]uint64, error)
	write        func(ctx context.Context, targetID uint64, ids []uint64) error
}

func normalizeCreateRoleInput(request openapicontract.PostRolesJSONRequestBody) (rbacstore.CreateRoleInput, bool) {
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

func normalizeUpdateRoleInput(roleID uint64, request openapicontract.PostRoleUpdateJSONRequestBody) (rbacstore.UpdateRoleInput, bool) {
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
	var request openapicontract.PostRolePermissionAssignJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return nil, err
	}
	return optionalStableIDs(request.PermissionIds), nil
}

func readGeneratedRolePermissionAssignRequest(ginCtx *gin.Context) (rbacopenapi.PostRolePermissionAssignJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostRolePermissionAssignJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostRolePermissionAssignJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.PermissionIds), nil
}

func readGeneratedUserRoleAssignRequest(ginCtx *gin.Context) (rbacopenapi.PostUserRolesAssignJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostUserRolesAssignJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostUserRolesAssignJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.RoleIds), nil
}

func optionalStableIDs(ids []int64) []uint64 {
	if ids == nil {
		return nil
	}
	stableIDs := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id < 0 {
			return append(stableIDs, 0)
		}
		stableIDs = append(stableIDs, uint64(id))
	}
	return stableIDs
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
