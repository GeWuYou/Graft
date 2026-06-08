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

func TestRegistryRejectsDuplicateQuickLinkID(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	definition := testQuickLinkDefinition("core.runtime")
	if err := registry.RegisterQuickLink(definition); err != nil {
		t.Fatalf("register first quick link: %v", err)
	}

	err := registry.RegisterQuickLink(definition)
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

func TestRegistryValidatesQuickLinkRequiredFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		definition QuickLinkDefinition
	}{
		{name: "missing id", definition: QuickLinkDefinition{ModuleKey: "core", Title: "Runtime", RouteLocation: "/modules/runtime"}},
		{name: "missing module", definition: QuickLinkDefinition{ID: "core.runtime", Title: "Runtime", RouteLocation: "/modules/runtime"}},
		{name: "missing title", definition: QuickLinkDefinition{ID: "core.runtime", ModuleKey: "core", RouteLocation: "/modules/runtime"}},
		{name: "missing route", definition: QuickLinkDefinition{ID: "core.runtime", ModuleKey: "core", Title: "Runtime"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if err := NewRegistry().RegisterQuickLink(testCase.definition); err == nil {
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
	assertIDsInOrder(t, []string{items[0].ID, items[1].ID, items[2].ID}, []string{"c.widget", "a.widget", "b.widget"})
}

func TestRegistryOrdersQuickLinksByOrderThenID(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, definition := range []QuickLinkDefinition{
		testQuickLinkDefinitionWithOrder("b.link", 20),
		testQuickLinkDefinitionWithOrder("a.link", 20),
		testQuickLinkDefinitionWithOrder("c.link", 10),
	} {
		if err := registry.RegisterQuickLink(definition); err != nil {
			t.Fatalf("register quick link: %v", err)
		}
	}

	items := registry.QuickLinks()
	assertIDsInOrder(t, []string{items[0].ID, items[1].ID, items[2].ID}, []string{"c.link", "a.link", "b.link"})
}

func TestRegistryKeepsQuickLinksAndWidgetsIndependent(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id := "core.same-id"
	if err := registry.Register(testWidgetDefinition(id)); err != nil {
		t.Fatalf("register widget: %v", err)
	}
	if err := registry.RegisterQuickLink(testQuickLinkDefinition(id)); err != nil {
		t.Fatalf("register quick link with widget id: %v", err)
	}

	if _, ok := registry.Get(id); !ok {
		t.Fatalf("expected widget to remain registered")
	}
	if _, ok := registry.GetQuickLink(id); !ok {
		t.Fatalf("expected quick link to remain registered")
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

func testQuickLinkDefinition(id string) QuickLinkDefinition {
	return testQuickLinkDefinitionWithOrder(id, 10)
}

func testQuickLinkDefinitionWithOrder(id string, order int) QuickLinkDefinition {
	return QuickLinkDefinition{
		ID:            id,
		ModuleKey:     "core",
		Title:         "Runtime",
		RouteLocation: "/modules/runtime",
		Order:         order,
	}
}
