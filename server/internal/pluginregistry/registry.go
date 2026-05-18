package pluginregistry

import (
	"fmt"

	"graft/server/internal/plugin"
)

// DefaultCoreMigrationDir 保留当前 core-owned Atlas 版本化迁移目录。
//
// Phase 1 先让 migrate wiring 从 compile-time registry 读取完整目录集合，
// 但当前 core migration 真相位置保持不变。
const DefaultCoreMigrationDir = "internal/ent/migrate/migrations"

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
// 目录顺序固定为 core 先于 plugin-owned 目录，plugin-owned 部分按依赖排序
// 展开，避免 CLI 再手写第二份插件顺序真相。
func MigrationDirs() ([]string, error) {
	ordered, err := OrderedDescriptors()
	if err != nil {
		return nil, err
	}

	dirs := []string{DefaultCoreMigrationDir}
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
