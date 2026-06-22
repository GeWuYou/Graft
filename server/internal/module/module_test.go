package module

import (
	"reflect"
	"strings"
	"testing"

	"graft/server/internal/buildinfo"
)

type testModule struct{}

func (m testModule) Register(_ *Context) error { return nil }

func (m testModule) Boot(_ *Context) error { return nil }

func (m testModule) Shutdown(_ *Context) error { return nil }

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

func TestNewRuntimeMetadataPreservesOrderedDescriptorSnapshot(t *testing.T) {
	metadata := NewRuntimeMetadata([]Spec{
		{ID: "audit"},
		{ID: "user"},
		{ID: "rbac", Dependencies: []string{"user"}},
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
		{Name: "rbac", DependsOn: []string{"user"}},
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
