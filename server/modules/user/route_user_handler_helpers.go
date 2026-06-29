package user

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpheader "graft/server/internal/contract/httpheader"
	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/httpx"
	userstore "graft/server/modules/user/store"
)

func userHeaderPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func bindGeneratedHeaders(ginCtx *gin.Context) (*string, *string) {
	return userHeaderPointer(ginCtx.GetHeader(string(httpheader.Locale))),
		userHeaderPointer(ginCtx.GetHeader(httpx.RequestIDHeader))
}

func generatedUserParams[P any](ginCtx *gin.Context, build func(*string, *string) P) P {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return build(locale, requestID)
}

func bindGeneratedUserCreateParams(ginCtx *gin.Context) useropenapi.PostUsersParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUsersParams {
		return useropenapi.PostUsersParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

func bindGeneratedUserListParams(ginCtx *gin.Context) useropenapi.GetUsersParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.GetUsersParams {
		return useropenapi.GetUsersParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

func bindGeneratedUserDetailParams(ginCtx *gin.Context) useropenapi.GetUserByIdParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.GetUserByIdParams {
		return useropenapi.GetUserByIdParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

func bindGeneratedUserUpdateParams(ginCtx *gin.Context) useropenapi.PostUserUpdateParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUserUpdateParams {
		return useropenapi.PostUserUpdateParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

func bindGeneratedUserStatusParams(ginCtx *gin.Context) useropenapi.PostUserStatusParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUserStatusParams {
		return useropenapi.PostUserStatusParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

func bindGeneratedUserResetPasswordParams(ginCtx *gin.Context) useropenapi.PostUserResetPasswordParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUserResetPasswordParams {
		return useropenapi.PostUserResetPasswordParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

func bindGeneratedUserDeleteParams(ginCtx *gin.Context) useropenapi.PostUserDeleteParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUserDeleteParams {
		return useropenapi.PostUserDeleteParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

func (r userRouteRegistrar) writeUserItemResponse(
	ginCtx *gin.Context,
	message string,
	user userstore.User,
) bool {
	payload, mapErr := toUserListItem(user, nil)
	if mapErr != nil {
		r.runtime().writeResponseMappingError(ginCtx, message, mapErr, zap.Uint64("userID", user.ID))
		return false
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	return true
}
