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
		{ID: "platform.error-code.ok", Name: "OK", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.OK},
		{ID: "platform.error-code.auth-forbidden", Name: "AUTH_FORBIDDEN", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.AuthForbidden},
		{ID: "platform.error-code.common-internal-error", Name: "COMMON_INTERNAL_ERROR", Kind: KindErrorCode, Owner: "server/internal/contract/errorcode", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: errorcodecontract.CommonInternalError},
		{ID: "platform.http-header.authorization", Name: "AUTHORIZATION", Kind: KindHTTPHeader, Owner: "server/internal/contract/httpheader", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: httpheadercontract.Authorization},
		{ID: "platform.http-header.locale", Name: "LOCALE", Kind: KindHTTPHeader, Owner: "server/internal/contract/httpheader", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: httpheadercontract.Locale},
		{ID: "platform.http-header.trace-id", Name: "TRACE_ID", Kind: KindHTTPHeader, Owner: "server/internal/contract/httpheader", Lifecycle: LifecycleActive, Visibility: VisibilityInternal, Value: httpheadercontract.TraceID},
		{ID: "platform.message-key.auth-forbidden", Name: "AUTH_FORBIDDEN", Kind: KindMessageKey, Owner: "server/internal/contract/message", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: messagecontract.AuthForbidden},
		{ID: "platform.message-key.common-internal-error", Name: "COMMON_INTERNAL_ERROR", Kind: KindMessageKey, Owner: "server/internal/contract/message", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: messagecontract.CommonInternalError},
	}
}
