package rbacopenapi

import "testing"

func TestGetPermissionsHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params GetPermissionsParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestGetRolesHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params GetRolesParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestGetRolePermissionsHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params GetRolePermissionsParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestGetUserRolesHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params GetUserRolesParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostUserRolesAssignHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostUserRolesAssignParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostUserRolesAssignRequestBodyKeepsRoleIDsOptional(t *testing.T) {
	t.Parallel()

	var body PostUserRolesAssignJSONRequestBody
	if body.RoleIds != nil {
		t.Fatalf("expected zero-value request body to keep role ids nil, got %#v", body)
	}
}
