package logger

import (
	"encoding/json"
	"testing"

	"graft/server/internal/httpx"
)

func TestValidateAppLogSavedViewRejectsInvalidSorterAndCurrentPage(t *testing.T) {
	t.Parallel()
	valid, _ := json.Marshal(map[string]any{"severity": "error", "sort": []string{"occurred_at:desc"}})
	if err := validateAppLogSavedView(httpx.SavedViewRequest{Name: "Errors", QueryState: valid, PageSize: 20, VisibleColumns: []string{"occurred_at", "severity"}}); err != nil {
		t.Fatalf("valid saved view rejected: %v", err)
	}
	invalidSort, _ := json.Marshal(map[string]any{"sort": []string{"message:asc"}})
	if err := validateAppLogSavedView(httpx.SavedViewRequest{Name: "Invalid", QueryState: invalidSort, PageSize: 20}); err == nil {
		t.Fatal("unsupported sorter must be rejected")
	}
	currentPage, _ := json.Marshal(map[string]any{"page": 2})
	if err := validateAppLogSavedView(httpx.SavedViewRequest{Name: "Current page", QueryState: currentPage, PageSize: 20}); err == nil {
		t.Fatal("current page must not be persisted")
	}
}
