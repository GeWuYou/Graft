package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestAgentTrustGenerationRotationAndRevocationRejectOldGenerations(t *testing.T) {
	db := openAgentTrustTestDB(t)
	repository := NewSQLRepository(db)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	identity := testAgentTrustIdentity()
	first := testPendingAgentTrustGeneration(1, now.Add(time.Hour))
	if _, err := repository.CreatePendingAgentTrustGeneration(context.Background(), identity, first); err != nil {
		t.Fatalf("create first generation: %v", err)
	}
	if err := repository.ActivateAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 1, "vault-pki-intermediate", "serial-1", "sha256:first", 42, now); err != nil {
		t.Fatalf("activate first generation: %v", err)
	}
	if _, err := repository.ReadActiveAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 1, now); err != nil {
		t.Fatalf("read active first generation: %v", err)
	}

	second := testPendingAgentTrustGeneration(2, now.Add(2*time.Hour))
	if _, err := repository.CreatePendingAgentTrustGeneration(context.Background(), identity, second); err != nil {
		t.Fatalf("create second generation: %v", err)
	}
	if _, err := repository.ReadActiveAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 1, now); !errors.Is(err, ErrAgentTrustNotFound) {
		t.Fatalf("rotation-pending first generation error = %v, want %v", err, ErrAgentTrustNotFound)
	}
	if err := repository.ActivateAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 2, "vault-pki-intermediate", "serial-2", "sha256:second", 42, now.Add(time.Minute)); err != nil {
		t.Fatalf("activate second generation: %v", err)
	}
	if _, err := repository.ReadActiveAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 1, now.Add(time.Minute)); !errors.Is(err, ErrAgentTrustNotFound) {
		t.Fatalf("retired first generation error = %v, want %v", err, ErrAgentTrustNotFound)
	}
	if _, err := repository.ReadActiveAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 2, now.Add(time.Minute)); err != nil {
		t.Fatalf("read active second generation: %v", err)
	}
	if err := repository.RevokeAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 2, "operator_revoke", 42, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("revoke second generation: %v", err)
	}
	if err := repository.RevokeAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 2, "operator_revoke", 42, now.Add(3*time.Minute)); err != nil {
		t.Fatalf("repeat revoke second generation: %v", err)
	}
	if _, err := repository.ReadActiveAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID, 2, now.Add(3*time.Minute)); !errors.Is(err, ErrAgentTrustNotFound) {
		t.Fatalf("revoked second generation error = %v, want %v", err, ErrAgentTrustNotFound)
	}
	current, err := repository.ReadCurrentAgentTrustGeneration(context.Background(), identity.TargetID, identity.AgentID)
	if err != nil || current.Status != "revoked" || current.RevokedAt == nil {
		t.Fatalf("current revoked generation = %#v, err=%v", current, err)
	}
}

func TestAgentTrustGenerationRejectsReuseAfterReset(t *testing.T) {
	db := openAgentTrustTestDB(t)
	repository := NewSQLRepository(db)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	identity := testAgentTrustIdentity()
	if _, err := repository.CreatePendingAgentTrustGeneration(context.Background(), identity, testPendingAgentTrustGeneration(1, now.Add(time.Hour))); err != nil {
		t.Fatalf("create generation: %v", err)
	}
	if err := repository.RevokeAllAgentTrustGenerations(context.Background(), identity.TargetID, identity.AgentID, "target_reset", 42, now); err != nil {
		t.Fatalf("reset trust: %v", err)
	}
	if _, err := repository.CreatePendingAgentTrustGeneration(context.Background(), identity, testPendingAgentTrustGeneration(1, now.Add(2*time.Hour))); !errors.Is(err, ErrAgentTrustNotActive) {
		t.Fatalf("reused generation error = %v, want %v", err, ErrAgentTrustNotActive)
	}
	if _, err := repository.CreatePendingAgentTrustGeneration(context.Background(), identity, testPendingAgentTrustGeneration(2, now.Add(2*time.Hour))); err != nil {
		t.Fatalf("create fresh generation after reset: %v", err)
	}
}

func testAgentTrustIdentity() AgentTrustIdentity {
	return AgentTrustIdentity{IdentityID: "vault:identity-7", TargetID: 7, AgentID: "agent-7", ProviderID: "docker", BuilderScope: "builder-agent-7", CapabilityProfile: "oci-build", CapabilityVersion: "v1", ImageDigest: "sha256:image", AgentVersion: "v1.0.0"}
}

func testPendingAgentTrustGeneration(generation int64, expiresAt time.Time) AgentTrustGeneration {
	return AgentTrustGeneration{Generation: generation, EnrollmentRef: "vault:enrollment", TrustBundleRef: "vault:trust-bundle", TrustBundleVersion: "2026-08", ExpiresAt: expiresAt}
}

func openAgentTrustTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE runtime_target_agent_identities (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, identity_id TEXT NOT NULL, agent_id TEXT NOT NULL, provider_id TEXT NOT NULL, builder_scope TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, image_digest TEXT NOT NULL DEFAULT '', agent_version TEXT NOT NULL DEFAULT '', updated_at DATETIME, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0)`,
		`CREATE UNIQUE INDEX uq_runtime_target_agent_identities_identity_live ON runtime_target_agent_identities (identity_id) WHERE deleted_at = 0`,
		`CREATE UNIQUE INDEX uq_runtime_target_agent_identities_target_agent_live ON runtime_target_agent_identities (runtime_target_id, agent_id) WHERE deleted_at = 0`,
		`CREATE TABLE runtime_target_agent_generations (id INTEGER PRIMARY KEY AUTOINCREMENT, identity_id INTEGER NOT NULL, generation INTEGER NOT NULL, enrollment_ref TEXT NOT NULL, trust_bundle_ref TEXT NOT NULL, trust_bundle_version TEXT NOT NULL, certificate_issuer TEXT NOT NULL DEFAULT '', certificate_serial TEXT NOT NULL DEFAULT '', public_key_fingerprint TEXT NOT NULL DEFAULT '', expires_at DATETIME NOT NULL, status TEXT NOT NULL, activated_at DATETIME, retired_at DATETIME, revoked_at DATETIME, revoked_reason TEXT NOT NULL DEFAULT '', updated_at DATETIME, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, UNIQUE(identity_id, generation))`,
		`CREATE UNIQUE INDEX uq_runtime_target_agent_generations_active_identity ON runtime_target_agent_generations (identity_id) WHERE status = 'active' AND deleted_at = 0`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create agent trust test schema: %v", err)
		}
	}
	return db
}
