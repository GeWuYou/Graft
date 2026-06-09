// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/httpx"
	applog "graft/server/internal/logger"
	usercontract "graft/server/modules/user/contract"
)

func (r userRouteRegistrar) registerUserReadRoutes(group *gin.RouterGroup) {
	group.GET(usercontract.UserCollection, r.guards.userRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		userReadGeneratedHandler{}.GetUsers(bindGeneratedUserListParams(ginCtx))

		users, err := r.userSvc.ListUsers(ginCtx.Request.Context())
		if err != nil {
			r.runtime().appLogger().Named("read").Error(
				ginCtx.Request.Context(),
				"list users failed",
				applog.StringField("module", r.moduleName),
				applog.ErrorField(err),
			)
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			return
		}

		userIDs := make([]uint64, 0, len(users))
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
		}

		roleSummariesByUserID, err := r.userSvc.ListUserRoleSummaries(ginCtx.Request.Context(), userIDs)
		if err != nil {
			r.runtime().appLogger().Named("read").Error(
				ginCtx.Request.Context(),
				"list user role summaries failed",
				applog.StringField("module", r.moduleName),
				applog.ErrorField(err),
			)
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			return
		}

		payload, mapErr := toUserListResponse(users, roleSummariesByUserID)
		if mapErr != nil {
			r.runtime().writeResponseMappingError(ginCtx, "map user list response failed", mapErr)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
	group.GET(usercontract.UserByID, r.guards.userRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}
		userReadGeneratedHandler{}.GetUserByID(rawID, bindGeneratedUserDetailParams(ginCtx))

		record, err := r.userSvc.GetUser(ginCtx.Request.Context(), rawID)
		if err != nil {
			r.runtime().writeUserLookupError(ginCtx, rawID, "get user by id failed", err)
			return
		}

		payload, mapErr := toUserListItem(record, nil)
		if mapErr != nil {
			r.runtime().writeResponseMappingError(ginCtx, "map user detail response failed", mapErr, zap.Uint64("userID", rawID))
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
}

type userReadGeneratedHandler struct{}

func (h userReadGeneratedHandler) GetUsers(params useropenapi.GetUsersParams) {
	_ = h
	_ = params
}

func (h userReadGeneratedHandler) GetUserByID(id uint64, params useropenapi.GetUserByIdParams) {
	_ = h
	_ = id
	_ = params
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
		requestCtx := ginCtx.Request.Context()
		var request useropenapi.PostUsersJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		userWriteGeneratedHandler{}.PostUsers(bindGeneratedUserCreateParams(ginCtx), request)
		if field, ok := invalidCreateUserField(request); ok {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, field)
			return
		}

		command := toCreateUserCommand(request, requestActorID(requestCtx))
		created, err := r.userSvc.CreateUser(requestCtx, r.passwords, r.policy, command)
		if err != nil {
			r.runtime().writeCreateUserError(ginCtx, "create user failed", err)
			return
		}

		payload, mapErr := toUserListItem(created, nil)
		if mapErr != nil {
			r.runtime().writeResponseMappingError(ginCtx, "map created user response failed", mapErr, zap.Uint64("userID", created.ID))
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
}

func invalidCreateUserField(request useropenapi.PostUsersJSONRequestBody) (string, bool) {
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
		requestCtx := ginCtx.Request.Context()
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		var request useropenapi.PostUserUpdateJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		userWriteGeneratedHandler{}.PostUserUpdate(userID, bindGeneratedUserUpdateParams(ginCtx), request)

		command := toUpdateUserCommand(request, userID, requestActorID(requestCtx))
		updated, err := r.userSvc.UpdateUser(requestCtx, command)
		if err != nil {
			r.runtime().writeUserManagementError(ginCtx, userID, "update user failed", err)
			return
		}

		payload, mapErr := toUserListItem(updated, nil)
		if mapErr != nil {
			r.runtime().writeResponseMappingError(ginCtx, "map updated user response failed", mapErr, zap.Uint64("userID", updated.ID))
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
}

type userWriteGeneratedHandler struct{}

func (h userWriteGeneratedHandler) PostUsers(
	params useropenapi.PostUsersParams,
	body useropenapi.PostUsersJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h userWriteGeneratedHandler) PostUserUpdate(
	id uint64,
	params useropenapi.PostUserUpdateParams,
	body useropenapi.PostUserUpdateJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h userWriteGeneratedHandler) PostUserStatus(
	id uint64,
	params useropenapi.PostUserStatusParams,
	body useropenapi.PostUserStatusJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h userWriteGeneratedHandler) PostUserResetPassword(
	id uint64,
	params useropenapi.PostUserResetPasswordParams,
	body useropenapi.PostUserResetPasswordJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h userWriteGeneratedHandler) PostUserDelete(
	id uint64,
	params useropenapi.PostUserDeleteParams,
) {
	_ = h
	_ = id
	_ = params
}

func bindGeneratedUserCreateParams(ginCtx *gin.Context) useropenapi.PostUsersParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.PostUsersParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserListParams(ginCtx *gin.Context) useropenapi.GetUsersParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.GetUsersParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserDetailParams(ginCtx *gin.Context) useropenapi.GetUserByIdParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.GetUserByIdParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserUpdateParams(ginCtx *gin.Context) useropenapi.PostUserUpdateParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.PostUserUpdateParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserStatusParams(ginCtx *gin.Context) useropenapi.PostUserStatusParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.PostUserStatusParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserResetPasswordParams(ginCtx *gin.Context) useropenapi.PostUserResetPasswordParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.PostUserResetPasswordParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserDeleteParams(ginCtx *gin.Context) useropenapi.PostUserDeleteParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.PostUserDeleteParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := headerPointer(ginCtx.GetHeader(string(httpheader.Locale)))
	requestID := headerPointer(ginCtx.GetHeader(httpx.RequestIDHeader))
	return locale, requestID
}

func headerPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func (r userRouteRegistrar) registerSetUserStatusRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserStatusRoute, r.guards.userDisable, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		requestCtx := ginCtx.Request.Context()
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		var request useropenapi.PostUserStatusJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		userWriteGeneratedHandler{}.PostUserStatus(userID, bindGeneratedUserStatusParams(ginCtx), request)
		command, ok := toUpdateUserStatusCommand(request, userID, requestActorID(requestCtx))
		if !ok {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "status")
			return
		}

		updated, err := r.userSvc.SetUserStatus(requestCtx, r.authRepo, command)
		if err != nil {
			r.runtime().writeUserManagementError(ginCtx, userID, "set user status failed", err)
			return
		}

		payload, mapErr := toUserListItem(updated, nil)
		if mapErr != nil {
			r.runtime().writeResponseMappingError(ginCtx, "map user status response failed", mapErr, zap.Uint64("userID", updated.ID))
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
}

func (r userRouteRegistrar) registerResetUserPasswordRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserResetPasswordRoute, r.guards.userUpdate, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		requestCtx := ginCtx.Request.Context()
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		var request useropenapi.PostUserResetPasswordJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		userWriteGeneratedHandler{}.PostUserResetPassword(userID, bindGeneratedUserResetPasswordParams(ginCtx), request)

		if err := r.userSvc.ResetUserPassword(
			requestCtx,
			r.authRepo,
			r.passwords,
			r.policy,
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
		requestCtx := ginCtx.Request.Context()
		userID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}
		userWriteGeneratedHandler{}.PostUserDelete(userID, bindGeneratedUserDeleteParams(ginCtx))

		if err := r.userSvc.DeleteUser(requestCtx, r.authRepo, userID); err != nil {
			r.runtime().writeUserManagementError(ginCtx, userID, "delete user failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}
