package rbac

import "graft/server/internal/module"

// registerDashboardWidgets 注册 RBAC 仪表盘组件；组件属于可选展示能力，不改变授权主流程。
func registerDashboardWidgets(_ *module.Context, _ managementReader) error {
	return nil
}
