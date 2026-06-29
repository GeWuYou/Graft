package rbac

import (
	"github.com/gin-gonic/gin"

	rbacopenapi "graft/server/internal/contract/openapi/rbac"
)

type rbacWriteGeneratedHandler struct{}

func (h rbacWriteGeneratedHandler) PostRoles(
	params rbacopenapi.PostRolesParams,
	body rbacopenapi.PostRolesJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRoleUpdate(
	id uint64,
	params rbacopenapi.PostRoleUpdateParams,
	body rbacopenapi.PostRoleUpdateJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRoleDelete(
	id uint64,
	params rbacopenapi.PostRoleDeleteParams,
) {
	_ = h
	_ = id
	_ = params
}

func (h rbacWriteGeneratedHandler) PostRolePermissionsAdd(
	id uint64,
	params rbacopenapi.PostRolePermissionsAddParams,
	body rbacopenapi.PostRolePermissionsAddJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRolePermissionsRemove(
	id uint64,
	params rbacopenapi.PostRolePermissionsRemoveParams,
	body rbacopenapi.PostRolePermissionsRemoveJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRolePermissionsReplace(
	id uint64,
	params rbacopenapi.PostRolePermissionsReplaceParams,
	body rbacopenapi.PostRolePermissionsReplaceJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRoleStatus(
	id uint64,
	params rbacopenapi.PostRoleStatusParams,
	body rbacopenapi.PostRoleStatusJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUserRolesAdd(
	id uint64,
	params rbacopenapi.PostUserRolesAddParams,
	body rbacopenapi.PostUserRolesAddJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUserRolesRemove(
	id uint64,
	params rbacopenapi.PostUserRolesRemoveParams,
	body rbacopenapi.PostUserRolesRemoveJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUserRolesReplace(
	id uint64,
	params rbacopenapi.PostUserRolesReplaceParams,
	body rbacopenapi.PostUserRolesReplaceJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUsersRolesAdd(
	params rbacopenapi.PostUsersRolesAddParams,
	body rbacopenapi.PostUsersRolesAddJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUsersRolesRemove(
	params rbacopenapi.PostUsersRolesRemoveParams,
	body rbacopenapi.PostUsersRolesRemoveJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUsersRolesReplace(
	params rbacopenapi.PostUsersRolesReplaceParams,
	body rbacopenapi.PostUsersRolesReplaceJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func bindGeneratedRoleCreateParams(ginCtx *gin.Context) rbacopenapi.PostRolesParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolesParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRoleUpdateParams(ginCtx *gin.Context) rbacopenapi.PostRoleUpdateParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleUpdateParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRoleDeleteParams(ginCtx *gin.Context) rbacopenapi.PostRoleDeleteParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleDeleteParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRolePermissionReplaceParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRolePermissionAddParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsAddParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRolePermissionRemoveParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRoleStatusParams(ginCtx *gin.Context) rbacopenapi.PostRoleStatusParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleStatusParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUserRoleReplaceParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUserRoleAddParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesAddParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUserRoleRemoveParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUsersRoleReplaceParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUsersRoleAddParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesAddParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUsersRoleRemoveParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}
