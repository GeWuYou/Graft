package network

import (
	"testing"

	"graft/server/internal/menu"
)

func TestRegisterMenuUsesPlatformNetworkIcon(t *testing.T) {
	registry := menu.NewRegistry()
	if err := registerMenu(registry); err != nil {
		t.Fatalf("register network menu: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one network menu item, got %#v", items)
	}
	if items[0].Code != moduleID || items[0].Icon != "platform-network" {
		t.Fatalf("unexpected network menu item: %#v", items[0])
	}
}
