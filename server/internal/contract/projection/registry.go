package projection

import (
	authcontract "graft/server/internal/contract/auth"
	errorcodecontract "graft/server/internal/contract/errorcode"
	httpheadercontract "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
)

// Registry 返回平台级 canonical contract 的 web 导出索引副本。
//
// 每个 Value 都直接引用其 owner 中的 typed constant，避免 metadata 成为第二份字符串定义。
func Registry() []Entry {
	return []Entry{
		{ID: "platform.auth-scheme.bearer", Name: "BEARER", Kind: KindAuthScheme, Owner: "server/internal/contract/auth", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: authcontract.Bearer},
		{ID: "platform.error-code.auth-current-password-invalid", Name: "AUTH_CURRENT_PASSWORD_INVALID", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthCurrentPasswordInvalid},
		{ID: "platform.error-code.ok", Name: "OK", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.OK},
		{ID: "platform.error-code.auth-forbidden", Name: "AUTH_FORBIDDEN", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthForbidden},
		{ID: "platform.error-code.auth-invalid-credentials", Name: "AUTH_INVALID_CREDENTIALS", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthInvalidCredentials},
		{ID: "platform.error-code.auth-invalid-refresh-session", Name: "AUTH_INVALID_REFRESH_SESSION", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthInvalidRefreshSession},
		{ID: "platform.error-code.auth-missing-actor", Name: "AUTH_MISSING_ACTOR", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthMissingActor},
		{ID: "platform.error-code.auth-missing-permission", Name: "AUTH_MISSING_PERMISSION", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthMissingPermission},
		{ID: "platform.error-code.auth-password-policy-violation", Name: "AUTH_PASSWORD_POLICY_VIOLATION", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthPasswordPolicyViolation},
		{ID: "platform.error-code.auth-password-reuse-forbidden", Name: "AUTH_PASSWORD_REUSE_FORBIDDEN", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthPasswordReuseForbidden},
		{ID: "platform.error-code.auth-session-not-found", Name: "AUTH_SESSION_NOT_FOUND", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthSessionNotFound},
		{ID: "platform.error-code.auth-token-expired", Name: "AUTH_TOKEN_EXPIRED", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthTokenExpired},
		{ID: "platform.error-code.auth-token-invalid", Name: "AUTH_TOKEN_INVALID", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthTokenInvalid},
		{ID: "platform.error-code.auth-token-missing", Name: "AUTH_TOKEN_MISSING", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthTokenMissing},
		{ID: "platform.error-code.common-internal-error", Name: "COMMON_INTERNAL_ERROR", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.CommonInternalError},
		{ID: "platform.error-code.common-invalid-argument", Name: "COMMON_INVALID_ARGUMENT", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.CommonInvalidArgument},
		{ID: "platform.error-code.common-not-found", Name: "COMMON_NOT_FOUND", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.CommonNotFound},
		{ID: "platform.error-code.user-not-found", Name: "USER_NOT_FOUND", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.UserNotFound},
		{ID: "platform.http-header.authorization", Name: "AUTHORIZATION", Kind: KindHTTPHeader, Owner: "server/internal/contract/httpheader", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: httpheadercontract.Authorization},
		{ID: "platform.http-header.locale", Name: "LOCALE", Kind: KindHTTPHeader, Owner: "server/internal/contract/httpheader", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: httpheadercontract.Locale},
		{ID: "platform.http-header.trace-id", Name: "TRACE_ID", Kind: KindHTTPHeader, Owner: "server/internal/contract/httpheader", Lifecycle: LifecycleActive, Visibility: VisibilityInternal, Value: httpheadercontract.TraceID},
		{ID: "platform.message-key.auth-forbidden", Name: "AUTH_FORBIDDEN", Kind: KindMessageKey, Owner: "server/internal/contract/message", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: messagecontract.AuthForbidden},
		{ID: "platform.message-key.common-conjunction", Name: "COMMON_CONJUNCTION", Kind: KindMessageKey, Owner: "server/internal/contract/message", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: messagecontract.CommonConjunction},
		{ID: "platform.message-key.common-copyright", Name: "COMMON_COPYRIGHT", Kind: KindMessageKey, Owner: "server/internal/contract/message", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: messagecontract.CommonCopyright},
		{ID: "platform.message-key.common-internal-error", Name: "COMMON_INTERNAL_ERROR", Kind: KindMessageKey, Owner: "server/internal/contract/message", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: messagecontract.CommonInternalError},
	}
}
