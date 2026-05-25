package authopenapi

import "testing"

func TestPostAuthLoginHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostAuthLoginParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestGetAuthBootstrapHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params GetAuthBootstrapParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostAuthRefreshHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostAuthRefreshParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostAuthLogoutHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostAuthLogoutParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestGetAuthSessionsHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params GetAuthSessionsParams
	if params.XGraftLocale != nil || params.XRequestId != nil || params.Limit != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers/query nil, got %#v", params)
	}
}

func TestPostAuthSessionsRevokeAllHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostAuthSessionsRevokeAllParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostAuthSessionsRevokeOthersHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostAuthSessionsRevokeOthersParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostAuthSessionRevokeRequiresSessionIDPathParam(t *testing.T) {
	t.Parallel()

	var params PostAuthSessionRevokeParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostAuthLoginRequestBodyRequiresConcreteFieldsOnly(t *testing.T) {
	t.Parallel()

	var body PostAuthLoginJSONRequestBody
	if body.Username != "" || body.Password != "" {
		t.Fatalf("expected zero-value login body fields to stay empty strings, got %#v", body)
	}
}

func TestGetAuthSessionsLimitRemainsGeneratedOptionalPointer(t *testing.T) {
	t.Parallel()

	limit := 10
	params := GetAuthSessionsParams{Limit: &limit}
	if params.Limit == nil || *params.Limit != 10 {
		t.Fatalf("expected generated session list limit to remain an optional int pointer, got %#v", params)
	}
}
