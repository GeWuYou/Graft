package update

import (
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
	updatelocales "graft/server/modules/update/locales"
)

func TestRegisterMessagesIncludesPlatformUpdateScheduledTaskKeys(t *testing.T) {
	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	resources, err := updatelocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load platform-update locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register platform-update locale resources: %v", err)
	}
	if err := registerMessages(localizer); err != nil {
		t.Fatalf("register platform-update messages: %v", err)
	}

	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleZHCN, "scheduledTask.platformUpdateCheck.title", "检查平台更新")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleENUS, "scheduledTask.platformUpdateCheck.title", "Check Platform Updates")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleZHCN, "scheduledTask.platformUpdateCheck.description", "检查发布源中是否存在经过验证的较新平台版本。")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleENUS, "scheduledTask.platformUpdateCheck.description", "Checks the release source for a newer verified platform version.")
}

func assertRegisteredUpdateMessage(t *testing.T, localizer *i18n.Service, locale i18n.LocaleTag, key string, expected string) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 {
		t.Fatalf("expected one platform-update message for %s %q, got %#v", locale, key, matches)
	}
	if matches[0].Text != expected {
		t.Fatalf("expected platform-update message %q for %s %q, got %#v", expected, locale, key, matches[0])
	}
}
