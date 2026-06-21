// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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

// HealthReporter reports narrow Redis runtime health without exposing the raw client to modules.
type HealthReporter interface {
	Report(context.Context) (HealthReport, error)
}

// HealthReport captures Redis availability, latency, and pool stats.
type HealthReport struct {
	Configured bool
	Reachable  bool
	Latency    time.Duration
	Pool       PoolStats
}

// PoolStats describes Redis connection-pool behavior in core-owned terms.
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

// NewHealthReporter returns a HealthReporter for monitoring the given Redis client.
func NewHealthReporter(client *redis.Client) HealthReporter {
	return reporter{client: client}
}

// Report checks Redis reachability and current pool stats.
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

// poolStatsFromClient builds PoolStats from a Redis client's pool configuration and current metrics.
// If the client is nil, it returns a zero-value PoolStats.
// poolStatsFromClient extracts and computes connection pool statistics from a Redis client.
// If the client is nil, it returns a zero-valued PoolStats.
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
// It returns 0 if inUse or capacity is not positive, otherwise the usage
usagePercent calculates the connection pool usage as a percentage between 0 and 100, returning 0 if inUse or capacity is non-positive.
func usagePercent(inUse int, capacity int) float64 {
	if inUse <= 0 || capacity <= 0 {
		return 0
	}
	return float64(inUse) / float64(capacity) * percentageScale
}
