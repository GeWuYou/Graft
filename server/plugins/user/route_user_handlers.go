package user

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	usercontract "graft/server/plugins/user/contract"
	userstore "graft/server/plugins/user/store"
)

func (r userRouteRegistrar) registerUserReadRoutes(group *gin.RouterGroup) {
	group.GET(usercontract.UserCollection, r.guards.userRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		users, err := r.userSvc.ListUsers(ginCtx.Request.Context())
		if err != nil {
			r.runtime().logger.Error("list users failed",
				zap.String("plugin", r.pluginName),
				zap.Error(err),
			)
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			return
		}

		items := make([]userListItem, 0, len(users))
		for _, user := range users {
			items = append(items, toUserListItem(user))
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, userListResponse{Items: items})
	})
	group.GET(usercontract.UserByID, r.guards.userRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		summary, err := r.userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			r.runtime().writeUserLookupError(ginCtx, rawID, "get user by id failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, summary)
	})
}

func (r userRouteRegistrar) registerUserWriteRoutes(group *gin.RouterGroup) {
	r.registerCreateUserRoute(group)
	r.registerUpdateUserRoute(group)
	r.registerSetUserStatusRoute(group)
	r.registerResetUserPasswordRoute(group)
	r.registerDeleteUserRoute(group)
}

func (r userRouteRegistrar) registerCreateUserRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserCollection, r.guards.userCreate, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		var request createUserRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		if field, ok := invalidCreateUserField(request); ok {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, field)
			return
		}

		created, err := r.userSvc.CreateUser(ginCtx.Request.Context(), r.authSvc.passwords, r.authSvc.policy, userstore.CreateUserInput{
			Username:     request.Username,
			Display:      request.Display,
			Status:       usercontract.UserStatusEnabled,
			ActorID:      requestActorID(ginCtx.Request.Context()),
			PasswordHash: request.Password,
		})
		if err != nil {
			r.runtime().writeCreateUserError(ginCtx, "create user failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toUserListItem(created))
	})
}

func invalidCreateUserField(request createUserRequest) (string, bool) {
	switch {
	case strings.TrimSpace(request.Username) == "":
		return "username", true
	case strings.TrimSpace(request.Display) == "":
		return "display", true
	case strings.TrimSpace(request.Password) == "":
		return "password", true
	default:
		return "", false
	}
}

func (r userRouteRegistrar) registerUpdateUserRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserUpdateRoute, r.guards.userUpdate, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		var request updateUserRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}

		updated, err := r.userSvc.UpdateUser(ginCtx.Request.Context(), userstore.UpdateUserInput{
			ID:       userID,
			Username: request.Username,
			Display:  request.Display,
			ActorID:  requestActorID(ginCtx.Request.Context()),
		})
		if err != nil {
			r.runtime().writeUserManagementError(ginCtx, userID, "update user failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toUserListItem(updated))
	})
}

func (r userRouteRegistrar) registerSetUserStatusRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserStatusRoute, r.guards.userDisable, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		var request updateUserStatusRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}

		updated, err := r.userSvc.SetUserStatus(ginCtx.Request.Context(), r.authSvc.auth, userstore.SetUserStatusInput{
			ID:      userID,
			Status:  request.Status,
			ActorID: requestActorID(ginCtx.Request.Context()),
		})
		if err != nil {
			r.runtime().writeUserManagementError(ginCtx, userID, "set user status failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toUserListItem(updated))
	})
}

func (r userRouteRegistrar) registerResetUserPasswordRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserResetPasswordRoute, r.guards.userUpdate, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		var request resetUserPasswordRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}

		if err := r.userSvc.ResetUserPassword(
			ginCtx.Request.Context(),
			r.authSvc.auth,
			r.authSvc.passwords,
			r.authSvc.policy,
			userID,
			request.NewPassword,
		); err != nil {
			r.runtime().writeUserManagementError(ginCtx, userID, "reset user password failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r userRouteRegistrar) registerDeleteUserRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserDeleteRoute, r.guards.userDisable, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		if err := r.userSvc.DeleteUser(ginCtx.Request.Context(), r.authSvc.auth, userID); err != nil {
			r.runtime().writeUserManagementError(ginCtx, userID, "delete user failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}
