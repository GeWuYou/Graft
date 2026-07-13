// Package contract declares runtime-target stable navigation and permission identifiers.
package contract

// Stable module contract identifiers.
const (
	MenuTitle         = "menu.runtimeTargets.title"
	MenuPath          = "/infrastructure/runtime-targets"
	ViewPermission    = "runtime_target.view"
	ManagePermission  = "runtime_target.manage"
	RefreshPermission = "runtime_target.refresh"
	// SummaryTopic is the realtime topic for target-level resource snapshots.
	SummaryTopic = "runtime-target.summary.list"
)
