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

func TestLegacyBuilderTelemetryIngressIsDisabled(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate legacy key: %v", err)
	}
	ingress := controlPlaneBuilderTelemetryIngress{}
	registration := moduleapi.BuilderTelemetryAgentRegistration{TargetID: 7, AgentID: "agent:7", ProviderID: "docker", BuilderScope: "builder-agent:7", CapabilityProfile: "oci-build", CapabilityVersion: "v1", PublicKey: publicKey, Enabled: true}
	if err := ingress.ProvisionBuilderTelemetryAgent(context.Background(), registration); !errors.Is(err, store.ErrLegacyAgentTrustDisabled) {
		t.Fatalf("legacy registration error = %v, want %v", err, store.ErrLegacyAgentTrustDisabled)
	}
	if err := ingress.SubmitBuilderTelemetry(context.Background(), moduleapi.BuilderTelemetryReport{}); !errors.Is(err, store.ErrLegacyAgentTrustDisabled) {
		t.Fatalf("legacy report error = %v, want %v", err, store.ErrLegacyAgentTrustDisabled)
	}
}

func TestLegacyBuilderTelemetryBindingsRemainReadableButCannotAdmit(t *testing.T) {
	db := openBuilderTelemetryTestDB(t)
	repository := store.NewSQLRepository(db)
	if _, err := db.Exec(`INSERT INTO runtime_target_builder_telemetry_agents (runtime_target_id, agent_id, provider_id, builder_scope, capability_profile, capability_version, public_key, enabled) VALUES (7, 'agent:7', 'docker', 'builder-agent:7', 'oci-build', 'v1', X'0000000000000000000000000000000000000000000000000000000000000000', true)`); err != nil {
		t.Fatalf("insert legacy binding: %v", err)
	}
	legacy, err := repository.ReadLegacyBuilderTelemetryAgent(context.Background(), 7, "agent:7")
	if err != nil || legacy.AgentID != "agent:7" || !legacy.Enabled {
		t.Fatalf("legacy binding = %#v, err=%v", legacy, err)
	}
	if _, err := repository.GetBuilderTelemetryAgent(context.Background(), 7, "agent:7"); !errors.Is(err, store.ErrLegacyAgentTrustDisabled) {
		t.Fatalf("legacy trust lookup error = %v, want %v", err, store.ErrLegacyAgentTrustDisabled)
	}
	provider := controlPlaneBuilderTelemetryProvider{repository: repository}
	snapshots, err := provider.ListBuilderTelemetry(context.Background(), []int64{7})
	if err != nil || len(snapshots) != 0 {
		t.Fatalf("legacy telemetry snapshots = %#v, err=%v", snapshots, err)
	}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("legacy telemetry admission = (%t, %v), want (false, nil)", admitted, err)
	}
}

//nolint:gocyclo // 该场景固定验证完整账本投影与撤销后的动态准入拒绝。
func TestBuilderTelemetryProviderAdmitsOnlyFreshActiveDockerLedgerReceipt(t *testing.T) {
	db := openBuilderTelemetryTestDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO runtime_target_agent_identities (id, identity_id, runtime_target_id, agent_id, provider_id, builder_scope, capability_profile, capability_version) VALUES (1, 'identity-1', 7, 'agent-7', 'docker', 'builder-agent:7', 'oci-build', 'docker/v1')`); err != nil {
		t.Fatalf("insert agent identity: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO runtime_target_agent_generations (id, identity_id, generation, expires_at, status) VALUES (1, 1, 1, ?, 'active')`, now.Add(time.Hour)); err != nil {
		t.Fatalf("insert active generation: %v", err)
	}
	snapshotID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digest := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := db.Exec(`INSERT INTO runtime_target_agent_ledger_snapshots (generation_id, snapshot_id, snapshot_digest, sequence, builder_scope, provider_id, capability_profile, capability_version, affinity_key, available, running, queued, allocatable_slots, observed_at, expires_at, issued_at, consumed_at, receipt_available, receipt_observed_at, receipt_expires_at) VALUES (1, ?, ?, 1, 'builder-agent:7', 'docker', 'oci-build', 'docker/v1', 'docker-agent:agent-7', true, 1, 0, 2, ?, ?, ?, ?, true, ?, ?)`, snapshotID, digest, now.Add(-time.Second), now.Add(time.Minute), now.Add(-time.Second), now, now.Add(-time.Second), now.Add(time.Minute)); err != nil {
		t.Fatalf("insert consumed ledger receipt: %v", err)
	}
	provider := controlPlaneBuilderTelemetryProvider{repository: store.NewSQLRepository(db)}
	snapshots, err := provider.ListBuilderTelemetry(context.Background(), []int64{7})
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("ledger telemetry snapshots = %#v, err=%v", snapshots, err)
	}
	snapshot := snapshots[0]
	if snapshot.ProviderID != "docker" || snapshot.Provenance != "runtime-target-controlled-execution-ledger" || snapshot.SourceRef != "ledger:"+snapshotID || snapshot.Integrity != "sha256:"+digest || snapshot.Running != 1 || snapshot.AllocatableSlots != 2 {
		t.Fatalf("ledger telemetry snapshot = %#v", snapshot)
	}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || !admitted {
		t.Fatalf("active ledger admission = (%t, %v)", admitted, err)
	}
	if _, err := db.Exec(`UPDATE runtime_target_agent_generations SET status = 'revoked' WHERE id = 1`); err != nil {
		t.Fatalf("revoke generation: %v", err)
	}
	if admitted, err := provider.ConformBuilderTelemetry(context.Background(), []int64{7}); err != nil || admitted {
		t.Fatalf("revoked ledger admission = (%t, %v)", admitted, err)
	}
}

//nolint:dupl // 遥测测试需要独立声明代理与账本约束，保持迁移 parity 可见。
func openBuilderTelemetryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openRuntimeTargetTestDB(t)
	if _, err := db.Exec(`CREATE TABLE runtime_target_builder_telemetry_agents (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, agent_id TEXT NOT NULL, provider_id TEXT NOT NULL, builder_scope TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, public_key BLOB NOT NULL, last_sequence INTEGER NOT NULL DEFAULT 0, enabled BOOLEAN NOT NULL, updated_at DATETIME, UNIQUE(runtime_target_id, agent_id))`); err != nil {
		t.Fatalf("create builder telemetry agents: %v", err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX uq_runtime_target_builder_telemetry_agents_active_target ON runtime_target_builder_telemetry_agents (runtime_target_id) WHERE enabled = true`); err != nil {
		t.Fatalf("create active telemetry agent index: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE runtime_target_builder_telemetry_observations (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, agent_id TEXT NOT NULL, telemetry_sequence INTEGER NOT NULL CHECK (telemetry_sequence > 0), builder_scope TEXT NOT NULL, provider_id TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, affinity_key TEXT NOT NULL DEFAULT '', available BOOLEAN NOT NULL, running_builds INTEGER NOT NULL CHECK (running_builds >= 0), queued_builds INTEGER NOT NULL CHECK (queued_builds >= 0), allocatable_slots INTEGER NOT NULL CHECK (allocatable_slots >= 0), observed_at DATETIME NOT NULL, expires_at DATETIME NOT NULL, source_ref TEXT NOT NULL, provenance TEXT NOT NULL, integrity TEXT NOT NULL, unsupported_dimensions_json BLOB NOT NULL DEFAULT '[]', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, CONSTRAINT runtime_target_builder_telemetry_window_check CHECK (expires_at > observed_at), UNIQUE(runtime_target_id, agent_id, telemetry_sequence), FOREIGN KEY(runtime_target_id, agent_id) REFERENCES runtime_target_builder_telemetry_agents(runtime_target_id, agent_id))`); err != nil {
		t.Fatalf("create builder telemetry observations: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE runtime_target_builder_execution_ledgers (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, agent_id TEXT NOT NULL, slot_budget INTEGER NOT NULL CHECK (slot_budget > 0), queued_builds INTEGER NOT NULL DEFAULT 0 CHECK (queued_builds >= 0), running_builds INTEGER NOT NULL DEFAULT 0 CHECK (running_builds >= 0), telemetry_sequence INTEGER NOT NULL DEFAULT 0 CHECK (telemetry_sequence >= 0), created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, UNIQUE(runtime_target_id, agent_id), CHECK (running_builds <= slot_budget), FOREIGN KEY(runtime_target_id, agent_id) REFERENCES runtime_target_builder_telemetry_agents(runtime_target_id, agent_id))`); err != nil {
		t.Fatalf("create builder execution ledgers: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE runtime_target_agent_identities (id INTEGER PRIMARY KEY AUTOINCREMENT, identity_id TEXT NOT NULL, runtime_target_id INTEGER NOT NULL, agent_id TEXT NOT NULL, provider_id TEXT NOT NULL, builder_scope TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE runtime_target_agent_generations (id INTEGER PRIMARY KEY AUTOINCREMENT, identity_id INTEGER NOT NULL, generation INTEGER NOT NULL, expires_at DATETIME NOT NULL, status TEXT NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE runtime_target_agent_ledger_snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, generation_id INTEGER NOT NULL, snapshot_id TEXT NOT NULL, snapshot_digest TEXT NOT NULL, sequence INTEGER NOT NULL, builder_scope TEXT NOT NULL, provider_id TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, affinity_key TEXT NOT NULL, available BOOLEAN NOT NULL, running INTEGER NOT NULL, queued INTEGER NOT NULL, allocatable_slots INTEGER NOT NULL, observed_at DATETIME NOT NULL, expires_at DATETIME NOT NULL, issued_at DATETIME NOT NULL, consumed_at DATETIME, receipt_available BOOLEAN, receipt_observed_at DATETIME, receipt_expires_at DATETIME, receipt_diagnostic TEXT NOT NULL DEFAULT '', deleted_at INTEGER NOT NULL DEFAULT 0)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create active ledger projection schema: %v", err)
		}
	}
	return db
}
