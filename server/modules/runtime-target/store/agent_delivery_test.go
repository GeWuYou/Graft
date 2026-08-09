package store

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestAgentDeliveryGrantHandoffAndReceiptReplay(t *testing.T) {
	db := openAgentDeliveryTestDB(t)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO runtime_target_agent_generations (id, status, deleted_at) VALUES (9, 'pending', 0)`); err != nil {
		t.Fatal(err)
	}
	repository := NewSQLRepository(db)
	grant, err := repository.CreatePendingAgentDeliveryGrant(context.Background(), AgentDeliveryGrant{GenerationID: 9, GrantID: "grant-1", TokenVerifier: strings.Repeat("a", 64), ExpectedAutomationID: "automation-1", DockerInstallationRef: "docker:service-1", ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("create delivery grant: %v", err)
	}
	if _, err := repository.AcceptAgentDeliveryHandoff(context.Background(), grant.GrantID, "automation-1", "handoff-1", now); err != nil {
		t.Fatalf("accept handoff: %v", err)
	}
	receipt := AgentDeliveryReceipt{GrantID: grant.GrantID, ReceiptID: "receipt-1", ProtocolVersion: "graft.delivery-receipt.v1", AutomationID: "automation-1", HandoffID: "handoff-1", AssertedDeliveredAt: now, DockerInstallationRef: "docker:service-1", DockerSecretRef: "secret:version-1", PayloadFingerprint: strings.Repeat("b", 64)}
	first, replay, err := repository.RecordAgentDeliveryReceipt(context.Background(), receipt, now.Add(time.Minute))
	if err != nil || replay || first.DeliveryGrantID != grant.ID {
		t.Fatalf("record receipt = %#v, replay=%t, err=%v", first, replay, err)
	}
	second, replay, err := repository.RecordAgentDeliveryReceipt(context.Background(), receipt, now.Add(2*time.Minute))
	if err != nil || !replay || second.ID != first.ID {
		t.Fatalf("replay receipt = %#v, replay=%t, err=%v", second, replay, err)
	}
	receipt.DockerSecretRef = "secret:changed"
	if _, _, err := repository.RecordAgentDeliveryReceipt(context.Background(), receipt, now.Add(3*time.Minute)); err != ErrAgentDeliveryRejected {
		t.Fatalf("changed receipt error = %v, want %v", err, ErrAgentDeliveryRejected)
	}
}

func openAgentDeliveryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE runtime_target_agent_generations (id INTEGER PRIMARY KEY, status TEXT NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE runtime_target_agent_delivery_grants (id INTEGER PRIMARY KEY AUTOINCREMENT, generation_id INTEGER NOT NULL, grant_id TEXT NOT NULL UNIQUE, token_verifier TEXT NOT NULL, expected_automation_id TEXT NOT NULL, docker_installation_ref TEXT NOT NULL, expires_at DATETIME NOT NULL, status TEXT NOT NULL, handoff_id TEXT NOT NULL DEFAULT '', handed_off_at DATETIME, delivered_at DATETIME, consumed_at DATETIME, revoked_at DATETIME, revoked_reason TEXT NOT NULL DEFAULT '', updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, deleted_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE runtime_target_agent_delivery_receipts (id INTEGER PRIMARY KEY AUTOINCREMENT, delivery_grant_id INTEGER NOT NULL, receipt_id TEXT NOT NULL UNIQUE, protocol_version TEXT NOT NULL, automation_id TEXT NOT NULL, handoff_id TEXT NOT NULL, asserted_delivered_at DATETIME NOT NULL, accepted_at DATETIME NOT NULL, docker_installation_ref TEXT NOT NULL, docker_secret_ref TEXT NOT NULL, payload_fingerprint TEXT NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
}
