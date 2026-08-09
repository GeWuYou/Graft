package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AgentCertificateIssuance 是与单个已投递授权绑定的可恢复证书签发事实。
type AgentCertificateIssuance struct {
	ID                              int64
	DeliveryGrantID                 int64
	IssuanceKey                     string
	CSRPublicKeyFingerprint         string
	Status                          string
	CertificateIssuer               string
	CertificateSerial               string
	CertificatePublicKeyFingerprint string
	CertificateExpiresAt            *time.Time
	TrustBundleRef                  string
	TrustBundleVersion              string
	TrustBundleExpiresAt            *time.Time
	AuthorizedAt                    time.Time
	IssuedAt                        *time.Time
	CompletedAt                     *time.Time
}

// AgentBootstrapAuthorization 返回已通过 token、交付和 CSR 绑定检查的签发上下文。
type AgentBootstrapAuthorization struct {
	Issuance   AgentCertificateIssuance
	Grant      AgentDeliveryGrant
	Generation AgentTrustGeneration
}

// AuthorizeAgentCertificateIssuance 以单次 delivered grant 和 CSR 公钥指纹建立可恢复签发授权。
// 同一个 CSR 指纹的重试返回原签发键；不同指纹、过期或非 delivered 授权均被拒绝。
//
//nolint:cyclop // 授权、幂等重放与指纹冲突必须在同一事务边界失败关闭。
func (r *SQLRepository) AuthorizeAgentCertificateIssuance(ctx context.Context, tokenVerifier, csrFingerprint, issuanceKey string, now time.Time) (AgentBootstrapAuthorization, bool, error) {
	if r == nil || r.db == nil || !validSHA256Hex(tokenVerifier) || !validSHA256Hex(csrFingerprint) || strings.TrimSpace(issuanceKey) == "" || now.IsZero() {
		return AgentBootstrapAuthorization{}, false, ErrAgentDeliveryRejected
	}
	var authorization AgentBootstrapAuthorization
	var replay bool
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		result, err := r.executor(txCtx).ExecContext(txCtx, `INSERT INTO runtime_target_agent_certificate_issuances (delivery_grant_id, issuance_key, csr_public_key_fingerprint, status, authorized_at) SELECT g.id, $1, $2, 'authorized', $3 FROM runtime_target_agent_delivery_grants g WHERE g.token_verifier = $4 AND g.status = 'delivered' AND g.expires_at > $3 AND g.deleted_at = 0 ON CONFLICT (delivery_grant_id) DO NOTHING`, strings.TrimSpace(issuanceKey), strings.TrimSpace(csrFingerprint), now.UTC(), strings.TrimSpace(tokenVerifier))
		if err != nil {
			return fmt.Errorf("authorize agent certificate issuance: %w", err)
		}
		created, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read agent certificate issuance authorization result: %w", err)
		}
		authorization, err = r.readAgentBootstrapAuthorization(txCtx, tokenVerifier)
		if err != nil {
			return err
		}
		if authorization.Issuance.CSRPublicKeyFingerprint != strings.TrimSpace(csrFingerprint) {
			return ErrAgentDeliveryRejected
		}
		replay = created == 0
		return nil
	})
	return authorization, replay, err
}

// RecordIssuedAgentCertificate 保存 Vault 已完成的非秘密签发证据，并允许相同证据的幂等恢复。
func (r *SQLRepository) RecordIssuedAgentCertificate(ctx context.Context, issuance AgentCertificateIssuance, now time.Time) (AgentCertificateIssuance, bool, error) {
	if r == nil || r.db == nil || !validIssuedAgentCertificate(issuance, now) {
		return AgentCertificateIssuance{}, false, ErrAgentDeliveryRejected
	}
	var recorded AgentCertificateIssuance
	var replay bool
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		result, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_certificate_issuances SET status = 'issued', certificate_issuer = $1, certificate_serial = $2, certificate_public_key_fingerprint = $3, certificate_expires_at = $4, trust_bundle_ref = $5, trust_bundle_version = $6, trust_bundle_expires_at = $7, issued_at = $8, updated_at = $8 WHERE issuance_key = $9 AND status = 'authorized' AND deleted_at = 0`, strings.TrimSpace(issuance.CertificateIssuer), strings.TrimSpace(issuance.CertificateSerial), strings.TrimSpace(issuance.CertificatePublicKeyFingerprint), issuance.CertificateExpiresAt.UTC(), strings.TrimSpace(issuance.TrustBundleRef), strings.TrimSpace(issuance.TrustBundleVersion), issuance.TrustBundleExpiresAt.UTC(), now.UTC(), strings.TrimSpace(issuance.IssuanceKey))
		if err != nil {
			return fmt.Errorf("record issued agent certificate: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read issued agent certificate result: %w", err)
		}
		recorded, err = r.readAgentCertificateIssuanceByKey(txCtx, issuance.IssuanceKey)
		if err != nil {
			return err
		}
		if affected == 0 {
			if !sameIssuedAgentCertificate(recorded, issuance) {
				return ErrAgentDeliveryRejected
			}
			replay = true
		}
		return nil
	})
	return recorded, replay, err
}

// CompleteAgentCertificateIssuance 原子激活对应世代、消费投递授权并完成签发协调状态。
//
//nolint:gocognit,gocyclo,cyclop // 证书激活、令牌消费和协调完成不可拆成跨事务可观察步骤。
func (r *SQLRepository) CompleteAgentCertificateIssuance(ctx context.Context, issuanceKey string, now time.Time) (AgentBootstrapAuthorization, bool, error) {
	if r == nil || r.db == nil || strings.TrimSpace(issuanceKey) == "" || now.IsZero() {
		return AgentBootstrapAuthorization{}, false, ErrAgentDeliveryRejected
	}
	var authorization AgentBootstrapAuthorization
	var replay bool
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		var err error
		authorization, err = r.readAgentBootstrapAuthorizationByKey(txCtx, issuanceKey)
		if err != nil {
			return err
		}
		if authorization.Issuance.Status == "completed" {
			replay = true
			return nil
		}
		if authorization.Issuance.Status != "issued" || authorization.Grant.Status != "delivered" {
			return ErrAgentDeliveryRejected
		}
		generationResult, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'active', certificate_issuer = $1, certificate_serial = $2, public_key_fingerprint = $3, activated_at = $4, updated_at = $4 WHERE id = $5 AND status = 'pending' AND expires_at > $4 AND deleted_at = 0`, authorization.Issuance.CertificateIssuer, authorization.Issuance.CertificateSerial, authorization.Issuance.CertificatePublicKeyFingerprint, now.UTC(), authorization.Generation.ID)
		if err != nil {
			return fmt.Errorf("activate agent generation after certificate issuance: %w", err)
		}
		if affected, err := generationResult.RowsAffected(); err != nil || affected != 1 {
			return ErrAgentDeliveryRejected
		}
		grantResult, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_delivery_grants SET status = 'consumed', consumed_at = $1, updated_at = $1 WHERE id = $2 AND status = 'delivered' AND deleted_at = 0`, now.UTC(), authorization.Grant.ID)
		if err != nil {
			return fmt.Errorf("consume agent delivery grant: %w", err)
		}
		if affected, err := grantResult.RowsAffected(); err != nil || affected != 1 {
			return ErrAgentDeliveryRejected
		}
		completionResult, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_certificate_issuances SET status = 'completed', completed_at = $1, updated_at = $1 WHERE id = $2 AND status = 'issued' AND deleted_at = 0`, now.UTC(), authorization.Issuance.ID)
		if err != nil {
			return fmt.Errorf("complete agent certificate issuance: %w", err)
		}
		if affected, err := completionResult.RowsAffected(); err != nil || affected != 1 {
			return ErrAgentDeliveryRejected
		}
		authorization.Grant.Status = "consumed"
		authorization.Issuance.Status = "completed"
		authorization.Issuance.CompletedAt = timePtr(now.UTC())
		authorization.Generation.Status = "active"
		return nil
	})
	return authorization, replay, err
}

func (r *SQLRepository) readAgentBootstrapAuthorization(ctx context.Context, tokenVerifier string) (AgentBootstrapAuthorization, error) {
	return r.readAgentBootstrapAuthorizationWhere(ctx, "d.token_verifier = $1", strings.TrimSpace(tokenVerifier))
}

func (r *SQLRepository) readAgentBootstrapAuthorizationByKey(ctx context.Context, issuanceKey string) (AgentBootstrapAuthorization, error) {
	return r.readAgentBootstrapAuthorizationWhere(ctx, "s.issuance_key = $1", strings.TrimSpace(issuanceKey))
}

func (r *SQLRepository) readAgentBootstrapAuthorizationWhere(ctx context.Context, predicate string, argument string) (AgentBootstrapAuthorization, error) {
	var result AgentBootstrapAuthorization
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT s.id, s.delivery_grant_id, s.issuance_key, s.csr_public_key_fingerprint, s.status, s.certificate_issuer, s.certificate_serial, s.certificate_public_key_fingerprint, s.certificate_expires_at, s.trust_bundle_ref, s.trust_bundle_version, s.trust_bundle_expires_at, s.authorized_at, s.issued_at, s.completed_at, d.id, d.generation_id, d.grant_id, d.token_verifier, d.expected_automation_id, d.docker_installation_ref, d.expires_at, d.status, d.handoff_id, d.handed_off_at, d.delivered_at, d.consumed_at, d.revoked_at, d.revoked_reason, `+agentTrustGenerationSelectColumns+` FROM runtime_target_agent_certificate_issuances s INNER JOIN runtime_target_agent_delivery_grants d ON d.id = s.delivery_grant_id INNER JOIN runtime_target_agent_generations g ON g.id = d.generation_id INNER JOIN runtime_target_agent_identities i ON i.id = g.identity_id WHERE `+predicate+` AND s.deleted_at = 0 AND d.deleted_at = 0 AND g.deleted_at = 0 AND i.deleted_at = 0`, argument).Scan(&result.Issuance.ID, &result.Issuance.DeliveryGrantID, &result.Issuance.IssuanceKey, &result.Issuance.CSRPublicKeyFingerprint, &result.Issuance.Status, &result.Issuance.CertificateIssuer, &result.Issuance.CertificateSerial, &result.Issuance.CertificatePublicKeyFingerprint, &result.Issuance.CertificateExpiresAt, &result.Issuance.TrustBundleRef, &result.Issuance.TrustBundleVersion, &result.Issuance.TrustBundleExpiresAt, &result.Issuance.AuthorizedAt, &result.Issuance.IssuedAt, &result.Issuance.CompletedAt, &result.Grant.ID, &result.Grant.GenerationID, &result.Grant.GrantID, &result.Grant.TokenVerifier, &result.Grant.ExpectedAutomationID, &result.Grant.DockerInstallationRef, &result.Grant.ExpiresAt, &result.Grant.Status, &result.Grant.HandoffID, &result.Grant.HandedOffAt, &result.Grant.DeliveredAt, &result.Grant.ConsumedAt, &result.Grant.RevokedAt, &result.Grant.RevokedReason, &result.Generation.Identity.ID, &result.Generation.Identity.IdentityID, &result.Generation.Identity.TargetID, &result.Generation.Identity.AgentID, &result.Generation.Identity.ProviderID, &result.Generation.Identity.BuilderScope, &result.Generation.Identity.CapabilityProfile, &result.Generation.Identity.CapabilityVersion, &result.Generation.Identity.ImageDigest, &result.Generation.Identity.AgentVersion, &result.Generation.ID, &result.Generation.Generation, &result.Generation.EnrollmentRef, &result.Generation.TrustBundleRef, &result.Generation.TrustBundleVersion, &result.Generation.CertificateIssuer, &result.Generation.CertificateSerial, &result.Generation.PublicKeyFingerprint, &result.Generation.ExpiresAt, &result.Generation.Status, &result.Generation.ActivatedAt, &result.Generation.RetiredAt, &result.Generation.RevokedAt, &result.Generation.RevokedReason)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentBootstrapAuthorization{}, ErrAgentDeliveryRejected
	}
	if err != nil {
		return AgentBootstrapAuthorization{}, fmt.Errorf("read agent bootstrap authorization: %w", err)
	}
	return result, nil
}

func (r *SQLRepository) readAgentCertificateIssuanceByKey(ctx context.Context, issuanceKey string) (AgentCertificateIssuance, error) {
	var issuance AgentCertificateIssuance
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT id, delivery_grant_id, issuance_key, csr_public_key_fingerprint, status, certificate_issuer, certificate_serial, certificate_public_key_fingerprint, certificate_expires_at, trust_bundle_ref, trust_bundle_version, trust_bundle_expires_at, authorized_at, issued_at, completed_at FROM runtime_target_agent_certificate_issuances WHERE issuance_key = $1 AND deleted_at = 0`, strings.TrimSpace(issuanceKey)).Scan(&issuance.ID, &issuance.DeliveryGrantID, &issuance.IssuanceKey, &issuance.CSRPublicKeyFingerprint, &issuance.Status, &issuance.CertificateIssuer, &issuance.CertificateSerial, &issuance.CertificatePublicKeyFingerprint, &issuance.CertificateExpiresAt, &issuance.TrustBundleRef, &issuance.TrustBundleVersion, &issuance.TrustBundleExpiresAt, &issuance.AuthorizedAt, &issuance.IssuedAt, &issuance.CompletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentCertificateIssuance{}, ErrAgentDeliveryRejected
	}
	return issuance, err
}

func validIssuedAgentCertificate(issuance AgentCertificateIssuance, now time.Time) bool {
	return strings.TrimSpace(issuance.IssuanceKey) != "" && strings.TrimSpace(issuance.CertificateIssuer) != "" && strings.TrimSpace(issuance.CertificateSerial) != "" && strings.TrimSpace(issuance.CertificatePublicKeyFingerprint) != "" && issuance.CertificateExpiresAt != nil && issuance.CertificateExpiresAt.After(now) && strings.TrimSpace(issuance.TrustBundleRef) != "" && strings.TrimSpace(issuance.TrustBundleVersion) != "" && issuance.TrustBundleExpiresAt != nil && issuance.TrustBundleExpiresAt.After(now)
}

//nolint:cyclop // 幂等重放需要逐一比较所有持久化的非秘密签发证据。
func sameIssuedAgentCertificate(left, right AgentCertificateIssuance) bool {
	return left.IssuanceKey == strings.TrimSpace(right.IssuanceKey) && left.CertificateIssuer == strings.TrimSpace(right.CertificateIssuer) && left.CertificateSerial == strings.TrimSpace(right.CertificateSerial) && left.CertificatePublicKeyFingerprint == strings.TrimSpace(right.CertificatePublicKeyFingerprint) && left.CertificateExpiresAt != nil && right.CertificateExpiresAt != nil && left.CertificateExpiresAt.Equal(right.CertificateExpiresAt.UTC()) && left.TrustBundleRef == strings.TrimSpace(right.TrustBundleRef) && left.TrustBundleVersion == strings.TrimSpace(right.TrustBundleVersion) && left.TrustBundleExpiresAt != nil && right.TrustBundleExpiresAt != nil && left.TrustBundleExpiresAt.Equal(right.TrustBundleExpiresAt.UTC())
}

func timePtr(value time.Time) *time.Time { return &value }
