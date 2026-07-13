package httpx

import (
	"encoding/json"
	"testing"
)

func TestValidateAccessLogSavedViewRejectsCurrentPageAndUnknownColumns(t *testing.T) {
	t.Parallel()
	valid, _ := json.Marshal(map[string]any{"status_group": []string{"5xx"}, "sort": []string{"started_at:desc"}})
	if err := validateAccessLogSavedView(SavedViewRequest{Name: "Server errors", QueryState: valid, PageSize: 20, VisibleColumns: []string{"started_at", "status_code"}}); err != nil {
		t.Fatalf("valid saved view rejected: %v", err)
	}
	currentPage, _ := json.Marshal(map[string]any{"page": 2})
	if err := validateAccessLogSavedView(SavedViewRequest{Name: "Current page", QueryState: currentPage, PageSize: 20}); err == nil {
		t.Fatal("current page must not be persisted")
	}
	if err := validateAccessLogSavedView(SavedViewRequest{Name: "Unknown column", QueryState: valid, PageSize: 20, VisibleColumns: []string{"unknown"}}); err == nil {
		t.Fatal("unknown visible column must be rejected")
	}
}
