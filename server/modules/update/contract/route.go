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
	// UpdateMenuPath 是 Update Center 的稳定前端路由。
	UpdateMenuPath = "/platform/updates"
)
