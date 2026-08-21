package container

import (
	"errors"
	"net/http"
	"testing"
)

func TestStatusForErrorClassifiesRuntimeFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "disabled policy", err: errRuntimeDisabled, want: http.StatusForbidden},
		{name: "socket missing", err: errRuntimeSocketMissing, want: http.StatusServiceUnavailable},
		{name: "permission denied", err: errRuntimePermissionDenied, want: http.StatusForbidden},
		{name: "daemon unavailable", err: errRuntimeDaemonUnavailable, want: http.StatusServiceUnavailable},
		{name: "timeout", err: errContainerRuntimeTimeout, want: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := statusForError(test.err); got != test.want {
				t.Fatalf("statusForError(%v) = %d, want %d", test.err, got, test.want)
			}
		})
	}
}

func TestRuntimeErrorsKeepStableMessageKeys(t *testing.T) {
	t.Parallel()
	if got := messageKeyForError(errors.Join(errRuntimeDaemonUnavailable, errors.New("socket"))); got != "ops.container.error.runtimeUnavailable" {
		t.Fatalf("unexpected runtime message key: %s", got)
	}
}
