package monitor

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/statex"
	statexkeys "graft/server/internal/statex/keys"
	monitorcontract "graft/server/modules/monitor/contract"
)

const dependencyHistoryStorageKeyPrefix = "graft:monitor:server-status:dependency-history"

type dependencyHistorySampleInput struct {
	appName      string
	hostName     string
	deploymentID string
	observedAt   time.Time
	database     generated.ServerStatusDependency
	redis        generated.ServerStatusDependency
}

// storeDependencyHistoryPoint 写入单个依赖探活的聚合桶。
// 保留策略与 monitor 趋势一致，严格限制在一小时内且写入失败不影响当前快照。
func storeDependencyHistoryPoint(
	ctx context.Context,
	trendStore statex.TimeSeriesStore,
	storageKey string,
	observedAt time.Time,
	point generated.ServerStatusDependencyHistoryPoint,
) error {
	payload, err := json.Marshal(point)
	if err != nil {
		return fmt.Errorf("marshal dependency history point: %w", err)
	}
	return trendStore.Append(ctx, storageKey, statex.TimeSeriesSample{
		ObservedAt: observedAt,
		Payload:    payload,
	}, statex.RetentionPolicy{
		TrimBefore:   observedAt.Add(-maxTrendRetentionWindow),
		ExpiresAfter: trendStorageTTL,
	})
}

func dependencyHistoryPoint(observedAt time.Time, dependency generated.ServerStatusDependency) (generated.ServerStatusDependencyHistoryPoint, bool) {
	probeCount := int64(1)
	point := generated.ServerStatusDependencyHistoryPoint{
		ObservedAt: observedAt,
		ProbeCount: &probeCount,
	}
	switch dependency.Status {
	case statusHealthy:
		availability := float32(percentageScale)
		failures := int64(0)
		point.AvailabilityPercent = &availability
		point.FailureCount = &failures
		if dependency.LatencyMs != nil {
			point.LatencyAverageMs = dependency.LatencyMs
			point.LatencyP95Ms = dependency.LatencyMs
		}
		return point, true
	case statusDegraded:
		availability := float32(0)
		failures := int64(1)
		point.AvailabilityPercent = &availability
		point.FailureCount = &failures
		return point, true
	default:
		return generated.ServerStatusDependencyHistoryPoint{}, false
	}
}

func recordDependencyHistorySamples(
	ctx context.Context,
	trendStore statex.TimeSeriesStore,
	input dependencyHistorySampleInput,
) {
	if trendStore == nil {
		return
	}
	for _, item := range []struct {
		kind       monitorcontract.DependencyKind
		dependency generated.ServerStatusDependency
	}{
		{kind: monitorcontract.DependencyKindPostgreSQL, dependency: input.database},
		{kind: monitorcontract.DependencyKindRedis, dependency: input.redis},
	} {
		point, ok := dependencyHistoryPoint(input.observedAt, item.dependency)
		if !ok {
			continue
		}
		if err := storeDependencyHistoryPoint(
			ctx,
			trendStore,
			dependencyHistoryStorageKey(input.appName, input.hostName, input.deploymentID, item.kind),
			input.observedAt,
			point,
		); err != nil {
			logger.Category(zap.L(), logger.CategoryRuntimeMetrics).Warn("store dependency history sample failed",
				zap.String("dependency", string(item.kind)),
				zap.Error(err),
			)
		}
	}
}

// buildDependencyHistory 读取指定依赖和窗口的短期聚合历史。
// Redis 历史不可用时保持当前快照可用，并以显式状态返回原因。
func buildDependencyHistory(
	ctx context.Context,
	moduleCtx *module.Context,
	instance *Module,
	observedAt time.Time,
	historyRange monitorcontract.TrendRange,
	dependencyKind monitorcontract.DependencyKind,
) *generated.ServerStatusDependencyHistory {
	retention := historyRange.Duration()
	history := &generated.ServerStatusDependencyHistory{
		Range:                 generated.ServerStatusDependencyHistoryRange(historyRange.String()),
		RetentionSeconds:      int64(retention.Seconds()),
		SampleIntervalSeconds: int64(trendSampleInterval.Seconds()),
		Status:                generated.ServerStatusDependencyHistoryStatus(monitorcontract.DependencyHistoryStatusUnavailable),
		Points:                make([]generated.ServerStatusDependencyHistoryPoint, 0),
	}

	trendStore := resolveTrendStore(moduleCtx, instance)
	if trendStore == nil {
		reason := generated.ServerStatusDependencyHistoryUnavailableReason(monitorcontract.DependencyHistoryUnavailableRedisNotConfigured)
		history.UnavailableReason = &reason
		return history
	}

	points, err := loadDependencyHistoryPoints(
		ctx,
		trendStore,
		dependencyHistoryStorageKey(resolveAppName(moduleCtx), resolveHostName(), monitorDeploymentID(instance), dependencyKind),
		observedAt,
		retention,
	)
	if err != nil {
		logTrendWarning(instance, moduleCtx, "load dependency history points failed", err)
		reason := generated.ServerStatusDependencyHistoryUnavailableReason(monitorcontract.DependencyHistoryUnavailableReadFailed)
		history.UnavailableReason = &reason
		return history
	}

	history.Status = generated.ServerStatusDependencyHistoryStatus(monitorcontract.DependencyHistoryStatusAvailable)
	if points.skipped > 0 {
		history.Status = generated.ServerStatusDependencyHistoryStatus(monitorcontract.DependencyHistoryStatusPartial)
	}
	history.Points = points.points
	return history
}

func loadDependencyHistoryPoints(
	ctx context.Context,
	trendStore statex.TimeSeriesStore,
	storageKey string,
	observedAt time.Time,
	retention time.Duration,
) (dependencyHistoryPoints, error) {
	samples, err := trendStore.Range(ctx, storageKey, statex.TimeSeriesQuery{
		StartAt: observedAt.Add(-retention),
		EndAt:   observedAt,
	})
	if err != nil {
		return dependencyHistoryPoints{}, fmt.Errorf("range dependency history points: %w", err)
	}

	points := make([]generated.ServerStatusDependencyHistoryPoint, 0, len(samples))
	skipped := 0
	for _, sample := range samples {
		var point generated.ServerStatusDependencyHistoryPoint
		if err := json.Unmarshal(sample.Payload, &point); err != nil {
			skipped++
			logger.Category(zap.L(), logger.CategoryRuntimeMetrics).Warn("decode stored dependency history point failed",
				zap.Time("observedAt", sample.ObservedAt),
				zap.String("storageKey", storageKey),
				zap.Error(err),
			)
			continue
		}
		points = append(points, point)
	}

	return dependencyHistoryPoints{points: points, skipped: skipped}, nil
}

type dependencyHistoryPoints struct {
	points  []generated.ServerStatusDependencyHistoryPoint
	skipped int
}

func monitorDeploymentID(instance *Module) string {
	if instance == nil {
		return "unstarted"
	}
	startedAt := instance.startedAtUnixNs.Load()
	if startedAt == 0 {
		return "unstarted"
	}
	return strconv.FormatInt(startedAt, 10)
}

func dependencyHistoryStorageKey(
	appName string,
	hostName string,
	deploymentID string,
	dependencyKind monitorcontract.DependencyKind,
) string {
	return fmt.Sprintf(
		"%s:%s:%s:%s:%s",
		dependencyHistoryStorageKeyPrefix,
		dependencyHistoryIdentitySegment(appName, "app"),
		dependencyHistoryIdentitySegment(hostName, "host"),
		statexkeys.Segment(deploymentID, "unstarted"),
		dependencyKind,
	)
}

func dependencyHistoryIdentitySegment(value string, fallback string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return statexkeys.Segment(value, fallback)
	}
	digest := sha256.Sum256([]byte(trimmed))
	return fmt.Sprintf("%s-%x", statexkeys.Segment(value, fallback), digest[:6])
}
