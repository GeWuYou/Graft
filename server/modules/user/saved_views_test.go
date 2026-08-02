package user

import (
	"encoding/json"
	"testing"

	"graft/server/internal/httpx"
)

func TestValidateUserSavedView(t *testing.T) {
	valid := httpx.SavedViewRequest{Name: "Enabled operators", QueryState: json.RawMessage(`{"keyword":"ops","status":"enabled","roleId":2}`), PageSize: 20, VisibleColumns: []string{"user", "roles"}}
	if err := validateUserSavedView(valid); err != nil {
		t.Fatalf("validate user saved view: %v", err)
	}
	invalidState := valid
	invalidState.QueryState = json.RawMessage(`{"status":"pending"}`)
	if err := validateUserSavedView(invalidState); err == nil {
		t.Fatal("expected unsupported user status to fail")
	}
	invalidColumn := valid
	invalidColumn.VisibleColumns = []string{"email"}
	if err := validateUserSavedView(invalidColumn); err == nil {
		t.Fatal("expected unsupported user column to fail")
	}
}
