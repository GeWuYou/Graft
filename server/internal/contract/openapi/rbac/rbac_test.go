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
