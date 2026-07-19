package projection

import (
	authcontract "graft/server/internal/contract/auth"
	errorcodecontract "graft/server/internal/contract/errorcode"
	httpheadercontract "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	httpx "graft/server/internal/httpx"
	logger "graft/server/internal/logger"
	announcementcontract "graft/server/modules/announcement/contract"
	auditcontract "graft/server/modules/audit/contract"
	containercontract "graft/server/modules/container/contract"
	monitorcontract "graft/server/modules/monitor/contract"
	notificationcontract "graft/server/modules/notification/contract"
	projectcontract "graft/server/modules/project/contract"
	rbaccontract "graft/server/modules/rbac/contract"
	runtimecontract "graft/server/modules/runtime-target/contract"
	schedulercontract "graft/server/modules/scheduler/contract"
	securitycontract "graft/server/modules/security/contract"
	systemconfigcontract "graft/server/modules/system-config/contract"
	taskcontract "graft/server/modules/task/contract"
	usercontract "graft/server/modules/user/contract"
)

// Target 定义一组生成到同一 web 派生产物的 descriptor。
//
// Path 相对于 web/src/contracts/generated，避免模块值被提升为平台级 authority。
type Target struct {
	Path    string
	Groups  []Group
	Entries []Entry
}

// Group 定义一个 target 中按契约 kind 输出的 TypeScript 导出名称。
type Group struct {
	Kind     Kind
	Constant string
	TypeName string
}

// Targets 返回所有需要生成的 web contract targets。
func Targets() []Target {
	return []Target{
		{Path: "platform.ts", Entries: Registry()},
		{Path: "modules/access-log.ts", Groups: permissionGroups("ACCESS_LOG", "AccessLog"), Entries: accessLogRegistry()},
		{Path: "modules/announcement.ts", Groups: permissionGroups("ANNOUNCEMENT", "Announcement"), Entries: announcementRegistry()},
		{Path: "modules/app-log.ts", Groups: permissionGroups("APP_LOG", "AppLog"), Entries: appLogRegistry()},
		{Path: "modules/audit.ts", Groups: permissionGroups("AUDIT", "Audit"), Entries: auditRegistry()},
		{Path: "modules/container.ts", Groups: containerGroups(), Entries: ContainerRegistry()},
		{Path: "modules/monitor.ts", Groups: permissionGroups("MONITOR", "Monitor"), Entries: monitorRegistry()},
		{Path: "modules/notification.ts", Groups: permissionGroups("NOTIFICATION", "Notification"), Entries: notificationRegistry()},
		{Path: "modules/project.ts", Groups: projectGroups(), Entries: projectRegistry()},
		{Path: "modules/rbac.ts", Groups: permissionGroups("RBAC", "Rbac"), Entries: rbacRegistry()},
		{Path: "modules/runtime-target.ts", Groups: realtimeGroups("RUNTIME_TARGET", "RuntimeTarget"), Entries: runtimeTargetRegistry()},
		{Path: "modules/scheduled-task.ts", Groups: permissionGroups("SCHEDULED_TASK", "ScheduledTask"), Entries: scheduledTaskRegistry()},
		{Path: "modules/security.ts", Groups: permissionGroups("SECURITY", "Security"), Entries: securityRegistry()},
		{Path: "modules/system-config.ts", Groups: permissionGroups("SYSTEM_CONFIG", "SystemConfig"), Entries: systemConfigRegistry()},
		{Path: "modules/task.ts", Groups: taskGroups(), Entries: taskRegistry()},
		{Path: "modules/user.ts", Groups: permissionGroups("USER", "User"), Entries: userRegistry()},
	}
}

func permissionGroups(prefix string, typePrefix string) []Group {
	return []Group{{Kind: KindPermissionCode, Constant: prefix + "_PERMISSION_CODE", TypeName: typePrefix + "PermissionCode"}}
}

func realtimeGroups(prefix string, typePrefix string) []Group {
	return []Group{{Kind: KindRealtimeTopic, Constant: prefix + "_REALTIME_TOPIC", TypeName: typePrefix + "RealtimeTopic"}}
}

func projectGroups() []Group {
	return []Group{
		{Kind: KindErrorCode, Constant: "PROJECT_ERROR_CODE", TypeName: "ProjectErrorCode"},
		{Kind: KindRealtimeTopic, Constant: "PROJECT_REALTIME_TOPIC", TypeName: "ProjectRealtimeTopic"},
	}
}

func taskGroups() []Group {
	return []Group{
		{Kind: KindRealtimeTopic, Constant: "TASK_REALTIME_TOPIC", TypeName: "TaskRealtimeTopic"},
		{Kind: KindRealtimeEvent, Constant: "TASK_REALTIME_EVENT", TypeName: "TaskRealtimeEvent"},
	}
}

func containerGroups() []Group {
	return []Group{
		{Kind: KindPermissionCode, Constant: "CONTAINER_PERMISSION_CODE", TypeName: "ContainerPermissionCode"},
		{Kind: KindRealtimeTopic, Constant: "CONTAINER_REALTIME_TOPIC", TypeName: "ContainerRealtimeTopic"},
		{Kind: KindDockerImageRemoveErrorCode, Constant: "DOCKER_IMAGE_REMOVE_ERROR_CODES", TypeName: "DockerImageRemoveErrorCode"},
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

func accessLogRegistry() []Entry {
	return []Entry{{ID: "access-log.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/internal/httpx", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: httpx.AccessLogReadPermission}}
}

//nolint:dupl // 描述符清单必须保留 owner、生命周期和常量引用，不能隐藏为第二份值定义。
func announcementRegistry() []Entry {
	return []Entry{
		{ID: "announcement.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/modules/announcement/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: announcementcontract.AnnouncementReadPermission},
		{ID: "announcement.permission.create", Name: "CREATE", Kind: KindPermissionCode, Owner: "server/modules/announcement/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: announcementcontract.AnnouncementCreatePermission},
		{ID: "announcement.permission.update", Name: "UPDATE", Kind: KindPermissionCode, Owner: "server/modules/announcement/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: announcementcontract.AnnouncementUpdatePermission},
		{ID: "announcement.permission.publish", Name: "PUBLISH", Kind: KindPermissionCode, Owner: "server/modules/announcement/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: announcementcontract.AnnouncementPublishPermission},
		{ID: "announcement.permission.delete", Name: "DELETE", Kind: KindPermissionCode, Owner: "server/modules/announcement/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: announcementcontract.AnnouncementDeletePermission},
	}
}

func appLogRegistry() []Entry {
	return []Entry{
		{ID: "app-log.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/internal/logger", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: logger.AppLogReadPermission},
		{ID: "app-log.permission.delete", Name: "DELETE", Kind: KindPermissionCode, Owner: "server/internal/logger", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: logger.AppLogDeletePermission},
	}
}

func auditRegistry() []Entry {
	return []Entry{
		{ID: "audit.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/modules/audit/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: auditcontract.AuditReadPermission},
		{ID: "audit.permission.manage", Name: "MANAGE", Kind: KindPermissionCode, Owner: "server/modules/audit/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: auditcontract.AuditManagePermission},
	}
}

func monitorRegistry() []Entry {
	return []Entry{{ID: "monitor.permission.server-status-read", Name: "SERVER_STATUS_READ", Kind: KindPermissionCode, Owner: "server/modules/monitor/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: monitorcontract.ServerStatusReadPermission}}
}

func notificationRegistry() []Entry {
	return []Entry{
		{ID: "notification.permission.view", Name: "VIEW", Kind: KindPermissionCode, Owner: "server/modules/notification/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: notificationcontract.NotificationViewPermission},
		{ID: "notification.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/modules/notification/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: notificationcontract.NotificationReadPermission},
		{ID: "notification.permission.manage", Name: "MANAGE", Kind: KindPermissionCode, Owner: "server/modules/notification/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: notificationcontract.NotificationManagePermission},
	}
}

//nolint:dupl // 描述符清单必须保留 owner、生命周期和常量引用，不能隐藏为第二份值定义。
func projectRegistry() []Entry {
	return []Entry{
		{ID: "project.error-code.inspection-expired", Name: "INSPECTION_EXPIRED", Kind: KindErrorCode, Owner: "server/modules/project/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: projectcontract.ApplicationInspectionExpired},
		{ID: "project.realtime-topic.list-summary", Name: "LIST_SUMMARY", Kind: KindRealtimeTopic, Owner: "server/modules/project/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: projectcontract.ApplicationListSummaryTopic},
		{ID: "project.realtime-topic.runtime-prefix", Name: "RUNTIME_PREFIX", Kind: KindRealtimeTopic, Owner: "server/modules/project/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: projectcontract.ApplicationRuntimeTopicPrefix},
		{ID: "project.realtime-topic.lifecycle-config-prefix", Name: "LIFECYCLE_CONFIG_PREFIX", Kind: KindRealtimeTopic, Owner: "server/modules/project/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: projectcontract.ApplicationLifecycleConfigTopicPrefix},
		{ID: "project.realtime-topic.logs-prefix", Name: "LOGS_PREFIX", Kind: KindRealtimeTopic, Owner: "server/modules/project/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: projectcontract.ApplicationLogsTopicPrefix},
	}
}

//nolint:dupl // 描述符清单必须保留 owner、生命周期和常量引用，不能隐藏为第二份值定义。
func rbacRegistry() []Entry {
	return []Entry{
		{ID: "rbac.permission.role-read", Name: "ROLE_READ", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.RoleReadPermission},
		{ID: "rbac.permission.role-create", Name: "ROLE_CREATE", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.RoleCreatePermission},
		{ID: "rbac.permission.role-update", Name: "ROLE_UPDATE", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.RoleUpdatePermission},
		{ID: "rbac.permission.role-status-update", Name: "ROLE_STATUS_UPDATE", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.RoleStatusUpdatePermission},
		{ID: "rbac.permission.role-delete", Name: "ROLE_DELETE", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.RoleDeletePermission},
		{ID: "rbac.permission.role-permission-assign", Name: "ROLE_PERMISSION_ASSIGN", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.RolePermissionAssignPermission},
		{ID: "rbac.permission.permission-read", Name: "PERMISSION_READ", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.PermissionReadPermission},
		{ID: "rbac.permission.user-role-read", Name: "USER_ROLE_READ", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.UserRoleReadPermission},
		{ID: "rbac.permission.user-role-assign", Name: "USER_ROLE_ASSIGN", Kind: KindPermissionCode, Owner: "server/modules/rbac/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: rbaccontract.UserRoleAssignPermission},
	}
}

func runtimeTargetRegistry() []Entry {
	return []Entry{{ID: "runtime-target.realtime-topic.summary", Name: "SUMMARY", Kind: KindRealtimeTopic, Owner: "server/modules/runtime-target/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: runtimecontract.SummaryTopic}}
}

func scheduledTaskRegistry() []Entry {
	return []Entry{
		{ID: "scheduled-task.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/modules/scheduler/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: schedulercontract.ScheduledTaskReadPermission},
		{ID: "scheduled-task.permission.create", Name: "CREATE", Kind: KindPermissionCode, Owner: "server/modules/scheduler/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: schedulercontract.ScheduledTaskCreatePermission},
		{ID: "scheduled-task.permission.update", Name: "UPDATE", Kind: KindPermissionCode, Owner: "server/modules/scheduler/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: schedulercontract.ScheduledTaskUpdatePermission},
		{ID: "scheduled-task.permission.delete", Name: "DELETE", Kind: KindPermissionCode, Owner: "server/modules/scheduler/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: schedulercontract.ScheduledTaskDeletePermission},
		{ID: "scheduled-task.permission.run", Name: "RUN", Kind: KindPermissionCode, Owner: "server/modules/scheduler/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: schedulercontract.ScheduledTaskRunPermission},
		{ID: "scheduled-task.permission.enable", Name: "ENABLE", Kind: KindPermissionCode, Owner: "server/modules/scheduler/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: schedulercontract.ScheduledTaskEnablePermission},
	}
}

func securityRegistry() []Entry {
	return []Entry{{ID: "security.permission.overview-read", Name: "OVERVIEW_READ", Kind: KindPermissionCode, Owner: "server/modules/security/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: securitycontract.OverviewReadPermission}}
}

func systemConfigRegistry() []Entry {
	return []Entry{
		{ID: "system-config.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/modules/system-config/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: systemconfigcontract.SystemConfigReadPermission},
		{ID: "system-config.permission.write", Name: "WRITE", Kind: KindPermissionCode, Owner: "server/modules/system-config/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: systemconfigcontract.SystemConfigWritePermission},
	}
}

//nolint:dupl // 描述符清单必须保留 owner、生命周期和常量引用，不能隐藏为第二份值定义。
func taskRegistry() []Entry {
	return []Entry{
		{ID: "task.realtime-topic.prefix", Name: "PREFIX", Kind: KindRealtimeTopic, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeTopicPrefix},
		{ID: "task.realtime-event.created", Name: "CREATED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventCreated},
		{ID: "task.realtime-event.cancelled", Name: "CANCELLED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventCancelled},
		{ID: "task.realtime-event.cancel-requested", Name: "CANCEL_REQUESTED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventCancelRequested},
		{ID: "task.realtime-event.retry-requested", Name: "RETRY_REQUESTED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventRetryRequested},
		{ID: "task.realtime-event.stage-started", Name: "STAGE_STARTED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventStageStarted},
		{ID: "task.realtime-event.stage-completed", Name: "STAGE_COMPLETED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventStageCompleted},
		{ID: "task.realtime-event.stage-failed", Name: "STAGE_FAILED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventStageFailed},
		{ID: "task.realtime-event.log-appended", Name: "LOG_APPENDED", Kind: KindRealtimeEvent, Owner: "server/modules/task/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: taskcontract.TaskRealtimeEventLogAppended},
	}
}

func userRegistry() []Entry {
	return []Entry{
		{ID: "user.permission.read", Name: "READ", Kind: KindPermissionCode, Owner: "server/modules/user/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: usercontract.UserReadPermission},
		{ID: "user.permission.create", Name: "CREATE", Kind: KindPermissionCode, Owner: "server/modules/user/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: usercontract.UserCreatePermission},
		{ID: "user.permission.update", Name: "UPDATE", Kind: KindPermissionCode, Owner: "server/modules/user/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: usercontract.UserUpdatePermission},
		{ID: "user.permission.disable", Name: "DISABLE", Kind: KindPermissionCode, Owner: "server/modules/user/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: usercontract.UserDisablePermission},
		{ID: "user.permission.session-read", Name: "SESSION_READ", Kind: KindPermissionCode, Owner: "server/modules/user/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: usercontract.UserSessionReadPermission},
		{ID: "user.permission.session-revoke", Name: "SESSION_REVOKE", Kind: KindPermissionCode, Owner: "server/modules/user/contract", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: usercontract.UserSessionRevokePermission},
	}
}
