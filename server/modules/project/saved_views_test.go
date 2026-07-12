package project

import (
	"encoding/json"
	"testing"
)

func TestValidateProjectListSavedViewRejectsUnknownConsumerFields(t *testing.T) {
	t.Parallel()
	valid, _ := json.Marshal(map[string]string{"source_kind": "managed"})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Managed", QueryState: valid, PageSize: 20, VisibleColumns: []string{"name", "runtime_status"}}); err != nil {
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
}

func TestValidateProjectListSavedViewAcceptsApplicationFilters(t *testing.T) {
	t.Parallel()
	state, _ := json.Marshal(map[string]any{"keyword": "api", "application_type": "compose", "runtime_target_id": 7, "provider": "docker", "runtime_status": "running", "source_kind": "managed", "drift_status": "clean"})
	if err := validateProjectListSavedView(savedViewRequest{Name: "Docker API", QueryState: state, PageSize: 50, VisibleColumns: []string{"application_type", "runtime_target", "provider"}}); err != nil {
		t.Fatalf("valid application filters rejected: %v", err)
	}
}
