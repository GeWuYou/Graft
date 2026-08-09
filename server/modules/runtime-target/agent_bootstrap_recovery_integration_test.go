package runtimetarget

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

//nolint:gocyclo,cyclop // 测试在一个场景中固定验证交付、恢复、激活与单次消费的完整边界。
func TestAgentBootstrapRecoversVaultIssuanceAndConsumesDeliveryGrant(t *testing.T) {
	now := time.Date(2026, 8, 9, 14, 0, 0, 0, time.UTC)
	db := openAgentEnrollmentAuthorityTestDB(t)
	createAgentDeliveryAuthoritySchema(t, db)
	createAgentCertificateIssuanceSchema(t, db)
	repository := store.NewSQLRepository(db)
	enrollment := createPendingAgentDeliveryGeneration(t, repository, now)
	delivery := newTestAgentDeliveryAuthority(t, repository, now)
	grant, err := delivery.CreateDeliveryGrant(context.Background(), moduleapi.AgentDeliveryGrantRequest{
		TargetID:              enrollment.TargetID,
		AgentID:               enrollment.AgentID,
		Generation:            enrollment.Generation,
		ExpectedAutomationID:  "github-actions-prod",
		DockerInstallationRef: "docker:prod:agent-7",
		ExpiresAt:             now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create delivery grant: %v", err)
	}
	actor := moduleapi.DeliveryActor{ID: "github-actions-prod", Type: "service"}
	handoff := assertAgentDeliveryHandoff(t, delivery, actor, grant)
	assertAgentDeliveryReceipt(t, delivery, actor, grant, handoff, now)

	csrDER := createBootstrapRecoveryCSR(t)
	_, fingerprint, err := parseBootstrapCSR(csrDER)
	if err != nil {
		t.Fatalf("parse bootstrap CSR: %v", err)
	}
	issuanceKey := strings.Repeat("a", agentDeliveryTokenBytes*2)
	authorization, replay, err := repository.AuthorizeAgentCertificateIssuance(context.Background(), tokenVerifier(handoff.BootstrapToken, delivery.enrollmentPepper()), fingerprint, issuanceKey, now)
	if err != nil || replay {
		t.Fatalf("authorize original issuance = %#v, replay=%t, err=%v", authorization, replay, err)
	}
	issuer := &recoveryAgentCertificateIssuer{issued: moduleapi.IssuedAgentCertificate{
		IssuanceKey:          issuanceKey,
		CertificateSerial:    "vault-serial-7",
		CertificateChainDER:  [][]byte{{1, 2, 3}},
		PublicKeyFingerprint: "sha256:" + fingerprint,
		ExpiresAt:            now.Add(time.Hour),
		TrustBundle:          moduleapi.TrustBundleReference{Reference: "vault:bundle-7", Version: "bundle-7", ExpiresAt: now.Add(2 * time.Hour)},
	}}
	authority := runtimeTargetAgentBootstrapAuthority{
		repository: repository,
		pepper:     delivery.pepper,
		issuer:     issuer,
		now:        func() time.Time { return now },
		random: bytes.NewReader(append(
			bytes.Repeat([]byte{0xbb}, agentDeliveryTokenBytes),
			bytes.Repeat([]byte{0xaa}, agentDeliveryTokenBytes)...,
		)),
	}

	changedCSR := createBootstrapRecoveryCSR(t)
	if _, err := authority.BootstrapAgent(context.Background(), moduleapi.AgentBootstrapRequest{BootstrapToken: handoff.BootstrapToken, CSRDER: changedCSR}); !errors.Is(err, errAgentBootstrapRejected) {
		t.Fatalf("bootstrap changed CSR = %v", err)
	}
	if issuer.reconcileCalls != 0 || issuer.issueCalls != 0 {
		t.Fatalf("issuer was called for changed CSR: reconcile=%d issue=%d", issuer.reconcileCalls, issuer.issueCalls)
	}

	result, err := authority.BootstrapAgent(context.Background(), moduleapi.AgentBootstrapRequest{BootstrapToken: handoff.BootstrapToken, CSRDER: csrDER})
	if err != nil {
		t.Fatalf("recover bootstrap issuance: %v", err)
	}
	if issuer.reconcileCalls != 1 || issuer.issueCalls != 0 {
		t.Fatalf("issuer calls = reconcile=%d issue=%d", issuer.reconcileCalls, issuer.issueCalls)
	}
	if got := result.CertificateChainDER; len(got) != 1 || !bytes.Equal(got[0], issuer.issued.CertificateChainDER[0]) || result.TrustBundle != issuer.issued.TrustBundle || !result.ExpiresAt.Equal(issuer.issued.ExpiresAt) {
		t.Fatalf("bootstrap result = %#v", result)
	}

	var generationStatus, grantStatus, issuanceStatus string
	if err := db.QueryRow(`SELECT status FROM runtime_target_agent_generations WHERE id = $1`, authorization.Generation.ID).Scan(&generationStatus); err != nil {
		t.Fatalf("read generation status: %v", err)
	}
	if err := db.QueryRow(`SELECT status FROM runtime_target_agent_delivery_grants WHERE id = $1`, authorization.Grant.ID).Scan(&grantStatus); err != nil {
		t.Fatalf("read grant status: %v", err)
	}
	if err := db.QueryRow(`SELECT status FROM runtime_target_agent_certificate_issuances WHERE issuance_key = $1`, issuanceKey).Scan(&issuanceStatus); err != nil {
		t.Fatalf("read issuance status: %v", err)
	}
	if generationStatus != "active" || grantStatus != "consumed" || issuanceStatus != "completed" {
		t.Fatalf("post-bootstrap statuses: generation=%q grant=%q issuance=%q", generationStatus, grantStatus, issuanceStatus)
	}
	if _, err := authority.BootstrapAgent(context.Background(), moduleapi.AgentBootstrapRequest{BootstrapToken: handoff.BootstrapToken, CSRDER: csrDER}); !errors.Is(err, errAgentBootstrapRejected) {
		t.Fatalf("bootstrap consumed grant = %v", err)
	}
}

func createBootstrapRecoveryCSR(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate bootstrap CSR key: %v", err)
	}
	encoded, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	if err != nil {
		t.Fatalf("create bootstrap CSR: %v", err)
	}
	return encoded
}

func createAgentCertificateIssuanceSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE runtime_target_agent_certificate_issuances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		delivery_grant_id INTEGER NOT NULL UNIQUE,
		issuance_key TEXT NOT NULL UNIQUE,
		csr_public_key_fingerprint TEXT NOT NULL,
		status TEXT NOT NULL,
		certificate_issuer TEXT NOT NULL DEFAULT '',
		certificate_serial TEXT NOT NULL DEFAULT '',
		certificate_public_key_fingerprint TEXT NOT NULL DEFAULT '',
		certificate_expires_at DATETIME,
		trust_bundle_ref TEXT NOT NULL DEFAULT '',
		trust_bundle_version TEXT NOT NULL DEFAULT '',
		trust_bundle_expires_at DATETIME,
		authorized_at DATETIME NOT NULL,
		issued_at DATETIME,
		completed_at DATETIME,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create certificate issuance test schema: %v", err)
	}
}

type recoveryAgentCertificateIssuer struct {
	issued         moduleapi.IssuedAgentCertificate
	reconcileCalls int
	issueCalls     int
}

func (i *recoveryAgentCertificateIssuer) IssueCSR(context.Context, moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	i.issueCalls++
	return moduleapi.IssuedAgentCertificate{}, errors.New("unexpected first Vault issuance")
}

func (i *recoveryAgentCertificateIssuer) ReconcileCSR(_ context.Context, issuanceKey string) (moduleapi.IssuedAgentCertificate, error) {
	i.reconcileCalls++
	if issuanceKey != i.issued.IssuanceKey {
		return moduleapi.IssuedAgentCertificate{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	return i.issued, nil
}

func (i *recoveryAgentCertificateIssuer) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return i.issued.TrustBundle, nil
}

func (*recoveryAgentCertificateIssuer) RevokeCertificate(context.Context, moduleapi.AgentCertificateRevocation) error {
	return nil
}

var _ moduleapi.AgentCertificateIssuer = (*recoveryAgentCertificateIssuer)(nil)
