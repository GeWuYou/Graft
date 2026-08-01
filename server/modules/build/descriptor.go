// Package build 定义 Build domain 的模块生命周期与编译期接线入口，保持任务、执行器和 HTTP 能力在各自后续边界中实现。
package build

import "graft/server/internal/module"

// NewModuleSpec 返回 Build domain 的编译期模块描述符。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"auth", "rbac", "project", "task", "container"},
		MigrationPath: []string{"modules/build/migrations"},
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return NewModule(), nil
		}),
	}
}
