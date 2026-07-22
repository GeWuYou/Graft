package redisx

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"graft/server/internal/config"
)

const redisPingTimeout = 3 * time.Second
const redisMetricsTimeout = 2 * time.Second

const defaultPoolSizePerCPU = 10
const percentageScale = 100

// HealthReporter 报告受限的 Redis 运行时健康信息，不向模块暴露底层客户端。
type HealthReporter interface {
	Report(context.Context) (HealthReport, error)
	// ReportMetrics 返回经过白名单过滤的可选 Redis 运行指标，不暴露底层客户端或原始 INFO 内容。
	ReportMetrics(context.Context) (Metrics, error)
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

// Metrics 是从 Redis INFO 白名单字段解析出的当前运行指标。
// 每个可空字段仅在 Redis 实际返回并成功解析时设置，避免把不可用指标误报为零值。
type Metrics struct {
	ConnectedClients          *int64
	BlockedClients            *int64
	MaxClients                *int64
	UsedMemoryBytes           *int64
	UsedMemoryPeakBytes       *int64
	MaxMemoryBytes            *int64
	MemoryFragmentationRatio  *float32
	TotalConnectionsReceived  *int64
	TotalCommandsProcessed    *int64
	InstantaneousOpsPerSecond *float32
	KeyspaceHitsTotal         *int64
	KeyspaceMissesTotal       *int64
	ExpiredKeysTotal          *int64
	EvictedKeysTotal          *int64
	RdbLastSaveAt             *time.Time
	RdbBgsaveInProgress       *bool
	AofEnabled                *bool
	AofRewriteInProgress      *bool
	ReplicationRole           *string
	MasterLinkStatus          *string
	Keyspaces                 *[]KeyspaceMetrics
}

// KeyspaceMetrics 是一个逻辑 Redis 数据库的聚合键空间统计，不包含任何键内容。
type KeyspaceMetrics struct {
	Database     string
	Keys         *int64
	Expires      *int64
	AverageTTLMs *int64
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

// ReportMetrics 通过 Redis INFO 的固定非敏感分区读取当前指标。
// 原始 INFO 文本不会离开 redisx，避免将地址、配置或未受支持字段传递到模块层。
func (r reporter) ReportMetrics(ctx context.Context) (Metrics, error) {
	if r.client == nil {
		return Metrics{}, nil
	}

	metricsCtx, cancel := context.WithTimeout(ctx, redisMetricsTimeout)
	defer cancel()

	raw, err := r.client.Info(metricsCtx, "clients", "memory", "stats", "keyspace", "persistence", "replication").Result()
	if err != nil {
		return Metrics{}, fmt.Errorf("read redis metrics: %w", err)
	}
	return parseMetrics(raw), nil
}

func parseMetrics(raw string) Metrics {
	values := parseInfoValues(raw)
	metrics := Metrics{
		ConnectedClients:          infoInt64(values, "connected_clients"),
		BlockedClients:            infoInt64(values, "blocked_clients"),
		MaxClients:                infoInt64(values, "maxclients"),
		UsedMemoryBytes:           infoInt64(values, "used_memory"),
		UsedMemoryPeakBytes:       infoInt64(values, "used_memory_peak"),
		MaxMemoryBytes:            infoInt64(values, "maxmemory"),
		MemoryFragmentationRatio:  infoFloat32(values, "mem_fragmentation_ratio"),
		TotalConnectionsReceived:  infoInt64(values, "total_connections_received"),
		TotalCommandsProcessed:    infoInt64(values, "total_commands_processed"),
		InstantaneousOpsPerSecond: infoFloat32(values, "instantaneous_ops_per_sec"),
		KeyspaceHitsTotal:         infoInt64(values, "keyspace_hits"),
		KeyspaceMissesTotal:       infoInt64(values, "keyspace_misses"),
		ExpiredKeysTotal:          infoInt64(values, "expired_keys"),
		EvictedKeysTotal:          infoInt64(values, "evicted_keys"),
		RdbBgsaveInProgress:       infoBool(values, "rdb_bgsave_in_progress"),
		AofEnabled:                infoBool(values, "aof_enabled"),
		AofRewriteInProgress:      infoBool(values, "aof_rewrite_in_progress"),
	}
	if savedAt := infoInt64(values, "rdb_last_save_time"); savedAt != nil && *savedAt > 0 {
		value := time.Unix(*savedAt, 0).UTC()
		metrics.RdbLastSaveAt = &value
	}
	metrics.ReplicationRole, metrics.MasterLinkStatus = parseReplicationMetrics(values)
	keyspaces := parseKeyspaces(values)
	if keyspaces != nil {
		metrics.Keyspaces = &keyspaces
	}
	return metrics
}

func parseReplicationMetrics(values map[string]string) (*string, *string) {
	var roleValue *string
	if role, ok := values["role"]; ok {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "master":
			value := "master"
			roleValue = &value
		case "slave", "replica":
			value := "replica"
			roleValue = &value
		default:
			value := "unknown"
			roleValue = &value
		}
	}
	if status, ok := values["master_link_status"]; ok {
		value := strings.ToLower(strings.TrimSpace(status))
		if value == "up" || value == "down" {
			return roleValue, &value
		}
	}
	return roleValue, nil
}

func parseInfoValues(raw string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values
}

func parseKeyspaces(values map[string]string) []KeyspaceMetrics {
	databases := make([]string, 0)
	for database := range values {
		if strings.HasPrefix(database, "db") && isDecimal(strings.TrimPrefix(database, "db")) {
			databases = append(databases, database)
		}
	}
	if len(databases) == 0 {
		return nil
	}
	sort.Strings(databases)
	keyspaces := make([]KeyspaceMetrics, 0, len(databases))
	for _, database := range databases {
		fields := parseKeyspaceFields(values[database])
		keyspaces = append(keyspaces, KeyspaceMetrics{
			Database:     database,
			Keys:         infoInt64(fields, "keys"),
			Expires:      infoInt64(fields, "expires"),
			AverageTTLMs: infoInt64(fields, "avg_ttl"),
		})
	}
	return keyspaces
}

func parseKeyspaceFields(raw string) map[string]string {
	fields := make(map[string]string)
	for _, field := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return fields
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func infoInt64(values map[string]string, key string) *int64 {
	value, ok := values[key]
	if !ok {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return nil
	}
	return &parsed
}

func infoFloat32(values map[string]string, key string) *float32 {
	value, ok := values[key]
	if !ok {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 32)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 {
		return nil
	}
	converted := float32(parsed)
	return &converted
}

func infoBool(values map[string]string, key string) *bool {
	value, ok := values[key]
	if !ok {
		return nil
	}
	switch strings.TrimSpace(value) {
	case "1":
		parsed := true
		return &parsed
	case "0":
		parsed := false
		return &parsed
	default:
		return nil
	}
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
