package monitor

import (
	"testing"

	"graft/server/internal/menu"
	monitorcontract "graft/server/plugins/monitor/contract"
)

func TestRegisterMonitorMenuIncludesThreeLevelEntries(t *testing.T) {
	t.Parallel()

	registry := menu.NewRegistry()
	registerMonitorMenu(registry, pluginID)

	menus := registry.Items()
	if len(menus) != 5 {
		t.Fatalf("expected 5 registered monitor menus, got %#v", menus)
	}

	sectionMenu := menus[0]
	assertMenuItem(t, sectionMenu, expectedMenuItem{
		code:       "monitor.section",
		titleKey:   monitorcontract.MonitorSectionTitle.String(),
		path:       monitorcontract.MonitorGroup,
		icon:       "server",
		permission: "",
	})

	serverStatusMenu := menus[1]
	assertMenuItem(t, serverStatusMenu, expectedMenuItem{
		code:       "monitor.server-status",
		titleKey:   monitorcontract.ServerStatusMenuTitle.String(),
		path:       monitorcontract.ServerStatusMenuPath,
		icon:       "activity",
		permission: "",
	})

	overviewMenu := menus[2]
	assertMenuItem(t, overviewMenu, expectedMenuItem{
		code:       "monitor.server-status.overview",
		titleKey:   monitorcontract.ServerStatusOverviewMenuTitle.String(),
		path:       monitorcontract.ServerStatusOverviewMenuPath,
		icon:       "dashboard",
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})

	runtimeMenu := menus[3]
	assertMenuItem(t, runtimeMenu, expectedMenuItem{
		code:       "monitor.server-status.runtime",
		titleKey:   monitorcontract.ServerStatusRuntimeMenuTitle.String(),
		path:       monitorcontract.ServerStatusRuntimeMenuPath,
		icon:       "time",
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})

	dependenciesMenu := menus[4]
	assertMenuItem(t, dependenciesMenu, expectedMenuItem{
		code:       "monitor.server-status.dependencies",
		titleKey:   monitorcontract.ServerStatusDependenciesMenuTitle.String(),
		path:       monitorcontract.ServerStatusDependenciesMenuPath,
		icon:       "data-base",
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})
}

type expectedMenuItem struct {
	code       string
	titleKey   string
	path       string
	icon       string
	permission string
}

func assertMenuItem(t *testing.T, actual menu.Item, expected expectedMenuItem) {
	t.Helper()

	if actual.Code != expected.code ||
		actual.TitleKey != expected.titleKey ||
		actual.Path != expected.path ||
		actual.Icon != expected.icon ||
		actual.Permission != expected.permission {
		t.Fatalf("expected canonical monitor menu contract, got %#v", actual)
	}
}
