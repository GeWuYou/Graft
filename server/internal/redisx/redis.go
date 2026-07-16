package redisx

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"

	"graft/server/internal/config"
)

const redisPingTimeout = 3 * time.Second

const defaultPoolSizePerCPU = 10
const percentageScale = 100

// HealthReporter 报告受限的 Redis 运行时健康信息，不向模块暴露底层客户端。
type HealthReporter interface {
	Report(context.Context) (HealthReport, error)
}

// HealthReport 汇总 Redis 可达性、探活延迟和连接池统计。
type HealthReport struct {
	Configured bool
	Reachable  bool
	Latency    time.Duration
	Pool       PoolStats
}

// PoolStats 以 core-owned 语义描述 Redis 连接池运行状态。
type PoolStats struct {
	Capacity             int
	MaxActiveConnections int
	OpenConnections      int
	InUseConnections     int
	IdleConnections      int
	UsagePercent         float64
	WaitCount            int64
	WaitDuration         time.Duration
	TimeoutCount         int64
	StaleCount           int64
}

type reporter struct {
	client *redis.Client
}

// Open 创建并验证服务端运行时所需的 Redis 客户端。
//
// 该函数会在给定上下文之上追加 3 秒探活超时；若 Ping 失败，会在返回前主动关闭客户端，
// Open 创建 Redis 客户端，验证与服务器的连通性。验证成功返回初始化的客户端；验证失败时关闭客户端并返回错误。
func Open(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		MaxActiveConns:  cfg.MaxActiveConns,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	})

	pingCtx, cancel := context.WithTimeout(ctx, redisPingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	return client, nil
}

// NewHealthReporter 为给定 Redis 客户端创建健康信息报告器。
func NewHealthReporter(client *redis.Client) HealthReporter {
	return reporter{client: client}
}

// Report 在有界超时内检查 Redis 可达性，并读取当前连接池统计。
func (r reporter) Report(ctx context.Context) (HealthReport, error) {
	if r.client == nil {
		return HealthReport{}, nil
	}

	pingCtx, cancel := context.WithTimeout(ctx, redisPingTimeout)
	defer cancel()

	startedAt := time.Now()
	if err := r.client.Ping(pingCtx).Err(); err != nil {
		return HealthReport{
			Configured: true,
			Reachable:  false,
			Pool:       poolStatsFromClient(r.client),
		}, fmt.Errorf("ping redis: %w", err)
	}

	return HealthReport{
		Configured: true,
		Reachable:  true,
		Latency:    time.Since(startedAt),
		Pool:       poolStatsFromClient(r.client),
	}, nil
}

// poolStatsFromClient 从 Redis 客户端配置和当前指标计算连接池统计；客户端为空时返回零值。
func poolStatsFromClient(client *redis.Client) PoolStats {
	if client == nil {
		return PoolStats{}
	}

	options := client.Options()
	stats := client.PoolStats()
	capacity := options.PoolSize
	if capacity <= 0 {
		capacity = defaultPoolSizePerCPU * runtime.GOMAXPROCS(0)
	}
	inUseConnections := int(stats.TotalConns - stats.IdleConns)

	return PoolStats{
		Capacity:             capacity,
		MaxActiveConnections: options.MaxActiveConns,
		OpenConnections:      int(stats.TotalConns),
		InUseConnections:     inUseConnections,
		IdleConnections:      int(stats.IdleConns),
		UsagePercent:         usagePercent(inUseConnections, capacity),
		WaitCount:            int64(stats.WaitCount),
		WaitDuration:         time.Duration(stats.WaitDurationNs),
		TimeoutCount:         int64(stats.Timeouts),
		StaleCount:           int64(stats.StaleConns),
	}
}

// usagePercent calculates the usage percentage of a connection pool.
// It returns 0 if inUse or capacity is not positive.
func usagePercent(inUse int, capacity int) float64 {
	if inUse <= 0 || capacity <= 0 {
		return 0
	}
	return float64(inUse) / float64(capacity) * percentageScale
}
