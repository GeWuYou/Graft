package rbac

import (
	"encoding/json"
	"testing"

	"graft/server/internal/httpx"
)

func TestValidateRBACSavedViews(t *testing.T) {
	role := httpx.SavedViewRequest{Name: "Builtin", QueryState: json.RawMessage(`{"type":"builtin"}`), PageSize: 20, VisibleColumns: []string{"role", "builtin"}}
	if err := validateRBACSavedView(role, roleSavedViewDefinition); err != nil {
		t.Fatalf("validate role saved view: %v", err)
	}
	permission := httpx.SavedViewRequest{Name: "Security", QueryState: json.RawMessage(`{"keyword":"read","module":"rbac"}`), PageSize: 50, VisibleColumns: []string{"permission", "module"}}
	if err := validateRBACSavedView(permission, permissionSavedViewDefinition); err != nil {
		t.Fatalf("validate permission saved view: %v", err)
	}
	invalid := permission
	invalid.QueryState = json.RawMessage(`{"unknown":true}`)
	if err := validateRBACSavedView(invalid, permissionSavedViewDefinition); err == nil {
		t.Fatal("expected unknown permission query field to fail")
	}
}
