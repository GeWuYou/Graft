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

	if envelope.Code != "ok" {
		t.Fatalf("expected generated envelope code field to stay addressable")
	}
	if createUser.Username != "alice" {
		t.Fatalf("expected generated create-user request to expose username field")
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

func stringPtr(value string) *string {
	return &value
}
