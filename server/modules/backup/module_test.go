package backup

import (
	"context"
	"database/sql"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/container"
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
	if err := registerMenu(registry); err != nil {
		t.Fatalf("register backup menu: %v", err)
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

func (testArtifactWriter) Verify(backupTaskInput) (moduleapi.CreateBackupInput, error) {
	return moduleapi.CreateBackupInput{}, nil
}
