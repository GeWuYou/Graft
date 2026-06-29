package monitor

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"graft/server/internal/buildinfo"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/module"
	monitorcontract "graft/server/modules/monitor/contract"
)

// buildServerStatusResponse 构建包含当前运行时指标的服务器状态响应。
func buildServerStatusResponse(
	ctx context.Context,
	moduleCtx *module.Context,
	instance *Module,
	trendRange monitorcontract.TrendRange,
) (generated.ServerStatusResponse, error) {
	runtimeSnapshot, err := collectRuntimeSnapshot(ctx)
	if err != nil {
		return generated.ServerStatusResponse{}, err
	}
	return buildServerStatusResponseWithRuntimeSnapshot(ctx, moduleCtx, instance, trendRange, runtimeSnapshot)
}

// buildServerStatusResponseWithRuntimeSnapshot keeps the production response assembly.
// buildServerStatusResponseWithRuntimeSnapshot constructs a server status response
// using the provided runtime snapshot and returns dependency health, module status,
// buildServerStatusResponseWithRuntimeSnapshot 根据运行时快照和趋势范围构建服务器状态响应。
// 该函数聚合数据库和 Redis 健康状态、模块信息、趋势数据和系统异常，返回完整的服务器状态响应或在依赖项健康检查失败时返回错误。
func buildServerStatusResponseWithRuntimeSnapshot(
	ctx context.Context,
	moduleCtx *module.Context,
	instance *Module,
	trendRange monitorcontract.TrendRange,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusResponse, error) {
	observedAt := time.Now().UTC()
	startedAt := observedAt
	if instance != nil {
		if startedAtUnixNs := instance.startedAtUnixNs.Load(); startedAtUnixNs > 0 {
			startedAt = time.Unix(0, startedAtUnixNs).UTC()
		}
	}

	databaseStatus, err := databaseHealth(ctx, instance)
	if err != nil {
		return generated.ServerStatusResponse{}, err
	}
	redisStatus, err := redisHealth(ctx, moduleCtx, instance)
	if err != nil {
		return generated.ServerStatusResponse{}, err
	}
	modules := runtimeModuleSummaries(moduleCtx, databaseStatus, redisStatus)
	summary := buildServerStatusSummary(databaseStatus, redisStatus, modules)
	trend := buildServerStatusTrend(ctx, moduleCtx, instance, observedAt, trendRange)
	serverBuildInfo := resolveServerBuildInfo(moduleCtx)
	anomalies := buildServerStatusAnomalies(observedAt, trendRange, serverStatusAnomalyInputs{
		runtimeSnapshot: runtimeSnapshot,
		dependencies: generated.ServerStatusDependencies{
			Database: databaseStatus,
			Redis:    redisStatus,
		},
		modules: modules,
		trend:   trend,
	})

	return generated.ServerStatusResponse{
		Status:     deriveOverallStatus(databaseStatus.Status, redisStatus.Status, anomalies),
		ObservedAt: observedAt,
		Server: generated.ServerStatusServer{
			Build: generated.ServerStatusServerBuildInfo{
				Version:      serverBuildInfo.Version,
				BuildTimeUtc: serverBuildInfo.BuildTimeUTC,
				GitCommit:    serverBuildInfo.GitCommit,
				GitTreeState: generated.ServerStatusServerBuildInfoGitTreeState(serverBuildInfo.GitTreeState),
			},
			StartedAt:     startedAt,
			UptimeSeconds: int64(observedAt.Sub(startedAt).Seconds()),
			GoVersion:     runtime.Version(),
			AppName:       resolveAppName(moduleCtx),
			AppEnv:        resolveAppEnv(moduleCtx),
		},
		Runtime: runtimeSnapshot,
		Dependencies: generated.ServerStatusDependencies{
			Database: databaseStatus,
			Redis:    redisStatus,
		},
		Summary:   summary,
		Trend:     trend,
		Modules:   modules,
		Anomalies: anomalies,
	}, nil
}

// ResolveServerBuildInfo 从模块上下文中解析服务器构建信息，如果上下文不可用则返回规范化的空信息。
func resolveServerBuildInfo(moduleCtx *module.Context) buildinfo.Info {
	if moduleCtx == nil {
		return buildinfo.Normalize(buildinfo.Info{})
	}

	return moduleCtx.RuntimeMetadata.BuildInfo()
}

func runtimeModuleSummaries(
	moduleCtx *module.Context,
	database generated.ServerStatusDependency,
	redis generated.ServerStatusDependency,
) []generated.ServerStatusModule {
	if moduleCtx == nil {
		return nil
	}

	descriptors := moduleCtx.RuntimeMetadata.OrderedModuleDescriptors()
	available := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		name := strings.TrimSpace(descriptor.Name)
		if name == "" {
			continue
		}
		available[name] = struct{}{}
	}

	platformStatus := deriveOverallStatus(database.Status, redis.Status, nil)
	items := make([]generated.ServerStatusModule, 0, len(descriptors))
	for _, descriptor := range descriptors {
		dependsOn := append([]string(nil), descriptor.DependsOn...)
		status, statusDetail, missingDependencies := deriveRuntimeModuleObservation(descriptor, available, platformStatus)
		item := generated.ServerStatusModule{
			Name:         descriptor.Name,
			Status:       status,
			StatusDetail: statusDetail,
			DependsOn:    dependsOn,
		}
		if len(missingDependencies) > 0 {
			missing := append([]string(nil), missingDependencies...)
			item.MissingDependencies = &missing
		}
		items = append(items, item)
	}

	return items
}

// deriveRuntimeModuleObservation keeps module runtime semantics explicit and narrow:
// a module is healthy only when its runtime metadata is complete, its declared
// module dependencies are present, and the current shared runtime signals are not
// degraded. When that cannot be confirmed, the returned detail explains the most
// useful operator-facing reason instead of collapsing everything into a coarse summary.
func deriveRuntimeModuleObservation(
	descriptor module.DescriptorSnapshot,
	available map[string]struct{},
	platformStatus string,
) (status string, detail string, missingDependencies []string) {
	if strings.TrimSpace(descriptor.Name) == "" {
		return statusUnknown, "Runtime metadata is incomplete", nil
	}

	for _, dependency := range descriptor.DependsOn {
		dependencyName := strings.TrimSpace(dependency)
		if dependencyName == "" {
			continue
		}
		if _, ok := available[dependencyName]; !ok {
			missingDependencies = append(missingDependencies, dependencyName)
		}
	}

	if len(missingDependencies) > 0 {
		return statusDegraded,
			fmt.Sprintf("Missing runtime dependencies: %s", strings.Join(missingDependencies, ", ")),
			missingDependencies
	}

	switch platformStatus {
	case statusHealthy:
		return statusHealthy, "Runtime metadata is present and platform signals are healthy", nil
	case statusDegraded:
		return statusDegraded, "Runtime metadata is present, but shared runtime signals are degraded", nil
	default:
		return statusUnknown, "Runtime status is not fully observable from shared platform signals", nil
	}
}

// buildServerStatusSummary aggregates the health status of dependencies and modules into a summary containing total and categorized counts.
func buildServerStatusSummary(
	database generated.ServerStatusDependency,
	redis generated.ServerStatusDependency,
	modules []generated.ServerStatusModule,
) generated.ServerStatusSummary {
	summary := generated.ServerStatusSummary{
		TotalDependencies: len([]generated.ServerStatusDependency{database, redis}),
		TotalModules:      len(modules),
	}

	for _, dependency := range []generated.ServerStatusDependency{database, redis} {
		switch dependency.Status {
		case statusHealthy:
			summary.HealthyDependencies++
		case statusDegraded:
			summary.DegradedDependencies++
		case statusDisabled:
			summary.DisabledDependencies++
		default:
			summary.UnknownDependencies++
		}
	}

	for _, moduleSummary := range modules {
		if moduleSummary.Status == statusHealthy {
			summary.HealthyModules++
		}
	}

	return summary
}

func resolveAppName(moduleCtx *module.Context) string {
	if moduleCtx == nil || moduleCtx.Config == nil {
		return ""
	}
	return strings.TrimSpace(moduleCtx.Config.App.Name)
}

func resolveAppEnv(moduleCtx *module.Context) string {
	if moduleCtx == nil || moduleCtx.Config == nil {
		return ""
	}
	return strings.TrimSpace(moduleCtx.Config.App.Env)
}

func deriveOverallStatus(databaseStatus string, redisStatus string, anomalies []generated.ServerStatusAnomaly) string {
	for _, status := range []string{databaseStatus, redisStatus} {
		if status == statusDegraded {
			return statusDegraded
		}
	}

	if len(anomalies) > 0 {
		return statusDegraded
	}

	if databaseStatus == statusHealthy || redisStatus == statusHealthy {
		return statusHealthy
	}

	return statusUnknown
}
