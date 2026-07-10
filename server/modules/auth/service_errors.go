package auth

import (
	"errors"
	"net/http"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/moduleapi"
)

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

func authErrorDetails(err error) map[string]any {
	if errors.Is(err, errCurrentPasswordRequired) {
		return map[string]any{"field": "current_password"}
	}
	return nil
}
