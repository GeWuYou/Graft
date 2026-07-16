package contract

// AnomalyKey 标识一种 monitor 模块拥有的规范异常类别。
type AnomalyKey string

const (
	// DependencyStatusDegraded 表示依赖可达但处于不健康状态。
	DependencyStatusDegraded AnomalyKey = "dependency_status_degraded"
	// DependencyStatusUnknown 表示无法确定依赖健康状态。
	DependencyStatusUnknown AnomalyKey = "dependency_status_unknown"
	// ModuleDependencyMissing 表示模块声明的必需依赖未解析完成。
	ModuleDependencyMissing AnomalyKey = "module_dependency_missing"
	// ResourceCPUPressure 表示限定监控窗口内 CPU 使用率升高。
	ResourceCPUPressure AnomalyKey = "resource_cpu_pressure"
	// ResourceMemoryPressure 表示主机内存使用率升高。
	ResourceMemoryPressure AnomalyKey = "resource_memory_pressure"
	// ResourceDiskPressure 表示监控路径上的磁盘使用率升高。
	ResourceDiskPressure AnomalyKey = "resource_disk_pressure"
	// RuntimeGoroutinePressure 表示 goroutine 数量升高。
	RuntimeGoroutinePressure AnomalyKey = "runtime_goroutine_pressure"
	// RuntimeHeapPressure 表示 Go 堆使用量升高。
	RuntimeHeapPressure AnomalyKey = "runtime_heap_pressure"
	// SystemLoadPressure 表示相对于 CPU 核数偏高的系统负载。
	SystemLoadPressure AnomalyKey = "system_load_pressure"
)

// Severity 标识面向运维人员的限定异常严重级别。
type Severity string

const (
	// SeverityWarning 表示需要关注但尚未达到严重级别的异常。
	SeverityWarning Severity = "warning"
	// SeverityCritical 表示需要立即处理的异常。
	SeverityCritical Severity = "critical"
)
