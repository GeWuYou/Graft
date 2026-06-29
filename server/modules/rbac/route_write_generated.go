package rbac

import (
	"github.com/gin-gonic/gin"

	rbacopenapi "graft/server/internal/contract/openapi/rbac"
)

type rbacWriteGeneratedHandler struct{}

// rbacWriteGeneratedHandler 保留 OpenAPI 生成的写入接口绑定点。
// 这些方法只负责让手写路由继续显式记录生成请求参数；真正的 RBAC 写入生命周期由 route_write_stable_ids.go 中的手写 handler 拥有。

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

// bindGeneratedRoleCreateParams 组装创建角色接口的通用请求参数。
// 它从请求头中读取语言和请求 ID，并填充到对应的 OpenAPI 参数结构体中。
func bindGeneratedRoleCreateParams(ginCtx *gin.Context) rbacopenapi.PostRolesParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolesParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedRoleUpdateParams 构造角色更新接口的参数。
// @returns 填充了 `XGraftLocale` 和 `XRequestId` 的 `PostRoleUpdateParams`。
func bindGeneratedRoleUpdateParams(ginCtx *gin.Context) rbacopenapi.PostRoleUpdateParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleUpdateParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedRoleDeleteParams 读取通用请求头并构造删除角色请求参数。
// 返回填充了 `XGraftLocale` 和 `XRequestId` 的 `PostRoleDeleteParams`。
func bindGeneratedRoleDeleteParams(ginCtx *gin.Context) rbacopenapi.PostRoleDeleteParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleDeleteParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedRolePermissionReplaceParams 从请求头构造角色权限替换接口的参数结构体。
// 它会填充 `XGraftLocale` 和 `XRequestId`。
func bindGeneratedRolePermissionReplaceParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedRolePermissionAddParams 构造角色权限新增接口的请求参数。
// 它会填充从请求头读取的 `XGraftLocale` 和 `XRequestId`。
func bindGeneratedRolePermissionAddParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsAddParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedRolePermissionRemoveParams 从请求头构造角色权限移除接口的参数。
// 它填充 `XGraftLocale` 和 `XRequestId`。
func bindGeneratedRolePermissionRemoveParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedRoleStatusParams 从请求头构造角色状态接口的参数。
// 它填充 `XGraftLocale` 和 `XRequestId`。
func bindGeneratedRoleStatusParams(ginCtx *gin.Context) rbacopenapi.PostRoleStatusParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleStatusParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedUserRoleReplaceParams 根据请求头构造用户角色替换接口的参数。
// 它填充 `XGraftLocale` 和 `XRequestId` 字段。
func bindGeneratedUserRoleReplaceParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedUserRoleAddParams 生成用户角色新增接口所需的参数结构体。
// @param ginCtx Gin 上下文。
// @returns 填充了 `XGraftLocale` 和 `XRequestId` 的 `PostUserRolesAddParams`。
func bindGeneratedUserRoleAddParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesAddParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedUserRoleRemoveParams 构造移除用户角色请求的参数。
// 它会填充通用请求头中的 `XGraftLocale` 和 `XRequestId`。
func bindGeneratedUserRoleRemoveParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedUsersRoleReplaceParams 构造批量用户角色替换接口的请求参数。
// 返回值包含从请求头读取的 `XGraftLocale` 和 `XRequestId`。
func bindGeneratedUsersRoleReplaceParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedUsersRoleAddParams 构造批量用户角色新增请求的参数结构体。
// 它会从请求头读取通用的语言和请求标识，并填充到 OpenAPI 参数中。
func bindGeneratedUsersRoleAddParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesAddParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGeneratedUsersRoleRemoveParams 生成批量移除用户角色所需的请求参数。
// 它会从请求头中读取通用信息并填充到 OpenAPI 参数结构体中。
func bindGeneratedUsersRoleRemoveParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}
