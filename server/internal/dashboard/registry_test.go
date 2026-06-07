package dashboard

import (
	"context"
	"strings"
	"testing"
)

func TestRegistryRejectsDuplicateWidgetID(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	definition := testWidgetDefinition("core.module-runtime-health")
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register first widget: %v", err)
	}

	err := registry.Register(definition)
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate registration error, got %v", err)
	}
}

func TestRegistryValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		definition WidgetDefinition
	}{
		{name: "missing id", definition: WidgetDefinition{ModuleKey: "core", Type: WidgetTypeHealth, Loader: noopLoader()}},
		{name: "missing module", definition: WidgetDefinition{ID: "core.health", Type: WidgetTypeHealth, Loader: noopLoader()}},
		{name: "missing type", definition: WidgetDefinition{ID: "core.health", ModuleKey: "core", Loader: noopLoader()}},
		{name: "missing loader", definition: WidgetDefinition{ID: "core.health", ModuleKey: "core", Type: WidgetTypeHealth}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if err := NewRegistry().Register(testCase.definition); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestRegistryOrdersByOrderThenID(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, definition := range []WidgetDefinition{
		testWidgetDefinitionWithOrder("b.widget", 20),
		testWidgetDefinitionWithOrder("a.widget", 20),
		testWidgetDefinitionWithOrder("c.widget", 10),
	} {
		if err := registry.Register(definition); err != nil {
			t.Fatalf("register widget: %v", err)
		}
	}

	items := registry.Items()
	got := []string{items[0].ID, items[1].ID, items[2].ID}
	want := []string{"c.widget", "a.widget", "b.widget"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected order %v, got %v", want, got)
		}
	}
}

func testWidgetDefinition(id string) WidgetDefinition {
	return testWidgetDefinitionWithOrder(id, 10)
}

func testWidgetDefinitionWithOrder(id string, order int) WidgetDefinition {
	return WidgetDefinition{
		ID:        id,
		ModuleKey: "core",
		Type:      WidgetTypeHealth,
		Size:      WidgetSizeMedium,
		Order:     order,
		Loader:    noopLoader(),
	}
}

func noopLoader() WidgetLoader {
	return WidgetLoaderFunc(func(_ context.Context, _ WidgetRequest) (WidgetPayload, error) {
		return WidgetPayload{}, nil
	})
}
