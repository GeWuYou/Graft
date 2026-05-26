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

func TestPostRolesHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostRolesParams
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

func TestPostRoleUpdateHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostRoleUpdateParams
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

func TestPostUserRolesReplaceHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostUserRolesReplaceParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostRolePermissionsReplaceHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostRolePermissionsReplaceParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostUserRolesReplaceRequestBodyKeepsRoleIDsOptional(t *testing.T) {
	t.Parallel()

	var body PostUserRolesReplaceJSONRequestBody
	if body.RoleIds != nil {
		t.Fatalf("expected zero-value request body to keep role ids nil, got %#v", body)
	}
}

func TestPostRolePermissionsReplaceRequestBodyKeepsPermissionIDsOptional(t *testing.T) {
	t.Parallel()

	var body PostRolePermissionsReplaceJSONRequestBody
	if body.PermissionIds != nil {
		t.Fatalf("expected zero-value request body to keep permission ids nil, got %#v", body)
	}
}

func TestPostRolesRequestBodyKeepsDescriptionOptional(t *testing.T) {
	t.Parallel()

	var body PostRolesJSONRequestBody
	if body.Description != nil {
		t.Fatalf("expected zero-value request body to keep description nil, got %#v", body)
	}
}

func TestPostRoleUpdateRequestBodyKeepsDescriptionOptional(t *testing.T) {
	t.Parallel()

	var body PostRoleUpdateJSONRequestBody
	if body.Description != nil {
		t.Fatalf("expected zero-value request body to keep description nil, got %#v", body)
	}
}
