package rbacopenapi

import "testing"

func TestGetPermissionsHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params GetPermissionsParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}
