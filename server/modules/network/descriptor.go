package network

import "graft/server/internal/module"

// NewModuleSpec 返回 platform-network 的编译期模块描述符。
func NewModuleSpec() module.Spec {
	return module.Spec{ID: moduleID, Dependencies: []string{"user", "rbac", "system-config"}, MigrationPath: []string{"modules/network/migrations"}, Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
		return NewModule(), nil
	})}
}
