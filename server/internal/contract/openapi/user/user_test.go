package useropenapi

import "testing"

func TestPostUsersHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostUsersParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostUserUpdateHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostUserUpdateParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostUserStatusHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostUserStatusParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostUserResetPasswordHeadersRemainOptional(t *testing.T) {
	t.Parallel()

	var params PostUserResetPasswordParams
	if params.XGraftLocale != nil || params.XRequestId != nil {
		t.Fatalf("expected zero-value generated params to keep optional headers nil, got %#v", params)
	}
}

func TestPostUsersRequestBodyRequiresConcreteFieldsOnly(t *testing.T) {
	t.Parallel()

	var body PostUsersJSONRequestBody
	if body.Username != "" || body.Display != "" || body.Password != "" {
		t.Fatalf("expected zero-value create body fields to stay empty strings, got %#v", body)
	}
}

func TestPostUserUpdateRequestBodyRequiresConcreteFieldsOnly(t *testing.T) {
	t.Parallel()

	var body PostUserUpdateJSONRequestBody
	if body.Username != "" || body.Display != "" {
		t.Fatalf("expected zero-value update body fields to stay empty strings, got %#v", body)
	}
}

func TestPostUserStatusRequestBodyKeepsGeneratedStatusEnum(t *testing.T) {
	t.Parallel()

	var body PostUserStatusJSONRequestBody
	if body.Status != "" {
		t.Fatalf("expected zero-value status body to stay empty, got %#v", body)
	}
}

func TestPostUserResetPasswordRequestBodyRequiresConcreteFieldOnly(t *testing.T) {
	t.Parallel()

	var body PostUserResetPasswordJSONRequestBody
	if body.NewPassword != "" {
		t.Fatalf("expected zero-value reset-password body field to stay empty string, got %#v", body)
	}
}
