// Package moduleruntime 暴露 compile-time module runtime 的只读快照。
package moduleruntime

import (
	"slices"
	"strings"

	"graft/server/internal/config"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/module"
)

const (
	// PermissionRead 是读取模块运行时快照所需的稳定权限。
	PermissionRead = "modules.runtime.read"

	enablementSourceAll       = "all"
	enablementSourceAllowlist = "allowlist"

	runtimeStatusRegistered = "registered"
	runtimeStatusDisabled   = "disabled"
	runtimeStatusDegraded   = "degraded"
	runtimeStatusUnknown    = "unknown"

	healthHealthy  = "healthy"
	healthDegraded = "degraded"
	healthUnknown  = "unknown"
	healthDisabled = "disabled"

	dependencyStatusSatisfied = "satisfied"
	dependencyStatusMissing   = "missing"
	dependencyStatusDisabled  = "disabled"

	migrationStatusDeclared    = "declared"
	migrationStatusNotDeclared = "not_declared"

	schemaStatusDeclared = "declared"
	schemaStatusUnknown  = "unknown"

	configStatusNotRequired = "not_required"
	configStatusUnknown     = "unknown"
)

// Snapshot 是与 OpenAPI 对齐的模块运行时快照响应体。
type Snapshot = generated.ModuleRuntimeSnapshot

// Summary 是与 OpenAPI 对齐的模块运行时汇总。
type Summary = generated.ModuleRuntimeSummary

// Item 是与 OpenAPI 对齐的模块运行时条目。
type Item = generated.ModuleRuntimeItem

// Dependency 是与 OpenAPI 对齐的模块依赖状态。
type Dependency = generated.ModuleRuntimeDependency

// MigrationStatus 是与 OpenAPI 对齐的模块迁移声明状态。
type MigrationStatus = generated.ModuleRuntimeMigrationStatus

// SchemaStatus 是与 OpenAPI 对齐的模块 schema 声明状态。
type SchemaStatus = generated.ModuleRuntimeSchemaStatus

// ConfigStatus 是与 OpenAPI 对齐的模块配置要求状态。
type ConfigStatus = generated.ModuleRuntimeConfigStatus

// BuildSnapshot 根据 compile-time module spec 和运行时配置构建只读模块运行时快照；输入 spec 会先复制以隔离调用方切片。
func BuildSnapshot(cfg *config.Config, specs []module.Spec) Snapshot {
	specs = cloneSpecs(specs)
	enablementSource := enablementSourceAll
	enabledSet := make(map[string]struct{})
	if cfg != nil && len(cfg.Modules.Enabled) > 0 {
		enablementSource = enablementSourceAllowlist
		for _, moduleID := range cfg.Modules.Enabled {
			moduleID = strings.TrimSpace(moduleID)
			if moduleID == "" {
				continue
			}
			enabledSet[moduleID] = struct{}{}
		}
	}

	presentSet := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		if spec.Name() == "" {
			continue
		}
		presentSet[spec.Name()] = struct{}{}
	}

	items := make([]Item, 0, len(specs))
	for _, spec := range specs {
		moduleKey := spec.Name()
		if moduleKey == "" {
			continue
		}

		enabled := enablementSource == enablementSourceAll
		if enablementSource == enablementSourceAllowlist {
			_, enabled = enabledSet[moduleKey]
		}

		dependencies := buildDependencies(spec.DependsOn(), presentSet, enabledSet, enablementSource)
		migrationStatus := buildMigrationStatus(spec.MigrationDirs())
		item := Item{
			ModuleKey:        moduleKey,
			Registered:       true,
			Enabled:          enabled,
			EnablementSource: generated.ModuleRuntimeItemEnablementSource(enablementSource),
			Dependencies:     dependencies,
			MigrationStatus:  migrationStatus,
			SchemaStatus:     buildSchemaStatus(migrationStatus),
			ConfigStatus:     ConfigStatus{Status: generated.ModuleRuntimeConfigStatusStatus(configStatusUnknown)},
			Diagnostics:      map[string]string{},
		}
		item.RuntimeStatus, item.Health = resolveModuleStatus(enabled, dependencies)
		items = append(items, item)
	}

	return Snapshot{
		Summary: buildSummary(items),
		Items:   items,
	}
}

// cloneSpecs 复制一组模块规格，并为每个规格创建独立副本。
func cloneSpecs(specs []module.Spec) []module.Spec {
	cloned := make([]module.Spec, 0, len(specs))
	for _, spec := range specs {
		cloned = append(cloned, cloneSpec(spec))
	}
	return cloned
}

func buildDependencies(
	dependencies []string,
	presentSet map[string]struct{},
	enabledSet map[string]struct{},
	enablementSource string,
) []Dependency {
	items := make([]Dependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		dependency = strings.TrimSpace(dependency)
		if dependency == "" {
			continue
		}

		_, present := presentSet[dependency]
		enabled := present && enablementSource == enablementSourceAll
		if present && enablementSource == enablementSourceAllowlist {
			_, enabled = enabledSet[dependency]
		}

		status := dependencyStatusSatisfied
		switch {
		case !present:
			status = dependencyStatusMissing
		case !enabled:
			status = dependencyStatusDisabled
		}

		items = append(items, Dependency{
			ModuleKey: dependency,
			Required:  true,
			Present:   present,
			Enabled:   enabled,
			Status:    generated.ModuleRuntimeDependencyStatus(status),
		})
	}
	return items
}

// buildMigrationStatus 汇总迁移目录并生成迁移状态。
// 它会去重、去除空白后的目录值，并根据是否存在已声明目录设置状态。
// 返回包含已声明目录和对应迁移状态的 MigrationStatus。
func buildMigrationStatus(dirs []string) MigrationStatus {
	declaredDirs := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" || containsString(declaredDirs, dir) {
			continue
		}
		declaredDirs = append(declaredDirs, dir)
	}

	status := migrationStatusNotDeclared
	if len(declaredDirs) > 0 {
		status = migrationStatusDeclared
	}

	return MigrationStatus{
		DeclaredDirs: declaredDirs,
		Status:       generated.ModuleRuntimeMigrationStatusStatus(status),
	}
}

func buildSchemaStatus(migrationStatus MigrationStatus) SchemaStatus {
	if string(migrationStatus.Status) == migrationStatusDeclared && len(migrationStatus.DeclaredDirs) > 0 {
		return SchemaStatus{Status: generated.ModuleRuntimeSchemaStatusStatus(schemaStatusDeclared)}
	}

	return SchemaStatus{Status: generated.ModuleRuntimeSchemaStatusStatus(schemaStatusUnknown)}
}

func resolveModuleStatus(enabled bool, dependencies []Dependency) (
	generated.ModuleRuntimeItemRuntimeStatus,
	generated.ModuleRuntimeItemHealth,
) {
	if !enabled {
		return generated.ModuleRuntimeItemRuntimeStatus(runtimeStatusDisabled),
			generated.ModuleRuntimeItemHealth(healthDisabled)
	}

	for _, dependency := range dependencies {
		if string(dependency.Status) != dependencyStatusSatisfied {
			return generated.ModuleRuntimeItemRuntimeStatus(runtimeStatusDegraded),
				generated.ModuleRuntimeItemHealth(healthDegraded)
		}
	}

	return generated.ModuleRuntimeItemRuntimeStatus(runtimeStatusRegistered),
		generated.ModuleRuntimeItemHealth(healthHealthy)
}

// 它统计模块总数、已启用数量、已注册数量以及各健康状态数量。
func buildSummary(items []Item) Summary {
	summary := Summary{TotalModules: len(items)}
	for _, item := range items {
		if item.Enabled {
			summary.EnabledModules++
		}
		if item.Registered {
			summary.RegisteredModules++
		}

		switch string(item.Health) {
		case healthHealthy:
			summary.HealthyModules++
		case healthDegraded:
			summary.DegradedModules++
		case healthUnknown:
			summary.UnknownModules++
		}
	}

	return summary
}

// cloneSpec 返回一个 module.Spec 的副本，并独立复制依赖和迁移路径切片。
func cloneSpec(spec module.Spec) module.Spec {
	current := spec
	current.Dependencies = append([]string(nil), spec.Dependencies...)
	current.MigrationPath = append([]string(nil), spec.MigrationPath...)
	return current
}

// containsString 判断 items 是否包含 target。
func containsString(items []string, target string) bool {
	return slices.Contains(items, target)
}
