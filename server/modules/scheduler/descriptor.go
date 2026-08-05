package scheduler

import "graft/server/internal/module"

// NewModuleSpec 返回 scheduler 模块的稳定编译期元数据和构建器；模块实例的启动与关闭仍由宿主生命周期驱动。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"notification", "system-config", "saved-view"},
		MigrationPath: []string{"modules/scheduler/migrations"},
		Builder:       module.BuilderFunc(func(module.BuildContext) (module.Module, error) { return NewModule(), nil }),
	}
}
