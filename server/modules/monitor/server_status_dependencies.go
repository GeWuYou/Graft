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

const (
	postgresActivityMetricsQuery = `SELECT
		count(*) FILTER (WHERE state = 'active'),
		count(*) FILTER (WHERE state = 'idle'),
		count(*) FILTER (WHERE state = 'idle in transaction'),
		count(*) FILTER (WHERE wait_event_type IS NOT NULL)
	FROM pg_stat_activity`
	postgresMaxConnectionsQuery  = `SELECT setting::bigint FROM pg_settings WHERE name = 'max_connections'`
	postgresDatabaseMetricsQuery = `SELECT
		pg_database_size(current_database()),
		xact_commit,
		xact_rollback,
		blks_read,
		blks_hit,
		tup_returned,
		tup_fetched,
		tup_inserted,
		tup_updated,
		tup_deleted,
		conflicts,
		deadlocks,
		temp_files,
		temp_bytes
	FROM pg_stat_database
	WHERE datname = current_database()`
)

// databaseHealth 检查数据库连接的健康状态。
// 若实例或数据库句柄为空，返回未知状态。
// 通过 Ping 测试连接可达性：失败返回降级状态，成功返回健康状态及延迟信息。
// 仅当延迟转换失败时返回错误；Ping 失败作为可观测的降级状态返回。
func databaseHealth(ctx context.Context, instance *Module) (generated.ServerStatusDependency, error) {
	dependency, err := databaseHealthProbe(ctx, instance)
	if err != nil || dependency.Status != statusHealthy {
		return dependency, err
	}

	if metrics, err := collectPostgreSQLMetrics(ctx, instance.db); err != nil {
		logTrendWarning(instance, nil, "collect PostgreSQL metrics failed", err)
	} else {
		dependency.PostgresqlMetrics = metrics
	}
	return dependency, nil
}

func databaseHealthProbe(ctx context.Context, instance *Module) (generated.ServerStatusDependency, error) {
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

// redisHealth 检查 Redis 连接状态；未配置时返回 disabled，检查失败或不可达时返回 degraded，成功时返回 healthy。
// 可用时同时返回连接池统计和延迟；连接检查失败作为状态返回，仅延迟转换失败才返回错误。
func redisHealth(ctx context.Context, moduleCtx *module.Context, instance *Module) (generated.ServerStatusDependency, error) {
	dependency, err := redisHealthProbe(ctx, moduleCtx, instance)
	if err != nil || dependency.Status != statusHealthy {
		return dependency, err
	}

	reporter := resolveRedisHealthReporter(moduleCtx, instance)
	if metrics, err := reporter.ReportMetrics(ctx); err != nil {
		logTrendWarning(instance, moduleCtx, "collect Redis metrics failed", err)
	} else {
		dependency.RedisMetrics = mapRedisMetrics(metrics)
	}
	return dependency, nil
}

func redisHealthProbe(ctx context.Context, moduleCtx *module.Context, instance *Module) (generated.ServerStatusDependency, error) {
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
		logTrendWarning(instance, moduleCtx, "redis ping failed", err)
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

// collectPostgreSQLMetrics 通过固定只读聚合查询收集当前诊断数据。
// 每个查询组独立失败，受限统计权限不会使已获得的当前健康快照或其它指标失效。
func collectPostgreSQLMetrics(ctx context.Context, db *sql.DB) (*generated.ServerStatusPostgresqlCurrentMetrics, error) {
	if db == nil {
		return nil, nil
	}
	metricsCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()
	metrics := &generated.ServerStatusPostgresqlCurrentMetrics{}
	collected := false
	var firstErr error

	var active, idle, idleInTransaction, waiting sql.NullInt64
	if err := db.QueryRowContext(metricsCtx, postgresActivityMetricsQuery).Scan(&active, &idle, &idleInTransaction, &waiting); err != nil {
		firstErr = fmt.Errorf("read PostgreSQL activity metrics: %w", err)
	} else {
		metrics.ActiveConnections = nullInt64Ptr(active)
		metrics.IdleConnections = nullInt64Ptr(idle)
		metrics.IdleInTransactionConnections = nullInt64Ptr(idleInTransaction)
		metrics.WaitingConnections = nullInt64Ptr(waiting)
		collected = true
	}

	var maxConnections sql.NullInt64
	if err := db.QueryRowContext(metricsCtx, postgresMaxConnectionsQuery).Scan(&maxConnections); err != nil {
		if firstErr == nil {
			firstErr = fmt.Errorf("read PostgreSQL connection limit: %w", err)
		}
	} else {
		metrics.MaxConnections = nullInt64Ptr(maxConnections)
		collected = true
	}

	var databaseSize, commits, rollbacks, blocksRead, blocksHit sql.NullInt64
	var tuplesReturned, tuplesFetched, tuplesInserted, tuplesUpdated, tuplesDeleted sql.NullInt64
	var conflicts, deadlocks, tempFiles, tempBytes sql.NullInt64
	if err := db.QueryRowContext(metricsCtx, postgresDatabaseMetricsQuery).Scan(
		&databaseSize, &commits, &rollbacks, &blocksRead, &blocksHit,
		&tuplesReturned, &tuplesFetched, &tuplesInserted, &tuplesUpdated, &tuplesDeleted,
		&conflicts, &deadlocks, &tempFiles, &tempBytes,
	); err != nil {
		if firstErr == nil {
			firstErr = fmt.Errorf("read PostgreSQL database metrics: %w", err)
		}
	} else {
		metrics.DatabaseSizeBytes = nullInt64Ptr(databaseSize)
		metrics.TransactionCommitTotal = nullInt64Ptr(commits)
		metrics.TransactionRollbackTotal = nullInt64Ptr(rollbacks)
		metrics.BlocksReadTotal = nullInt64Ptr(blocksRead)
		metrics.BlocksHitTotal = nullInt64Ptr(blocksHit)
		metrics.TuplesReturnedTotal = nullInt64Ptr(tuplesReturned)
		metrics.TuplesFetchedTotal = nullInt64Ptr(tuplesFetched)
		metrics.TuplesInsertedTotal = nullInt64Ptr(tuplesInserted)
		metrics.TuplesUpdatedTotal = nullInt64Ptr(tuplesUpdated)
		metrics.TuplesDeletedTotal = nullInt64Ptr(tuplesDeleted)
		metrics.ConflictsTotal = nullInt64Ptr(conflicts)
		metrics.DeadlocksTotal = nullInt64Ptr(deadlocks)
		metrics.TempFilesTotal = nullInt64Ptr(tempFiles)
		metrics.TempBytesTotal = nullInt64Ptr(tempBytes)
		metrics.CacheHitPercent = cacheHitPercent(metrics.BlocksHitTotal, metrics.BlocksReadTotal)
		collected = true
	}

	if !collected {
		return nil, firstErr
	}
	return metrics, nil
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}
	converted := value.Int64
	return &converted
}

func cacheHitPercent(hits, reads *int64) *float32 {
	if hits == nil || reads == nil || *hits+*reads <= 0 {
		return nil
	}
	percent := float32(roundUsagePercent(float64(*hits) / float64(*hits+*reads) * percentageScale))
	return &percent
}

func mapRedisMetrics(metrics redisx.Metrics) *generated.ServerStatusRedisCurrentMetrics {
	ret := &generated.ServerStatusRedisCurrentMetrics{
		ConnectedClients:          metrics.ConnectedClients,
		BlockedClients:            metrics.BlockedClients,
		MaxClients:                metrics.MaxClients,
		UsedMemoryBytes:           metrics.UsedMemoryBytes,
		UsedMemoryPeakBytes:       metrics.UsedMemoryPeakBytes,
		MaxMemoryBytes:            metrics.MaxMemoryBytes,
		MemoryFragmentationRatio:  metrics.MemoryFragmentationRatio,
		TotalConnectionsReceived:  metrics.TotalConnectionsReceived,
		TotalCommandsProcessed:    metrics.TotalCommandsProcessed,
		InstantaneousOpsPerSecond: metrics.InstantaneousOpsPerSecond,
		KeyspaceHitsTotal:         metrics.KeyspaceHitsTotal,
		KeyspaceMissesTotal:       metrics.KeyspaceMissesTotal,
		KeyspaceHitPercent:        cacheHitPercent(metrics.KeyspaceHitsTotal, metrics.KeyspaceMissesTotal),
		ExpiredKeysTotal:          metrics.ExpiredKeysTotal,
		EvictedKeysTotal:          metrics.EvictedKeysTotal,
		RdbLastSaveAt:             metrics.RdbLastSaveAt,
		RdbBgsaveInProgress:       metrics.RdbBgsaveInProgress,
		AofEnabled:                metrics.AofEnabled,
		AofRewriteInProgress:      metrics.AofRewriteInProgress,
	}
	if metrics.ReplicationRole != nil {
		role := generated.ServerStatusRedisCurrentMetricsReplicationRole(*metrics.ReplicationRole)
		ret.ReplicationRole = &role
	}
	if metrics.MasterLinkStatus != nil {
		status := generated.ServerStatusRedisCurrentMetricsMasterLinkStatus(*metrics.MasterLinkStatus)
		ret.MasterLinkStatus = &status
	}
	if metrics.Keyspaces != nil {
		keyspaces := make([]generated.ServerStatusRedisKeyspaceMetrics, 0, len(*metrics.Keyspaces))
		for _, keyspace := range *metrics.Keyspaces {
			keyspaces = append(keyspaces, generated.ServerStatusRedisKeyspaceMetrics{
				Database:     keyspace.Database,
				Keys:         keyspace.Keys,
				Expires:      keyspace.Expires,
				AverageTtlMs: keyspace.AverageTTLMs,
			})
		}
		ret.Keyspaces = &keyspaces
	}
	return ret
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
