package update

import (
	"testing"

	"github.com/robfig/cron/v3"
	"graft/server/internal/config"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	updatecontract "graft/server/modules/update/contract"
	updatelocales "graft/server/modules/update/locales"
)

func TestRegisterMessagesIncludesPlatformUpdateScheduledTaskKeys(t *testing.T) {
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})
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
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleZHCN, "menu.platform.maintenance", "系统维护")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleENUS, "menu.platform.maintenance", "System Maintenance")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleZHCN, "scheduledTask.platformUpdateCheck.title", "检查平台更新")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleENUS, "scheduledTask.platformUpdateCheck.title", "Check Platform Updates")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleZHCN, "scheduledTask.platformUpdateCheck.description", "检查发布源中是否存在经过验证的较新平台版本。")
	assertRegisteredUpdateMessage(t, localizer, i18n.LocaleENUS, "scheduledTask.platformUpdateCheck.description", "Checks the release source for a newer verified platform version.")
}

func TestRegisterMessagesRejectsMissingScheduledTaskMessage(t *testing.T) {
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})
	resources := []i18n.EmbeddedLocaleResource{{Namespace: "update", Locale: i18n.LocaleZHCN, Source: "update/zh-CN.yaml", Data: []byte("menu.platform.maintenance: 系统维护\nmenu.platform.update: 更新中心\nscheduledTask.platformUpdateCheck.title: 检查平台更新\n")}, {Namespace: "update", Locale: i18n.LocaleENUS, Source: "update/en-US.yaml", Data: []byte("menu.platform.maintenance: Maintenance\nmenu.platform.update: Update Center\nscheduledTask.platformUpdateCheck.title: Check Platform Updates\n")}}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register synthetic platform-update locale resources: %v", err)
	}
	err := registerMessages(localizer)
	if err == nil || err.Error() != "platform-update locale resource missing scheduledTask.platformUpdateCheck.description for zh-CN" {
		t.Fatalf("expected missing scheduled task description error, got %v", err)
	}
}

func TestPlatformUpdateCheckScheduleUsesSeconds(t *testing.T) {
	if platformUpdateCheckSchedule != "0 0 */4 * * *" {
		t.Fatalf("expected platform update check every four hours, got %q", platformUpdateCheckSchedule)
	}
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(platformUpdateCheckSchedule); err != nil {
		t.Fatalf("parse platform update check schedule: %v", err)
	}
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

func TestRegisterMenuGroupsUpdateUnderPlatformMaintenance(t *testing.T) {
	registry := menu.NewRegistry()
	menu.RegisterDomainGroups(registry)
	if err := registerMenu(registry); err != nil {
		t.Fatalf("register menu: %v", err)
	}

	items := map[string]menu.Item{}
	for _, item := range registry.Items() {
		items[item.Code] = item
	}
	maintenance := items["platform-maintenance"]
	if maintenance.ParentCode != "domain.platform" || maintenance.Kind != menu.NodeKindGroup || maintenance.TitleKey != "menu.platform.maintenance" || maintenance.Icon != "system-maintenance" {
		t.Fatalf("unexpected maintenance group: %#v", maintenance)
	}
	update := items["platform-update.center"]
	if update.ParentCode != maintenance.Code || update.Path != updatecontract.UpdateMenuPath || update.Permission != updatecontract.UpdateReadPermission.String() || update.Icon != "platform-update" {
		t.Fatalf("unexpected update entry: %#v", update)
	}
}
