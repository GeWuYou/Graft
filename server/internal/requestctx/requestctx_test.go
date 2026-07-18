package requestctx

import (
	"context"
	"testing"
)

func TestAuditContextRoundTrip(t *testing.T) {
	want := AuditContext{RequestID: "req-1", TraceID: "trace-1", Route: "/items/:id", Method: "GET"}
	got, ok := AuditContextFromContext(WithAuditContext(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("expected stored audit context %#v, got %#v, ok=%v", want, got, ok)
	}
}
