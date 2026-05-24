package openapi

import (
	"encoding/json"
	"testing"
)

func TestGeneratedTypesExposeCoveredEnvelopeAndCreateUserShapes(t *testing.T) {
	var envelope APIEnvelope
	envelope.Code = "ok"
	envelope.MessageKey = stringPtr("user.created")

	var createUser PostUsersJSONRequestBody
	createUser.Username = "alice"
	createUser.Password = "secret"
	var updateUser PostUserUpdateJSONRequestBody
	updateUser.Username = "alice.ops"
	updateUser.Display = "Alice Ops"
	var updateStatus PostUserStatusJSONRequestBody
	updateStatus.Status = PostUserStatusJSONBodyStatusDisabled

	if envelope.Code != "ok" {
		t.Fatalf("expected generated envelope code field to stay addressable")
	}
	if createUser.Username != "alice" {
		t.Fatalf("expected generated create-user request to expose username field")
	}
	if updateUser.Username != "alice.ops" || updateUser.Display != "Alice Ops" {
		t.Fatalf("expected generated update-user request to expose username/display fields")
	}
	if updateStatus.Status != PostUserStatusJSONBodyStatusDisabled {
		t.Fatalf("expected generated user-status request to expose route-local status enum")
	}
}

func TestPostUsersJSONRequestBodyUnmarshalFollowsOpenAPIJSONShape(t *testing.T) {
	var body PostUsersJSONRequestBody
	if err := json.Unmarshal([]byte(`{"username":"alice","display":"Alice","password":"Password12345"}`), &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}

	if body.Username != "alice" || body.Display != "Alice" || body.Password != "Password12345" {
		t.Fatalf("unexpected unmarshaled request body: %#v", body)
	}
}

func TestPostUserUpdateJSONRequestBodyUnmarshalFollowsOpenAPIJSONShape(t *testing.T) {
	var body PostUserUpdateJSONRequestBody
	if err := json.Unmarshal([]byte(`{"username":"alice.ops","display":"Alice Ops"}`), &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}

	if body.Username != "alice.ops" || body.Display != "Alice Ops" {
		t.Fatalf("unexpected unmarshaled update-user request body: %#v", body)
	}
}

func TestPostUserStatusJSONRequestBodyUnmarshalFollowsOpenAPIJSONShape(t *testing.T) {
	var body PostUserStatusJSONRequestBody
	if err := json.Unmarshal([]byte(`{"status":"disabled"}`), &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}

	if body.Status != PostUserStatusJSONBodyStatusDisabled {
		t.Fatalf("unexpected unmarshaled user-status request body: %#v", body)
	}
}

func TestPostUserStatusJSONBodyStatusAliasMatchesGeneratedEnumMembers(t *testing.T) {
	cases := []PostUserStatusJSONBodyStatus{
		PostUserStatusJSONBodyStatusEnabled,
		PostUserStatusJSONBodyStatusDisabled,
	}

	for _, status := range cases {
		switch status {
		case PostUserStatusJSONBodyStatusEnabled, PostUserStatusJSONBodyStatusDisabled:
		default:
			t.Fatalf("unexpected generated user-status enum alias member: %q", status)
		}
	}
}

func stringPtr(value string) *string {
	return &value
}
