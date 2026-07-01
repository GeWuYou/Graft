package moduleregistry

import (
	"fmt"
	"slices"

	"graft/server/internal/i18n"
	"graft/server/internal/module"
	announcementlocales "graft/server/modules/announcement/locales"
	auditlocales "graft/server/modules/audit/locales"
	containerlocales "graft/server/modules/container/locales"
	monitorlocales "graft/server/modules/monitor/locales"
	projectlocales "graft/server/modules/project/locales"
	rbaclocales "graft/server/modules/rbac/locales"
	schedulerlocales "graft/server/modules/scheduler/locales"
	systemconfiglocales "graft/server/modules/system-config/locales"
	userlocales "graft/server/modules/user/locales"
)

// DefaultMigrationDir 是 `graft migrate` 默认链路使用的 owner-aligned 选择器。
//
// 它不是一个真实目录；CLI 在看到这个值时会回到 compile-time registry，
// 按模块依赖顺序展开默认迁移目录集合。
const DefaultMigrationDir = "default"

const accessLogMigrationDir = "internal/httpx/migrations"
const appLogMigrationDir = "internal/logger/migrations"
const drilldownMigrationDir = "internal/drilldown/migrations"

// EmbeddedMigrationFile 表示 compile-time 嵌入 registry 的单个迁移文件快照。
type EmbeddedMigrationFile struct {
	Name     string
	Contents []byte
}

// EmbeddedMigrationDir 表示 compile-time 嵌入 registry 的单个迁移目录快照。
type EmbeddedMigrationDir struct {
	Path  string
	Files []EmbeddedMigrationFile
}

// EmbeddedLocaleResources 返回按固定模块顺序聚合后的编译期本地化资源。
// 任一模块资源加载失败时直接 panic，避免运行时带着不完整资源启动。
func EmbeddedLocaleResources() []i18n.EmbeddedLocaleResource {
	providers := []struct {
		name string
		load func() ([]i18n.EmbeddedLocaleResource, error)
	}{
		{name: "announcement", load: announcementlocales.EmbeddedLocaleResources},
		{name: "audit", load: auditlocales.EmbeddedLocaleResources},
		{name: "container", load: containerlocales.EmbeddedLocaleResources},
		{name: "monitor", load: monitorlocales.EmbeddedLocaleResources},
		{name: "project", load: projectlocales.EmbeddedLocaleResources},
		{name: "rbac", load: rbaclocales.EmbeddedLocaleResources},
		{name: "scheduler", load: schedulerlocales.EmbeddedLocaleResources},
		{name: "system-config", load: systemconfiglocales.EmbeddedLocaleResources},
		{name: "user", load: userlocales.EmbeddedLocaleResources},
	}

	type loadedProviderResources struct {
		name      string
		resources []i18n.EmbeddedLocaleResource
	}

	loaded := make([]loadedProviderResources, 0, len(providers))
	capacity := 0
	for _, provider := range providers {
		items, err := provider.load()
		if err != nil {
			panic(fmt.Sprintf("load %s embedded locale resources: %v", provider.name, err))
		}
		loaded = append(loaded, loadedProviderResources{
			name:      provider.name,
			resources: items,
		})
		capacity += len(items)
	}

	resources := make([]i18n.EmbeddedLocaleResource, 0, capacity)
	for _, provider := range loaded {
		resources = append(resources, provider.resources...)
	}

	return resources
}

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

// CoreMigrationDirs returns the core live migration directories.
func CoreMigrationDirs() []string {
	return []string{accessLogMigrationDir, appLogMigrationDir, drilldownMigrationDir}
}

// EmbeddedMigrationDirs 返回 compile-time 生成的嵌入式迁移目录快照的深拷贝。
func EmbeddedMigrationDirs() []EmbeddedMigrationDir {
	dirs := make([]EmbeddedMigrationDir, 0, len(generatedEmbeddedMigrationDirs))
	for _, dir := range generatedEmbeddedMigrationDirs {
		cloned := EmbeddedMigrationDir{
			Path:  dir.Path,
			Files: make([]EmbeddedMigrationFile, 0, len(dir.Files)),
		}
		for _, file := range dir.Files {
			cloned.Files = append(cloned.Files, EmbeddedMigrationFile{
				Name:     file.Name,
				Contents: append([]byte(nil), file.Contents...),
			})
		}
		dirs = append(dirs, cloned)
	}

	return dirs
}

// EmbeddedMigrationDirByPath 根据指定路径从编译期嵌入迁移目录中查找匹配项。找到时返回目录的深拷贝和 true；未找到时返回零值和 false。
func EmbeddedMigrationDirByPath(path string) (EmbeddedMigrationDir, bool) {
	for _, dir := range generatedEmbeddedMigrationDirs {
		if dir.Path != path {
			continue
		}

		cloned := EmbeddedMigrationDir{
			Path:  dir.Path,
			Files: make([]EmbeddedMigrationFile, 0, len(dir.Files)),
		}
		for _, file := range dir.Files {
			cloned.Files = append(cloned.Files, EmbeddedMigrationFile{
				Name:     file.Name,
				Contents: append([]byte(nil), file.Contents...),
			})
		}

		return cloned, true
	}

	return EmbeddedMigrationDir{}, false
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
