package moduleruntime

import (
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
	moduleruntimelocales "graft/server/internal/moduleruntime/locales"
)

func TestRegisterMessagesUsesEmbeddedLocaleResources(t *testing.T) {
	localizer := mustNewModuleRuntimeTestLocalizer(t)

	if err := registerMessages(localizer); err != nil {
		t.Fatalf("register module runtime messages: %v", err)
	}

	assertRegisteredRuntimeMessage(t, localizer, i18n.LocaleZHCN, menuModulesRuntimeTitleKey, "模块运行时")
	assertRegisteredRuntimeMessage(t, localizer, i18n.LocaleENUS, menuModulesRuntimeTitleKey, "Module Runtime")
}

func assertRegisteredRuntimeMessage(
	t *testing.T,
	localizer *i18n.Service,
	locale i18n.LocaleTag,
	key string,
	expected string,
) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 {
		t.Fatalf("expected one module-runtime message for %s %q, got %#v", locale, key, matches)
	}
	if matches[0].Text != expected {
		t.Fatalf("expected module-runtime message %q for %s %q, got %#v", expected, locale, key, matches[0])
	}
}

func mustNewModuleRuntimeTestLocalizer(t *testing.T) *i18n.Service {
	t.Helper()

	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})

	resources, err := moduleruntimelocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load module-runtime locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register module-runtime locale resources: %v", err)
	}

	return localizer
}
