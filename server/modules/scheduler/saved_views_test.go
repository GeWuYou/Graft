package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	schedulercore "graft/server/internal/scheduler"
)

type schedulerSavedViewServiceStub struct{}

func (schedulerSavedViewServiceStub) List(context.Context, uint64, string) ([]moduleapi.SavedView, error) {
	return nil, nil
}

func (schedulerSavedViewServiceStub) Create(context.Context, moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	return moduleapi.SavedView{}, nil
}

func (schedulerSavedViewServiceStub) Update(context.Context, moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	return moduleapi.SavedView{}, nil
}

func (schedulerSavedViewServiceStub) Delete(context.Context, uint64, string, uint64) error {
	return nil
}

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

func TestRegisterSchedulerRoutesPropagatesMissingSavedViewService(t *testing.T) {
	ctx := newModuleTestContext()
	ctx.Services = container.New()

	err := registerSchedulerRoutesWithRuntime(ctx, moduleID, testAuthService{}, allowAllAuthorizer{}, func() (schedulercore.Runtime, error) {
		return &stopContextRecorderRuntime{}, nil
	})
	if !errors.Is(err, container.ErrServiceNotRegistered) {
		t.Fatalf("register scheduler routes error = %v, want missing saved-view service", err)
	}
}

var _ moduleapi.SavedViewService = schedulerSavedViewServiceStub{}
