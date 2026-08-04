package update

import (
	"context"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	updatecontract "graft/server/modules/update/contract"
	updatelocales "graft/server/modules/update/locales"
)

type deploymentProfileRuntimeStub struct {
	current func(context.Context) moduleapi.DeploymentContext
}

type outboundDiagnosticRegistryStub struct {
	targets             []moduleapi.OutboundDiagnosticTarget
	connectivityTargets []moduleapi.ConnectivityTarget
}

func (s *outboundDiagnosticRegistryStub) RegisterOutboundDiagnosticTarget(target moduleapi.OutboundDiagnosticTarget) error {
	s.targets = append(s.targets, target)
	if connectivityTarget, ok := target.(moduleapi.ConnectivityTarget); ok {
		return s.RegisterConnectivityTarget(connectivityTarget)
	}
	return nil
}

func (s *outboundDiagnosticRegistryStub) OutboundDiagnosticTarget(name string) (moduleapi.OutboundDiagnosticTarget, bool) {
	for _, target := range s.targets {
		if target.Name() == name {
			return target, true
		}
	}
	return nil, false
}

func (s *outboundDiagnosticRegistryStub) OutboundDiagnosticTargets() []moduleapi.OutboundDiagnosticTarget {
	return append([]moduleapi.OutboundDiagnosticTarget(nil), s.targets...)
}

func (s *outboundDiagnosticRegistryStub) RegisterConnectivityTarget(target moduleapi.ConnectivityTarget) error {
	s.connectivityTargets = append(s.connectivityTargets, target)
	return nil
}

func (s *outboundDiagnosticRegistryStub) ConnectivityTarget(id moduleapi.ConnectivityTargetID) (moduleapi.ConnectivityTarget, bool) {
	for _, target := range s.connectivityTargets {
		if target.ConnectivityTargetDescriptor().ID == id {
			return target, true
		}
	}
	return nil, false
}

func (s *outboundDiagnosticRegistryStub) ConnectivityTargets() []moduleapi.ConnectivityTarget {
	return append([]moduleapi.ConnectivityTarget(nil), s.connectivityTargets...)
}

func (s *outboundDiagnosticRegistryStub) ConnectivityTargetDescriptors() []moduleapi.ConnectivityTargetDescriptor {
	items := make([]moduleapi.ConnectivityTargetDescriptor, 0, len(s.connectivityTargets))
	for _, target := range s.connectivityTargets {
		items = append(items, target.ConnectivityTargetDescriptor().Snapshot())
	}
	return items
}

type outboundNetworkConsumerRegistryStub struct {
	consumers []moduleapi.OutboundNetworkConsumer
}

func (s *outboundNetworkConsumerRegistryStub) RegisterOutboundNetworkConsumer(consumer moduleapi.OutboundNetworkConsumer) error {
	s.consumers = append(s.consumers, consumer)
	return nil
}

func (s *outboundNetworkConsumerRegistryStub) OutboundNetworkConsumers() []moduleapi.OutboundNetworkConsumer {
	return append([]moduleapi.OutboundNetworkConsumer(nil), s.consumers...)
}

func (s deploymentProfileRuntimeStub) Current(ctx context.Context) moduleapi.DeploymentContext {
	return s.current(ctx)
}

func (deploymentProfileRuntimeStub) Freeze(context.Context, moduleapi.DeploymentFreezeRequest) (moduleapi.DeploymentSnapshot, error) {
	return moduleapi.DeploymentSnapshot{}, nil
}

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

func TestConfigureDeploymentRuntimeBoundsAndCancelsProfileLookup(t *testing.T) {
	services := container.New()
	var profileCtx context.Context
	runtime := deploymentProfileRuntimeStub{current: func(ctx context.Context) moduleapi.DeploymentContext {
		profileCtx = ctx
		return moduleapi.NewDeploymentContext("compose", "docker_discovered", false, nil, nil)
	}}
	if err := services.RegisterSingleton((*moduleapi.DeploymentRuntime)(nil), func(container.Resolver) (any, error) { return runtime, nil }); err != nil {
		t.Fatalf("register deployment runtime: %v", err)
	}
	instance := NewModule(&memoryOperationStore{}, failureDiagnosticStoreStub{}, nil)
	if err := instance.configureDeploymentRuntime(&module.Context{Services: services, LifecycleContext: context.Background()}); err != nil {
		t.Fatalf("configure deployment runtime: %v", err)
	}

	instance.service.profile()
	if profileCtx == nil {
		t.Fatal("expected deployment runtime profile lookup")
	}
	deadline, ok := profileCtx.Deadline()
	if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > deploymentProfileTimeout {
		t.Fatalf("expected profile lookup deadline within %s, got %v", deploymentProfileTimeout, deadline)
	}
	select {
	case <-profileCtx.Done():
	default:
		t.Fatal("expected profile lookup context to be canceled after the callback returns")
	}
}

func TestConfigureDeploymentRuntimePropagatesLifecycleCancellation(t *testing.T) {
	services := container.New()
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	lifecycleCancel()
	var lookupErr error
	runtime := deploymentProfileRuntimeStub{current: func(ctx context.Context) moduleapi.DeploymentContext {
		lookupErr = ctx.Err()
		return moduleapi.NewDeploymentContext("compose", "docker_discovered", false, nil, nil)
	}}
	if err := services.RegisterSingleton((*moduleapi.DeploymentRuntime)(nil), func(container.Resolver) (any, error) { return runtime, nil }); err != nil {
		t.Fatalf("register deployment runtime: %v", err)
	}
	instance := NewModule(&memoryOperationStore{}, failureDiagnosticStoreStub{}, nil)
	if err := instance.configureDeploymentRuntime(&module.Context{Services: services, LifecycleContext: lifecycleCtx}); err != nil {
		t.Fatalf("configure deployment runtime: %v", err)
	}

	instance.service.profile()
	if lookupErr != context.Canceled {
		t.Fatalf("expected canceled lifecycle context to reach profile lookup, got %v", lookupErr)
	}
}

func TestConfigureOutboundNetworkRejectsMissingFactory(t *testing.T) {
	instance := NewModule(&memoryOperationStore{}, failureDiagnosticStoreStub{}, nil)
	if err := instance.configureOutboundNetwork(&module.Context{Services: container.New()}); err == nil {
		t.Fatal("expected missing outbound HTTP client factory to reject platform-update registration")
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
