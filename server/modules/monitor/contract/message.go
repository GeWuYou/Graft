// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MenuMessageKey identifies a stable monitor module menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// MonitorSectionTitle identifies the localized title for the monitor navigation group.
	MonitorSectionTitle MenuMessageKey = "monitor.sectionTitle"
	// ServerStatusMenuTitle identifies the localized title for the server management root menu.
	ServerStatusMenuTitle MenuMessageKey = "menu.server.title"
	// ServerStatusOverviewMenuTitle identifies the localized title for the server overview menu.
	ServerStatusOverviewMenuTitle MenuMessageKey = "menu.server.overview.title"
	// ServerStatusRuntimeMenuTitle identifies the localized title for the server runtime menu.
	ServerStatusRuntimeMenuTitle MenuMessageKey = "menu.server.runtime.title"
	// ServerStatusDependenciesMenuTitle identifies the localized title for the server dependencies menu.
	ServerStatusDependenciesMenuTitle MenuMessageKey = "menu.server.dependencies.title"
)
