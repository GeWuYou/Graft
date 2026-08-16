package module

import (
	"context"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"graft/server/internal/buildinfo"
)

type testModule struct{}

type lifecycleCallRecorder struct {
	register atomic.Int32
	boot     atomic.Int32
	shutdown atomic.Int32
}

func (r *lifecycleCallRecorder) Register(_ *Context) error {
	r.register.Add(1)
	return nil
}

func (r *lifecycleCallRecorder) Boot(_ *Context) error {
	r.boot.Add(1)
	return nil
}

func (r *lifecycleCallRecorder) Shutdown(_ *Context) error {
	r.shutdown.Add(1)
	return nil
}

type testRequiredCapability interface {
	Run(context.Context) error
}

type testExposedCapability interface {
	Read(context.Context) error
}

func (m testModule) Register(_ *Context) error { return nil }

func (m testModule) Boot(_ *Context) error { return nil }

func (m testModule) Shutdown(_ *Context) error { return nil }

// TestManagerRegistrationDoesNotInvokeModuleLifecycle 验证模块登记与描述符构造
// 只建立运行时元数据，不提前执行 Register、Boot 或 Shutdown，长期资源只能由
// Runtime 的显式生命周期阶段拥有。
func TestManagerRegistrationDoesNotInvokeModuleLifecycle(t *testing.T) {
	recorder := &lifecycleCallRecorder{}
	runtimeModule, err := (Spec{
		ID: "lifecycle-recorder",
		Builder: BuilderFunc(func(BuildContext) (Module, error) {
			return recorder, nil
		}),
	}).Build(BuildContext{})
	if err != nil {
		t.Fatalf("build module: %v", err)
	}

	manager := NewManager()
	if err := manager.RegisterModule(runtimeModule); err != nil {
		t.Fatalf("register module: %v", err)
	}
	if _, err := manager.Ordered(); err != nil {
		t.Fatalf("order modules: %v", err)
	}

	if got := recorder.register.Load(); got != 0 {
		t.Fatalf("expected Register to remain untouched during registration, got %d calls", got)
	}
	if got := recorder.boot.Load(); got != 0 {
		t.Fatalf("expected Boot to remain untouched during registration, got %d calls", got)
	}
	if got := recorder.shutdown.Load(); got != 0 {
		t.Fatalf("expected Shutdown to remain untouched during registration, got %d calls", got)
	}
}

// TestManagerOrderedUsesDependencyOrderAndAlphabeticalTieBreak 验证同一批模块在
// 不同注册顺序下仍会按依赖和字母序得到稳定的运行时顺序。
func TestManagerOrderedUsesDependencyOrderAndAlphabeticalTieBreak(t *testing.T) {
	manager := NewManager()
	input := []RuntimeModule{
		NewModule(Spec{ID: "user"}, testModule{}),
		NewModule(Spec{ID: "scheduler"}, testModule{}),
		NewModule(Spec{ID: "rbac", Dependencies: []string{"user"}}, testModule{}),
		NewModule(Spec{ID: "audit"}, testModule{}),
	}

	for _, current := range input {
		if err := manager.RegisterModule(current); err != nil {
			t.Fatalf("register module %s: %v", current.Name(), err)
		}
	}

	ordered, err := manager.Ordered()
	if err != nil {
		t.Fatalf("order modules: %v", err)
	}

	got := make([]string, 0, len(ordered))
	for _, current := range ordered {
		got = append(got, current.Name())
	}

	expected := []string{"audit", "scheduler", "user", "rbac"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

// TestManagerOrderedRejectsMissingDependency 验证缺失依赖会在排序阶段直接阻断。
func TestManagerOrderedRejectsMissingDependency(t *testing.T) {
	manager := NewManager()
	if err := manager.RegisterModule(NewModule(Spec{ID: "rbac", Dependencies: []string{"user"}}, testModule{})); err != nil {
		t.Fatalf("register module: %v", err)
	}

	_, err := manager.Ordered()
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
	if !strings.Contains(err.Error(), "depends on missing module user") {
		t.Fatalf("expected missing dependency error, got %v", err)
	}
}

// TestManagerOrderedRejectsDependencyCycle 验证循环依赖会被明确识别。
func TestManagerOrderedRejectsDependencyCycle(t *testing.T) {
	manager := NewManager()
	for _, current := range []RuntimeModule{
		NewModule(Spec{ID: "user", Dependencies: []string{"rbac"}}, testModule{}),
		NewModule(Spec{ID: "rbac", Dependencies: []string{"user"}}, testModule{}),
	} {
		if err := manager.RegisterModule(current); err != nil {
			t.Fatalf("register module %s: %v", current.Name(), err)
		}
	}

	_, err := manager.Ordered()
	if err == nil {
		t.Fatal("expected dependency cycle error")
	}
	if !strings.Contains(err.Error(), "module dependency cycle detected") {
		t.Fatalf("expected dependency cycle error, got %v", err)
	}
}

// TestOrderSpecsIsIndependentFromInputOrder 验证模块定义排序不依赖生成输入顺序。
func TestOrderSpecsIsIndependentFromInputOrder(t *testing.T) {
	input := []Spec{
		{ID: "scheduler", Builder: BuilderFunc(func(BuildContext) (Module, error) {
			return testModule{}, nil
		})},
		{ID: "rbac", Dependencies: []string{"user"}, Builder: BuilderFunc(func(BuildContext) (Module, error) {
			return testModule{}, nil
		})},
		{ID: "audit", Builder: BuilderFunc(func(BuildContext) (Module, error) {
			return testModule{}, nil
		})},
		{ID: "user", Builder: BuilderFunc(func(BuildContext) (Module, error) {
			return testModule{}, nil
		})},
	}

	ordered, err := OrderSpecs(input)
	if err != nil {
		t.Fatalf("order module specs: %v", err)
	}

	got := make([]string, 0, len(ordered))
	for _, current := range ordered {
		got = append(got, current.Name())
	}

	expected := []string{"audit", "scheduler", "user", "rbac"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

// TestSpecBuildWrapsCanonicalMetadata 验证模块定义构造出的运行时模块以
// 模块定义元数据为 canonical truth。
func TestSpecBuildWrapsCanonicalMetadata(t *testing.T) {
	descriptor := Spec{
		ID:           "rbac",
		Dependencies: []string{"user"},
		Builder: BuilderFunc(func(BuildContext) (Module, error) {
			return testModule{}, nil
		}),
	}

	built, err := descriptor.Build(BuildContext{})
	if err != nil {
		t.Fatalf("build descriptor: %v", err)
	}

	if built.Name() != "rbac" {
		t.Fatalf("expected descriptor name, got %q", built.Name())
	}
	if !reflect.DeepEqual(built.DependsOn(), []string{"user"}) {
		t.Fatalf("expected descriptor dependencies, got %v", built.DependsOn())
	}
}

func TestSpecValidatesTypedCompositionDeclarations(t *testing.T) {
	descriptor := Spec{
		ID:                   "runtime-target",
		RequiredCapabilities: []CapabilityDeclaration{{Key: TypedCapability[testRequiredCapability]()}},
		ExposedCapabilities:  []CapabilityDeclaration{{Key: TypedCapability[testExposedCapability]()}},
		ConfigurationOwner:   "runtime-target",
		Resources: []ResourceDeclaration{{
			Name: "summary collector", Owner: "runtime-target", Cleanup: "Service.Close",
		}},
		Builder: BuilderFunc(func(BuildContext) (Module, error) { return testModule{}, nil }),
	}

	if err := descriptor.Validate(); err != nil {
		t.Fatalf("validate composition declaration: %v", err)
	}

	if err := (Spec{
		ID:                   "duplicate",
		RequiredCapabilities: []CapabilityDeclaration{{Key: TypedCapability[testRequiredCapability]()}, {Key: TypedCapability[testRequiredCapability]()}},
		Builder:              descriptor.Builder,
	}).Validate(); err == nil {
		t.Fatal("expected duplicate capability declaration error")
	}

	for _, resource := range []ResourceDeclaration{
		{Owner: "runtime-target", Cleanup: "Service.Close"},
		{Name: "summary collector", Cleanup: "Service.Close"},
		{Name: "summary collector", Owner: "runtime-target"},
	} {
		descriptor.Resources = []ResourceDeclaration{resource}
		if err := descriptor.Validate(); err == nil {
			t.Fatalf("expected incomplete resource declaration error for %#v", resource)
		}
	}
}

func TestNewRuntimeMetadataPreservesOrderedDescriptorSnapshot(t *testing.T) {
	metadata := NewRuntimeMetadata([]Spec{
		{ID: "audit"},
		{ID: "user"},
		{ID: "rbac", Dependencies: []string{"user"}, ConfigurationOwner: "rbac", RequiredCapabilities: []CapabilityDeclaration{{Key: TypedCapability[testRequiredCapability]()}}, Resources: []ResourceDeclaration{{Name: "role cache", Owner: "rbac", Cleanup: "Close"}}},
	}, buildinfo.Info{
		Version:      "0.1.0",
		GitCommit:    "abc1234",
		BuildTimeUTC: "2026-06-22T00:00:00Z",
		GitTreeState: "clean",
	})

	got := metadata.OrderedModuleDescriptors()
	expected := []DescriptorSnapshot{
		{Name: "audit"},
		{Name: "user"},
		{Name: "rbac", DependsOn: []string{"user"}, ConfigurationOwner: "rbac", RequiredCapabilities: []CapabilityDeclaration{{Key: TypedCapability[testRequiredCapability]()}}, Resources: []ResourceDeclaration{{Name: "role cache", Owner: "rbac", Cleanup: "Close"}}},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	got[0].Name = "mutated"
	got[2].DependsOn[0] = "mutated"

	unchanged := metadata.OrderedModuleDescriptors()
	if !reflect.DeepEqual(unchanged, expected) {
		t.Fatalf("expected runtime metadata to remain immutable, got %v", unchanged)
	}

	if got := metadata.BuildInfo(); got.Version != "0.1.0" || got.GitCommit != "abc1234" {
		t.Fatalf("expected runtime metadata to expose build info snapshot, got %+v", got)
	}
}

func TestRuntimeMetadataBuildInfoNormalizesZeroValueSnapshot(t *testing.T) {
	var metadata RuntimeMetadata

	got := metadata.BuildInfo()
	if got.Version != "dev" {
		t.Fatalf("expected normalized version %q, got %q", "dev", got.Version)
	}
	if got.GitCommit != "unknown" {
		t.Fatalf("expected normalized git commit %q, got %q", "unknown", got.GitCommit)
	}
	if got.BuildTimeUTC != "unknown" {
		t.Fatalf("expected normalized build time %q, got %q", "unknown", got.BuildTimeUTC)
	}
	if got.GitTreeState != "unknown" {
		t.Fatalf("expected normalized tree state %q, got %q", "unknown", got.GitTreeState)
	}
}
