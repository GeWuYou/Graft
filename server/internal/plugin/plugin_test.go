package plugin

import (
	"reflect"
	"strings"
	"testing"
)

type testPlugin struct {
}

func (p testPlugin) Register(_ *Context) error { return nil }

func (p testPlugin) Boot(_ *Context) error { return nil }

func (p testPlugin) Shutdown(_ *Context) error { return nil }

// TestManagerOrderedUsesDependencyOrderAndAlphabeticalTieBreak 验证同一批插件在
// 不同注册顺序下仍会按依赖和字母序得到稳定的运行时顺序。
func TestManagerOrderedUsesDependencyOrderAndAlphabeticalTieBreak(t *testing.T) {
	manager := NewManager()
	input := []Module{
		describedPlugin{moduleSpec: ModuleSpec{ID: "user"}, delegate: testPlugin{}},
		describedPlugin{moduleSpec: ModuleSpec{ID: "scheduler"}, delegate: testPlugin{}},
		describedPlugin{moduleSpec: ModuleSpec{ID: "rbac", Dependencies: []string{"user"}}, delegate: testPlugin{}},
		describedPlugin{moduleSpec: ModuleSpec{ID: "audit"}, delegate: testPlugin{}},
	}

	for _, current := range input {
		if err := manager.RegisterPlugin(current); err != nil {
			t.Fatalf("register plugin %s: %v", current.Name(), err)
		}
	}

	ordered, err := manager.Ordered()
	if err != nil {
		t.Fatalf("order plugins: %v", err)
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
	if err := manager.RegisterPlugin(describedPlugin{moduleSpec: ModuleSpec{ID: "rbac", Dependencies: []string{"user"}}, delegate: testPlugin{}}); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	_, err := manager.Ordered()
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
	if !strings.Contains(err.Error(), "depends on missing plugin user") {
		t.Fatalf("expected missing dependency error, got %v", err)
	}
}

// TestManagerOrderedRejectsDependencyCycle 验证循环依赖会被明确识别。
func TestManagerOrderedRejectsDependencyCycle(t *testing.T) {
	manager := NewManager()
	for _, current := range []Module{
		describedPlugin{moduleSpec: ModuleSpec{ID: "user", Dependencies: []string{"rbac"}}, delegate: testPlugin{}},
		describedPlugin{moduleSpec: ModuleSpec{ID: "rbac", Dependencies: []string{"user"}}, delegate: testPlugin{}},
	} {
		if err := manager.RegisterPlugin(current); err != nil {
			t.Fatalf("register plugin %s: %v", current.Name(), err)
		}
	}

	_, err := manager.Ordered()
	if err == nil {
		t.Fatal("expected dependency cycle error")
	}
	if !strings.Contains(err.Error(), "plugin dependency cycle detected") {
		t.Fatalf("expected dependency cycle error, got %v", err)
	}
}

// TestOrderModuleSpecsIsIndependentFromInputOrder 验证模块定义排序不依赖生成输入顺序。
func TestOrderModuleSpecsIsIndependentFromInputOrder(t *testing.T) {
	input := []ModuleSpec{
		{ID: "scheduler", Builder: BuilderFunc(func(BuildContext) (Plugin, error) {
			return testPlugin{}, nil
		})},
		{ID: "rbac", Dependencies: []string{"user"}, Builder: BuilderFunc(func(BuildContext) (Plugin, error) {
			return testPlugin{}, nil
		})},
		{ID: "audit", Builder: BuilderFunc(func(BuildContext) (Plugin, error) {
			return testPlugin{}, nil
		})},
		{ID: "user", Builder: BuilderFunc(func(BuildContext) (Plugin, error) {
			return testPlugin{}, nil
		})},
	}

	ordered, err := OrderModuleSpecs(input)
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

// TestModuleSpecBuildWrapsCanonicalMetadata 验证模块定义构造出的运行时插件以
// 模块定义元数据为 canonical truth。
func TestModuleSpecBuildWrapsCanonicalMetadata(t *testing.T) {
	descriptor := ModuleSpec{
		ID:           "rbac",
		Dependencies: []string{"user"},
		Builder: BuilderFunc(func(BuildContext) (Plugin, error) {
			return testPlugin{}, nil
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
	metadata := NewRuntimeMetadata([]ModuleSpec{
		{ID: "audit"},
		{ID: "user"},
		{ID: "rbac", Dependencies: []string{"user"}},
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
}
