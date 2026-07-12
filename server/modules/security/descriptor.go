package security

import "graft/server/internal/module"

const moduleID = "security"

// NewModuleSpec 返回 security 模块的标识、依赖关系和构建器元数据。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:           moduleID,
		Dependencies: []string{"user", "rbac", "audit"},
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return NewModule(), nil
		}),
	}
}
