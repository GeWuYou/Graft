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
