package runtimetarget

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	store "graft/server/modules/runtime-target/store"
)

func TestControlPlaneBuilderTelemetryProviderReturnsLatestDurableBuilderAgentObservation(t *testing.T) {
	db := openBuilderTelemetryTestDB(t)
	repository := store.NewSQLRepository(db)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	if err := repository.RecordBuilderTelemetryObservation(context.Background(), testBuilderTelemetryObservation(now.Add(-time.Minute), now.Add(time.Minute))); err != nil {
		t.Fatalf("record builder telemetry: %v", err)
	}
	newer := testBuilderTelemetryObservation(now, now.Add(2*time.Minute))
	newer.Running = 2
	newer.AllocatableSlots = 1
	if err := repository.RecordBuilderTelemetryObservation(context.Background(), newer); err != nil {
		t.Fatalf("record newer builder telemetry: %v", err)
	}
	provider := controlPlaneBuilderTelemetryProvider{repository: repository, now: func() time.Time { return now }}
	snapshots, err := provider.ListBuilderTelemetry(context.Background(), []int64{7})
	if err != nil {
		t.Fatalf("list builder telemetry: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].BuilderScope != "builder-agent:7" || snapshots[0].Running != 2 || snapshots[0].AllocatableSlots != 1 {
		t.Fatalf("latest builder telemetry = %#v", snapshots)
	}
	admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7})
	if err != nil || !admitted {
		t.Fatalf("control-plane telemetry admission = (%t, %v), want (true, nil)", admitted, err)
	}
}

func TestControlPlaneBuilderTelemetryProviderFailsClosedForMissingExpiredAndUnsupportedObservations(t *testing.T) {
	db := openBuilderTelemetryTestDB(t)
	repository := store.NewSQLRepository(db)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	provider := controlPlaneBuilderTelemetryProvider{repository: repository, now: func() time.Time { return now }}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("missing observation admission = (%t, %v), want (false, nil)", admitted, err)
	}
	expired := testBuilderTelemetryObservation(now.Add(-2*time.Minute), now.Add(-time.Minute))
	if err := repository.RecordBuilderTelemetryObservation(context.Background(), expired); err != nil {
		t.Fatalf("record expired telemetry: %v", err)
	}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("expired observation admission = (%t, %v), want (false, nil)", admitted, err)
	}
	unsupported := testBuilderTelemetryObservation(now, now.Add(time.Minute))
	unsupported.UnsupportedDimensions = []string{"queue"}
	if err := repository.RecordBuilderTelemetryObservation(context.Background(), unsupported); err != nil {
		t.Fatalf("record unsupported telemetry: %v", err)
	}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("unsupported observation admission = (%t, %v), want (false, nil)", admitted, err)
	}
}

func testBuilderTelemetryObservation(observedAt, expiresAt time.Time) store.BuilderTelemetryObservation {
	return store.BuilderTelemetryObservation{TargetID: 7, BuilderScope: "builder-agent:7", ProviderID: "builder-agent", CapabilityProfile: "oci-build", CapabilityVersion: "v1", Available: true, Running: 1, Queued: 0, AllocatableSlots: 2, ObservedAt: observedAt, ExpiresAt: expiresAt, SourceRef: "control-plane:observation-7", Provenance: "builder-agent-control-plane", Integrity: "sha256:observation-7", UnsupportedDimensions: []string{"cache_state"}}
}

func openBuilderTelemetryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE runtime_target_builder_telemetry_observations (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, builder_scope TEXT NOT NULL, provider_id TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, available BOOLEAN NOT NULL, running_builds INTEGER NOT NULL, queued_builds INTEGER NOT NULL, allocatable_slots INTEGER NOT NULL, observed_at DATETIME NOT NULL, expires_at DATETIME NOT NULL, source_ref TEXT NOT NULL, provenance TEXT NOT NULL, integrity TEXT NOT NULL, unsupported_dimensions_json BLOB NOT NULL)`); err != nil {
		t.Fatalf("create builder telemetry observations: %v", err)
	}
	return db
}
