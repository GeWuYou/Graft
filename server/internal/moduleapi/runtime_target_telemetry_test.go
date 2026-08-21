package moduleapi_test

import (
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestBuilderTelemetrySnapshotFreshAtRequiresAuthoritativeFreshFacts(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	snapshot := moduleapi.BuilderTelemetrySnapshot{
		TargetID:         7,
		Available:        true,
		Running:          1,
		Queued:           0,
		AllocatableSlots: 3,
		ObservedAt:       now.Add(-time.Minute),
		ExpiresAt:        now.Add(time.Minute),
		SourceRef:        "runtime-target:7",
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
	incoherent.AllocatableSlots = -1
	if incoherent.FreshAt(now) {
		t.Fatal("expected an invalid allocatable-slot telemetry snapshot to fail closed")
	}
}

func TestBuilderTelemetrySnapshotConformantRequiresProviderEvidence(t *testing.T) {
	snapshot := moduleapi.BuilderTelemetrySnapshot{
		BuilderScope: "runtime-agent:1", ProviderID: "provider-test", CapabilityProfile: "oci-build", CapabilityVersion: "cap-v1",
		Provenance: "control-plane:1", Integrity: "sha256:evidence",
	}
	if !snapshot.Conformant() {
		t.Fatal("expected provider-backed evidence to be conformant")
	}
	snapshot.Integrity = ""
	if snapshot.Conformant() {
		t.Fatal("expected missing integrity evidence to fail closed")
	}
}

func TestBuilderTelemetrySnapshotDynamicPlacementConformanceRejectsUnsupportedRequiredDimension(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	snapshot := moduleapi.BuilderTelemetrySnapshot{
		TargetID: 7, BuilderScope: "runtime-agent:1", ProviderID: "provider-test", CapabilityProfile: "oci-build", CapabilityVersion: "cap-v1",
		Available: true, Running: 1, Queued: 0, AllocatableSlots: 3, ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		SourceRef: "control-plane:1", Provenance: "runtime-agent", Integrity: "sha256:evidence", UnsupportedDimensions: []string{"cache_state"},
	}
	if !snapshot.DynamicPlacementConformantAt(now) {
		t.Fatal("expected complete runtime-agent telemetry to be dynamically conformant")
	}
	snapshot.UnsupportedDimensions = append(snapshot.UnsupportedDimensions, "queue")
	if snapshot.DynamicPlacementConformantAt(now) {
		t.Fatal("expected an unsupported required dimension to fail closed")
	}
	snapshot.UnsupportedDimensions = []string{"future_dimension"}
	if snapshot.DynamicPlacementConformantAt(now) {
		t.Fatal("expected an unknown unsupported dimension to fail closed")
	}
}
