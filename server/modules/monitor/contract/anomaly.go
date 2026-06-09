// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// AnomalyKey identifies one canonical monitor anomaly class.
type AnomalyKey string

const (
	// DependencyStatusDegraded marks a dependency that is reachable but unhealthy.
	DependencyStatusDegraded AnomalyKey = "dependency_status_degraded"
	// DependencyStatusUnknown marks a dependency whose health cannot be determined.
	DependencyStatusUnknown AnomalyKey = "dependency_status_unknown"
	// ModuleDependencyMissing marks a module with unresolved required dependencies.
	ModuleDependencyMissing AnomalyKey = "module_dependency_missing"
	// ResourceCPUPressure marks elevated CPU usage in the bounded monitor window.
	ResourceCPUPressure AnomalyKey = "resource_cpu_pressure"
	// ResourceMemoryPressure marks elevated host memory usage.
	ResourceMemoryPressure AnomalyKey = "resource_memory_pressure"
	// ResourceDiskPressure marks elevated disk usage on the monitored path.
	ResourceDiskPressure AnomalyKey = "resource_disk_pressure"
	// RuntimeGoroutinePressure marks elevated goroutine counts.
	RuntimeGoroutinePressure AnomalyKey = "runtime_goroutine_pressure"
	// RuntimeHeapPressure marks elevated Go heap usage.
	RuntimeHeapPressure AnomalyKey = "runtime_heap_pressure"
	// SystemLoadPressure marks elevated system load relative to CPU cores.
	SystemLoadPressure AnomalyKey = "system_load_pressure"
)

// Severity identifies the bounded operator-facing anomaly severity.
type Severity string

const (
	// SeverityWarning marks an anomaly that needs operator attention but is not yet critical.
	SeverityWarning Severity = "warning"
	// SeverityCritical marks an anomaly that needs immediate operator attention.
	SeverityCritical Severity = "critical"
)
