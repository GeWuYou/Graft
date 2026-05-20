package pluginregistry

import (
	"fmt"

	"graft/server/internal/plugin"
)

// DefaultMigrationDir 是 `graft migrate` 默认链路使用的 owner-aligned 选择器。
//
// 它不是一个真实目录；CLI 在看到这个值时会回到 compile-time registry，
// 按插件依赖顺序展开默认迁移目录集合。
const DefaultMigrationDir = "default"

// HistoricalSharedMigrationDir 保留历史共享 Atlas 迁移目录的显式访问路径。
//
// 该目录不再属于默认 apply 链路，但仍可通过 `--dir` 手动执行历史共享链。
const HistoricalSharedMigrationDir = "internal/ent/migrate/migrations"

// Descriptors 返回 compile-time 生成的插件描述符快照。
func Descriptors() []plugin.Descriptor {
	return append([]plugin.Descriptor(nil), generatedDescriptors...)
}

// OrderedDescriptors 返回按依赖关系排序后的描述符集合。
func OrderedDescriptors() ([]plugin.Descriptor, error) {
	return plugin.OrderDescriptors(Descriptors())
}

// BuildPlugins 根据 compile-time 描述符构造运行时插件集合。
func BuildPlugins(buildCtx plugin.BuildContext) ([]plugin.Plugin, error) {
	ordered, err := OrderedDescriptors()
	if err != nil {
		return nil, err
	}

	built := make([]plugin.Plugin, 0, len(ordered))
	for _, descriptor := range ordered {
		instance, err := descriptor.Build(buildCtx)
		if err != nil {
			return nil, fmt.Errorf("build plugin %s: %w", descriptor.Name(), err)
		}

		built = append(built, instance)
	}

	return built, nil
}

// MigrationDirs 返回当前 compile-time registry 声明的默认迁移目录集合。
//
// 默认链路只展开 plugin-owned 目录，并按依赖排序展开，避免 CLI 再手写
// 第二份插件顺序真相。
func MigrationDirs() ([]string, error) {
	ordered, err := OrderedDescriptors()
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(ordered))
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
