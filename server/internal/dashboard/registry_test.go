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

func TestRegistryRejectsInvalidFrameworkFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		definition WidgetDefinition
		want       string
	}{
		{
			name: "invalid size",
			definition: func() WidgetDefinition {
				definition := testWidgetDefinition("core.invalid-size")
				definition.Size = WidgetSize("wide")
				return definition
			}(),
			want: "unsupported size",
		},
		{
			name: "invalid category",
			definition: func() WidgetDefinition {
				definition := testWidgetDefinition("core.invalid-category")
				definition.Category = WidgetCategory("operations")
				return definition
			}(),
			want: "unsupported category",
		},
		{
			name: "invalid priority",
			definition: func() WidgetDefinition {
				definition := testWidgetDefinition("core.invalid-priority")
				definition.Priority = WidgetPriority("urgent")
				return definition
			}(),
			want: "unsupported priority",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := NewRegistry().Register(testCase.definition)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected %q validation error, got %v", testCase.want, err)
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
	assertIDsInOrder(t, []string{items[0].ID, items[1].ID, items[2].ID}, []string{"c.widget", "a.widget", "b.widget"})
}

func TestRegistryNormalizesWidgetFrameworkFields(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register(WidgetDefinition{
		ID:            "core.normalized",
		ModuleKey:     "core",
		Type:          WidgetTypeHealth,
		Size:          WidgetSizeSmall,
		Category:      WidgetCategoryOperation,
		Priority:      WidgetPriorityWarning,
		RouteLocation: "/runtime",
		Action: WidgetAction{
			LabelKey: " dashboard.actions.details ",
			Label:    " View details ",
		},
		Loader: noopLoader(),
	}); err != nil {
		t.Fatalf("register widget: %v", err)
	}

	widget, ok := registry.Get("core.normalized")
	if !ok {
		t.Fatalf("expected widget to be registered")
	}
	if widget.Category != WidgetCategoryOperation || widget.Priority != WidgetPriorityWarning {
		t.Fatalf("unexpected framework fields: %#v", widget)
	}
	if widget.Action.LabelKey != "dashboard.actions.details" ||
		widget.Action.Label != "View details" ||
		widget.Action.Route != "/runtime" {
		t.Fatalf("expected action to default to route location, got %#v", widget.Action)
	}
}

func testWidgetDefinition(id string) WidgetDefinition {
	return testWidgetDefinitionWithOrder(id, 10)
}

func assertIDsInOrder(t *testing.T, got []string, want []string) {
	t.Helper()

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected order %v, got %v", want, got)
		}
	}
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
