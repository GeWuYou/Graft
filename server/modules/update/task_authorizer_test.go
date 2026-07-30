package update

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	updatecontract "graft/server/modules/update/contract"
	updatelocales "graft/server/modules/update/locales"
)

func TestPlatformUpdateTaskOwnerAuthorizerMapsTaskActionsToUpdatePermissions(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		action     moduleapi.TaskOwnerAction
		permission string
	}{
		{name: "view", action: moduleapi.TaskOwnerActionView, permission: updatecontract.UpdateReadPermission.String()},
		{name: "cancel", action: moduleapi.TaskOwnerActionCancel, permission: updatecontract.UpdateManagePermission.String()},
		{name: "retry", action: moduleapi.TaskOwnerActionRetry, permission: updatecontract.UpdateManagePermission.String()},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			authorizer := &recordingUpdateTaskAuthorizer{}
			err := (platformUpdateTaskOwnerAuthorizer{authorizer: authorizer}).AuthorizeTaskOwner(context.Background(), &moduleapi.CurrentUser{ID: 7}, testCase.action, moduleapi.TaskOwner{Type: platformUpdateTaskOwnerType, ID: "update-1785218824462138825"})
			if err != nil {
				t.Fatalf("authorize update task owner: %v", err)
			}
			if authorizer.permission != testCase.permission || authorizer.actor == nil || authorizer.actor.ID != 7 {
				t.Fatalf("unexpected authorization request: %#v", authorizer)
			}
		})
	}
}

func TestPlatformUpdateTaskOwnerAuthorizerFailsClosedForInvalidTaskOwner(t *testing.T) {
	authorizer := platformUpdateTaskOwnerAuthorizer{authorizer: &recordingUpdateTaskAuthorizer{}}
	validOwner := moduleapi.TaskOwner{Type: platformUpdateTaskOwnerType, ID: "update-1785218824462138825"}
	for _, testCase := range []struct {
		name   string
		actor  *moduleapi.CurrentUser
		action moduleapi.TaskOwnerAction
		owner  moduleapi.TaskOwner
	}{
		{name: "missing actor", action: moduleapi.TaskOwnerActionView, owner: validOwner},
		{name: "wrong owner type", actor: &moduleapi.CurrentUser{ID: 7}, action: moduleapi.TaskOwnerActionView, owner: moduleapi.TaskOwner{Type: "application", ID: validOwner.ID}},
		{name: "invalid operation identity", actor: &moduleapi.CurrentUser{ID: 7}, action: moduleapi.TaskOwnerActionView, owner: moduleapi.TaskOwner{Type: platformUpdateTaskOwnerType, ID: "bad operation id"}},
		{name: "unsupported action", actor: &moduleapi.CurrentUser{ID: 7}, action: "delete", owner: validOwner},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := authorizer.AuthorizeTaskOwner(context.Background(), testCase.actor, testCase.action, testCase.owner); err == nil {
				t.Fatal("expected task owner authorization to fail")
			}
		})
	}

	if err := (platformUpdateTaskOwnerAuthorizer{}).AuthorizeTaskOwner(context.Background(), &moduleapi.CurrentUser{ID: 7}, moduleapi.TaskOwnerActionView, validOwner); !errors.Is(err, moduleapi.ErrUnauthenticated) {
		t.Fatalf("expected missing authorizer to fail as unauthenticated, got %v", err)
	}
}

func TestModuleRegisterRegistersPlatformUpdateTaskOwnerAuthorizer(t *testing.T) {
	services := container.New()
	runtime := &updateTaskRuntimeStub{}
	if err := services.RegisterSingleton((*moduleapi.TaskService)(nil), func(container.Resolver) (any, error) { return runtime, nil }); err != nil {
		t.Fatalf("register task service: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.TaskRuntimeRegistrar)(nil), func(container.Resolver) (any, error) { return runtime, nil }); err != nil {
		t.Fatalf("register task runtime registrar: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.BackupService)(nil), func(container.Resolver) (any, error) { return &stubBackupService{}, nil }); err != nil {
		t.Fatalf("register backup service: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(container.Resolver) (any, error) { return &recordingUpdateTaskAuthorizer{}, nil }); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}
	deploymentCandidate := moduleapi.NewDeploymentComposeCandidate("compose-test", "/opt/graft", []string{"/opt/graft/compose.yml"}, "graft", "high", nil)
	deploymentRuntime := deploymentRuntimeStub{context: moduleapi.NewDeploymentContext("compose", "explicit_config", false, []moduleapi.DeploymentComposeCandidate{deploymentCandidate}, nil)}
	if err := services.RegisterSingleton((*moduleapi.DeploymentRuntime)(nil), func(container.Resolver) (any, error) { return deploymentRuntime, nil }); err != nil {
		t.Fatalf("register deployment runtime: %v", err)
	}
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})
	resources, err := updatelocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load update locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register update locale resources: %v", err)
	}

	instance := NewModule(&memoryOperationStore{}, failureDiagnosticStoreStub{}, nil)
	if err := instance.Register(&module.Context{Services: services, I18n: localizer, PermissionRegistry: permission.NewRegistry(), MenuRegistry: menu.NewRegistry()}); err != nil {
		t.Fatalf("register platform update module: %v", err)
	}
	defer func() { _ = instance.Shutdown(nil) }()
	if len(runtime.authorizers) != 1 || runtime.authorizers[0].OwnerType() != platformUpdateTaskOwnerType {
		t.Fatalf("expected one platform update task owner authorizer, got %#v", runtime.authorizers)
	}
}

type recordingUpdateTaskAuthorizer struct {
	actor      *moduleapi.CurrentUser
	permission string
	err        error
}

func (a *recordingUpdateTaskAuthorizer) Authorize(_ context.Context, request moduleapi.RequestAuthContext, permission string) error {
	a.actor = request.User
	a.permission = permission
	return a.err
}

type updateTaskRuntimeStub struct {
	authorizers []moduleapi.TaskOwnerAuthorizer
}

func (*updateTaskRuntimeStub) Submit(context.Context, moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	return moduleapi.TaskReceipt{}, nil
}

func (*updateTaskRuntimeStub) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, nil
}

func (*updateTaskRuntimeStub) Cancel(context.Context, uint64) error                { return nil }
func (*updateTaskRuntimeStub) RetryStage(context.Context, uint64, uint64) error    { return nil }
func (*updateTaskRuntimeStub) RegisterStageExecutor(moduleapi.StageExecutor) error { return nil }

func (s *updateTaskRuntimeStub) RegisterTaskOwnerAuthorizer(authorizer moduleapi.TaskOwnerAuthorizer) error {
	s.authorizers = append(s.authorizers, authorizer)
	return nil
}

type failureDiagnosticStoreStub struct{}

func (failureDiagnosticStoreStub) CreateFailureDiagnostic(context.Context, FailureDiagnostic, uint64) error {
	return nil
}

func (failureDiagnosticStoreStub) GetFailureDiagnostic(context.Context, string) (FailureDiagnostic, error) {
	return FailureDiagnostic{}, errUpdateFailureDiagnosticNotFound
}

func (failureDiagnosticStoreStub) GetFailureDiagnosticByOperation(context.Context, string) (FailureDiagnostic, error) {
	return FailureDiagnostic{}, errUpdateFailureDiagnosticNotFound
}
