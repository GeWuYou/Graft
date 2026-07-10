package auth

import (
	"errors"
	"net/http"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/moduleapi"
)

// mapAuthError 将认证相关错误映射为 HTTP 状态码和消息契约键。
// 对于未匹配的错误，返回 500 状态码和通用内部错误键。
func mapAuthError(err error) (int, messagecontract.Key) {
	for _, mapping := range []struct {
		match  error
		status int
		key    messagecontract.Key
	}{
		{moduleapi.ErrUnauthenticated, http.StatusUnauthorized, messagecontract.AuthTokenMissing},
		{errInvalidLoginCredentials, http.StatusBadRequest, messagecontract.AuthInvalidCredentials},
		{errRefreshTokenRequired, http.StatusUnauthorized, messagecontract.AuthTokenMissing},
		{errExpiredRefreshToken, http.StatusUnauthorized, messagecontract.AuthTokenExpired},
		{errInvalidRefreshToken, http.StatusUnauthorized, messagecontract.AuthTokenInvalid},
		{errSessionNotFound, http.StatusNotFound, messagecontract.AuthSessionNotFound},
		{errRequiredPasswordChangeOnly, http.StatusForbidden, messagecontract.AuthForbidden},
		{errCurrentPasswordRequired, http.StatusBadRequest, messagecontract.CommonInvalidArgument},
		{errPasswordPolicyViolation, http.StatusBadRequest, messagecontract.AuthPasswordPolicyViolation},
		{errPasswordReuseForbidden, http.StatusBadRequest, messagecontract.AuthPasswordReuseForbidden},
		{errCurrentPasswordInvalid, http.StatusBadRequest, messagecontract.AuthCurrentPasswordInvalid},
		{errRefreshSessionFailed, http.StatusUnauthorized, messagecontract.AuthTokenInvalid},
	} {
		if errors.Is(err, mapping.match) {
			return mapping.status, mapping.key
		}
	}
	return http.StatusInternalServerError, messagecontract.CommonInternalError
}

// authErrorDetails 返回当前密码错误所需的字段级明细；匹配当前密码必填错误时返回字段名，否则返回 nil。
func authErrorDetails(err error) map[string]any {
	if errors.Is(err, errCurrentPasswordRequired) {
		return map[string]any{"field": "current_password"}
	}
	return nil
}
