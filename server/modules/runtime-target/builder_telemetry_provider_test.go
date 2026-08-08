package runtimetarget

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

func TestControlPlaneBuilderTelemetryProviderReturnsLatestDurableBuilderAgentObservation(t *testing.T) {
	db := openBuilderTelemetryTestDB(t)
	repository := store.NewSQLRepository(db)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	ingress, privateKey := provisionBuilderTelemetryAgent(t, repository)
	if err := ingress.SubmitBuilderTelemetry(context.Background(), signedBuilderTelemetryReport(t, privateKey, now.Add(-time.Minute), now.Add(time.Minute))); err != nil {
		t.Fatalf("submit builder telemetry: %v", err)
	}
	newer := signedBuilderTelemetryReport(t, privateKey, now, now.Add(2*time.Minute))
	newer.Running = 2
	newer.AllocatableSlots = 1
	newer = signBuilderTelemetryReport(t, privateKey, newer)
	if err := ingress.SubmitBuilderTelemetry(context.Background(), newer); err != nil {
		t.Fatalf("submit newer builder telemetry: %v", err)
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
	ingress, privateKey := provisionBuilderTelemetryAgent(t, repository)
	provider := controlPlaneBuilderTelemetryProvider{repository: repository, now: func() time.Time { return now }}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("missing observation admission = (%t, %v), want (false, nil)", admitted, err)
	}
	expired := signedBuilderTelemetryReport(t, privateKey, now.Add(-2*time.Minute), now.Add(-time.Minute))
	if err := ingress.SubmitBuilderTelemetry(context.Background(), expired); err != nil {
		t.Fatalf("submit expired telemetry: %v", err)
	}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("expired observation admission = (%t, %v), want (false, nil)", admitted, err)
	}
	unsupported := signedBuilderTelemetryReport(t, privateKey, now, now.Add(time.Minute))
	unsupported.UnsupportedDimensions = []string{"queue"}
	unsupported = signBuilderTelemetryReport(t, privateKey, unsupported)
	if err := ingress.SubmitBuilderTelemetry(context.Background(), unsupported); err != nil {
		t.Fatalf("submit unsupported telemetry: %v", err)
	}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("unsupported observation admission = (%t, %v), want (false, nil)", admitted, err)
	}
}

func TestControlPlaneBuilderTelemetryIngressRejectsUnregisteredAndInvalidAgentReports(t *testing.T) {
	db := openBuilderTelemetryTestDB(t)
	repository := store.NewSQLRepository(db)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate agent key: %v", err)
	}
	ingress := controlPlaneBuilderTelemetryIngress{repository: repository}
	report := signedBuilderTelemetryReport(t, privateKey, now, now.Add(time.Minute))
	if err := ingress.SubmitBuilderTelemetry(context.Background(), report); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unregistered report error = %v, want %v", err, store.ErrNotFound)
	}
	registeredIngress, _ := provisionBuilderTelemetryAgent(t, repository)
	report.Signature[0] ^= 0xff
	if err := registeredIngress.SubmitBuilderTelemetry(context.Background(), report); err == nil {
		t.Fatal("invalid agent signature unexpectedly entered telemetry ledger")
	}
}

func provisionBuilderTelemetryAgent(t *testing.T, repository *store.SQLRepository) (controlPlaneBuilderTelemetryIngress, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate agent key: %v", err)
	}
	ingress := controlPlaneBuilderTelemetryIngress{repository: repository}
	if err := ingress.ProvisionBuilderTelemetryAgent(context.Background(), moduleapi.BuilderTelemetryAgentRegistration{TargetID: 7, AgentID: "agent:7", PublicKey: publicKey, Enabled: true}); err != nil {
		t.Fatalf("provision telemetry agent: %v", err)
	}
	return ingress, privateKey
}

func signedBuilderTelemetryReport(t *testing.T, privateKey ed25519.PrivateKey, observedAt, expiresAt time.Time) moduleapi.BuilderTelemetryReport {
	t.Helper()
	return signBuilderTelemetryReport(t, privateKey, moduleapi.BuilderTelemetryReport{AgentID: "agent:7", TargetID: 7, BuilderScope: "builder-agent:7", ProviderID: "builder-agent", CapabilityProfile: "oci-build", CapabilityVersion: "v1", Available: true, Running: 1, Queued: 0, AllocatableSlots: 2, ObservedAt: observedAt, ExpiresAt: expiresAt, SourceRef: "control-plane:observation-7", Provenance: "builder-agent-control-plane", UnsupportedDimensions: []string{"cache_state"}})
}

func signBuilderTelemetryReport(t *testing.T, privateKey ed25519.PrivateKey, report moduleapi.BuilderTelemetryReport) moduleapi.BuilderTelemetryReport {
	t.Helper()
	payload, err := canonicalBuilderTelemetryReport(report)
	if err != nil {
		t.Fatalf("canonical telemetry report: %v", err)
	}
	report.Signature = ed25519.Sign(privateKey, payload)
	return report
}

func openBuilderTelemetryTestDB(t *testing.T) *sql.DB {
	db := openRuntimeTargetTestDB(t)
	if _, err := db.Exec(`CREATE TABLE runtime_target_builder_telemetry_observations (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, builder_scope TEXT NOT NULL, provider_id TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, available BOOLEAN NOT NULL, running_builds INTEGER NOT NULL, queued_builds INTEGER NOT NULL, allocatable_slots INTEGER NOT NULL, observed_at DATETIME NOT NULL, expires_at DATETIME NOT NULL, source_ref TEXT NOT NULL, provenance TEXT NOT NULL, integrity TEXT NOT NULL, unsupported_dimensions_json BLOB NOT NULL)`); err != nil {
		t.Fatalf("create builder telemetry observations: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE runtime_target_builder_telemetry_agents (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, agent_id TEXT NOT NULL, public_key BLOB NOT NULL, enabled BOOLEAN NOT NULL, UNIQUE(runtime_target_id, agent_id))`); err != nil {
		t.Fatalf("create builder telemetry agents: %v", err)
	}
	return db
}
