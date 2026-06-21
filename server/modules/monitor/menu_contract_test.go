package monitor

import (
	"testing"

	"graft/server/internal/menu"
	monitorcontract "graft/server/modules/monitor/contract"
)

func TestRegisterMonitorMenuIncludesThreeLevelEntries(t *testing.T) {
	t.Parallel()

	registry := menu.NewRegistry()
	registerMonitorMenu(registry, moduleID)

	menus := registry.Items()
	if len(menus) != 4 {
		t.Fatalf("expected 4 registered monitor menus, got %#v", menus)
	}

	sectionMenu := menus[0]
	assertMenuItem(t, sectionMenu, expectedMenuItem{
		code:       "monitor.section",
		titleKey:   monitorcontract.ServerStatusMenuTitle.String(),
		path:       monitorcontract.ServerStatusMenuPath,
		icon:       "server",
		order:      100,
		permission: "",
	})

	overviewMenu := menus[1]
	assertMenuItem(t, overviewMenu, expectedMenuItem{
		code:       "monitor.server-status.overview",
		titleKey:   monitorcontract.ServerStatusOverviewMenuTitle.String(),
		path:       monitorcontract.ServerStatusOverviewMenuPath,
		icon:       "dashboard",
		order:      101,
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})

	runtimeMenu := menus[2]
	assertMenuItem(t, runtimeMenu, expectedMenuItem{
		code:       "monitor.server-status.runtime",
		titleKey:   monitorcontract.ServerStatusRuntimeMenuTitle.String(),
		path:       monitorcontract.ServerStatusRuntimeMenuPath,
		icon:       "time",
		order:      102,
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})

	dependenciesMenu := menus[3]
	assertMenuItem(t, dependenciesMenu, expectedMenuItem{
		code:       "monitor.server-status.dependencies",
		titleKey:   monitorcontract.ServerStatusDependenciesMenuTitle.String(),
		path:       monitorcontract.ServerStatusDependenciesMenuPath,
		icon:       "data-base",
		order:      103,
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})
}

type expectedMenuItem struct {
	code       string
	titleKey   string
	path       string
	icon       string
	order      int
	permission string
}

func assertMenuItem(t *testing.T, actual menu.Item, expected expectedMenuItem) {
	t.Helper()

	if actual.Code != expected.code ||
		actual.TitleKey != expected.titleKey ||
		actual.Path != expected.path ||
		actual.Icon != expected.icon ||
		actual.Order != expected.order ||
		actual.Permission != expected.permission {
		t.Fatalf("expected canonical monitor menu contract, got %#v", actual)
	}
}
