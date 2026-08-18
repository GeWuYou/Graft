package runtimetarget

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	contract "graft/server/modules/runtime-target/contract"
	store "graft/server/modules/runtime-target/store"
)

func TestRuntimeTargetDashboardWidgetDeclaresPermissionAndLoadsAggregate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := store.NewSQLRepository(db)
	registry := dashboard.NewRegistry()
	if err := registerRuntimeTargetDashboardWidget(&module.Context{DashboardRegistry: registry}, repository); err != nil {
		t.Fatalf("register runtime target dashboard widget: %v", err)
	}
	definition, ok := registry.Get(runtimeTargetSummaryWidgetID)
	assertRuntimeTargetDashboardDefinition(t, definition, ok)
	expectRuntimeTargetSummaryQuery(mock, 9, 7, 2, nil)
	payload, err := definition.Loader.Load(context.Background(), dashboard.WidgetRequest{})
	if err != nil {
		t.Fatalf("load runtime target dashboard widget: %v", err)
	}
	assertRuntimeTargetDashboardPayload(t, payload)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected runtime target aggregate queries: %v", err)
	}
}

func assertRuntimeTargetDashboardDefinition(t *testing.T, definition dashboard.WidgetDefinition, found bool) {
	t.Helper()
	if !found {
		t.Fatal("runtime target dashboard definition was not registered")
	}
	if definition.Type != dashboard.WidgetTypeStatGroup || definition.RouteLocation != contract.MenuPath {
		t.Fatalf("unexpected runtime target dashboard definition: %#v", definition)
	}
	if len(definition.RequiredPermissions) != 1 || definition.RequiredPermissions[0] != contract.ViewPermission {
		t.Fatalf("unexpected runtime target dashboard permission: %#v", definition.RequiredPermissions)
	}
}

func assertRuntimeTargetDashboardPayload(t *testing.T, payload dashboard.WidgetPayload) {
	t.Helper()
	items, ok := payload["items"].([]map[string]any)
	if !ok || len(items) != 3 {
		t.Fatalf("unexpected runtime target stats: %#v", payload["items"])
	}
	wantValues := []string{"9", "7", "2"}
	for index, want := range wantValues {
		if items[index]["value"] != want {
			t.Fatalf("runtime target stat %d value=%#v, want %q", index, items[index]["value"], want)
		}
	}
	if payload["state"] != string(dashboard.WidgetStateWarning) || payload["priority"] != string(dashboard.WidgetPriorityWarning) {
		t.Fatalf("unexpected runtime target attention state: %#v", payload)
	}
}

func TestRuntimeTargetDashboardWidgetPropagatesReadAndRegistrationErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := store.NewSQLRepository(db)
	loadErr := errors.New("summary read failed")
	expectRuntimeTargetSummaryQuery(mock, 0, 0, 0, loadErr)
	if _, err := loadRuntimeTargetSummaryWidget(context.Background(), repository); !errors.Is(err, loadErr) {
		t.Fatalf("expected summary read error, got %v", err)
	}
	ctx := &module.Context{DashboardRegistry: dashboard.NewRegistry()}
	if err := registerRuntimeTargetDashboardWidget(ctx, repository); err != nil {
		t.Fatalf("register first runtime target widget: %v", err)
	}
	if err := registerRuntimeTargetDashboardWidget(ctx, repository); err == nil {
		t.Fatal("expected duplicate runtime target widget registration error")
	}
}

func expectRuntimeTargetSummaryQuery(mock sqlmock.Sqlmock, total, healthy, unavailable int64, queryErr error) {
	query := `SELECT
COUNT(*),
COUNT(*) FILTER (WHERE availability = true),
COUNT(*) FILTER (WHERE availability = false)
FROM runtime_targets
WHERE deleted_at = 0`
	expectation := mock.ExpectQuery(regexp.QuoteMeta(query))
	if queryErr != nil {
		expectation.WillReturnError(queryErr)
		return
	}
	expectation.WillReturnRows(sqlmock.NewRows([]string{"total", "healthy", "unavailable"}).AddRow(total, healthy, unavailable))
}
