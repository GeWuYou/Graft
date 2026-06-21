package user

import (
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
	usercontract "graft/server/modules/user/contract"
	userlocales "graft/server/modules/user/locales"
)

func TestRegisterMessagesUsesEmbeddedLocaleResources(t *testing.T) {
	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	resources, err := userlocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load user locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register user locale resources: %v", err)
	}

	if err := registerMessages(localizer); err != nil {
		t.Fatalf("register user messages: %v", err)
	}

	assertRegisteredUserMessage(t, localizer, i18n.LocaleZHCN, usercontract.UserListMenuTitle.String(), "用户管理")
	assertRegisteredUserMessage(t, localizer, i18n.LocaleENUS, usercontract.UserListMenuTitle.String(), "User Management")
}

func assertRegisteredUserMessage(
	t *testing.T,
	localizer *i18n.Service,
	locale i18n.LocaleTag,
	key string,
	expected string,
) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 {
		t.Fatalf("expected one user message for %s %q, got %#v", locale, key, matches)
	}
	if matches[0].Text != expected {
		t.Fatalf("expected user message %q for %s %q, got %#v", expected, locale, key, matches[0])
	}
}
