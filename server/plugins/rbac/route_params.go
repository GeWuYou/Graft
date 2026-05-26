package rbac

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	rbacstore "graft/server/plugins/rbac/store"
)

type replaceStableIDsHandlerConfig struct {
	invalidField         string
	readAndBindGenerated func(ginCtx *gin.Context, targetID uint64) ([]uint64, error)
	write                func(ctx context.Context, targetID uint64, ids []uint64) error
}

type batchStableIDsHandlerConfig struct {
	invalidField         string
	readAndBindGenerated func(ginCtx *gin.Context) (batchStableIDSet, error)
	write                func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error
}

type batchStableIDSet struct {
	userIDs []uint64
	roleIDs []uint64
}

func normalizeCreateRoleInput(request rbacopenapi.PostRolesJSONRequestBody) (rbacstore.CreateRoleInput, bool) {
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

func normalizeUpdateRoleInput(roleID uint64, request rbacopenapi.PostRoleUpdateJSONRequestBody) (rbacstore.UpdateRoleInput, bool) {
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

func readGeneratedRolePermissionReplaceRequest(ginCtx *gin.Context) (rbacopenapi.PostRolePermissionsReplaceJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostRolePermissionsReplaceJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostRolePermissionsReplaceJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.PermissionIds), nil
}

func readGeneratedRolePermissionAddRequest(ginCtx *gin.Context) (rbacopenapi.PostRolePermissionsAddJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostRolePermissionsAddJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostRolePermissionsAddJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.PermissionIds), nil
}

func readGeneratedRolePermissionRemoveRequest(ginCtx *gin.Context) (rbacopenapi.PostRolePermissionsRemoveJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostRolePermissionsRemoveJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostRolePermissionsRemoveJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.PermissionIds), nil
}

func readGeneratedUserRoleReplaceRequest(ginCtx *gin.Context) (rbacopenapi.PostUserRolesReplaceJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostUserRolesReplaceJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostUserRolesReplaceJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.RoleIds), nil
}

func readGeneratedUserRoleAddRequest(ginCtx *gin.Context) (rbacopenapi.PostUserRolesAddJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostUserRolesAddJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostUserRolesAddJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.RoleIds), nil
}

func readGeneratedUserRoleRemoveRequest(ginCtx *gin.Context) (rbacopenapi.PostUserRolesRemoveJSONRequestBody, []uint64, error) {
	var request rbacopenapi.PostUserRolesRemoveJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostUserRolesRemoveJSONRequestBody{}, nil, err
	}
	return request, optionalStableIDs(request.RoleIds), nil
}

func readGeneratedBatchUserRoleReplaceRequest(ginCtx *gin.Context) (rbacopenapi.PostUsersRolesReplaceJSONRequestBody, batchStableIDSet, error) {
	var request rbacopenapi.PostUsersRolesReplaceJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostUsersRolesReplaceJSONRequestBody{}, batchStableIDSet{}, err
	}
	return request, batchStableIDSet{
		userIDs: optionalStableIDs(request.UserIds),
		roleIDs: optionalStableIDs(request.RoleIds),
	}, nil
}

func readGeneratedBatchUserRoleAddRequest(ginCtx *gin.Context) (rbacopenapi.PostUsersRolesAddJSONRequestBody, batchStableIDSet, error) {
	var request rbacopenapi.PostUsersRolesAddJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostUsersRolesAddJSONRequestBody{}, batchStableIDSet{}, err
	}
	return request, batchStableIDSet{
		userIDs: optionalStableIDs(request.UserIds),
		roleIDs: optionalStableIDs(request.RoleIds),
	}, nil
}

func readGeneratedBatchUserRoleRemoveRequest(ginCtx *gin.Context) (rbacopenapi.PostUsersRolesRemoveJSONRequestBody, batchStableIDSet, error) {
	var request rbacopenapi.PostUsersRolesRemoveJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		return rbacopenapi.PostUsersRolesRemoveJSONRequestBody{}, batchStableIDSet{}, err
	}
	return request, batchStableIDSet{
		userIDs: optionalStableIDs(request.UserIds),
		roleIDs: optionalStableIDs(request.RoleIds),
	}, nil
}

func normalizeRoleStatusInput(request rbacopenapi.PostRoleStatusJSONRequestBody) (string, bool) {
	switch request.Status {
	case rbacopenapi.PostRoleStatusJSONBodyStatusEnabled:
		return rbacstore.RoleStatusEnabled, true
	case rbacopenapi.PostRoleStatusJSONBodyStatusDisabled:
		return rbacstore.RoleStatusDisabled, true
	default:
		return "", false
	}
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
