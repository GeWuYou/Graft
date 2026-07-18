package apperror

import (
	"errors"
	"testing"

	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
)

func TestWrapDescribeAndReportedMarkerPreserveErrorChain(t *testing.T) {
	cause := errors.New("database unavailable")
	descriptor := Descriptor{
		Kind:       KindInternal,
		Code:       errorcode.CommonInternalError,
		MessageKey: messagecontract.CommonInternalError,
	}

	err := MarkReported(Wrap(cause, descriptor))
	if !errors.Is(err, cause) {
		t.Fatal("expected reported application error to preserve cause matching")
	}
	if !IsReported(err) {
		t.Fatal("expected reported marker")
	}
	resolved, ok := Describe(err)
	if !ok || resolved != descriptor {
		t.Fatalf("expected descriptor %#v, got %#v, ok=%v", descriptor, resolved, ok)
	}
	if MarkReported(err) != err {
		t.Fatal("expected marking an already reported error to be idempotent")
	}
}

func TestNewUsesStableDescriptorWhenNoCauseExists(t *testing.T) {
	descriptor := Descriptor{
		Kind:       KindNotFound,
		Code:       errorcode.CommonNotFound,
		MessageKey: messagecontract.CommonNotFound,
	}
	err := New(descriptor)
	if err.Error() != messagecontract.CommonNotFound.String() {
		t.Fatalf("expected stable message key text, got %q", err.Error())
	}
	if resolved, ok := Describe(err); !ok || resolved != descriptor {
		t.Fatalf("expected descriptor %#v, got %#v, ok=%v", descriptor, resolved, ok)
	}
}
