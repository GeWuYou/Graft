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

func TestPostAuthLoginRequestBodyRequiresConcreteFieldsOnly(t *testing.T) {
	t.Parallel()

	var body PostAuthLoginJSONRequestBody
	if body.Username != "" || body.Password != "" {
		t.Fatalf("expected zero-value login body fields to stay empty strings, got %#v", body)
	}
}
