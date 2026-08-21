package runtimetarget

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	contract "graft/server/modules/runtime-target/contract"
	store "graft/server/modules/runtime-target/store"
)

func TestNewAgentTrustAuditEventStatusMatchesSuccess(t *testing.T) {
	binding := moduleapi.RuntimeTargetAgentBinding{TargetID: 7, AgentID: "agent-7", Generation: 1}

	successEvent, err := NewAgentTrustAuditEvent(AgentTrustAuditActionRegistration, nil, binding, true, "registered")
	if err != nil {
		t.Fatalf("create successful audit event: %v", err)
	}
	if successEvent.StatusCode != http.StatusOK || !successEvent.Success {
		t.Fatalf("successful audit event = %#v", successEvent)
	}

	failureEvent, err := NewAgentTrustAuditEvent(AgentTrustAuditActionRegistration, nil, binding, false, "rejected")
	if err != nil {
		t.Fatalf("create failed audit event: %v", err)
	}
	if failureEvent.StatusCode != http.StatusUnprocessableEntity || failureEvent.Success {
		t.Fatalf("failed audit event = %#v", failureEvent)
	}
}

func TestAgentEnrollmentAuthorityRegistersAndPersistsLifecycle(t *testing.T) {
	db := openAgentEnrollmentAuthorityTestDB(t)
	repository := store.NewSQLRepository(db)
	assertAgentEnrollmentAuthorityRegistered(t, repository)

	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	authority := runtimeTargetAgentEnrollmentAuthority{repository: repository, events: &runtimeTargetRecordingPublisher{}, now: func() time.Time { return now }}
	first := createAgentEnrollment(t, authority, testAgentEnrollmentRequest(now.Add(time.Hour)))
	assertPendingAgentEnrollment(t, first, 1, "enrollment-1", "bundle-1")
	activateAgentEnrollment(t, authority, first, "serial-1", "sha256:first")
	assertUnsupportedAgentEnrollmentRotation(t, authority, first, now)
	second := rotateAgentEnrollment(t, authority, first, now)
	assertPendingAgentEnrollment(t, second, 2, "enrollment-2", "bundle-2")
	activateAgentEnrollment(t, authority, second, "serial-2", "sha256:second")
	assertAgentEnrollmentRevocationIsIdempotent(t, authority, repository, second)
}

func TestAgentEnrollmentRevocationPublishesDurableCertificateFactOnce(t *testing.T) {
	db := openAgentEnrollmentAuthorityTestDB(t)
	repository := store.NewSQLRepository(db)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	publisher := &runtimeTargetRecordingPublisher{}
	authority := runtimeTargetAgentEnrollmentAuthority{repository: repository, events: publisher, now: func() time.Time { return now }}
	enrollment := createAgentEnrollment(t, authority, testAgentEnrollmentRequest(now.Add(time.Hour)))
	activateAgentEnrollment(t, authority, enrollment, "serial-1", "sha256:first")
	revocation := moduleapi.AgentEnrollmentRevocation{IdentityID: enrollment.IdentityID, TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, Generation: enrollment.Generation, Reason: "operator_revoke"}
	if err := authority.RevokeGeneration(context.Background(), revocation); err != nil {
		t.Fatalf("revoke generation: %v", err)
	}
	if err := authority.RevokeGeneration(context.Background(), revocation); err != nil {
		t.Fatalf("repeat revoke generation: %v", err)
	}
	if len(publisher.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.published))
	}
	var payload contract.AgentCertificateRevocationEvent
	if err := json.Unmarshal(publisher.published[0].Payload, &payload); err != nil {
		t.Fatalf("decode revocation event: %v", err)
	}
	if publisher.published[0].Type != contract.AgentCertificateRevocationEventType || payload.CertificateSerial != "serial-1" || payload.IdentityID != enrollment.IdentityID {
		t.Fatalf("revocation event = %#v, payload = %#v", publisher.published[0], payload)
	}
}

func TestAgentEnrollmentRevocationRejectsUnavailablePublisherWithoutChangingTrust(t *testing.T) {
	db := openAgentEnrollmentAuthorityTestDB(t)
	repository := store.NewSQLRepository(db)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	authority := runtimeTargetAgentEnrollmentAuthority{repository: repository, now: func() time.Time { return now }}
	enrollment := createAgentEnrollment(t, authority, testAgentEnrollmentRequest(now.Add(time.Hour)))
	activateAgentEnrollment(t, authority, enrollment, "serial-1", "sha256:first")
	revocation := moduleapi.AgentEnrollmentRevocation{IdentityID: enrollment.IdentityID, TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, Generation: enrollment.Generation, Reason: "operator_revoke"}
	if err := authority.RevokeGeneration(context.Background(), revocation); err == nil {
		t.Fatal("revocation without a publisher unexpectedly succeeded")
	}
	current, err := repository.ReadCurrentAgentTrustGeneration(context.Background(), enrollment.TargetID, enrollment.AgentID)
	if err != nil || current.Status != "active" {
		t.Fatalf("trust state after rejected revocation = %#v, err=%v", current, err)
	}
}

func assertAgentEnrollmentAuthorityRegistered(t *testing.T, repository *store.SQLRepository) {
	t.Helper()
	services := containerdi.New()
	if err := NewModule(repository).registerReaders(&module.Context{Services: services}); err != nil {
		t.Fatalf("register runtime target readers: %v", err)
	}
	registered, err := module.ResolveService[moduleapi.AgentEnrollmentAuthority](services, (*moduleapi.AgentEnrollmentAuthority)(nil))
	if err != nil {
		t.Fatalf("resolve agent enrollment authority: %v", err)
	}
	if _, ok := registered.(runtimeTargetAgentEnrollmentAuthority); !ok {
		t.Fatalf("registered enrollment authority = %T, want runtimeTargetAgentEnrollmentAuthority", registered)
	}
}

func createAgentEnrollment(t *testing.T, authority moduleapi.AgentEnrollmentAuthority, request moduleapi.AgentEnrollmentRequest) moduleapi.AgentEnrollment {
	t.Helper()
	enrollment, err := authority.CreateEnrollment(context.Background(), request)
	if err != nil {
		t.Fatalf("create enrollment: %v", err)
	}
	return enrollment
}

func assertPendingAgentEnrollment(t *testing.T, enrollment moduleapi.AgentEnrollment, generation int64, enrollmentRef, trustBundleVersion string) {
	t.Helper()
	if enrollment.IdentityID != "runtime-target:7:agent:agent-7" || enrollment.Generation != generation || enrollment.Status != moduleapi.RuntimeTargetAgentStatusPending || enrollment.EnrollmentRef != enrollmentRef || enrollment.TrustBundleVersion != trustBundleVersion {
		t.Fatalf("pending enrollment = %#v", enrollment)
	}
}

func activateAgentEnrollment(t *testing.T, authority moduleapi.AgentEnrollmentAuthority, enrollment moduleapi.AgentEnrollment, serial, fingerprint string) {
	t.Helper()
	err := authority.ActivateGeneration(context.Background(), moduleapi.AgentEnrollmentActivation{IdentityID: enrollment.IdentityID, TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, Generation: enrollment.Generation, CertificateIssuer: "vault-pki-intermediate", CertificateSerial: serial, PublicKeyFingerprint: fingerprint})
	if err != nil {
		t.Fatalf("activate generation: %v", err)
	}
}

func assertUnsupportedAgentEnrollmentRotation(t *testing.T, authority moduleapi.AgentEnrollmentAuthority, enrollment moduleapi.AgentEnrollment, now time.Time) {
	t.Helper()
	_, err := authority.RotateGeneration(context.Background(), moduleapi.AgentEnrollmentRotationRequest{IdentityID: enrollment.IdentityID, TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, ProviderID: "podman", BuilderScope: enrollment.BuilderScope, CapabilityProfile: enrollment.CapabilityProfile, CapabilityVersion: enrollment.CapabilityVersion, Capabilities: enrollment.Capabilities, RuntimeProtocol: enrollment.RuntimeProtocol, EnrollmentRef: "unsupported-enrollment", TrustBundle: moduleapi.TrustBundleReference{Reference: "vault:unsupported", Version: "bundle-unsupported", ExpiresAt: now.Add(2 * time.Hour)}, ExpiresAt: now.Add(2 * time.Hour), Reason: "certificate_rotation"})
	if err == nil {
		t.Fatal("rotate enrollment for unsupported provider succeeded")
	}
}

func rotateAgentEnrollment(t *testing.T, authority moduleapi.AgentEnrollmentAuthority, enrollment moduleapi.AgentEnrollment, now time.Time) moduleapi.AgentEnrollment {
	t.Helper()
	rotated, err := authority.RotateGeneration(context.Background(), moduleapi.AgentEnrollmentRotationRequest{IdentityID: enrollment.IdentityID, TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, ProviderID: enrollment.ProviderID, BuilderScope: enrollment.BuilderScope, CapabilityProfile: enrollment.CapabilityProfile, CapabilityVersion: enrollment.CapabilityVersion, Capabilities: enrollment.Capabilities, RuntimeProtocol: enrollment.RuntimeProtocol, EnrollmentRef: "enrollment-2", TrustBundle: moduleapi.TrustBundleReference{Reference: "vault:bundle-2", Version: "bundle-2", ExpiresAt: now.Add(2 * time.Hour)}, ExpiresAt: now.Add(2 * time.Hour), Reason: "certificate_rotation"})
	if err != nil {
		t.Fatalf("rotate generation: %v", err)
	}
	return rotated
}

func assertAgentEnrollmentRevocationIsIdempotent(t *testing.T, authority moduleapi.AgentEnrollmentAuthority, repository *store.SQLRepository, enrollment moduleapi.AgentEnrollment) {
	t.Helper()
	revocation := moduleapi.AgentEnrollmentRevocation{IdentityID: enrollment.IdentityID, TargetID: enrollment.TargetID, AgentID: enrollment.AgentID, Generation: enrollment.Generation, Reason: "operator_revoke"}
	if err := authority.RevokeGeneration(context.Background(), revocation); err != nil {
		t.Fatalf("revoke generation: %v", err)
	}
	if err := authority.RevokeGeneration(context.Background(), revocation); err != nil {
		t.Fatalf("repeat revoke generation: %v", err)
	}
	current, err := repository.ReadCurrentAgentTrustGeneration(context.Background(), enrollment.TargetID, enrollment.AgentID)
	if err != nil {
		t.Fatalf("read revoked enrollment: %v", err)
	}
	if current.Status != string(moduleapi.RuntimeTargetAgentStatusRevoked) || current.RevokedAt == nil {
		t.Fatalf("current generation after revocation = %#v", current)
	}
}

func TestAgentEnrollmentAuthorityRejectsMissingPKIAttestationMetadata(t *testing.T) {
	db := openAgentEnrollmentAuthorityTestDB(t)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	authority := runtimeTargetAgentEnrollmentAuthority{repository: store.NewSQLRepository(db), now: func() time.Time { return now }}
	request := testAgentEnrollmentRequest(now.Add(time.Hour))
	request.TrustBundle.Reference = ""
	if _, err := authority.CreateEnrollment(context.Background(), request); err == nil {
		t.Fatal("create enrollment without trust bundle reference succeeded")
	}
	if _, err := authority.CreateEnrollment(context.Background(), testAgentEnrollmentRequest(now)); err == nil {
		t.Fatal("create enrollment with expired metadata succeeded")
	}
	expiredTrustBundle := testAgentEnrollmentRequest(now.Add(time.Hour))
	expiredTrustBundle.TrustBundle.ExpiresAt = now
	if _, err := authority.CreateEnrollment(context.Background(), expiredTrustBundle); err == nil {
		t.Fatal("create enrollment with expired trust bundle succeeded")
	}
	unsupportedProvider := testAgentEnrollmentRequest(now.Add(time.Hour))
	unsupportedProvider.ProviderID = "podman"
	if _, err := authority.CreateEnrollment(context.Background(), unsupportedProvider); err == nil {
		t.Fatal("create enrollment for unsupported provider succeeded")
	}
	invalidDigest := testAgentEnrollmentRequest(now.Add(time.Hour))
	invalidDigest.ImageDigest = "graft-runtime-agent:latest"
	if _, err := authority.CreateEnrollment(context.Background(), invalidDigest); err == nil {
		t.Fatal("create enrollment with mutable image reference succeeded")
	}
	missingVersion := testAgentEnrollmentRequest(now.Add(time.Hour))
	missingVersion.AgentVersion = ""
	if _, err := authority.CreateEnrollment(context.Background(), missingVersion); err == nil {
		t.Fatal("create enrollment without agent version succeeded")
	}
	nonHexDigest := testAgentEnrollmentRequest(now.Add(time.Hour))
	nonHexDigest.ImageDigest = "sha256:" + strings.Repeat("g", 64)
	if _, err := authority.CreateEnrollment(context.Background(), nonHexDigest); err == nil {
		t.Fatal("create enrollment with non-hex digest succeeded")
	}
	uppercaseDigest := testAgentEnrollmentRequest(now.Add(time.Hour))
	uppercaseDigest.ImageDigest = "sha256:" + strings.Repeat("A", 64)
	if _, err := authority.CreateEnrollment(context.Background(), uppercaseDigest); err == nil {
		t.Fatal("create enrollment with uppercase digest succeeded")
	}
	if err := authority.RevokeGeneration(context.Background(), moduleapi.AgentEnrollmentRevocation{IdentityID: "runtime-target:7:agent:agent-7", TargetID: 7, AgentID: "agent-7", Generation: 1}); err == nil {
		t.Fatal("revoke enrollment without reason succeeded")
	} else if errors.Is(err, store.ErrAgentTrustNotFound) {
		t.Fatalf("revocation validation reached persistence: %v", err)
	}
}

func testAgentEnrollmentRequest(expiresAt time.Time) moduleapi.AgentEnrollmentRequest {
	return moduleapi.AgentEnrollmentRequest{TargetID: 7, AgentID: "agent-7", ProviderID: "docker", BuilderScope: "runtime-agent-7", CapabilityProfile: "oci-build", CapabilityVersion: "v1", Capabilities: []string{"oci-build"}, RuntimeProtocol: "runtime/v1", ImageDigest: "sha256:" + strings.Repeat("a", 64), AgentVersion: "v1.0.0", EnrollmentRef: "enrollment-1", TrustBundle: moduleapi.TrustBundleReference{Reference: "vault:bundle-1", Version: "bundle-1", ExpiresAt: expiresAt}, ExpiresAt: expiresAt}
}

func openAgentEnrollmentAuthorityTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE runtime_target_agent_identities (id INTEGER PRIMARY KEY AUTOINCREMENT, runtime_target_id INTEGER NOT NULL, identity_id TEXT NOT NULL, agent_id TEXT NOT NULL, provider_id TEXT NOT NULL, builder_scope TEXT NOT NULL, capability_profile TEXT NOT NULL, capability_version TEXT NOT NULL, image_digest TEXT NOT NULL DEFAULT '', agent_version TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, created_by INTEGER NOT NULL DEFAULT 0, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0, CHECK (trim(identity_id) <> ''), CHECK (trim(agent_id) <> ''), CHECK (trim(provider_id) <> ''), CHECK (trim(builder_scope) <> ''), CHECK (trim(capability_profile) <> ''), CHECK (trim(capability_version) <> ''))`,
		`CREATE TABLE runtime_target_agent_capability_bindings (id INTEGER PRIMARY KEY AUTOINCREMENT, identity_id INTEGER NOT NULL, generation_id INTEGER NOT NULL UNIQUE, provider_id TEXT NOT NULL, capabilities TEXT NOT NULL, capability_version TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, created_by INTEGER NOT NULL DEFAULT 0, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0)`,
		`CREATE UNIQUE INDEX uq_runtime_target_agent_identities_identity_live ON runtime_target_agent_identities (identity_id) WHERE deleted_at = 0`,
		`CREATE UNIQUE INDEX uq_runtime_target_agent_identities_target_agent_live ON runtime_target_agent_identities (runtime_target_id, agent_id) WHERE deleted_at = 0`,
		`CREATE TABLE runtime_target_agent_generations (id INTEGER PRIMARY KEY AUTOINCREMENT, identity_id INTEGER NOT NULL, generation INTEGER NOT NULL, enrollment_ref TEXT NOT NULL, trust_bundle_ref TEXT NOT NULL, trust_bundle_version TEXT NOT NULL, certificate_issuer TEXT NOT NULL DEFAULT '', certificate_serial TEXT NOT NULL DEFAULT '', public_key_fingerprint TEXT NOT NULL DEFAULT '', expires_at DATETIME NOT NULL, status TEXT NOT NULL, activated_at DATETIME, retired_at DATETIME, revoked_at DATETIME, revoked_reason TEXT NOT NULL DEFAULT '', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, created_by INTEGER NOT NULL DEFAULT 0, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_by INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0, UNIQUE(identity_id, generation), CHECK (generation > 0), CHECK (status IN ('pending', 'active', 'revoked', 'retired')), CHECK (status <> 'pending' OR (certificate_serial = '' AND public_key_fingerprint = '' AND activated_at IS NULL)), CHECK (status <> 'active' OR (certificate_issuer <> '' AND certificate_serial <> '' AND public_key_fingerprint <> '' AND activated_at IS NOT NULL AND revoked_at IS NULL AND retired_at IS NULL)), CHECK (status <> 'revoked' OR revoked_at IS NOT NULL), CHECK (status <> 'retired' OR retired_at IS NOT NULL))`,
		`CREATE UNIQUE INDEX uq_runtime_target_agent_generations_active_identity ON runtime_target_agent_generations (identity_id) WHERE status = 'active' AND deleted_at = 0`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create agent enrollment authority test schema: %v", err)
		}
	}
	return db
}
