package scheduler

import (
	"math"
	"testing"
	"time"

	"graft/server/internal/cronx"
	schedulercore "graft/server/internal/scheduler"
)

func TestToScheduledTaskLastRunIncludesResultJSON(t *testing.T) {
	startedAt := time.Date(2026, 6, 7, 8, 0, 0, 0, time.UTC)
	mapped := toScheduledTaskLastRun(schedulercore.TaskRun{
		ID:          42,
		TriggerType: schedulercore.TriggerTypeManual,
		Status:      schedulercore.RunStatusSuccess,
		Result:      "deleted 3 rows",
		ResultJSON:  `{"summary":"deleted 3 rows","stage":"completed"}`,
		StartedAt:   startedAt,
	})

	if mapped.ResultJson == nil || *mapped.ResultJson != `{"summary":"deleted 3 rows","stage":"completed"}` {
		t.Fatalf("expected last run result_json to be mapped, got %#v", mapped)
	}
}

func TestToScheduledTaskActionResultUsesSerializableFallback(t *testing.T) {
	result := toScheduledTaskActionResult(schedulercore.JobActionResult{
		ActionKey: "dryRun",
		TaskKey:   "task",
		JobKey:    "job",
		Result: cronx.JobRunResult{
			Summary: "bad result",
			Metrics: map[string]any{
				"invalid": math.Inf(1),
			},
		},
	})

	if result.Result.Summary == nil || *result.Result.Summary != "job action result serialization failed" {
		t.Fatalf("expected serializable fallback result, got %#v", result.Result)
	}
	if result.ResultJson != `{"summary":"job action result serialization failed","stage":"failed","warnings":["job action result serialization failed"]}` {
		t.Fatalf("expected fallback result_json, got %s", result.ResultJson)
	}
}
