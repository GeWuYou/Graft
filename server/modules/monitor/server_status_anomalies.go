package monitor

import (
	"fmt"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	monitorcontract "graft/server/modules/monitor/contract"
)

// buildServerStatusAnomalies collects server status anomalies from dependencies, modules, and runtime metrics within the specified time window.
func buildServerStatusAnomalies(
	observedAt time.Time,
	trendRange monitorcontract.TrendRange,
	inputs serverStatusAnomalyInputs,
) []generated.ServerStatusAnomaly {
	windowStart := observedAt.Add(-trendRange.Duration())
	anomalies := make([]generated.ServerStatusAnomaly, 0)

	anomalies = append(anomalies, buildDependencyAnomalies(observedAt, windowStart, inputs.dependencies)...)
	anomalies = append(anomalies, buildModuleDependencyAnomalies(observedAt, windowStart, inputs.modules)...)
	anomalies = append(anomalies, buildRuntimeMetricAnomalies(observedAt, windowStart, inputs.runtimeSnapshot, inputs.trend)...)
	return anomalies
}

func buildDependencyAnomalies(
	observedAt time.Time,
	windowStart time.Time,
	dependencies generated.ServerStatusDependencies,
) []generated.ServerStatusAnomaly {
	anomalies := make([]generated.ServerStatusAnomaly, 0, serverDependencyCount)
	appendDependencyAnomaly := func(scopeRef string, dependency generated.ServerStatusDependency) {
		switch dependency.Status {
		case statusDegraded:
			anomalies = append(anomalies, generated.ServerStatusAnomaly{
				AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(monitorcontract.DependencyStatusDegraded),
				ScopeKind:  generated.ServerStatusAnomalyScopeKind(scopeKindDependency),
				ScopeRef:   scopeRef,
				Severity:   generated.ServerStatusAnomalySeverity(monitorcontract.SeverityCritical),
				Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
				ObservedAt: observedAt,
				Summary:    dependency.Detail,
				EvidenceLinks: []generated.EvidenceLink{
					unavailableEvidenceLink(windowStart, observedAt, "Audit evidence is not available for this dependency health issue."),
				},
			})
		case statusUnknown:
			anomalies = append(anomalies, generated.ServerStatusAnomaly{
				AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(monitorcontract.DependencyStatusUnknown),
				ScopeKind:  generated.ServerStatusAnomalyScopeKind(scopeKindDependency),
				ScopeRef:   scopeRef,
				Severity:   generated.ServerStatusAnomalySeverity(monitorcontract.SeverityWarning),
				Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
				ObservedAt: observedAt,
				Summary:    dependency.Detail,
				EvidenceLinks: []generated.EvidenceLink{
					unavailableEvidenceLink(windowStart, observedAt, "Audit evidence is not available for this dependency observability gap."),
				},
			})
		}
	}

	appendDependencyAnomaly("database", dependencies.Database)
	appendDependencyAnomaly("redis", dependencies.Redis)

	return anomalies
}

func buildModuleDependencyAnomalies(
	observedAt time.Time,
	windowStart time.Time,
	modules []generated.ServerStatusModule,
) []generated.ServerStatusAnomaly {
	anomalies := make([]generated.ServerStatusAnomaly, 0)
	for _, item := range modules {
		if item.MissingDependencies == nil || len(*item.MissingDependencies) == 0 {
			continue
		}
		anomalies = append(anomalies, generated.ServerStatusAnomaly{
			AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(monitorcontract.ModuleDependencyMissing),
			ScopeKind:  generated.ServerStatusAnomalyScopeKind(scopeKindModule),
			ScopeRef:   item.Name,
			Severity:   generated.ServerStatusAnomalySeverity(monitorcontract.SeverityCritical),
			Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
			ObservedAt: observedAt,
			Summary:    item.StatusDetail,
			EvidenceLinks: []generated.EvidenceLink{
				unavailableEvidenceLink(windowStart, observedAt, "Audit evidence is not available for this module dependency issue."),
			},
		})
	}
	return anomalies
}

func buildRuntimeMetricAnomalies(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
	trend generated.ServerStatusTrend,
) []generated.ServerStatusAnomaly {
	anomalies := make([]generated.ServerStatusAnomaly, 0)

	if cpuAnomaly, ok := buildCPUAnomaly(observedAt, windowStart, trend); ok {
		anomalies = append(anomalies, cpuAnomaly)
	}
	if memoryAnomaly, ok := buildMemoryAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, memoryAnomaly)
	}
	if diskAnomaly, ok := buildDiskAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, diskAnomaly)
	}
	if loadAnomaly, ok := buildLoadAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, loadAnomaly)
	}
	if goroutineAnomaly, ok := buildGoroutineAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, goroutineAnomaly)
	}
	if heapAnomaly, ok := buildHeapAnomaly(observedAt, windowStart, runtimeSnapshot); ok {
		anomalies = append(anomalies, heapAnomaly)
	}

	return anomalies
}

func buildCPUAnomaly(observedAt time.Time, windowStart time.Time, trend generated.ServerStatusTrend) (generated.ServerStatusAnomaly, bool) {
	cpuPercent, ok := latestTrendCPUPercent(trend)
	if !ok {
		return generated.ServerStatusAnomaly{}, false
	}
	severity, hit := classifyPercentSeverity(cpuPercent, cpuPressureWarningPercent, cpuPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.ResourceCPUPressure,
			scopeKind: scopeKindResource,
			scopeRef:  "runtime.cpu",
			severity:  severity,
			summary:   fmt.Sprintf("CPU usage reached %.1f%% in the current monitor window.", cpuPercent),
		},
	), true
}

func buildMemoryAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	severity, hit := classifyPercentSeverity(float64(runtimeSnapshot.HostMemoryUsedPercent), memoryPressureWarningPercent, memoryPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.ResourceMemoryPressure,
			scopeKind: scopeKindResource,
			scopeRef:  "runtime.host_memory",
			severity:  severity,
			summary:   fmt.Sprintf("Server memory usage reached %.1f%%.", float64(runtimeSnapshot.HostMemoryUsedPercent)),
		},
	), true
}

func buildDiskAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	if runtimeSnapshot.DiskUsage.TotalBytes <= 0 {
		return generated.ServerStatusAnomaly{}, false
	}
	severity, hit := classifyPercentSeverity(float64(runtimeSnapshot.DiskUsage.UsedPercent), diskPressureWarningPercent, diskPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.ResourceDiskPressure,
			scopeKind: scopeKindResource,
			scopeRef:  fmt.Sprintf("disk:%s", runtimeSnapshot.DiskUsage.Path),
			severity:  severity,
			summary:   fmt.Sprintf("Disk usage on %s reached %.1f%%.", runtimeSnapshot.DiskUsage.Path, float64(runtimeSnapshot.DiskUsage.UsedPercent)),
		},
	), true
}

func buildLoadAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	loadPercent := 0.0
	if runtimeSnapshot.CpuCores > 0 {
		loadPercent = (float64(runtimeSnapshot.LoadAverage.OneMinute) / float64(runtimeSnapshot.CpuCores)) * percentageScale
	}
	severity, hit := classifyPercentSeverity(loadPercent, loadPressureWarningPercent, loadPressureCriticalPercent)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.SystemLoadPressure,
			scopeKind: scopeKindRuntime,
			scopeRef:  "runtime.load",
			severity:  severity,
			summary:   fmt.Sprintf("1-minute load average reached %.2f against %d CPU cores.", float64(runtimeSnapshot.LoadAverage.OneMinute), runtimeSnapshot.CpuCores),
		},
	), true
}

func buildGoroutineAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	severity, hit := classifyCountSeverity(runtimeSnapshot.Goroutines, goroutinePressureWarningCount, goroutinePressureCriticalCount)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.RuntimeGoroutinePressure,
			scopeKind: scopeKindRuntime,
			scopeRef:  "runtime.goroutines",
			severity:  severity,
			summary:   fmt.Sprintf("Goroutine count reached %d.", runtimeSnapshot.Goroutines),
		},
	), true
}

func buildHeapAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	runtimeSnapshot generated.ServerStatusRuntime,
) (generated.ServerStatusAnomaly, bool) {
	severity, hit := classifyInt64Severity(runtimeSnapshot.RuntimeHeapInUseBytes, runtimeHeapWarningBytes, runtimeHeapCriticalBytes)
	if !hit {
		return generated.ServerStatusAnomaly{}, false
	}
	return newMetricAnomaly(
		observedAt,
		windowStart,
		metricAnomalySpec{
			key:       monitorcontract.RuntimeHeapPressure,
			scopeKind: scopeKindRuntime,
			scopeRef:  "runtime.heap_in_use",
			severity:  severity,
			summary:   fmt.Sprintf("Runtime heap usage reached %d bytes.", runtimeSnapshot.RuntimeHeapInUseBytes),
		},
	), true
}

func newMetricAnomaly(
	observedAt time.Time,
	windowStart time.Time,
	spec metricAnomalySpec,
) generated.ServerStatusAnomaly {
	return generated.ServerStatusAnomaly{
		AnomalyKey: generated.ServerStatusAnomalyAnomalyKey(spec.key),
		ScopeKind:  generated.ServerStatusAnomalyScopeKind(spec.scopeKind),
		ScopeRef:   spec.scopeRef,
		Severity:   generated.ServerStatusAnomalySeverity(spec.severity),
		Status:     generated.ServerStatusAnomalyStatus(anomalyStatusActive),
		ObservedAt: observedAt,
		Summary:    spec.summary,
		EvidenceLinks: []generated.EvidenceLink{
			availableEvidenceLink(windowStart, observedAt, "Review related audit activity", "Check audit records from the same bounded monitor window."),
		},
	}
}

func latestTrendCPUPercent(trend generated.ServerStatusTrend) (float64, bool) {
	if len(trend.Points) == 0 {
		return 0, false
	}
	return float64(trend.Points[len(trend.Points)-1].CpuPercent), true
}

func classifyPercentSeverity(value float64, warningThreshold float64, criticalThreshold float64) (monitorcontract.Severity, bool) {
	if value >= criticalThreshold {
		return monitorcontract.SeverityCritical, true
	}
	if value >= warningThreshold {
		return monitorcontract.SeverityWarning, true
	}
	return "", false
}

func classifyCountSeverity(value int, warningThreshold int, criticalThreshold int) (monitorcontract.Severity, bool) {
	if value >= criticalThreshold {
		return monitorcontract.SeverityCritical, true
	}
	if value >= warningThreshold {
		return monitorcontract.SeverityWarning, true
	}
	return "", false
}

func classifyInt64Severity(value int64, warningThreshold int64, criticalThreshold int64) (monitorcontract.Severity, bool) {
	if value >= criticalThreshold {
		return monitorcontract.SeverityCritical, true
	}
	if value >= warningThreshold {
		return monitorcontract.SeverityWarning, true
	}
	return "", false
}

func availableEvidenceLink(windowStart time.Time, windowEnd time.Time, title string, reason string) generated.EvidenceLink {
	return generated.EvidenceLink{
		TargetKind: generated.EvidenceLinkTargetKind(evidenceTargetAudit),
		LinkState:  generated.EvidenceLinkLinkState(evidenceStateAvailable),
		Title:      title,
		Reason:     stringPointer(reason),
		TimeWindow: &generated.EvidenceLinkTimeWindow{
			CreatedFrom: windowStart,
			CreatedTo:   windowEnd,
		},
		AuditContext: &generated.AuditEvidenceContext{
			CreatedFrom: &windowStart,
			CreatedTo:   &windowEnd,
		},
	}
}

func unavailableEvidenceLink(windowStart time.Time, windowEnd time.Time, reason string) generated.EvidenceLink {
	return generated.EvidenceLink{
		TargetKind: generated.EvidenceLinkTargetKind(evidenceTargetAudit),
		LinkState:  generated.EvidenceLinkLinkState(evidenceStateUnavailable),
		TitleKey:   stringPointer(monitorcontract.AuditEvidenceUnavailableTitle.String()),
		Title:      "",
		Reason:     stringPointer(reason),
		TimeWindow: &generated.EvidenceLinkTimeWindow{
			CreatedFrom: windowStart,
			CreatedTo:   windowEnd,
		},
	}
}

// stringPointer 返回 nil（若修剪后的 value 为空），否则返回指向 value 的指针。
func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
