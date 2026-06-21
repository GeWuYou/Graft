package testassert

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	errorcodecontract "graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
)

// DecodeSuccessData decodes and validates the shared HTTP success envelope.
func DecodeSuccessData[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var payload httpx.SuccessResponse[T]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode success envelope: %v", err)
	}
	if !payload.Success || payload.Code != errorcodecontract.OK.String() || payload.TraceID == "" {
		t.Fatalf("expected stable success envelope, got %#v", payload)
	}
	if recorder.Header().Get(httpx.RequestIDHeader) != payload.TraceID {
		t.Fatalf("expected response header trace id to match payload, got header=%q payload=%#v", recorder.Header().Get(httpx.RequestIDHeader), payload)
	}

	return payload.Data
}

// DecodeErrorResponse decodes the shared HTTP error envelope.
func DecodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) httpx.ErrorResponse {
	t.Helper()

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	return payload
}

// AssertErrorPayload checks the stable error message key, code, and locale fields.
func AssertErrorPayload(t *testing.T, payload httpx.ErrorResponse, messageKey string, code string, locale string) {
	t.Helper()

	if payload.MessageKey != messageKey || payload.Code != code || payload.Locale != locale {
		t.Fatalf("expected error payload key=%s code=%s locale=%s, got %#v", messageKey, code, locale, payload)
	}
}

// AssertContractErrorPayload checks error fields derived from a message contract key.
func AssertContractErrorPayload(t *testing.T, payload httpx.ErrorResponse, messageKey messagecontract.Key, locale string) {
	t.Helper()

	AssertErrorPayload(t, payload, messageKey.String(), errorcodecontract.FromMessageKey(messageKey).String(), locale)
}

// AssertErrorFieldDetail checks the conventional invalid field detail.
func AssertErrorFieldDetail(t *testing.T, payload httpx.ErrorResponse, field string) {
	t.Helper()

	if payload.Details["field"] != field {
		t.Fatalf("expected field detail %s, got %#v", field, payload)
	}
}
