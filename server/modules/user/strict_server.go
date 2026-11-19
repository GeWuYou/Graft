package user

import (
	"context"
	"github.com/gin-gonic/gin"
	"graft/server/internal/contract/openapi/strictsupport"
	useropenapi "graft/server/internal/contract/openapi/user"
)

// userStrictServer 将现有 HTTP 响应写入接入生成的请求契约。
type userStrictServer struct {
	registrar userRouteRegistrar
	saved     userSavedViewHandlers
}

var _ useropenapi.StrictServerInterface = userStrictServer{}

func (s userStrictServer) GetUsers(ctx context.Context, _ useropenapi.GetUsersRequestObject) (useropenapi.GetUsersResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleUserList)
}

// GetUserById 保留删除后用户详情的 404 行为。
//
//nolint:revive // 方法名必须与生成的 StrictServerInterface 一致。
func (s userStrictServer) GetUserById(ctx context.Context, _ useropenapi.GetUserByIdRequestObject) (useropenapi.GetUserByIdResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleUserDetail)
}

func (s userStrictServer) PostUsers(ctx context.Context, req useropenapi.PostUsersRequestObject) (useropenapi.PostUsersResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.registrar.handleUserCreate(c, req.Body) })
}

func (s userStrictServer) PostUserUpdate(ctx context.Context, req useropenapi.PostUserUpdateRequestObject) (useropenapi.PostUserUpdateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.registrar.handleUserUpdate(c, req.Body) })
}

func (s userStrictServer) PostUserStatus(ctx context.Context, req useropenapi.PostUserStatusRequestObject) (useropenapi.PostUserStatusResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.registrar.handleUserStatus(c, req.Body) })
}

func (s userStrictServer) PostUserResetPassword(ctx context.Context, req useropenapi.PostUserResetPasswordRequestObject) (useropenapi.PostUserResetPasswordResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.registrar.handleUserResetPassword(c, req.Body) })
}

func (s userStrictServer) DeleteUser(ctx context.Context, _ useropenapi.DeleteUserRequestObject) (useropenapi.DeleteUserResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleUserDelete)
}

func (s userStrictServer) GetUserSessions(ctx context.Context, _ useropenapi.GetUserSessionsRequestObject) (useropenapi.GetUserSessionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleUserSessions)
}

func (s userStrictServer) PostUserSessionRevoke(ctx context.Context, _ useropenapi.PostUserSessionRevokeRequestObject) (useropenapi.PostUserSessionRevokeResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleUserSessionRevoke)
}

func (s userStrictServer) PostUserSessionsRevokeAll(ctx context.Context, _ useropenapi.PostUserSessionsRevokeAllRequestObject) (useropenapi.PostUserSessionsRevokeAllResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleUserSessionsRevokeAll)
}

func (s userStrictServer) GetUserSavedViews(ctx context.Context, _ useropenapi.GetUserSavedViewsRequestObject) (useropenapi.GetUserSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.list)
}

func (s userStrictServer) PostUserSavedView(ctx context.Context, _ useropenapi.PostUserSavedViewRequestObject) (useropenapi.PostUserSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.create)
}

func (s userStrictServer) PutUserSavedView(ctx context.Context, _ useropenapi.PutUserSavedViewRequestObject) (useropenapi.PutUserSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.update)
}

func (s userStrictServer) DeleteUserSavedView(ctx context.Context, _ useropenapi.DeleteUserSavedViewRequestObject) (useropenapi.DeleteUserSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.remove)
}
