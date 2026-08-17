package backup

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/dashboard"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	backupcontract "graft/server/modules/backup/contract"
)

func TestModuleSpecBuildDoesNotRequireTaskServiceBeforeRegister(t *testing.T) {
	services := container.New()
	if err := services.RegisterSingleton((*sql.DB)(nil), func(container.Resolver) (any, error) { return &sql.DB{}, nil }); err != nil {
		t.Fatalf("register sql db: %v", err)
	}
	if err := services.RegisterSingleton((*config.Config)(nil), func(container.Resolver) (any, error) {
		return &config.Config{Backup: config.BackupConfig{ArtifactRoot: t.TempDir()}, Database: config.DatabaseConfig{URL: "database-url-from-test"}}, nil
	}); err != nil {
		t.Fatalf("register config: %v", err)
	}
	if _, err := NewModuleSpec().Build(module.BuildContext{Services: services}); err != nil {
		t.Fatalf("build backup module before task registration: %v", err)
	}
}

func TestModuleRegisterInjectsTaskServiceAfterTaskModuleRegistration(t *testing.T) {
	services := container.New()
	tasks := &backupTaskRuntimeStub{}
	if err := services.RegisterSingleton((*moduleapi.TaskService)(nil), func(container.Resolver) (any, error) { return tasks, nil }); err != nil {
		t.Fatalf("register task service: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.TaskRuntimeRegistrar)(nil), func(container.Resolver) (any, error) { return tasks, nil }); err != nil {
		t.Fatalf("register task registrar: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(container.Resolver) (any, error) { return backupAuthorizerStub{}, nil }); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}
	service := NewService(&serviceTestRepository{})
	service.setArtifactWriter(testArtifactWriter{})
	if err := NewModule(service).Register(&module.Context{Services: services, PermissionRegistry: permission.NewRegistry(), MenuRegistry: menu.NewRegistry()}); err != nil {
		t.Fatalf("register backup module: %v", err)
	}
	if service.tasks != tasks || len(tasks.executors) != 2 || len(tasks.authorizers) != 1 {
		t.Fatalf("expected TaskService injection and registrations, service=%#v executors=%d authorizers=%d", service, len(tasks.executors), len(tasks.authorizers))
	}
}

func TestRegisterMenuGroupsBackupUnderPlatformMaintenance(t *testing.T) {
	registry := menu.NewRegistry()
	menu.RegisterDomainGroups(registry)
	registry.Register(menu.Item{Code: "platform-maintenance", ParentCode: "domain.platform", Kind: menu.NodeKindGroup, TitleKey: "menu.platform.maintenance", Icon: "system-maintenance", Order: 103, Module: "update"})
	if err := registerMenu(registry); err != nil {
		t.Fatalf("register backup menu: %v", err)
	}
	if err := registry.Validate(); err != nil {
		t.Fatalf("validate combined platform-maintenance menu: %v", err)
	}
	items := map[string]menu.Item{}
	for _, item := range registry.Items() {
		items[item.Code] = item
	}
	backup := items["platform-backup.history"]
	if backup.ParentCode != "platform-maintenance" || backup.Path != backupcontract.BackupMenuPath || backup.Permission != backupcontract.BackupReadPermission || backup.Icon != "backup" {
		t.Fatalf("unexpected backup menu: %#v", backup)
	}
}

func TestBackupDashboardWidgetDeclaresPermissionAndLoadsLatestSummary(t *testing.T) {
	repository := &backupDashboardRepository{items: []moduleapi.BackupSummary{{
		ID: 7, Status: moduleapi.BackupStatusAvailable, RetainUntil: time.Now().UTC().Add(time.Hour),
	}}}
	service := NewService(repository)
	registry := dashboard.NewRegistry()
	if err := registerBackupDashboardWidget(&module.Context{DashboardRegistry: registry}, service); err != nil {
		t.Fatalf("register backup dashboard widget: %v", err)
	}
	definition, ok := registry.Get(backupHealthWidgetID)
	if !ok || definition.Type != dashboard.WidgetTypeHealth || definition.RouteLocation != backupcontract.BackupMenuPath || len(definition.RequiredPermissions) != 1 || definition.RequiredPermissions[0] != backupcontract.BackupReadPermission {
		t.Fatalf("unexpected backup dashboard definition: %#v", definition)
	}
	payload, err := definition.Loader.Load(context.Background(), dashboard.WidgetRequest{})
	if err != nil {
		t.Fatalf("load backup dashboard widget: %v", err)
	}
	summary := payload["summary"].(dashboard.HealthSummaryItem)
	if summary.Status != dashboard.HealthStatusHealthy || repository.calls != 1 || repository.limit != 1 || repository.offset != 0 {
		t.Fatalf("unexpected backup dashboard load: summary=%#v repository=%#v", summary, repository)
	}
}

func TestBackupDashboardWidgetDistinguishesLifecycleFacts(t *testing.T) {
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		items  []moduleapi.BackupSummary
		status dashboard.HealthStatus
		key    string
	}{
		{name: "available", items: []moduleapi.BackupSummary{{Status: moduleapi.BackupStatusAvailable, RetainUntil: now.Add(time.Hour)}}, status: dashboard.HealthStatusHealthy, key: "dashboard.widget.backupHealth.available.summary"},
		{name: "available without retention evidence", items: []moduleapi.BackupSummary{{Status: moduleapi.BackupStatusAvailable}}, status: dashboard.HealthStatusUnknown, key: "dashboard.widget.backupHealth.unknown.summary"},
		{name: "elapsed retention", items: []moduleapi.BackupSummary{{Status: moduleapi.BackupStatusAvailable, RetainUntil: now.Add(-time.Second)}}, status: dashboard.HealthStatusDegraded, key: "dashboard.widget.backupHealth.expired.summary"},
		{name: "expired", items: []moduleapi.BackupSummary{{Status: moduleapi.BackupStatusExpired}}, status: dashboard.HealthStatusDegraded, key: "dashboard.widget.backupHealth.expired.summary"},
		{name: "restored evidence", items: []moduleapi.BackupSummary{{Status: moduleapi.BackupStatusRestored}}, status: dashboard.HealthStatusUnknown, key: "dashboard.widget.backupHealth.restored.summary"},
		{name: "none", status: dashboard.HealthStatusUnknown, key: "dashboard.widget.backupHealth.none.summary"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := backupHealthPayload(test.items, now)
			summary := payload["summary"].(dashboard.HealthSummaryItem)
			if summary.Status != test.status || summary.LabelKey != test.key {
				t.Fatalf("unexpected lifecycle payload: %#v", payload)
			}
		})
	}
}

func TestBackupDashboardWidgetPropagatesReadAndRegistrationErrors(t *testing.T) {
	loadErr := errors.New("backup read failed")
	service := NewService(&backupDashboardRepository{err: loadErr})
	if _, err := loadBackupHealthWidget(context.Background(), service, time.Now().UTC()); !errors.Is(err, loadErr) {
		t.Fatalf("expected backup read error, got %v", err)
	}
	ctx := &module.Context{DashboardRegistry: dashboard.NewRegistry()}
	if err := registerBackupDashboardWidget(ctx, service); err != nil {
		t.Fatalf("register first backup widget: %v", err)
	}
	if err := registerBackupDashboardWidget(ctx, service); err == nil {
		t.Fatal("expected duplicate backup widget registration error")
	}
}

type backupDashboardRepository struct {
	serviceTestRepository
	items  []moduleapi.BackupSummary
	err    error
	calls  int
	limit  int
	offset int
}

func (r *backupDashboardRepository) ListSummaries(_ context.Context, limit, offset int) ([]moduleapi.BackupSummary, int64, error) {
	r.calls++
	r.limit = limit
	r.offset = offset
	if r.err != nil {
		return nil, 0, r.err
	}
	return r.items, int64(len(r.items)), nil
}

type backupTaskRuntimeStub struct {
	executors   []moduleapi.StageExecutor
	authorizers []moduleapi.TaskOwnerAuthorizer
}

func (*backupTaskRuntimeStub) Submit(context.Context, moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	return moduleapi.TaskReceipt{}, nil
}
func (*backupTaskRuntimeStub) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, nil
}
func (*backupTaskRuntimeStub) Cancel(context.Context, uint64) error             { return nil }
func (*backupTaskRuntimeStub) RetryStage(context.Context, uint64, uint64) error { return nil }
func (s *backupTaskRuntimeStub) RegisterStageExecutor(executor moduleapi.StageExecutor) error {
	s.executors = append(s.executors, executor)
	return nil
}
func (s *backupTaskRuntimeStub) RegisterTaskOwnerAuthorizer(authorizer moduleapi.TaskOwnerAuthorizer) error {
	s.authorizers = append(s.authorizers, authorizer)
	return nil
}

type backupAuthorizerStub struct{}

func (backupAuthorizerStub) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	return nil
}

type testArtifactWriter struct{}

func (testArtifactWriter) Create(context.Context, backupTaskInput) (moduleapi.CreateBackupInput, error) {
	return moduleapi.CreateBackupInput{}, nil
}

func (testArtifactWriter) Verify(context.Context, backupTaskInput) (moduleapi.CreateBackupInput, error) {
	return moduleapi.CreateBackupInput{}, nil
}
