package build

import (
	"context"
	"slices"
	"testing"

	containerdi "graft/server/internal/container"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	buildcontract "graft/server/modules/build/contract"
	buildstore "graft/server/modules/build/store"
)

type testBuildContexts struct{}

func (testBuildContexts) ResolveApplicationBuildContext(context.Context, string) (moduleapi.ApplicationBuildContext, error) {
	return moduleapi.ApplicationBuildContext{}, nil
}

type testBuildTasks struct{}

func (testBuildTasks) ReserveTask(context.Context, moduleapi.SubmitTaskInput) (moduleapi.TaskReservation, error) {
	return moduleapi.TaskReservation{TaskID: 1}, nil
}
func (testBuildTasks) ActivateTask(context.Context, moduleapi.TaskReservation) (moduleapi.TaskReceipt, error) {
	return moduleapi.TaskReceipt{TaskID: 1, Status: moduleapi.TaskStatusPending}, nil
}
func (testBuildTasks) DiscardTaskReservation(context.Context, moduleapi.TaskReservation) error {
	return nil
}

type testBuildDocker struct{}

func (testBuildDocker) BuildImage(context.Context, moduleapi.DockerImageBuildInput, moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	return moduleapi.DockerImageBuildResult{}, nil
}

type testBuildRepository struct{}

func (testBuildRepository) CreateJob(context.Context, buildstore.JobSnapshot) error { return nil }
func (testBuildRepository) GetJobByTaskID(context.Context, uint64) (buildstore.JobSnapshot, error) {
	return buildstore.JobSnapshot{}, nil
}
func (testBuildRepository) SettleDockerArtifact(context.Context, uint64, moduleapi.DockerImageBuildResult) error {
	return nil
}
func (testBuildRepository) ListJobs(context.Context, buildstore.ListQuery) (buildstore.ListResult, error) {
	return buildstore.ListResult{}, nil
}
func (testBuildRepository) GetJobByBuildID(context.Context, string) (buildstore.JobProjection, error) {
	return buildstore.JobProjection{}, buildstore.ErrNotFound
}

type testBuildRegistrar struct{}

func (testBuildRegistrar) RegisterStageExecutor(moduleapi.StageExecutor) error { return nil }
func (testBuildRegistrar) RegisterTaskOwnerAuthorizer(moduleapi.TaskOwnerAuthorizer) error {
	return nil
}

func TestModuleRegistersBuildPermissionsAndMenu(t *testing.T) {
	menuRegistry := menu.NewRegistry()
	menu.RegisterDomainGroups(menuRegistry)
	permissionRegistry := permission.NewRegistry()
	services := containerdi.New()
	for key, value := range map[any]any{
		(*moduleapi.ApplicationBuildContextResolver)(nil): testBuildContexts{},
		(*moduleapi.TaskReservationService)(nil):          testBuildTasks{},
		(*moduleapi.TaskRuntimeRegistrar)(nil):            testBuildRegistrar{},
		(*moduleapi.DockerImageBuildCapability)(nil):      testBuildDocker{},
	} {
		if err := services.RegisterSingleton(key, func(containerdi.Resolver) (any, error) { return value, nil }); err != nil {
			t.Fatalf("register test service: %v", err)
		}
	}

	if err := NewModule(testBuildRepository{}).Register(&module.Context{
		MenuRegistry:       menuRegistry,
		PermissionRegistry: permissionRegistry,
		Services:           services,
	}); err != nil {
		t.Fatalf("register build module: %v", err)
	}

	permissions := permissionRegistry.Items()
	assertBuildPermissionMetadata(t, permissions)

	if err := menuRegistry.Validate(); err != nil {
		t.Fatalf("validate build menu: %v", err)
	}
	menus := menuRegistry.Items()
	if !slices.ContainsFunc(menus, func(item menu.Item) bool {
		return item.Code == "build.jobs" &&
			item.ParentCode == "domain.build" &&
			item.Path == "/build/jobs" &&
			item.Permission == buildcontract.BuildReadPermission &&
			item.Module == moduleID
	}) {
		t.Fatalf("expected build jobs menu, got %#v", menus)
	}
}

func assertBuildPermissionMetadata(t *testing.T, permissions []permission.Item) {
	t.Helper()
	registered := make(map[string]permission.Item, len(permissions))
	for _, item := range permissions {
		registered[item.Code] = item
	}
	expected := []permission.Item{
		{Code: buildcontract.BuildReadPermission, Module: moduleID, Resource: "build", Action: "read", RiskLevel: permission.RiskLevelLow, RiskCategory: permission.RiskCategoryRead},
		{Code: buildcontract.BuildCreatePermission, Module: moduleID, Resource: "build", Action: "create", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryWrite},
		{Code: buildcontract.BuildCancelPermission, Module: moduleID, Resource: "build", Action: "cancel", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryDestructive},
		{Code: buildcontract.BuildRetryPermission, Module: moduleID, Resource: "build", Action: "retry", RiskLevel: permission.RiskLevelHigh, RiskCategory: permission.RiskCategoryWrite},
	}
	for _, want := range expected {
		got, ok := registered[want.Code]
		if !ok || got.Module != want.Module || got.Resource != want.Resource || got.Action != want.Action || got.RiskLevel != want.RiskLevel || got.RiskCategory != want.RiskCategory {
			t.Fatalf("expected build permission %#v, got %#v", want, got)
		}
	}
}
