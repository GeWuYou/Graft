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
	if err := validateProjectListSavedView(savedViewRequest{Name: "Columns", QueryState: valid, PageSize: 20, VisibleColumns: []string{"unknown"}}); err == nil {
		t.Fatal("unknown project-list column must be rejected")
	}
}
