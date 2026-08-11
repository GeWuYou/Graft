package cli

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"

	_ "github.com/mattn/go-sqlite3"
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

func TestRunDevResetAdminStopsBeforeResetWhenLocaleRegistrationFails(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	previousLoadConfig := devResetLoadConfig
	previousOpenDB := devResetOpenDB
	previousCloseDB := devResetCloseDB
	previousResources := devResetEmbeddedLocaleResources
	previousResetAdmin := devResetAdmin
	t.Cleanup(func() {
		devResetLoadConfig = previousLoadConfig
		devResetOpenDB = previousOpenDB
		devResetCloseDB = previousCloseDB
		devResetEmbeddedLocaleResources = previousResources
		devResetAdmin = previousResetAdmin
	})

	devResetLoadConfig = func() (*config.Config, error) {
		return &config.Config{App: config.AppConfig{Env: "local"}, I18n: config.I18nConfig{DefaultLocale: "en-US", FallbackLocale: "en-US", SupportedLocales: []string{"en-US", "zh-CN"}}}, nil
	}
	devResetOpenDB = func(config.DatabaseConfig) (*database.Resources, error) {
		return &database.Resources{SQL: db}, nil
	}
	devResetCloseDB = func(*database.Resources) error { return nil }
	devResetEmbeddedLocaleResources = func() []i18n.EmbeddedLocaleResource {
		return []i18n.EmbeddedLocaleResource{{Namespace: "rbac", Locale: i18n.LocaleENUS, Source: "modules/rbac/en-US.yaml", Data: []byte("not: [valid")}}
	}
	resetCalled := false
	devResetAdmin = func(context.Context, authstore.AuthRepository, moduleapi.UserIdentityProvider, *i18n.Service, moduleapi.RBACBootstrapService) error {
		resetCalled = true
		return nil
	}

	command := newDevResetAdminCommand()
	err = runDevResetAdmin(command)
	if err == nil || !strings.Contains(err.Error(), "register module locale resources") {
		t.Fatalf("expected locale registration error, got %v", err)
	}
	if resetCalled {
		t.Fatal("reset admin ran after locale registration failed")
	}
}
