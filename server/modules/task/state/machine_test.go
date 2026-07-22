package state

import (
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestTaskTransitionsRespectLifecycle(t *testing.T) {
	t.Parallel()

	for _, transition := range []struct {
		from moduleapi.TaskStatus
		to   moduleapi.TaskStatus
	}{
		{moduleapi.TaskStatusPending, moduleapi.TaskStatusRunning},
		{moduleapi.TaskStatusScheduled, moduleapi.TaskStatusCancelled},
		{moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention},
		{moduleapi.TaskStatusNeedsAttention, moduleapi.TaskStatusRunning},
		{moduleapi.TaskStatusNeedsAttention, moduleapi.TaskStatusSuccess},
		{moduleapi.TaskStatusNeedsAttention, moduleapi.TaskStatusFailed},
	} {
		if err := ValidateTaskTransition(transition.from, transition.to); err != nil {
			t.Fatalf("expected task transition %s -> %s to be valid: %v", transition.from, transition.to, err)
		}
	}

	if err := ValidateTaskTransition(moduleapi.TaskStatusSuccess, moduleapi.TaskStatusRunning); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected terminal task transition rejection, got %v", err)
	}
}

func TestStageTransitionsProtectUnknownRecovery(t *testing.T) {
	t.Parallel()

	if err := ValidateStageTransition(moduleapi.StageStatusRunning, moduleapi.StageStatusUnknown); err != nil {
		t.Fatalf("expected interrupted stage transition to be valid: %v", err)
	}
	if err := ValidateStageTransition(moduleapi.StageStatusUnknown, moduleapi.StageStatusPending); err != nil {
		t.Fatalf("expected manual retry reset to be valid: %v", err)
	}
	if err := ValidateStageTransition(moduleapi.StageStatusUnknown, moduleapi.StageStatusSuccess); err != nil {
		t.Fatalf("expected receipt-backed stage settlement to be valid: %v", err)
	}
	if err := ValidateStageTransition(moduleapi.StageStatusSuccess, moduleapi.StageStatusPending); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected completed stage rewrite rejection, got %v", err)
	}
}
