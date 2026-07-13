package audit

import (
	"encoding/json"
	"testing"

	"graft/server/internal/httpx"
)

func TestValidateAuditLogSavedViewRejectsScopeAndUnknownColumn(t *testing.T) {
	t.Parallel()
	valid, _ := json.Marshal(map[string]any{"result": "DENIED", "sort": []string{"created_at:desc"}})
	if err := validateAuditLogSavedView(httpx.SavedViewRequest{Name: "Denied", QueryState: valid, PageSize: 20, VisibleColumns: []string{"action", "result"}}); err != nil {
		t.Fatalf("valid saved view rejected: %v", err)
	}
	scope, _ := json.Marshal(map[string]any{"scope": "critical_security"})
	if err := validateAuditLogSavedView(httpx.SavedViewRequest{Name: "Scope", QueryState: scope, PageSize: 20}); err == nil {
		t.Fatal("drilldown scope must not be persisted")
	}
	if err := validateAuditLogSavedView(httpx.SavedViewRequest{Name: "Column", QueryState: valid, PageSize: 20, VisibleColumns: []string{"unknown"}}); err == nil {
		t.Fatal("unknown visible column must be rejected")
	}
}
