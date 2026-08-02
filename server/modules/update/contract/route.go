package contract

const (
	// UpdateGroup 标识更新 API 的路由组。
	UpdateGroup = "/platform/updates"
	// UpdateStatusRoute 返回当前发现快照。
	UpdateStatusRoute = "/status"
	// UpdateCheckRoute 请求立即刷新上游发布目录。
	UpdateCheckRoute = "/check"
	// UpdateOperationCollectionRoute 提交人工确认的更新，或读取 Update 历史。
	UpdateOperationCollectionRoute = "/operations"
	// UpdateOperationRoute 读取一条不含秘密部署数据的操作记录。
	UpdateOperationRoute = "/operations/:operationID"
	// UpdateOperationRecoveryRoute 启动已终止 runner 的一次性受保护恢复。
	UpdateOperationRecoveryRoute = "/operations/:operationID/recovery"
	// UpdateActiveOperationRoute 读取由 runner 接管的当前操作，供新标签页恢复升级会话。
	// 它不位于 :operationID 参数路由下，避免保留字与操作标识发生路由语义冲突。
	UpdateActiveOperationRoute = "/active-operation"
	// UpdateOperationEventsRoute 回放一次 operation 的受控节点事件，不代理 runner 原始输出。
	UpdateOperationEventsRoute = "/operations/:operationID/events"
	// UpdateFailureDiagnosticRoute 读取一次更新启动失败的受控诊断详情。
	UpdateFailureDiagnosticRoute = "/diagnostics/:requestID"
	// UpdateOperationDiagnosticRoute 读取一条 operation 终态的受控失败诊断详情。
	UpdateOperationDiagnosticRoute = "/operations/:operationID/diagnostic"
	// UpdateMenuPath 是 Update Center 的稳定前端路由。
	UpdateMenuPath = "/platform/updates"
)
