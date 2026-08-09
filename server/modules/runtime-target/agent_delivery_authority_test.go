package runtimetarget

import (
	"context"
	"database/sql"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"graft/server/internal/config"
	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

func TestAgentDeliveryAuthorityHandoffAndReceipt(t *testing.T) {
	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	db := openAgentEnrollmentAuthorityTestDB(t)
	repository := store.NewSQLRepository(db)
	createAgentDeliveryAuthoritySchema(t, db)
	enrollment := createPendingAgentDeliveryGeneration(t, repository, now)
	authority := newTestAgentDeliveryAuthority(t, repository, now)

	grant, err := authority.CreateDeliveryGrant(context.Background(), moduleapi.AgentDeliveryGrantRequest{TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, Generation: enrollment.Generation, ExpectedAutomationID: "github-actions-prod", DockerInstallationRef: "docker:prod:agent-7", ExpiresAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("create delivery grant: %v", err)
	}
	if grant.GrantID == "" || grant.TargetID != enrollment.TargetID || grant.AgentID != enrollment.AgentID || grant.Generation != enrollment.Generation || grant.ExpectedAutomationID != "github-actions-prod" {
		t.Fatalf("delivery grant = %#v", grant)
	}

	actor := moduleapi.DeliveryActor{ID: "github-actions-prod", Type: "service"}
	handoff := assertAgentDeliveryHandoff(t, authority, actor, grant)
	assertOnlyTokenVerifierPersisted(t, db, grant.GrantID, handoff.BootstrapToken, authority.enrollmentPepper())
	assertAgentDeliveryReceipt(t, authority, actor, grant, handoff, now)
}

func TestAgentDeliveryAuthorityRegistersWithoutConfiguredPepper(t *testing.T) {
	repository := store.NewSQLRepository(openAgentEnrollmentAuthorityTestDB(t))
	services := containerdi.New()
	if err := NewModule(repository).registerReaders(&module.Context{Services: services, Config: &config.Config{}}); err != nil {
		t.Fatalf("register runtime target delivery authority: %v", err)
	}
	registered, err := module.ResolveService[moduleapi.AgentDeliveryAuthority](services, (*moduleapi.AgentDeliveryAuthority)(nil))
	if err != nil {
		t.Fatalf("resolve agent delivery authority: %v", err)
	}
	if _, ok := registered.(runtimeTargetAgentDeliveryAuthority); !ok {
		t.Fatalf("registered delivery authority = %T, want runtimeTargetAgentDeliveryAuthority", registered)
	}
}

func assertAgentDeliveryHandoff(t *testing.T, authority runtimeTargetAgentDeliveryAuthority, actor moduleapi.DeliveryActor, grant moduleapi.AgentDeliveryGrant) moduleapi.AgentDeliveryHandoffMaterial {
	t.Helper()
	if _, err := authority.HandoffDeliveryGrant(context.Background(), moduleapi.DeliveryActor{ID: "wrong", Type: "service"}, moduleapi.AgentDeliveryHandoffRequest{GrantID: grant.GrantID}); err == nil {
		t.Fatal("handoff for an unexpected automation identity succeeded")
	}
	handoff, err := authority.HandoffDeliveryGrant(context.Background(), actor, moduleapi.AgentDeliveryHandoffRequest{GrantID: grant.GrantID})
	if err != nil {
		t.Fatalf("handoff delivery grant: %v", err)
	}
	if handoff.GrantID != grant.GrantID || handoff.HandoffID == "" || len(handoff.BootstrapToken) != 64 {
		t.Fatalf("handoff material = %#v", handoff)
	}
	if _, err := authority.HandoffDeliveryGrant(context.Background(), actor, moduleapi.AgentDeliveryHandoffRequest{GrantID: grant.GrantID}); err == nil {
		t.Fatal("second handoff succeeded")
	}
	return handoff
}

func assertAgentDeliveryReceipt(t *testing.T, authority runtimeTargetAgentDeliveryAuthority, actor moduleapi.DeliveryActor, grant moduleapi.AgentDeliveryGrant, handoff moduleapi.AgentDeliveryHandoffMaterial, now time.Time) {
	t.Helper()
	receiptRequest := moduleapi.AgentDeliveryReceiptRequest{GrantID: grant.GrantID, ReceiptID: "receipt-1", ProtocolVersion: "graft.delivery-receipt.v1", HandoffID: handoff.HandoffID, AssertedDeliveredAt: now, DockerInstallationRef: grant.DockerInstallationRef, DockerSecretRef: "docker-secret:agent-7:v1", PayloadFingerprint: strings.Repeat("a", 64)}
	receipt, err := authority.RecordDeliveryReceipt(context.Background(), actor, receiptRequest)
	if err != nil {
		t.Fatalf("record delivery receipt: %v", err)
	}
	if receipt.AutomationID != actor.ID || receipt.Replay {
		t.Fatalf("receipt = %#v", receipt)
	}
	replay, err := authority.RecordDeliveryReceipt(context.Background(), actor, receiptRequest)
	if err != nil {
		t.Fatalf("replay delivery receipt: %v", err)
	}
	if !replay.Replay {
		t.Fatalf("replay = %#v, want Replay=true", replay)
	}
}

func TestAgentDeliveryAuthorityRejectsMissingPepperAndInvalidReceipt(t *testing.T) {
	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	db := openAgentEnrollmentAuthorityTestDB(t)
	repository := store.NewSQLRepository(db)
	createAgentDeliveryAuthoritySchema(t, db)
	enrollment := createPendingAgentDeliveryGeneration(t, repository, now)
	authority := runtimeTargetAgentDeliveryAuthority{repository: repository, now: func() time.Time { return now }}
	if _, err := authority.CreateDeliveryGrant(context.Background(), moduleapi.AgentDeliveryGrantRequest{TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, Generation: enrollment.Generation, ExpectedAutomationID: "automation", DockerInstallationRef: "docker:prod", ExpiresAt: now.Add(time.Hour)}); err == nil {
		t.Fatal("create delivery grant without installation pepper succeeded")
	}
	if _, err := newTestAgentDeliveryAuthority(t, repository, now).RecordDeliveryReceipt(context.Background(), moduleapi.DeliveryActor{ID: "automation", Type: "service"}, moduleapi.AgentDeliveryReceiptRequest{ProtocolVersion: "graft.delivery-receipt.v1", AssertedDeliveredAt: now}); err == nil {
		t.Fatal("invalid receipt succeeded")
	}
}

func TestAgentDeliveryAuthorityRejectsNonDockerGeneration(t *testing.T) {
	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	db := openAgentEnrollmentAuthorityTestDB(t)
	repository := store.NewSQLRepository(db)
	createAgentDeliveryAuthoritySchema(t, db)
	enrollment := createPendingAgentDeliveryGeneration(t, repository, now)
	if _, err := db.Exec(`UPDATE runtime_target_agent_identities SET provider_id = 'podman' WHERE runtime_target_id = $1 AND agent_id = $2`, enrollment.TargetID, enrollment.AgentID); err != nil {
		t.Fatalf("change test provider: %v", err)
	}
	if _, err := newTestAgentDeliveryAuthority(t, repository, now).CreateDeliveryGrant(context.Background(), moduleapi.AgentDeliveryGrantRequest{TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, Generation: enrollment.Generation, ExpectedAutomationID: "automation", DockerInstallationRef: "docker:prod", ExpiresAt: now.Add(time.Hour)}); err == nil {
		t.Fatal("create delivery grant for non-Docker generation succeeded")
	}
}

func newTestAgentDeliveryAuthority(t *testing.T, repository *store.SQLRepository, now time.Time) runtimeTargetAgentDeliveryAuthority {
	t.Helper()
	pepperPath := t.TempDir() + "/enrollment-pepper"
	if err := os.WriteFile(pepperPath, []byte("test-installation-pepper"), 0o600); err != nil {
		t.Fatalf("write enrollment pepper: %v", err)
	}
	pepper, err := config.NewEnrollmentPepperProvider(config.EnrollmentSecurityConfig{PepperFile: pepperPath})
	if err != nil {
		t.Fatalf("construct enrollment pepper provider: %v", err)
	}
	return runtimeTargetAgentDeliveryAuthority{repository: repository, pepper: pepper, now: func() time.Time { return now }}
}

func createPendingAgentDeliveryGeneration(t *testing.T, repository *store.SQLRepository, now time.Time) moduleapi.AgentEnrollment {
	t.Helper()
	authority := runtimeTargetAgentEnrollmentAuthority{repository: repository, now: func() time.Time { return now }}
	enrollment := createAgentEnrollment(t, authority, testAgentEnrollmentRequest(now.Add(time.Hour)))
	return enrollment
}

func assertOnlyTokenVerifierPersisted(t *testing.T, db *sql.DB, grantID, token string, pepper []byte) {
	t.Helper()
	var verifier string
	if err := db.QueryRowContext(context.Background(), `SELECT token_verifier FROM runtime_target_agent_delivery_grants WHERE grant_id = $1`, grantID).Scan(&verifier); err != nil {
		t.Fatalf("read token verifier: %v", err)
	}
	if verifier == token || verifier != tokenVerifier(token, pepper) {
		t.Fatalf("stored verifier = %q", verifier)
	}
	if _, err := hex.DecodeString(verifier); err != nil {
		t.Fatalf("stored verifier is not SHA-256 hex: %v", err)
	}
}

func createAgentDeliveryAuthoritySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE runtime_target_agent_delivery_grants (id INTEGER PRIMARY KEY AUTOINCREMENT, generation_id INTEGER NOT NULL, grant_id TEXT NOT NULL UNIQUE, token_verifier TEXT NOT NULL, expected_automation_id TEXT NOT NULL, docker_installation_ref TEXT NOT NULL, expires_at DATETIME NOT NULL, status TEXT NOT NULL, handoff_id TEXT NOT NULL DEFAULT '', handed_off_at DATETIME, delivered_at DATETIME, consumed_at DATETIME, revoked_at DATETIME, revoked_reason TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, created_by INTEGER NOT NULL DEFAULT 0, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE runtime_target_agent_delivery_receipts (id INTEGER PRIMARY KEY AUTOINCREMENT, delivery_grant_id INTEGER NOT NULL, receipt_id TEXT NOT NULL UNIQUE, protocol_version TEXT NOT NULL, automation_id TEXT NOT NULL, handoff_id TEXT NOT NULL, asserted_delivered_at DATETIME NOT NULL, accepted_at DATETIME NOT NULL, docker_installation_ref TEXT NOT NULL, docker_secret_ref TEXT NOT NULL, payload_fingerprint TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, created_by INTEGER NOT NULL DEFAULT 0, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create agent delivery authority test schema: %v", err)
		}
	}
}
