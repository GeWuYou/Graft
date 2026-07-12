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
	if len(menus) != 3 {
		t.Fatalf("expected 3 registered monitor menus, got %#v", menus)
	}

	overviewMenu := menus[0]
	assertMenuItem(t, overviewMenu, expectedMenuItem{
		code:       "monitor.server-status.overview",
		titleKey:   monitorcontract.ServerStatusOverviewMenuTitle.String(),
		path:       monitorcontract.ServerStatusOverviewMenuPath,
		icon:       "dashboard",
		order:      101,
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})

	runtimeMenu := menus[1]
	assertMenuItem(t, runtimeMenu, expectedMenuItem{
		code:       "monitor.server-status.runtime",
		titleKey:   monitorcontract.ServerStatusServiceStatusMenuTitle.String(),
		path:       monitorcontract.ServerStatusServiceStatusMenuPath,
		icon:       "runtime-overview",
		order:      102,
		permission: monitorcontract.ServerStatusReadPermission.String(),
	})

	dependenciesMenu := menus[2]
	assertMenuItem(t, dependenciesMenu, expectedMenuItem{
		code:       "monitor.server-status.dependencies",
		titleKey:   monitorcontract.ServerStatusDependenciesMenuTitle.String(),
		path:       monitorcontract.ServerStatusDependenciesMenuPath,
		icon:       "dependencies",
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
