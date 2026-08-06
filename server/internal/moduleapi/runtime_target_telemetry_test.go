package moduleapi_test

import (
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestBuilderTelemetrySnapshotFreshAtRequiresAuthoritativeFreshFacts(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	snapshot := moduleapi.BuilderTelemetrySnapshot{
		TargetID:   7,
		Available:  true,
		Capacity:   4,
		Running:    1,
		Queued:     0,
		ObservedAt: now.Add(-time.Minute),
		ExpiresAt:  now.Add(time.Minute),
		SourceRef:  "runtime-target:7",
	}
	if !snapshot.FreshAt(now) {
		t.Fatal("expected a coherent, unexpired telemetry snapshot to be usable")
	}

	stale := snapshot
	stale.ExpiresAt = now
	if stale.FreshAt(now) {
		t.Fatal("expected an expired telemetry snapshot to fail closed")
	}

	incoherent := snapshot
	incoherent.Running = incoherent.Capacity + 1
	if incoherent.FreshAt(now) {
		t.Fatal("expected an over-capacity telemetry snapshot to fail closed")
	}
}
