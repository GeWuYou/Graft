package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

//nolint:gocognit,gocyclo,cyclop // 同一场景需连续验证签发、回执重放、准入投影和撤销边界。
func TestAgentLedgerSnapshotAndReceiptBindActiveCertificate(t *testing.T) {
	db := openAgentLedgerTestDB(t)
	repository := NewSQLRepository(db)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	identity := testAgentTrustIdentity()
	if _, err := repository.CreatePendingAgentTrustGeneration(context.Background(), identity, testPendingAgentTrustGeneration(1, now.Add(time.Hour))); err != nil {
		t.Fatalf("create generation: %v", err)
	}
	if err := repository.ActivateAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 1, "vault", "serial-1", "sha256:key-1", 0, now); err != nil {
		t.Fatalf("activate generation: %v", err)
	}
	if err := repository.EnsureBuilderAgentLedger(context.Background(), identity.TargetID, identity.AgentID, 2); err != nil {
		t.Fatalf("ensure execution ledger: %v", err)
	}
	mtlsIdentity := AgentLedgerIdentity{IdentityID: identity.IdentityID, TargetID: identity.TargetID, AgentID: identity.AgentID, Generation: 1, CertificateSerial: "serial-1", PublicKeyFingerprint: "sha256:key-1"}
	snapshot, err := repository.IssueAgentLedgerSnapshot(context.Background(), mtlsIdentity, testAgentLedgerSnapshotID(), now, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("issue ledger snapshot: %v", err)
	}
	if snapshot.Sequence != 1 || snapshot.SnapshotDigest == "" || !snapshot.Available {
		t.Fatalf("unexpected issued snapshot %#v", snapshot)
	}
	receipt := AgentTelemetryReceiptInput{SnapshotID: snapshot.SnapshotID, SnapshotDigest: snapshot.SnapshotDigest, ObservedAt: now, ExpiresAt: now.Add(3 * time.Minute), Available: true, ImplementationVersion: "v1"}
	if err := repository.RecordAgentTelemetryReceipt(context.Background(), mtlsIdentity, receipt, now.Add(time.Second)); err != nil {
		t.Fatalf("record receipt: %v", err)
	}
	admitted, err := repository.ListActiveDockerAgentLedgerSnapshots(context.Background(), []int64{identity.TargetID}, now.Add(2*time.Second))
	if err != nil || len(admitted) != 1 {
		t.Fatalf("admitted ledger snapshots = %#v, err=%v", admitted, err)
	}
	if admitted[0].TargetID != identity.TargetID || admitted[0].AgentID != identity.AgentID || admitted[0].ProviderID != "docker" || admitted[0].Sequence != snapshot.Sequence || !admitted[0].Available {
		t.Fatalf("admitted ledger snapshot = %#v", admitted[0])
	}
	if err := repository.RecordAgentTelemetryReceipt(context.Background(), mtlsIdentity, receipt, now.Add(2*time.Second)); err != nil {
		t.Fatalf("retry exact receipt: %v", err)
	}
	if err := repository.RecordAgentTelemetryReceipt(context.Background(), mtlsIdentity, receipt, now.Add(2*time.Minute)); !errors.Is(err, ErrAgentTrustNotActive) {
		t.Fatalf("expired snapshot retry = %v, want rejection", err)
	}
	receipt.Available = false
	if err := repository.RecordAgentTelemetryReceipt(context.Background(), mtlsIdentity, receipt, now.Add(3*time.Second)); !errors.Is(err, ErrAgentTrustNotActive) {
		t.Fatalf("changed receipt = %v, want rejection", err)
	}
	wrongCertificate := mtlsIdentity
	wrongCertificate.CertificateSerial = "forged"
	if _, err := repository.IssueAgentLedgerSnapshot(context.Background(), wrongCertificate, testAgentLedgerSnapshotID2(), now, now.Add(time.Minute)); !errors.Is(err, ErrAgentTrustNotActive) {
		t.Fatalf("wrong certificate = %v, want rejection", err)
	}
	if err := repository.RevokeAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 1, "operator_revoke", 0, now.Add(4*time.Second)); err != nil {
		t.Fatalf("revoke active generation: %v", err)
	}
	if admitted, err := repository.ListActiveDockerAgentLedgerSnapshots(context.Background(), []int64{identity.TargetID}, now.Add(5*time.Second)); err != nil || len(admitted) != 0 {
		t.Fatalf("revoked generation admission = %#v, err=%v", admitted, err)
	}
	if _, err := repository.IssueAgentLedgerSnapshot(context.Background(), mtlsIdentity, testAgentLedgerSnapshotID2(), now.Add(5*time.Second), now.Add(time.Minute)); !errors.Is(err, ErrAgentTrustNotActive) {
		t.Fatalf("revoked generation = %v, want rejection", err)
	}
}

func TestAgentTelemetryReceiptRejectsArbitraryDiagnosticText(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	receipt := AgentTelemetryReceiptInput{
		SnapshotID:     testAgentLedgerSnapshotID(),
		SnapshotDigest: testAgentLedgerSnapshotID2(),
		ObservedAt:     now,
		ExpiresAt:      now.Add(time.Minute),
		Diagnostic:     "token=secret-value",
	}
	if validAgentTelemetryReceipt(receipt, now) {
		t.Fatal("arbitrary diagnostic text unexpectedly accepted")
	}
	receipt.Diagnostic = agentLedgerDiagnosticCodeUnavailable
	if !validAgentTelemetryReceipt(receipt, now) {
		t.Fatal("stable diagnostic code rejected")
	}
}

func testAgentLedgerSnapshotID() string {
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
func testAgentLedgerSnapshotID2() string {
	return "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
}

func openAgentLedgerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openAgentTrustTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE runtime_target_builder_execution_ledgers (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, agent_id TEXT NOT NULL, slot_budget INTEGER NOT NULL, queued_builds INTEGER NOT NULL DEFAULT 0, running_builds INTEGER NOT NULL DEFAULT 0, telemetry_sequence INTEGER NOT NULL DEFAULT 0, created_at DATETIME, updated_at DATETIME, UNIQUE(runtime_target_id, agent_id))`,
		`CREATE TABLE runtime_target_agent_ledger_snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, generation_id INTEGER NOT NULL, snapshot_id TEXT NOT NULL, snapshot_digest TEXT NOT NULL, sequence INTEGER NOT NULL, builder_scope TEXT NOT NULL, provider_id TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, affinity_key TEXT NOT NULL, available BOOLEAN NOT NULL, running INTEGER NOT NULL, queued INTEGER NOT NULL, allocatable_slots INTEGER NOT NULL, observed_at DATETIME NOT NULL, expires_at DATETIME NOT NULL, issued_at DATETIME NOT NULL, consumed_at DATETIME, receipt_fingerprint TEXT NOT NULL DEFAULT '', receipt_observed_at DATETIME, receipt_expires_at DATETIME, receipt_available BOOLEAN, receipt_implementation_version TEXT NOT NULL DEFAULT '', receipt_diagnostic TEXT NOT NULL DEFAULT '', created_at DATETIME, created_by INTEGER NOT NULL DEFAULT 0, updated_at DATETIME, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0, UNIQUE(snapshot_id), UNIQUE(generation_id, sequence))`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create agent ledger test schema: %v", err)
		}
	}
	return db
}
