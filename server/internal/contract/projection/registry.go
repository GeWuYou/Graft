package projection

import (
	authcontract "graft/server/internal/contract/auth"
	errorcodecontract "graft/server/internal/contract/errorcode"
	httpheadercontract "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	containercontract "graft/server/modules/container/contract"
)

// Target 定义一组生成到同一 web 派生产物的 descriptor。
//
// Path 相对于 web/src/contracts/generated，避免模块值被提升为平台级 authority。
type Target struct {
	Path    string
	Entries []Entry
}

// Targets 返回所有需要生成的 web contract targets。
func Targets() []Target {
	return []Target{
		{Path: "platform.ts", Entries: Registry()},
		{Path: "modules/container.ts", Entries: ContainerRegistry()},
	}
}

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

// ContainerRegistry 返回 container 模块已经公开给 web 的 canonical contract 导出索引。
func ContainerRegistry() []Entry {
	return []Entry{
		{ID: "container.permission.view", Name: "VIEW", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerViewPermission},
		{ID: "container.permission.detail", Name: "DETAIL", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerDetailPermission},
		{ID: "container.permission.events", Name: "EVENTS", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerEventsPermission},
		{ID: "container.permission.environment", Name: "ENVIRONMENT", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerEnvironmentPermission},
		{ID: "container.permission.logs", Name: "LOGS", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerLogsPermission},
		{ID: "container.permission.shell", Name: "SHELL", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerShellPermission},
		{ID: "container.permission.start", Name: "START", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerStartPermission},
		{ID: "container.permission.stop", Name: "STOP", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerStopPermission},
		{ID: "container.permission.restart", Name: "RESTART", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerRestartPermission},
		{ID: "container.permission.remove", Name: "REMOVE", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerRemovePermission},
		{ID: "container.permission.image-pull", Name: "IMAGE_PULL", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImagePullPermission},
		{ID: "container.permission.image-tag", Name: "IMAGE_TAG", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImageTagPermission},
		{ID: "container.permission.image-untag", Name: "IMAGE_UNTAG", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImageUntagPermission},
		{ID: "container.permission.image-remove", Name: "IMAGE_REMOVE", Kind: KindPermissionCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImageRemovePermission},
		{ID: "container.realtime-topic.dashboard-summary", Name: "DASHBOARD_SUMMARY", Kind: KindRealtimeTopic, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerDashboardSummaryTopic},
		{ID: "container.realtime-topic.events-prefix", Name: "EVENTS_PREFIX", Kind: KindRealtimeTopic, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerEventsTopicPrefix},
		{ID: "container.realtime-topic.logs-prefix", Name: "LOGS_PREFIX", Kind: KindRealtimeTopic, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerLogsTopicPrefix},
		{ID: "container.realtime-topic.list-stats", Name: "LIST_STATS", Kind: KindRealtimeTopic, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerListStatsTopic},
		{ID: "container.realtime-topic.stats-prefix", Name: "STATS_PREFIX", Kind: KindRealtimeTopic, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.ContainerStatsTopicPrefix},
		{ID: "container.docker-image-remove-error.image-referenced-by-multiple-tags", Name: "IMAGE_REFERENCED_BY_MULTIPLE_TAGS", Kind: KindDockerImageRemoveErrorCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImageMultipleTagsError},
		{ID: "container.docker-image-remove-error.image-in-use", Name: "IMAGE_IN_USE", Kind: KindDockerImageRemoveErrorCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImageInUseError},
		{ID: "container.docker-image-remove-error.image-not-found", Name: "IMAGE_NOT_FOUND", Kind: KindDockerImageRemoveErrorCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImageNotFoundError},
		{ID: "container.docker-image-remove-error.runtime-unavailable", Name: "DOCKER_RUNTIME_UNAVAILABLE", Kind: KindDockerImageRemoveErrorCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerRuntimeUnavailable},
		{ID: "container.docker-image-remove-error.timeout", Name: "DOCKER_TIMEOUT", Kind: KindDockerImageRemoveErrorCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerTimeout},
		{ID: "container.docker-image-remove-error.communication", Name: "DOCKER_COMMUNICATION_ERROR", Kind: KindDockerImageRemoveErrorCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerCommunicationError},
		{ID: "container.docker-image-remove-error.unknown", Name: "UNKNOWN", Kind: KindDockerImageRemoveErrorCode, Owner: "server/modules/container/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: containercontract.DockerImageRemoveUnknown},
	}
}
