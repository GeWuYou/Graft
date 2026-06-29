package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/module"
	"graft/server/internal/redisx"
)

// databaseHealth evaluates the database connection status.
// Returns dependency status unknown if the database is unavailable, degraded if the connection check fails,
// or healthy with latency and pool statistics if the database responds successfully. An error is returned
// databaseHealth 检查数据库连接的健康状态。
// 若实例或数据库句柄为空，返回未知状态。
// 通过 Ping 测试连接可达性：失败返回降级状态，成功返回健康状态及延迟信息。
// 仅当延迟转换失败时返回错误。
func databaseHealth(ctx context.Context, instance *Module) (generated.ServerStatusDependency, error) {
	if instance == nil || instance.db == nil {
		return generated.ServerStatusDependency{
			Status: statusUnknown,
			Detail: "Database handle is unavailable",
			Pool:   nil,
		}, nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	startedAt := time.Now()
	if err := instance.db.PingContext(pingCtx); err != nil {
		logTrendWarning(instance, nil, "database ping failed", err)
		return generated.ServerStatusDependency{
			Status: statusDegraded,
			Detail: "Database ping failed",
			Pool:   databasePoolStats(instance.db),
		}, nil
	}

	latencyMs, err := toGeneratedFloat32(roundLatencyMilliseconds(time.Since(startedAt)), "database latency ms")
	if err != nil {
		return generated.ServerStatusDependency{}, fmt.Errorf("convert database latency: %w", err)
	}
	return generated.ServerStatusDependency{
		Status:    statusHealthy,
		Detail:    "Database ping succeeded",
		LatencyMs: &latencyMs,
		Pool:      databasePoolStats(instance.db),
	}, nil
}

// redisHealth determines the health status and connectivity of a Redis dependency.
// The returned status is disabled if Redis is not configured, degraded if the health
// check fails or Redis is unreachable, and healthy if Redis responds successfully.
// Connection pool statistics and latency are included when available. An error is
// Returns an error only if latency metric conversion fails.
func redisHealth(ctx context.Context, moduleCtx *module.Context, instance *Module) (generated.ServerStatusDependency, error) {
	reporter := resolveRedisHealthReporter(moduleCtx, instance)
	if reporter == nil {
		return generated.ServerStatusDependency{
			Status: statusDisabled,
			Detail: "Redis client is not configured",
			Pool:   nil,
		}, nil
	}

	report, err := reporter.Report(ctx)
	if err != nil {
		logTrendWarning(nil, moduleCtx, "redis ping failed", err)
		return generated.ServerStatusDependency{
			Status: statusDegraded,
			Detail: "Redis ping failed",
			Pool:   redisPoolStats(report.Pool),
		}, nil
	}
	if !report.Configured {
		return generated.ServerStatusDependency{
			Status: statusDisabled,
			Detail: "Redis client is not configured",
			Pool:   nil,
		}, nil
	}
	if !report.Reachable {
		return generated.ServerStatusDependency{
			Status: statusDegraded,
			Detail: "Redis ping failed",
			Pool:   redisPoolStats(report.Pool),
		}, nil
	}

	latencyMs, err := toGeneratedFloat32(roundLatencyMilliseconds(report.Latency), "redis latency ms")
	if err != nil {
		return generated.ServerStatusDependency{}, fmt.Errorf("convert redis latency: %w", err)
	}
	return generated.ServerStatusDependency{
		Status:    statusHealthy,
		Detail:    "Redis ping succeeded",
		LatencyMs: &latencyMs,
		Pool:      redisPoolStats(report.Pool),
	}, nil
}

// databasePoolStats 从数据库连接句柄中提取连接池统计信息。
func databasePoolStats(db *sql.DB) *generated.ServerStatusConnectionPool {
	if db == nil {
		return nil
	}

	stats := db.Stats()
	capacity := stats.MaxOpenConnections
	if capacity <= 0 {
		capacity = stats.OpenConnections
	}
	maxActiveConnections := optionalPositiveInt64(stats.MaxOpenConnections)

	return &generated.ServerStatusConnectionPool{
		Capacity:             int64(capacity),
		MaxActiveConnections: maxActiveConnections,
		OpenConnections:      int64(stats.OpenConnections),
		InUseConnections:     int64(stats.InUse),
		IdleConnections:      int64(stats.Idle),
		UsagePercent:         poolUsagePercent(stats.InUse, capacity),
		WaitCount:            stats.WaitCount,
		WaitDurationMs:       float32(roundLatencyMilliseconds(stats.WaitDuration)),
		TimeoutCount:         0,
		StaleCount:           stats.MaxIdleClosed + stats.MaxIdleTimeClosed + stats.MaxLifetimeClosed,
	}
}

// ResolveRedisHealthReporter 获取 Redis 健康报告器。若实例已缓存则返回缓存值，否则从模块上下文解析；解析失败时返回 nil。
func resolveRedisHealthReporter(moduleCtx *module.Context, instance *Module) redisx.HealthReporter {
	if instance != nil && instance.redisHealth != nil {
		return instance.redisHealth
	}

	reporter, err := resolveOptionalRedisHealthReporter(moduleCtx)
	if err != nil {
		logTrendWarning(instance, moduleCtx, "resolve redis health reporter failed", err)
		return nil
	}

	return reporter
}

// redisPoolStats 将 Redis 连接池统计映射为服务器状态响应结构，容量和打开连接数均为 0 或以下时返回 nil。
func redisPoolStats(pool redisx.PoolStats) *generated.ServerStatusConnectionPool {
	if pool.Capacity <= 0 && pool.OpenConnections <= 0 {
		return nil
	}

	maxActiveConnections := optionalPositiveInt64(pool.MaxActiveConnections)

	return &generated.ServerStatusConnectionPool{
		Capacity:             int64(pool.Capacity),
		MaxActiveConnections: maxActiveConnections,
		OpenConnections:      int64(pool.OpenConnections),
		InUseConnections:     int64(pool.InUseConnections),
		IdleConnections:      int64(pool.IdleConnections),
		UsagePercent:         float32(roundUsagePercent(pool.UsagePercent)),
		WaitCount:            pool.WaitCount,
		WaitDurationMs:       float32(roundLatencyMilliseconds(pool.WaitDuration)),
		TimeoutCount:         pool.TimeoutCount,
		StaleCount:           pool.StaleCount,
	}
}

func optionalPositiveInt64(value int) *int64 {
	if value <= 0 {
		return nil
	}
	converted := int64(value)
	return &converted
}

func poolUsagePercent(inUse int, capacity int) float32 {
	if capacity <= 0 || inUse <= 0 {
		return 0
	}
	percent := float64(inUse) / float64(capacity) * percentageScale
	return float32(roundUsagePercent(percent))
}
