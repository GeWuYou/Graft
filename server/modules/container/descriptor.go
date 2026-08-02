package container

import "graft/server/internal/module"

const moduleID = "container"

// NewModuleSpec 提供 container 模块的标识符、依赖模块列表和构建器。
// 构建器创建并返回 container 模块实例。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:           moduleID,
		Dependencies: []string{"user", "auth", "rbac", "saved-view", "system-config", "runtime-target", "task"},
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return NewModule(), nil
		}),
	}
}
