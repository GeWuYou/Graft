package moduleregistry

import (
	"fmt"
	"slices"

	"graft/server/internal/module"
)

// DefaultMigrationDir 是 `graft migrate` 默认链路使用的 owner-aligned 选择器。
//
// 它不是一个真实目录；CLI 在看到这个值时会回到 compile-time registry，
// 按模块依赖顺序展开默认迁移目录集合。
const DefaultMigrationDir = "default"

// HistoricalSharedMigrationDir 保留历史共享 Atlas 迁移目录的显式访问路径。
//
// 该目录不再属于默认 apply 链路，但仍可通过 `--dir` 手动执行历史共享链。
const HistoricalSharedMigrationDir = "internal/ent/migrate/migrations"

const accessLogMigrationDir = "internal/httpx/migrations"
const appLogMigrationDir = "internal/logger/migrations"
const drilldownMigrationDir = "internal/drilldown/migrations"

// ModuleSpecs 返回 compile-time 生成的模块定义快照。
func ModuleSpecs() []module.Spec {
	specs := make([]module.Spec, 0, len(generatedModuleSpecs))
	for _, spec := range generatedModuleSpecs {
		cloned := spec
		cloned.Dependencies = append([]string(nil), spec.Dependencies...)
		cloned.MigrationPath = append([]string(nil), spec.MigrationPath...)
		specs = append(specs, cloned)
	}

	return specs
}

// OrderedModuleSpecs 返回按依赖关系排序后的模块定义集合。
func OrderedModuleSpecs() ([]module.Spec, error) {
	return module.OrderSpecs(ModuleSpecs())
}

// FilteredOrderedModuleSpecs 返回按依赖排序且经过 enabled set 过滤后的模块定义集合。
//
// 当 enabled 为空时，表示当前运行时启用全部 compile-time modules。
func FilteredOrderedModuleSpecs(enabled []string) ([]module.Spec, error) {
	ordered, err := OrderedModuleSpecs()
	if err != nil {
		return nil, err
	}
	if len(enabled) == 0 {
		return ordered, nil
	}

	enabledSet := make(map[string]struct{}, len(enabled))
	for _, moduleID := range enabled {
		enabledSet[moduleID] = struct{}{}
	}

	if err := validateEnabledModulePresence(ordered, enabled); err != nil {
		return nil, err
	}

	filtered := filterOrderedSpecs(ordered, enabledSet)
	if err := validateFilteredModuleDependencies(filtered, enabledSet); err != nil {
		return nil, err
	}

	return filtered, nil
}

func validateEnabledModulePresence(ordered []module.Spec, enabled []string) error {
	for _, moduleID := range enabled {
		if slices.ContainsFunc(ordered, func(spec module.Spec) bool {
			return spec.Name() == moduleID
		}) {
			continue
		}

		return fmt.Errorf("enabled module %s is not present in compile-time registry", moduleID)
	}

	return nil
}

func filterOrderedSpecs(ordered []module.Spec, enabledSet map[string]struct{}) []module.Spec {
	filtered := make([]module.Spec, 0, len(ordered))
	for _, spec := range ordered {
		if _, ok := enabledSet[spec.Name()]; ok {
			filtered = append(filtered, spec)
		}
	}

	return filtered
}

func validateFilteredModuleDependencies(filtered []module.Spec, enabledSet map[string]struct{}) error {
	for _, spec := range filtered {
		for _, dependency := range spec.DependsOn() {
			if _, ok := enabledSet[dependency]; ok {
				continue
			}

			return fmt.Errorf("enabled module %s depends on disabled module %s", spec.Name(), dependency)
		}
	}

	return nil
}

// BuildModules 根据 compile-time 模块定义构造运行时模块集合。
func BuildModules(buildCtx module.BuildContext, enabled []string) ([]module.RuntimeModule, error) {
	ordered, err := FilteredOrderedModuleSpecs(enabled)
	if err != nil {
		return nil, err
	}

	built := make([]module.RuntimeModule, 0, len(ordered))
	for _, descriptor := range ordered {
		instance, err := descriptor.Build(buildCtx)
		if err != nil {
			return nil, fmt.Errorf("build module %s: %w", descriptor.Name(), err)
		}

		built = append(built, instance)
	}

	return built, nil
}

// CoreMigrationDirs 返回当前默认链路中的 core-owned live 迁移目录集合。
func CoreMigrationDirs() []string {
	return []string{accessLogMigrationDir, appLogMigrationDir, drilldownMigrationDir}
}

// MigrationDirs 返回当前 compile-time registry 声明的默认迁移目录集合。
//
// 默认链路先展开 live core-owned 目录，再按依赖排序展开 module-owned 目录，
// 避免 CLI 再手写第二份迁移顺序真相。
func MigrationDirs() ([]string, error) {
	ordered, err := FilteredOrderedModuleSpecs(nil)
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(CoreMigrationDirs())+len(ordered))
	dirs = append(dirs, CoreMigrationDirs()...)
	for _, descriptor := range ordered {
		dirs = append(dirs, descriptor.MigrationDirs()...)
	}

	return dedupePreserveOrder(dirs), nil
}

func dedupePreserveOrder(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}

	return deduped
}
