package project

import (
	"encoding/json"
	"testing"

	projectstore "graft/server/modules/project/store"
)

func TestValidateProjectListSavedViewRejectsUnknownConsumerFields(t *testing.T) {
	t.Parallel()
	valid, _ := json.Marshal(map[string]string{"source_type": "managed"})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Managed", QueryState: valid, PageSize: 20, VisibleColumns: []string{"row-select", "name", "runtime", "operation"}}); err != nil {
		t.Fatalf("valid project saved view rejected: %v", err)
	}
	invalid, _ := json.Marshal(map[string]int{"page": 3})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Paging", QueryState: invalid, PageSize: 20, VisibleColumns: []string{"name"}}); err == nil {
		t.Fatal("current page must not be persisted")
	}
	unsupportedFilter, _ := json.Marshal(map[string]string{"unsupported_filter": "x"})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Filter", QueryState: unsupportedFilter, PageSize: 20, VisibleColumns: []string{"name"}}); err == nil {
		t.Fatal("unknown project-list filter must be rejected")
	}
	if err := validateProjectListSavedView(savedViewRequest{Name: "Columns", QueryState: valid, PageSize: 20, VisibleColumns: []string{"unknown"}}); err == nil {
		t.Fatal("unknown project-list column must be rejected")
	}
	if err := validateProjectListSavedView(savedViewRequest{Name: "Legacy columns", QueryState: valid, PageSize: 20, VisibleColumns: []string{"runtime_status"}}); err == nil {
		t.Fatal("legacy project-list column must be rejected")
	}
}

func TestValidateProjectListSavedViewAcceptsApplicationFilters(t *testing.T) {
	t.Parallel()
	state, _ := json.Marshal(map[string]any{"keyword": "api", "deployment_adapter_kind": "compose", "runtime_target_id": 7, "provider": "docker", "runtime_status": "running", "source_type": "managed", "drift_status": "clean"})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Docker API", QueryState: state, PageSize: 50, VisibleColumns: []string{"applicationType", "runtimeTarget", "provider"}}); err != nil {
		t.Fatalf("valid application filters rejected: %v", err)
	}
}

func TestValidateProjectListSavedViewAcceptsOnlyProjectSortExpressions(t *testing.T) {
	t.Parallel()

	valid, _ := json.Marshal(map[string]any{"sort": []string{projectstore.ApplicationListSortCreatedAtAsc}})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Oldest", QueryState: valid, PageSize: 20, VisibleColumns: []string{"name"}}); err != nil {
		t.Fatalf("valid project sort rejected: %v", err)
	}
	invalid, _ := json.Marshal(map[string]any{"sort": []string{"updated_at:desc"}})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Unsafe", QueryState: invalid, PageSize: 20, VisibleColumns: []string{"name"}}); err == nil {
		t.Fatal("unknown project sort must be rejected")
	}
	duplicate, _ := json.Marshal(map[string]any{"sort": []string{projectstore.ApplicationListSortCreatedAtDesc, projectstore.ApplicationListSortCreatedAtAsc}})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Duplicate", QueryState: duplicate, PageSize: 20, VisibleColumns: []string{"name"}}); err == nil {
		t.Fatal("duplicate project sorts must be rejected")
	}
}
