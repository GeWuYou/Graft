package contract

// DependencyKind 标识依赖可观测性中稳定的依赖类别。
type DependencyKind string

const (
	// DependencyKindPostgreSQL 标识 PostgreSQL 依赖。
	DependencyKindPostgreSQL DependencyKind = "postgresql"
	// DependencyKindRedis 标识 Redis 依赖。
	DependencyKindRedis DependencyKind = "redis"
)

// DependencyHistoryStatus 标识请求窗口内 Redis 历史读取的可用性。
type DependencyHistoryStatus string

const (
	// DependencyHistoryStatusAvailable 表示历史存储已读取，点位可能为空。
	DependencyHistoryStatusAvailable DependencyHistoryStatus = "available"
	// DependencyHistoryStatusUnavailable 表示当前快照无法读取历史，但不影响当前依赖状态。
	DependencyHistoryStatusUnavailable DependencyHistoryStatus = "unavailable"
)

// DependencyHistoryUnavailableReason 标识不返回历史点位的稳定原因。
type DependencyHistoryUnavailableReason string

const (
	// DependencyHistoryUnavailableRedisNotConfigured 表示 Redis 历史存储未配置或未注册。
	DependencyHistoryUnavailableRedisNotConfigured DependencyHistoryUnavailableReason = "redis_not_configured"
	// DependencyHistoryUnavailableReadFailed 表示 Redis 历史读取失败。
	DependencyHistoryUnavailableReadFailed DependencyHistoryUnavailableReason = "read_failed"
)
