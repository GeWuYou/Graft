// Package moduleruntime exposes a read-only snapshot of the compile-time module runtime.
package moduleruntime

import (
	"slices"
	"strings"

	"graft/server/internal/config"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/module"
)

const (
	// PermissionRead is the stable permission required to read module runtime snapshots.
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

// Snapshot is the OpenAPI-aligned module runtime snapshot response body.
type Snapshot = generated.ModuleRuntimeSnapshot

// Summary is the OpenAPI-aligned module runtime summary.
type Summary = generated.ModuleRuntimeSummary

// Item is the OpenAPI-aligned module runtime item.
type Item = generated.ModuleRuntimeItem

// Dependency is the OpenAPI-aligned module dependency status.
type Dependency = generated.ModuleRuntimeDependency

// MigrationStatus is the OpenAPI-aligned module migration declaration status.
type MigrationStatus = generated.ModuleRuntimeMigrationStatus

// SchemaStatus is the OpenAPI-aligned module schema declaration status.
type SchemaStatus = generated.ModuleRuntimeSchemaStatus

// ConfigStatus is the OpenAPI-aligned module config requirement status.
type ConfigStatus = generated.ModuleRuntimeConfigStatus

// BuildSnapshot builds a read-only module runtime snapshot from compile-time module specs and runtime config.
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

func cloneSpec(spec module.Spec) module.Spec {
	current := spec
	current.Dependencies = append([]string(nil), spec.Dependencies...)
	current.MigrationPath = append([]string(nil), spec.MigrationPath...)
	return current
}

func containsString(items []string, target string) bool {
	return slices.Contains(items, target)
}
