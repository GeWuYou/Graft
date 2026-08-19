package contract

// JoinRoute 拼接路由组路径与路由片段，供审计模块和相关契约生成统一路径。
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// AuditGroup 是审计 API 的根路由组。
	AuditGroup = "/audit"

	// AuditIncidentParam 是审计事件详情路由使用的规范参数名。
	AuditIncidentParam = "event_id"

	// AuditLogParam 是审计日志详情路由使用的规范参数名。
	AuditLogParam = "id"

	// AuditCollection 是审计日志集合路由片段。
	AuditCollection = "/logs"

	// AuditItem 是审计日志详情路由片段。
	AuditItem = AuditCollection + "/:" + AuditLogParam
	// AuditBatchDeleteCollection 是审计日志批量删除路由片段。
	AuditBatchDeleteCollection = AuditCollection + "/batch-delete"

	// AuditVisibilityPolicyCollection 是审计可见性策略路由片段。
	AuditVisibilityPolicyCollection = "/policies/visibility"
	// AuditVisibilityOverrideCollection 是审计可见性覆盖规则路由片段。
	AuditVisibilityOverrideCollection = AuditVisibilityPolicyCollection + "/overrides"
	// AuditVisibilityOverrideBatchCollection 是审计可见性覆盖规则原子批量写入路由片段。
	AuditVisibilityOverrideBatchCollection = AuditVisibilityOverrideCollection + "/batch"

	// AuditIncidentItem 是审计事件详情路由片段。
	AuditIncidentItem = "/incidents/:" + AuditIncidentParam

	// AuditMenuPath 是审计 UI 菜单的规范路径。
	AuditMenuPath = "/security/audit"

	// AuditLogsMenuPath 是审计日志菜单的规范路径，与审计菜单根路径保持一致。
	AuditLogsMenuPath = AuditMenuPath

	// AuditLogDetailAPIPath 是审计日志详情 API 的规范路径模板。
	AuditLogDetailAPIPath = AuditGroup + AuditCollection + "/{" + AuditLogParam + "}"

	// AuditVisibilityPolicyAPIPath 是审计可见性策略 API 的规范路径。
	AuditVisibilityPolicyAPIPath = AuditGroup + AuditVisibilityPolicyCollection
	// AuditVisibilityOverrideAPIPath 是审计可见性覆盖规则 API 的规范路径。
	AuditVisibilityOverrideAPIPath = AuditGroup + AuditVisibilityOverrideCollection
	// AuditVisibilityOverrideBatchAPIPath 是审计可见性覆盖规则原子批量写入 API 的规范路径。
	AuditVisibilityOverrideBatchAPIPath = AuditGroup + AuditVisibilityOverrideBatchCollection

	// AuditIncidentAPIPath 是审计事件详情 API 的规范路径模板。
	AuditIncidentAPIPath = AuditGroup + "/incidents/{" + AuditIncidentParam + "}"
)
