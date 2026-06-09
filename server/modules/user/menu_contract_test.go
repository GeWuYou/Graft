// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"testing"

	"graft/server/internal/menu"
	usercontract "graft/server/modules/user/contract"
)

func TestRegisterUserMenuIncludesTitleKey(t *testing.T) {
	ctx, _ := newModuleTestContext(t, moduleTestUserRepository{}, &moduleTestAuthRepository{})

	menus := ctx.MenuRegistry.Items()
	if len(menus) == 0 {
		t.Fatalf("expected registered menu items, got %#v", menus)
	}

	menu := menus[0]
	if menu.Code != "user.list" ||
		menu.TitleKey != usercontract.UserListMenuTitle.String() ||
		menu.Path != "/access-control/users" ||
		menu.Order != 2 ||
		menu.Permission != usercontract.UserReadPermission.String() {
		t.Fatalf("expected canonical user menu contract, got %#v", menu)
	}
}

func TestFilterBootstrapMenusIncludesTitleKeyAndFallback(t *testing.T) {
	menus := filterBootstrapMenus(testMenuRegistry(), map[string]struct{}{
		usercontract.UserReadPermission.String(): {},
	})

	if len(menus) != 2 {
		t.Fatalf("expected filtered menus to keep user and public entries, got %#v", menus)
	}
	if menus[0].Code != "user.list" ||
		menus[0].TitleKey != usercontract.UserListMenuTitle.String() ||
		menus[0].Order != 2 ||
		menus[0].Title != "用户管理" {
		t.Fatalf("expected canonical user bootstrap menu, got %#v", menus[0])
	}
	if menus[1].Code != "profile.self" || menus[1].TitleKey != "" || menus[1].Order != 999 || menus[1].Title != "个人中心" {
		t.Fatalf("expected fallback-only public menu, got %#v", menus[1])
	}
}

func TestFilterBootstrapMenusDeduplicatesSharedRootMenu(t *testing.T) {
	registry := menu.NewRegistry()
	registry.Register(menu.Item{
		Code:     "log-center.root",
		Title:    "日志中心",
		TitleKey: "menu.logCenter.title",
		Path:     "/logs",
		Icon:     "bulletpoint",
		Order:    210,
		Module:   "core.httpx",
	})
	registry.Register(menu.Item{
		Code:     "log-center.root",
		Title:    "日志中心",
		TitleKey: "menu.logCenter.title",
		Path:     "/logs",
		Icon:     "bulletpoint",
		Order:    210,
		Module:   "core.logger",
	})
	registry.Register(menu.Item{
		Code:       "access-log.list",
		Title:      "访问日志",
		TitleKey:   "menu.accessLog.title",
		Path:       "/logs/access",
		Order:      211,
		Permission: "access_log.read",
	})
	registry.Register(menu.Item{
		Code:       "app-log.list",
		Title:      "应用日志",
		TitleKey:   "menu.appLog.title",
		Path:       "/logs/app",
		Order:      212,
		Permission: "app_log.read",
	})

	menus := filterBootstrapMenus(registry, map[string]struct{}{
		"access_log.read": {},
		"app_log.read":    {},
	})

	if len(menus) != 3 {
		t.Fatalf("expected one root and two leaf log menus, got %#v", menus)
	}
	if menus[0].Code != "log-center.root" || menus[0].Path != "/logs" {
		t.Fatalf("expected deduplicated log center root first, got %#v", menus[0])
	}
	if menus[1].Code != "access-log.list" || menus[2].Code != "app-log.list" {
		t.Fatalf("expected access-log then app-log leaves, got %#v", menus)
	}
}

func testMenuRegistry() *menu.Registry {
	registry := menu.NewRegistry()
	registerUserMenu(registry, "user")
	registry.Register(menu.Item{
		Code:  "profile.self",
		Title: "个人中心",
		Path:  "/profile",
		Icon:  "user",
		Order: 999,
	})

	return registry
}
