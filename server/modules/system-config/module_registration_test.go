// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
	systemconfigcontract "graft/server/modules/system-config/contract"
	systemconfiglocales "graft/server/modules/system-config/locales"
)

func TestRegisterMessagesUsesEmbeddedLocaleResources(t *testing.T) {
	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	resources, err := systemconfiglocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load system-config locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register system-config locale resources: %v", err)
	}

	if err := registerMessages(localizer); err != nil {
		t.Fatalf("register system-config messages: %v", err)
	}

	assertRegisteredSystemConfigMessage(t, localizer, i18n.LocaleZHCN, systemconfigcontract.SystemConfigMenuTitle.String(), "系统配置")
	assertRegisteredSystemConfigMessage(t, localizer, i18n.LocaleENUS, systemconfigcontract.SystemConfigMenuTitle.String(), "System Configuration")
	assertRegisteredSystemConfigMessage(t, localizer, i18n.LocaleZHCN, systemconfigcontract.SystemConfigNotFound.String(), "系统配置不存在")
	assertRegisteredSystemConfigMessage(t, localizer, i18n.LocaleENUS, systemconfigcontract.SystemConfigNotFound.String(), "System Configuration Not Found")
	assertRegisteredSystemConfigMessage(t, localizer, i18n.LocaleZHCN, systemconfigcontract.SystemConfigInvalidRequest.String(), "系统配置请求无效")
	assertRegisteredSystemConfigMessage(t, localizer, i18n.LocaleENUS, systemconfigcontract.SystemConfigInvalidRequest.String(), "Invalid System Configuration Request")
	assertRegisteredSystemConfigMessage(
		t,
		localizer,
		i18n.LocaleZHCN,
		"systemConfig.container.ops.container.environment.masked_copy_enabled.description",
		"开启后，已具备环境变量明文读取权限的用户可在环境变量列表、复制 .env 与原始 JSON 复制中获得敏感环境变量真实值；页面展示仍保持 *****。关闭后，包含敏感字段的复制操作会被禁止，不提供真实值。",
	)
}

func assertRegisteredSystemConfigMessage(
	t *testing.T,
	localizer *i18n.Service,
	locale i18n.LocaleTag,
	key string,
	expected string,
) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 {
		t.Fatalf("expected one system-config message for %s %q, got %#v", locale, key, matches)
	}
	if matches[0].Text != expected {
		t.Fatalf("expected system-config message %q for %s %q, got %#v", expected, locale, key, matches[0])
	}
}
