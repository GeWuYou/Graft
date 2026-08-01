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
)

type testBuildContexts struct{}

func (testBuildContexts) ResolveApplicationBuildContext(context.Context, uint64) (moduleapi.ApplicationBuildContext, error) {
	return moduleapi.ApplicationBuildContext{}, nil
}

type testBuildTasks struct{}

func (testBuildTasks) Submit(context.Context, moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	return moduleapi.TaskReceipt{}, nil
}
func (testBuildTasks) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, nil
}
func (testBuildTasks) Cancel(context.Context, uint64) error             { return nil }
func (testBuildTasks) RetryStage(context.Context, uint64, uint64) error { return nil }

type testBuildDocker struct{}

func (testBuildDocker) BuildImage(context.Context, moduleapi.DockerImageBuildInput, moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	return moduleapi.DockerImageBuildResult{}, nil
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
		(*moduleapi.TaskService)(nil):                     testBuildTasks{},
		(*moduleapi.TaskRuntimeRegistrar)(nil):            testBuildRegistrar{},
		(*moduleapi.DockerImageBuildCapability)(nil):      testBuildDocker{},
	} {
		if err := services.RegisterSingleton(key, func(containerdi.Resolver) (any, error) { return value, nil }); err != nil {
			t.Fatalf("register test service: %v", err)
		}
	}

	if err := NewModule().Register(&module.Context{
		MenuRegistry:       menuRegistry,
		PermissionRegistry: permissionRegistry,
		Services:           services,
	}); err != nil {
		t.Fatalf("register build module: %v", err)
	}

	permissions := permissionRegistry.Items()
	for _, code := range []string{
		buildcontract.BuildReadPermission,
		buildcontract.BuildCreatePermission,
		buildcontract.BuildCancelPermission,
		buildcontract.BuildRetryPermission,
	} {
		if !slices.ContainsFunc(permissions, func(item permission.Item) bool {
			return item.Code == code && item.Module == moduleID
		}) {
			t.Fatalf("expected build permission %q, got %#v", code, permissions)
		}
	}

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
