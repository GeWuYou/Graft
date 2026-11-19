package rbac

import (
	"context"
	"github.com/gin-gonic/gin"
	messagecontract "graft/server/internal/contract/message"
	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	"graft/server/internal/contract/openapi/strictsupport"
	"graft/server/internal/module"
)

// rbacStrictServer 将现有 HTTP 响应写入接入生成的请求契约。
type rbacStrictServer struct {
	ctx         *module.Context
	moduleName  string
	reader      readManagementService
	writer      writeManagementService
	roles       rbacSavedViewHandlers
	permissions rbacSavedViewHandlers
}

var _ rbacopenapi.StrictServerInterface = rbacStrictServer{}

func (s rbacStrictServer) GetRoles(ctx context.Context, _ rbacopenapi.GetRolesRequestObject) (rbacopenapi.GetRolesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListRoles)
}

func (s rbacStrictServer) GetRole(ctx context.Context, _ rbacopenapi.GetRoleRequestObject) (rbacopenapi.GetRoleResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleGetRole)
}

func (s rbacStrictServer) GetRolePermissions(ctx context.Context, _ rbacopenapi.GetRolePermissionsRequestObject) (rbacopenapi.GetRolePermissionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListRolePermissionBindings)
}

func (s rbacStrictServer) PostRoles(ctx context.Context, _ rbacopenapi.PostRolesRequestObject) (rbacopenapi.PostRolesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleCreateRoleRoute)
}

func (s rbacStrictServer) PostRoleUpdate(ctx context.Context, _ rbacopenapi.PostRoleUpdateRequestObject) (rbacopenapi.PostRoleUpdateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleUpdateRoleRoute)
}

func (s rbacStrictServer) PostRoleClone(ctx context.Context, _ rbacopenapi.PostRoleCloneRequestObject) (rbacopenapi.PostRoleCloneResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleCloneRoleRoute)
}

func (s rbacStrictServer) PostRoleStatus(ctx context.Context, _ rbacopenapi.PostRoleStatusRequestObject) (rbacopenapi.PostRoleStatusResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleUpdateRoleStatusRoute)
}

func (s rbacStrictServer) PostRoleDelete(ctx context.Context, _ rbacopenapi.PostRoleDeleteRequestObject) (rbacopenapi.PostRoleDeleteResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleDeleteRoleRoute)
}

func (s rbacStrictServer) PostRolePermissionsReplace(ctx context.Context, _ rbacopenapi.PostRolePermissionsReplaceRequestObject) (rbacopenapi.PostRolePermissionsReplaceResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleReplaceRolePermissionsRoute)
}

func (s rbacStrictServer) PostRolePermissionsAdd(ctx context.Context, _ rbacopenapi.PostRolePermissionsAddRequestObject) (rbacopenapi.PostRolePermissionsAddResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleAddRolePermissionsRoute)
}

func (s rbacStrictServer) DeleteRolePermissions(ctx context.Context, _ rbacopenapi.DeleteRolePermissionsRequestObject) (rbacopenapi.DeleteRolePermissionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleRemoveRolePermissionsRoute)
}

func (s rbacStrictServer) GetPermissions(ctx context.Context, _ rbacopenapi.GetPermissionsRequestObject) (rbacopenapi.GetPermissionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListPermissions)
}

func (s rbacStrictServer) GetPermission(ctx context.Context, _ rbacopenapi.GetPermissionRequestObject) (rbacopenapi.GetPermissionResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleGetPermission)
}

func (s rbacStrictServer) GetUserRoles(ctx context.Context, _ rbacopenapi.GetUserRolesRequestObject) (rbacopenapi.GetUserRolesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListUserRoleBindings)
}

func (s rbacStrictServer) PostUserRolesReplace(ctx context.Context, _ rbacopenapi.PostUserRolesReplaceRequestObject) (rbacopenapi.PostUserRolesReplaceResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleReplaceUserRolesRoute)
}

func (s rbacStrictServer) PostUserRolesAdd(ctx context.Context, _ rbacopenapi.PostUserRolesAddRequestObject) (rbacopenapi.PostUserRolesAddResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleAddUserRolesRoute)
}

func (s rbacStrictServer) DeleteUserRoles(ctx context.Context, _ rbacopenapi.DeleteUserRolesRequestObject) (rbacopenapi.DeleteUserRolesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleRemoveUserRolesRoute)
}

func (s rbacStrictServer) PostUsersRolesReplace(ctx context.Context, _ rbacopenapi.PostUsersRolesReplaceRequestObject) (rbacopenapi.PostUsersRolesReplaceResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleBatchReplaceUserRolesRoute)
}

func (s rbacStrictServer) PostUsersRolesAdd(ctx context.Context, _ rbacopenapi.PostUsersRolesAddRequestObject) (rbacopenapi.PostUsersRolesAddResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleBatchAddUserRolesRoute)
}

func (s rbacStrictServer) DeleteUsersRoles(ctx context.Context, _ rbacopenapi.DeleteUsersRolesRequestObject) (rbacopenapi.DeleteUsersRolesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleBatchRemoveUserRolesRoute)
}

func (s rbacStrictServer) GetRoleSavedViews(ctx context.Context, _ rbacopenapi.GetRoleSavedViewsRequestObject) (rbacopenapi.GetRoleSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.roles.list)
}

func (s rbacStrictServer) PostRoleSavedView(ctx context.Context, _ rbacopenapi.PostRoleSavedViewRequestObject) (rbacopenapi.PostRoleSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.roles.create)
}

func (s rbacStrictServer) PutRoleSavedView(ctx context.Context, _ rbacopenapi.PutRoleSavedViewRequestObject) (rbacopenapi.PutRoleSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.roles.update)
}

func (s rbacStrictServer) DeleteRoleSavedView(ctx context.Context, _ rbacopenapi.DeleteRoleSavedViewRequestObject) (rbacopenapi.DeleteRoleSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.roles.remove)
}

func (s rbacStrictServer) GetPermissionSavedViews(ctx context.Context, _ rbacopenapi.GetPermissionSavedViewsRequestObject) (rbacopenapi.GetPermissionSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.permissions.list)
}

func (s rbacStrictServer) PostPermissionSavedView(ctx context.Context, _ rbacopenapi.PostPermissionSavedViewRequestObject) (rbacopenapi.PostPermissionSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.permissions.create)
}

func (s rbacStrictServer) PutPermissionSavedView(ctx context.Context, _ rbacopenapi.PutPermissionSavedViewRequestObject) (rbacopenapi.PutPermissionSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.permissions.update)
}

func (s rbacStrictServer) DeletePermissionSavedView(ctx context.Context, _ rbacopenapi.DeletePermissionSavedViewRequestObject) (rbacopenapi.DeletePermissionSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.permissions.remove)
}

func (s rbacStrictServer) handleListRoles(c *gin.Context) {
	handleListRoles(s.ctx, s.moduleName, s.reader)(c)
}

func (s rbacStrictServer) handleGetRole(c *gin.Context) {
	handleGetRole(s.ctx, s.moduleName, s.reader)(c)
}

func (s rbacStrictServer) handleListRolePermissionBindings(c *gin.Context) {
	handleListRolePermissionBindings(s.ctx, s.moduleName, s.reader)(c)
}

func (s rbacStrictServer) handleCreateRoleRoute(c *gin.Context) {
	handleCreateRoleRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleUpdateRoleRoute(c *gin.Context) {
	handleUpdateRoleRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleCloneRoleRoute(c *gin.Context) {
	handleCloneRoleRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleUpdateRoleStatusRoute(c *gin.Context) {
	handleUpdateRoleStatusRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleDeleteRoleRoute(c *gin.Context) {
	handleDeleteRoleRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleReplaceRolePermissionsRoute(c *gin.Context) {
	handleReplaceRolePermissionsRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleAddRolePermissionsRoute(c *gin.Context) {
	handleAddRolePermissionsRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleRemoveRolePermissionsRoute(c *gin.Context) {
	handleRemoveRolePermissionsRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleListPermissions(c *gin.Context) {
	handleListPermissions(s.ctx, s.moduleName, s.reader)(c)
}

func (s rbacStrictServer) handleGetPermission(c *gin.Context) {
	handleGetPermission(s.ctx, s.moduleName, s.reader)(c)
}

func (s rbacStrictServer) handleListUserRoleBindings(c *gin.Context) {
	handleListUserRoleBindings(s.ctx, s.moduleName, s.reader)(c)
}

func (s rbacStrictServer) handleReplaceUserRolesRoute(c *gin.Context) {
	handleReplaceUserRolesRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleAddUserRolesRoute(c *gin.Context) {
	handleAddUserRolesRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleRemoveUserRolesRoute(c *gin.Context) {
	handleRemoveUserRolesRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleBatchReplaceUserRolesRoute(c *gin.Context) {
	handleBatchReplaceUserRolesRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleBatchAddUserRolesRoute(c *gin.Context) {
	handleBatchAddUserRolesRoute(c, s.ctx, s.moduleName, s.writer)
}

func (s rbacStrictServer) handleBatchRemoveUserRolesRoute(c *gin.Context) {
	handleBatchRemoveUserRolesRoute(c, s.ctx, s.moduleName, s.writer)
}

func newRBACStrictWrapper(ctx *module.Context, server rbacStrictServer) rbacopenapi.ServerInterfaceWrapper {
	return rbacopenapi.ServerInterfaceWrapper{
		Handler: rbacopenapi.NewStrictHandlerWithOptions(server, nil, rbacopenapi.StrictGinServerOptions{
			RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String(), nil),
			HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
		}),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String()),
	}
}
