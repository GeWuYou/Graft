package monitor

import "graft/server/internal/module"

const (
	moduleID = "monitor"
)

// NewModuleSpec 返回 monitor 模块稳定的编译期元数据和构造器。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:           moduleID,
		Dependencies: []string{"user", "rbac"},
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return NewModule(), nil
		}),
	}
}
