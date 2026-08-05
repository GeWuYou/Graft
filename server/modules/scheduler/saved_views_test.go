package scheduler

import (
	"encoding/json"
	"testing"

	"graft/server/internal/httpx"
)

func TestValidateSchedulerSavedView(t *testing.T) {
	t.Parallel()

	valid := httpx.SavedViewRequest{
		Name:           "Failed jobs",
		QueryState:     json.RawMessage(`{"keyword":"cleanup","jobKey":"audit.audit-log-retention-cleanup","status":"failed"}`),
		PageSize:       20,
		VisibleColumns: []string{"task", "job_key", "status", "last_run"},
	}
	if err := validateSchedulerSavedView(valid); err != nil {
		t.Fatalf("valid scheduler saved view rejected: %v", err)
	}

	for name, request := range map[string]httpx.SavedViewRequest{
		"unsupported status": {
			Name:       "Invalid status",
			QueryState: json.RawMessage(`{"status":"paused"}`),
			PageSize:   20,
		},
		"current page": {
			Name:       "Current page",
			QueryState: json.RawMessage(`{"page":2}`),
			PageSize:   20,
		},
		"unsupported page size": {
			Name:       "Unsupported size",
			QueryState: json.RawMessage(`{}`),
			PageSize:   25,
		},
		"unsupported column": {
			Name:           "Unsupported column",
			QueryState:     json.RawMessage(`{}`),
			PageSize:       20,
			VisibleColumns: []string{"operation"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateSchedulerSavedView(request); err == nil {
				t.Fatal("expected invalid scheduler saved view to be rejected")
			}
		})
	}
}
