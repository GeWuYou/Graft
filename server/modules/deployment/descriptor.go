package deployment

import "graft/server/internal/module"

const moduleID = "deployment"

// NewModuleSpec 声明只消费 Docker 原始事实的 capability-only Deployment Runtime 模块。
func NewModuleSpec() module.Spec {
	return module.Spec{ID: moduleID, Dependencies: []string{"container"}, Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
		return NewModule(), nil
	})}
}
