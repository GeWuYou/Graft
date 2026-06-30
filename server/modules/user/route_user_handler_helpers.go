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

// userHeaderPointer 将非空白的头部值转换为字符串指针。
// 如果去除空白后为空，则返回 nil；否则返回去除空白后的值指针。
func userHeaderPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// 返回的 locale 和 requestID 会在空白时置为 nil。
func bindGeneratedHeaders(ginCtx *gin.Context) (*string, *string) {
	return userHeaderPointer(ginCtx.GetHeader(string(httpheader.Locale))),
		userHeaderPointer(ginCtx.GetHeader(httpx.RequestIDHeader))
}

// 它会读取 `XGraft-Locale` 和 `X-Request-Id`，将它们转换为可选字符串后传入 build。
func generatedUserParams[P any](ginCtx *gin.Context, build func(*string, *string) P) P {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return build(locale, requestID)
}

// bindGeneratedUserCreateParams 从 Gin 上下文构造创建用户接口的 OpenAPI 请求参数。
func bindGeneratedUserCreateParams(ginCtx *gin.Context) useropenapi.PostUsersParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUsersParams {
		return useropenapi.PostUsersParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

// bindGeneratedUserListParams 根据请求头生成用户列表接口参数。
// 它将 `XGraft-Locale` 和 `X-Request-Id` 绑定到 `useropenapi.GetUsersParams` 中。
func bindGeneratedUserListParams(ginCtx *gin.Context) useropenapi.GetUsersParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.GetUsersParams {
		return useropenapi.GetUsersParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

// bindGeneratedUserDetailParams 绑定用户详情接口所需的请求头参数，并构造对应的 OpenAPI 参数结构体。
func bindGeneratedUserDetailParams(ginCtx *gin.Context) useropenapi.GetUserByIdParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.GetUserByIdParams {
		return useropenapi.GetUserByIdParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

// bindGeneratedUserUpdateParams 将请求头中的用户相关参数绑定为用户更新接口的 OpenAPI 参数。
// 结果包含 `XGraftLocale` 和 `XRequestId` 字段。
func bindGeneratedUserUpdateParams(ginCtx *gin.Context) useropenapi.PostUserUpdateParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUserUpdateParams {
		return useropenapi.PostUserUpdateParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

// bindGeneratedUserStatusParams 绑定用户状态接口所需的请求头参数。
//
// XGraftLocale 和 XRequestId 来自 Gin 上下文中的对应请求头，空白值会被忽略。
func bindGeneratedUserStatusParams(ginCtx *gin.Context) useropenapi.PostUserStatusParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUserStatusParams {
		return useropenapi.PostUserStatusParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

// bindGeneratedUserResetPasswordParams 从 Gin 上下文构造重置密码接口的请求参数。
// 将 `XGraft-Locale` 和 `X-Request-Id` 头映射到 `useropenapi.PostUserResetPasswordParams` 的对应字段。
func bindGeneratedUserResetPasswordParams(ginCtx *gin.Context) useropenapi.PostUserResetPasswordParams {
	return generatedUserParams(ginCtx, func(locale *string, requestID *string) useropenapi.PostUserResetPasswordParams {
		return useropenapi.PostUserResetPasswordParams{XGraftLocale: locale, XRequestId: requestID}
	})
}

// bindGeneratedUserDeleteParams 从请求头生成用户删除接口的 OpenAPI 参数。
// 它将 `XGraft-Locale` 和 `X-Request-Id` 绑定到返回值的 `XGraftLocale` 与 `XRequestId` 字段。
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
