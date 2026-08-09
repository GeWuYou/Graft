package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrAgentTrustNotFound 表示指定的 Agent 信任记录不存在或不再可读取。
var ErrAgentTrustNotFound = errors.New("runtime target agent trust not found")

// ErrAgentTrustNotActive 表示身份世代不是当前可受信任的活动世代。
var ErrAgentTrustNotActive = errors.New("runtime target agent trust generation is not active")

// AgentTrustIdentity 是 Runtime Target 持有的非秘密稳定 Agent 绑定。
type AgentTrustIdentity struct {
	ID                int64
	IdentityID        string
	TargetID          int64
	AgentID           string
	ProviderID        string
	BuilderScope      string
	CapabilityProfile string
	CapabilityVersion string
	ImageDigest       string
	AgentVersion      string
}

// AgentTrustGeneration 是与稳定 Agent 绑定关联的一次不可复用信任世代。
type AgentTrustGeneration struct {
	Identity             AgentTrustIdentity
	Generation           int64
	EnrollmentRef        string
	TrustBundleRef       string
	TrustBundleVersion   string
	CertificateIssuer    string
	CertificateSerial    string
	PublicKeyFingerprint string
	ExpiresAt            time.Time
	Status               string
	ActivatedAt          *time.Time
	RetiredAt            *time.Time
	RevokedAt            *time.Time
	RevokedReason        string
}

// CreatePendingAgentTrustGeneration 持久化由外部身份 authority 创建的非秘密待激活世代。
// 它不签发证书、不保存引导材料，也不会把待激活世代暴露为可信身份。
//
//nolint:cyclop // 世代创建在同一事务边界验证并分配连续编号。
func (r *SQLRepository) CreatePendingAgentTrustGeneration(ctx context.Context, identity AgentTrustIdentity, generation AgentTrustGeneration) (AgentTrustGeneration, error) {
	if r == nil || r.db == nil || !validAgentTrustIdentity(identity) || !validPendingAgentTrustGeneration(generation) {
		return AgentTrustGeneration{}, ErrAgentTrustNotActive
	}
	var result AgentTrustGeneration
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		identityID, err := r.ensureAgentTrustIdentity(txCtx, identity)
		if err != nil {
			return err
		}
		var nextGeneration int64
		if err := r.executor(txCtx).QueryRowContext(txCtx, `SELECT COALESCE(MAX(generation), 0) + 1 FROM runtime_target_agent_generations WHERE identity_id = $1`, identityID).Scan(&nextGeneration); err != nil {
			return fmt.Errorf("allocate runtime target agent trust generation: %w", err)
		}
		if generation.Generation != nextGeneration {
			return ErrAgentTrustNotActive
		}
		if generation.Generation > 1 {
			if _, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'retired', retired_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE identity_id = $1 AND status = 'active' AND deleted_at = 0`, identityID); err != nil {
				return fmt.Errorf("retire active runtime target agent trust generation before rotation: %w", err)
			}
		}
		_, err = r.executor(txCtx).ExecContext(txCtx, `INSERT INTO runtime_target_agent_generations (identity_id, generation, enrollment_ref, trust_bundle_ref, trust_bundle_version, expires_at, status) VALUES ($1,$2,$3,$4,$5,$6,'pending')`, identityID, generation.Generation, generation.EnrollmentRef, generation.TrustBundleRef, generation.TrustBundleVersion, generation.ExpiresAt.UTC())
		if err != nil {
			return fmt.Errorf("create runtime target agent trust generation: %w", err)
		}
		result = generation
		result.Identity = identity
		result.Status = "pending"
		return nil
	})
	return result, err
}

// ActivateAgentTrustGeneration 原子停用先前活动世代并激活已被外部 authority 绑定证书元数据的新世代。
//
//nolint:cyclop,gocyclo,revive // 激活必须在同一事务内处理 retire 与 pending-to-active 转换。
func (r *SQLRepository) ActivateAgentTrustGeneration(ctx context.Context, targetID int64, agentID string, generation int64, certificateIssuer, certificateSerial, fingerprint string, actorID int64, now time.Time) error {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" || generation < 1 || strings.TrimSpace(certificateIssuer) == "" || strings.TrimSpace(certificateSerial) == "" || strings.TrimSpace(fingerprint) == "" || now.IsZero() {
		return ErrAgentTrustNotActive
	}
	return r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		identity, err := r.findAgentTrustIdentity(txCtx, targetID, agentID)
		if err != nil {
			return err
		}
		result, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'retired', retired_at = $1, updated_at = $1, updated_by = $2 WHERE identity_id = $3 AND status = 'active' AND deleted_at = 0`, now.UTC(), actorID, identity.ID)
		if err != nil {
			return fmt.Errorf("retire active runtime target agent trust generation: %w", err)
		}
		if _, err := result.RowsAffected(); err != nil {
			return fmt.Errorf("read retired runtime target agent trust generations: %w", err)
		}
		result, err = r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'active', certificate_issuer = $1, certificate_serial = $2, public_key_fingerprint = $3, activated_at = $4, updated_at = $4, updated_by = $5 WHERE identity_id = $6 AND generation = $7 AND status = 'pending' AND deleted_at = 0 AND expires_at > $4`, strings.TrimSpace(certificateIssuer), strings.TrimSpace(certificateSerial), strings.TrimSpace(fingerprint), now.UTC(), actorID, identity.ID, generation)
		if err != nil {
			return fmt.Errorf("activate runtime target agent trust generation: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read activated runtime target agent trust generation: %w", err)
		}
		if affected != 1 {
			return ErrAgentTrustNotActive
		}
		return nil
	})
}

// RevokeAgentTrustGeneration 将一个世代不可逆地移出可信集合；重复撤销保持幂等。
//
//nolint:cyclop,revive // 撤销在同一事务内处理幂等状态与审计事实。
func (r *SQLRepository) RevokeAgentTrustGeneration(ctx context.Context, targetID int64, agentID string, generation int64, reason string, actorID int64, now time.Time) error {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" || generation < 1 || now.IsZero() {
		return ErrAgentTrustNotActive
	}
	return r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		identity, err := r.findAgentTrustIdentity(txCtx, targetID, agentID)
		if err != nil {
			return err
		}
		result, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'revoked', revoked_at = COALESCE(revoked_at, $1), revoked_reason = CASE WHEN revoked_reason = '' THEN $2 ELSE revoked_reason END, updated_at = $1, updated_by = $3 WHERE identity_id = $4 AND generation = $5 AND deleted_at = 0 AND status <> 'revoked'`, now.UTC(), strings.TrimSpace(reason), actorID, identity.ID, generation)
		if err != nil {
			return fmt.Errorf("revoke runtime target agent trust generation: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read revoked runtime target agent trust generation: %w", err)
		}
		if affected == 0 {
			var status string
			err := r.executor(txCtx).QueryRowContext(txCtx, `SELECT status FROM runtime_target_agent_generations WHERE identity_id = $1 AND generation = $2 AND deleted_at = 0`, identity.ID, generation).Scan(&status)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAgentTrustNotFound
			}
			if err != nil {
				return fmt.Errorf("read runtime target agent trust generation: %w", err)
			}
		}
		return nil
	})
}

// RevokeAllAgentTrustGenerations 用于目标重置或重新绑定，确保任何旧世代不能恢复活动状态。
func (r *SQLRepository) RevokeAllAgentTrustGenerations(ctx context.Context, targetID int64, agentID, reason string, actorID int64, now time.Time) error {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" || now.IsZero() {
		return ErrAgentTrustNotActive
	}
	return r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		identity, err := r.findAgentTrustIdentity(txCtx, targetID, agentID)
		if err != nil {
			return err
		}
		_, err = r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'revoked', revoked_at = COALESCE(revoked_at, $1), revoked_reason = CASE WHEN revoked_reason = '' THEN $2 ELSE revoked_reason END, updated_at = $1, updated_by = $3 WHERE identity_id = $4 AND deleted_at = 0 AND status <> 'revoked'`, now.UTC(), strings.TrimSpace(reason), actorID, identity.ID)
		if err != nil {
			return fmt.Errorf("revoke all runtime target agent trust generations: %w", err)
		}
		return nil
	})
}

// ReadActiveAgentTrustGeneration 只返回当前可被后续 mTLS 核验使用的精确世代。
func (r *SQLRepository) ReadActiveAgentTrustGeneration(ctx context.Context, targetID int64, agentID string, generation int64, now time.Time) (AgentTrustGeneration, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" || generation < 1 || now.IsZero() {
		return AgentTrustGeneration{}, ErrAgentTrustNotActive
	}
	row := r.executor(ctx).QueryRowContext(ctx, `SELECT i.id, i.identity_id, i.runtime_target_id, i.agent_id, i.provider_id, i.builder_scope, i.capability_profile, i.capability_version, i.image_digest, i.agent_version, g.generation, g.enrollment_ref, g.trust_bundle_ref, g.trust_bundle_version, g.certificate_issuer, g.certificate_serial, g.public_key_fingerprint, g.expires_at, g.status, g.activated_at, g.retired_at, g.revoked_at, g.revoked_reason FROM runtime_target_agent_identities i INNER JOIN runtime_target_agent_generations g ON g.identity_id = i.id WHERE i.runtime_target_id = $1 AND i.agent_id = $2 AND g.generation = $3 AND i.deleted_at = 0 AND g.deleted_at = 0 AND g.status = 'active' AND g.revoked_at IS NULL AND g.retired_at IS NULL AND g.expires_at > $4`, targetID, agentID, generation, now.UTC())
	return scanAgentTrustGeneration(row)
}

// ReadCurrentAgentTrustGeneration 返回最新世代的可读状态投影，不把它视为活动信任判断。
func (r *SQLRepository) ReadCurrentAgentTrustGeneration(ctx context.Context, targetID int64, agentID string) (AgentTrustGeneration, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" {
		return AgentTrustGeneration{}, ErrAgentTrustNotFound
	}
	row := r.executor(ctx).QueryRowContext(ctx, `SELECT i.id, i.identity_id, i.runtime_target_id, i.agent_id, i.provider_id, i.builder_scope, i.capability_profile, i.capability_version, i.image_digest, i.agent_version, g.generation, g.enrollment_ref, g.trust_bundle_ref, g.trust_bundle_version, g.certificate_issuer, g.certificate_serial, g.public_key_fingerprint, g.expires_at, g.status, g.activated_at, g.retired_at, g.revoked_at, g.revoked_reason FROM runtime_target_agent_identities i INNER JOIN runtime_target_agent_generations g ON g.identity_id = i.id WHERE i.runtime_target_id = $1 AND i.agent_id = $2 AND i.deleted_at = 0 AND g.deleted_at = 0 ORDER BY g.generation DESC LIMIT 1`, targetID, agentID)
	return scanAgentTrustGeneration(row)
}

type agentTrustGenerationScanner interface {
	Scan(...any) error
}

func scanAgentTrustGeneration(row agentTrustGenerationScanner) (AgentTrustGeneration, error) {
	var generation AgentTrustGeneration
	err := row.Scan(&generation.Identity.ID, &generation.Identity.IdentityID, &generation.Identity.TargetID, &generation.Identity.AgentID, &generation.Identity.ProviderID, &generation.Identity.BuilderScope, &generation.Identity.CapabilityProfile, &generation.Identity.CapabilityVersion, &generation.Identity.ImageDigest, &generation.Identity.AgentVersion, &generation.Generation, &generation.EnrollmentRef, &generation.TrustBundleRef, &generation.TrustBundleVersion, &generation.CertificateIssuer, &generation.CertificateSerial, &generation.PublicKeyFingerprint, &generation.ExpiresAt, &generation.Status, &generation.ActivatedAt, &generation.RetiredAt, &generation.RevokedAt, &generation.RevokedReason)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentTrustGeneration{}, ErrAgentTrustNotFound
	}
	if err != nil {
		return AgentTrustGeneration{}, fmt.Errorf("read runtime target agent trust generation: %w", err)
	}
	return generation, nil
}

//nolint:cyclop // 现有身份与新建身份必须在同一处执行完整不可变字段比对。
func (r *SQLRepository) ensureAgentTrustIdentity(ctx context.Context, identity AgentTrustIdentity) (int64, error) {
	var existing AgentTrustIdentity
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT id, identity_id, runtime_target_id, agent_id, provider_id, builder_scope, capability_profile, capability_version, image_digest, agent_version FROM runtime_target_agent_identities WHERE identity_id = $1 AND deleted_at = 0`, identity.IdentityID).Scan(&existing.ID, &existing.IdentityID, &existing.TargetID, &existing.AgentID, &existing.ProviderID, &existing.BuilderScope, &existing.CapabilityProfile, &existing.CapabilityVersion, &existing.ImageDigest, &existing.AgentVersion)
	if err == nil {
		if existing.TargetID != identity.TargetID || existing.AgentID != identity.AgentID || existing.ProviderID != identity.ProviderID || existing.BuilderScope != identity.BuilderScope || existing.CapabilityProfile != identity.CapabilityProfile || existing.CapabilityVersion != identity.CapabilityVersion || existing.ImageDigest != identity.ImageDigest || existing.AgentVersion != identity.AgentVersion {
			return 0, ErrAgentTrustNotActive
		}
		return existing.ID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("read runtime target agent identity: %w", err)
	}
	var identityID int64
	err = r.executor(ctx).QueryRowContext(ctx, `INSERT INTO runtime_target_agent_identities (runtime_target_id, identity_id, agent_id, provider_id, builder_scope, capability_profile, capability_version, image_digest, agent_version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, identity.TargetID, identity.IdentityID, identity.AgentID, identity.ProviderID, identity.BuilderScope, identity.CapabilityProfile, identity.CapabilityVersion, identity.ImageDigest, identity.AgentVersion).Scan(&identityID)
	if err != nil {
		return 0, fmt.Errorf("create runtime target agent identity: %w", err)
	}
	return identityID, nil
}

func (r *SQLRepository) findAgentTrustIdentity(ctx context.Context, targetID int64, agentID string) (AgentTrustIdentity, error) {
	var identity AgentTrustIdentity
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT id, identity_id, runtime_target_id, agent_id, provider_id, builder_scope, capability_profile, capability_version, image_digest, agent_version FROM runtime_target_agent_identities WHERE runtime_target_id = $1 AND agent_id = $2 AND deleted_at = 0`, targetID, agentID).Scan(&identity.ID, &identity.IdentityID, &identity.TargetID, &identity.AgentID, &identity.ProviderID, &identity.BuilderScope, &identity.CapabilityProfile, &identity.CapabilityVersion, &identity.ImageDigest, &identity.AgentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentTrustIdentity{}, ErrAgentTrustNotFound
	}
	if err != nil {
		return AgentTrustIdentity{}, fmt.Errorf("read runtime target agent identity: %w", err)
	}
	return identity, nil
}

func validAgentTrustIdentity(identity AgentTrustIdentity) bool {
	return identity.TargetID > 0 && strings.TrimSpace(identity.IdentityID) != "" && strings.TrimSpace(identity.AgentID) != "" && strings.TrimSpace(identity.ProviderID) != "" && strings.TrimSpace(identity.BuilderScope) != "" && strings.TrimSpace(identity.CapabilityProfile) != "" && strings.TrimSpace(identity.CapabilityVersion) != ""
}

//nolint:cyclop // 待激活世代的完整性条件与空证书证据属于同一安全断言。
func validPendingAgentTrustGeneration(generation AgentTrustGeneration) bool {
	return generation.Generation > 0 && strings.TrimSpace(generation.EnrollmentRef) != "" && strings.TrimSpace(generation.TrustBundleRef) != "" && strings.TrimSpace(generation.TrustBundleVersion) != "" && !generation.ExpiresAt.IsZero() && generation.CertificateIssuer == "" && generation.CertificateSerial == "" && generation.PublicKeyFingerprint == "" && generation.ActivatedAt == nil && generation.RetiredAt == nil && generation.RevokedAt == nil
}
