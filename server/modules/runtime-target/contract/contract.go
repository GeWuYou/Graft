// Package contract 定义 runtime-target 稳定的导航和权限标识。
package contract

// 稳定的模块契约标识。
const (
	MenuTitle         = "menu.runtimeTargets.title"
	MenuPath          = "/infrastructure/runtime-targets"
	ViewPermission    = "runtime_target.view"
	ManagePermission  = "runtime_target.manage"
	RefreshPermission = "runtime_target.refresh"
	// SummaryTopic 是目标级资源快照使用的实时主题。
	SummaryTopic = "runtime-target.summary.list"
)
