package cli

import (
	"strings"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
)

func TestRegisterDevResetModuleLocaleResourcesMakesModuleMessagesAvailable(t *testing.T) {
	localizer, err := i18n.New(config.I18nConfig{
		DefaultLocale:    "en-US",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"en-US", "zh-CN"},
	})
	if err != nil {
		t.Fatalf("create localizer: %v", err)
	}

	if err := registerDevResetModuleLocaleResources(localizer); err != nil {
		t.Fatalf("register module locale resources: %v", err)
	}

	matches := localizer.RegisteredMessageResources(i18n.LocaleENUS, "rbac.permissionCatalog.buildRead.display")
	if len(matches) != 1 || matches[0].Text == "" {
		t.Fatalf("registered module message = %#v, want rbac user permission text", matches)
	}
}

func TestRegisterDevResetModuleLocaleResourcesPropagatesRegistrationError(t *testing.T) {
	previousResources := devResetEmbeddedLocaleResources
	devResetEmbeddedLocaleResources = func() []i18n.EmbeddedLocaleResource {
		return []i18n.EmbeddedLocaleResource{{
			Namespace: "rbac",
			Locale:    i18n.LocaleENUS,
			Source:    "modules/rbac/en-US.yaml",
			Data:      []byte("not: [valid"),
		}}
	}
	t.Cleanup(func() { devResetEmbeddedLocaleResources = previousResources })

	localizer, err := i18n.New(config.I18nConfig{
		DefaultLocale:    "en-US",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"en-US", "zh-CN"},
	})
	if err != nil {
		t.Fatalf("create localizer: %v", err)
	}

	err = registerDevResetModuleLocaleResources(localizer)
	if err == nil || !strings.Contains(err.Error(), "register module locale resources") {
		t.Fatalf("expected module resource registration error, got %v", err)
	}
}
