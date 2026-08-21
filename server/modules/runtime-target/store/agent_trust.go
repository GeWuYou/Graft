package store

import (
	"context"
	"database/sql"
	"encoding/json"
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
	Capabilities      []string
	RuntimeProtocol   string
	ImageDigest       string
	AgentVersion      string
}

// AgentCapabilityBinding 是 Runtime Target 持有的 provider-neutral Agent 能力集合。
// 执行准入只读取该事实，不再从实验期单 profile 字段推导能力。
type AgentCapabilityBinding struct {
	IdentityID        int64
	GenerationID      int64
	ProviderID        string
	Capabilities      []string
	CapabilityVersion string
}

// AgentTrustGeneration 是与稳定 Agent 绑定关联的一次不可复用信任世代。
type AgentTrustGeneration struct {
	ID                   int64
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

const agentTrustGenerationSelectColumns = `i.id, i.identity_id, i.runtime_target_id, i.agent_id, i.provider_id, i.builder_scope, i.capability_profile, i.capability_version, i.image_digest, i.agent_version, g.id, g.generation, g.enrollment_ref, g.trust_bundle_ref, g.trust_bundle_version, g.certificate_issuer, g.certificate_serial, g.public_key_fingerprint, g.expires_at, g.status, g.activated_at, g.retired_at, g.revoked_at, g.revoked_reason`

// CreatePendingAgentTrustGeneration 持久化由 Runtime Target 登记 authority 创建的非秘密待激活世代。
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
		var generationID int64
		err = r.executor(txCtx).QueryRowContext(txCtx, `INSERT INTO runtime_target_agent_generations (identity_id, generation, enrollment_ref, trust_bundle_ref, trust_bundle_version, expires_at, status) VALUES ($1,$2,$3,$4,$5,$6,'pending') RETURNING id`, identityID, generation.Generation, generation.EnrollmentRef, generation.TrustBundleRef, generation.TrustBundleVersion, generation.ExpiresAt.UTC()).Scan(&generationID)
		if err != nil {
			return fmt.Errorf("create runtime target agent trust generation: %w", err)
		}
		if err := r.UpsertAgentCapabilityBinding(txCtx, AgentCapabilityBinding{IdentityID: identityID, GenerationID: generationID, ProviderID: identity.ProviderID, Capabilities: identity.Capabilities, CapabilityVersion: identity.CapabilityVersion}); err != nil {
			return err
		}
		result = generation
		result.ID = generationID
		result.Identity = identity
		result.Status = "pending"
		return nil
	})
	return result, err
}

// UpsertAgentCapabilityBinding 在身份登记事务中建立不可改写的世代级 capability binding。
// 能力扩容只能随新世代写入，旧证书关联的世代事实不会被原地授予新能力。
//
//nolint:cyclop // 写入前校验、JSON 编码和冲突行数共同保证 binding 不被静默改写。
func (r *SQLRepository) UpsertAgentCapabilityBinding(ctx context.Context, binding AgentCapabilityBinding) error {
	capabilities, err := normalizeAgentCapabilities(binding.Capabilities)
	if r == nil || r.db == nil || binding.IdentityID < 1 || binding.GenerationID < 1 || strings.TrimSpace(binding.ProviderID) == "" || strings.TrimSpace(binding.CapabilityVersion) == "" || err != nil {
		return ErrAgentTrustNotActive
	}
	payload, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("encode runtime target agent capability binding: %w", err)
	}
	result, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_target_agent_capability_bindings (identity_id, generation_id, provider_id, capabilities, capability_version) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (generation_id) DO UPDATE SET updated_at = CURRENT_TIMESTAMP WHERE runtime_target_agent_capability_bindings.identity_id = excluded.identity_id AND runtime_target_agent_capability_bindings.provider_id = excluded.provider_id AND runtime_target_agent_capability_bindings.capabilities = excluded.capabilities AND runtime_target_agent_capability_bindings.capability_version = excluded.capability_version AND runtime_target_agent_capability_bindings.deleted_at = 0`, binding.IdentityID, binding.GenerationID, strings.TrimSpace(binding.ProviderID), string(payload), strings.TrimSpace(binding.CapabilityVersion))
	if err != nil {
		return fmt.Errorf("write runtime target agent capability binding: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read runtime target agent capability binding result: %w", err)
	}
	if affected != 1 {
		return ErrAgentTrustNotActive
	}
	return nil
}

// ReadAgentCapabilityBinding 读取 execution admission 使用的精确世代能力集合。
func (r *SQLRepository) ReadAgentCapabilityBinding(ctx context.Context, generationID int64) (AgentCapabilityBinding, error) {
	if r == nil || r.db == nil || generationID < 1 {
		return AgentCapabilityBinding{}, ErrAgentTrustNotFound
	}
	var binding AgentCapabilityBinding
	var payload []byte
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT identity_id, generation_id, provider_id, capabilities, capability_version FROM runtime_target_agent_capability_bindings WHERE generation_id = $1 AND deleted_at = 0`, generationID).Scan(&binding.IdentityID, &binding.GenerationID, &binding.ProviderID, &payload, &binding.CapabilityVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentCapabilityBinding{}, ErrAgentTrustNotFound
	}
	if err != nil {
		return AgentCapabilityBinding{}, fmt.Errorf("read runtime target agent capability binding: %w", err)
	}
	if err := json.Unmarshal(payload, &binding.Capabilities); err != nil {
		return AgentCapabilityBinding{}, fmt.Errorf("decode runtime target agent capability binding: %w", err)
	}
	capabilities, err := normalizeAgentCapabilities(binding.Capabilities)
	if err != nil {
		return AgentCapabilityBinding{}, ErrAgentTrustNotActive
	}
	binding.Capabilities = capabilities
	return binding, nil
}

func normalizeAgentCapabilities(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, ErrAgentTrustNotActive
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		capability := strings.TrimSpace(value)
		if capability == "" {
			return nil, ErrAgentTrustNotActive
		}
		if _, exists := seen[capability]; exists {
			continue
		}
		seen[capability] = struct{}{}
		result = append(result, capability)
	}
	return result, nil
}

// ActivateAgentTrustGeneration 原子停用先前活动世代并激活已由 Runtime Target 核验过证书元数据的新世代。
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
		_, err = r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'retired', retired_at = $1, updated_at = $1, updated_by = $2 WHERE identity_id = $3 AND status = 'active' AND deleted_at = 0`, now.UTC(), actorID, identity.ID)
		if err != nil {
			return fmt.Errorf("retire active runtime target agent trust generation: %w", err)
		}
		result, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_generations SET status = 'active', certificate_issuer = $1, certificate_serial = $2, public_key_fingerprint = $3, activated_at = $4, updated_at = $4, updated_by = $5 WHERE identity_id = $6 AND generation = $7 AND status = 'pending' AND deleted_at = 0 AND expires_at > $4`, strings.TrimSpace(certificateIssuer), strings.TrimSpace(certificateSerial), strings.TrimSpace(fingerprint), now.UTC(), actorID, identity.ID, generation)
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
		_, _, err := r.RevokeAgentTrustGenerationTx(txCtx, nil, targetID, agentID, generation, reason, actorID, now)
		return err
	})
}

// RevokeAgentTrustGenerationTx 在调用方事务中撤销世代，并返回证书序列号与是否发生状态变化。
// 调用方可据此把撤销事实与本地状态放入同一 durable event 事务。
//
//nolint:revive,cyclop // 事务 helper 需要显式携带完整审计与世代边界，避免引入宽泛请求对象。
func (r *SQLRepository) RevokeAgentTrustGenerationTx(ctx context.Context, tx *sql.Tx, targetID int64, agentID string, generation int64, reason string, actorID int64, now time.Time) (string, bool, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" || generation < 1 || now.IsZero() {
		return "", false, ErrAgentTrustNotActive
	}
	if tx != nil {
		ctx = context.WithValue(ctx, transactionContextKey{}, tx)
	}
	identity, err := r.findAgentTrustIdentity(ctx, targetID, agentID)
	if err != nil {
		return "", false, err
	}
	var serial, status string
	err = r.executor(ctx).QueryRowContext(ctx, `SELECT certificate_serial, status FROM runtime_target_agent_generations WHERE identity_id = $1 AND generation = $2 AND deleted_at = 0`, identity.ID, generation).Scan(&serial, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, ErrAgentTrustNotFound
	}
	if err != nil {
		return "", false, fmt.Errorf("read runtime target agent trust generation: %w", err)
	}
	if status == "revoked" {
		return serial, false, nil
	}
	result, err := r.executor(ctx).ExecContext(ctx, `UPDATE runtime_target_agent_generations SET status = 'revoked', revoked_at = COALESCE(revoked_at, $1), revoked_reason = CASE WHEN revoked_reason = '' THEN $2 ELSE revoked_reason END, updated_at = $1, updated_by = $3 WHERE identity_id = $4 AND generation = $5 AND deleted_at = 0 AND status <> 'revoked'`, now.UTC(), strings.TrimSpace(reason), actorID, identity.ID, generation)
	if err != nil {
		return "", false, fmt.Errorf("revoke runtime target agent trust generation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return "", false, fmt.Errorf("read runtime target agent trust revocation result: %w", err)
	}
	return serial, affected == 1, nil
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
	row := r.executor(ctx).QueryRowContext(ctx, `SELECT `+agentTrustGenerationSelectColumns+` FROM runtime_target_agent_identities i INNER JOIN runtime_target_agent_generations g ON g.identity_id = i.id WHERE i.runtime_target_id = $1 AND i.agent_id = $2 AND g.generation = $3 AND i.deleted_at = 0 AND g.deleted_at = 0 AND g.status = 'active' AND g.revoked_at IS NULL AND g.retired_at IS NULL AND g.expires_at > $4`, targetID, agentID, generation, now.UTC())
	return scanAgentTrustGeneration(row)
}

// ReadActiveAgentTrustGenerationByCertificate resolves lifecycle generation from certificate evidence.
// The workload URI identifies only the stable Agent; serial and public-key fingerprint bind the current generation.
func (r *SQLRepository) ReadActiveAgentTrustGenerationByCertificate(ctx context.Context, targetID int64, agentID, certificateSerial, publicKeyFingerprint string, now time.Time) (AgentTrustGeneration, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" || strings.TrimSpace(certificateSerial) == "" || strings.TrimSpace(publicKeyFingerprint) == "" || now.IsZero() {
		return AgentTrustGeneration{}, ErrAgentTrustNotActive
	}
	row := r.executor(ctx).QueryRowContext(ctx, `SELECT `+agentTrustGenerationSelectColumns+` FROM runtime_target_agent_identities i INNER JOIN runtime_target_agent_generations g ON g.identity_id = i.id WHERE i.runtime_target_id = $1 AND i.agent_id = $2 AND g.certificate_serial = $3 AND g.public_key_fingerprint = $4 AND i.deleted_at = 0 AND g.deleted_at = 0 AND g.status = 'active' AND g.revoked_at IS NULL AND g.retired_at IS NULL AND g.expires_at > $5`, targetID, agentID, strings.TrimSpace(certificateSerial), strings.TrimSpace(publicKeyFingerprint), now.UTC())
	return scanAgentTrustGeneration(row)
}

// ReadActiveAgentTrustGenerationForLedgerMutation 在同一事务中固定活动世代，避免撤销与账本写入交错。
func (r *SQLRepository) ReadActiveAgentTrustGenerationForLedgerMutation(ctx context.Context, targetID int64, agentID string, generation int64, now time.Time) (AgentTrustGeneration, error) {
	active, err := r.ReadActiveAgentTrustGeneration(ctx, targetID, agentID, generation, now)
	if err != nil {
		return AgentTrustGeneration{}, err
	}
	result, err := r.executor(ctx).ExecContext(ctx, `UPDATE runtime_target_agent_generations SET status = status WHERE id = $1 AND status = 'active' AND revoked_at IS NULL AND retired_at IS NULL AND expires_at > $2 AND deleted_at = 0`, active.ID, now.UTC())
	if err != nil {
		return AgentTrustGeneration{}, fmt.Errorf("lock active runtime target agent trust generation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AgentTrustGeneration{}, fmt.Errorf("read active runtime target agent trust lock result: %w", err)
	}
	if affected != 1 {
		return AgentTrustGeneration{}, ErrAgentTrustNotActive
	}
	return active, nil
}

// ReadCurrentAgentTrustGeneration 返回最新世代的可读状态投影，不把它视为活动信任判断。
func (r *SQLRepository) ReadCurrentAgentTrustGeneration(ctx context.Context, targetID int64, agentID string) (AgentTrustGeneration, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" {
		return AgentTrustGeneration{}, ErrAgentTrustNotFound
	}
	row := r.executor(ctx).QueryRowContext(ctx, `SELECT `+agentTrustGenerationSelectColumns+` FROM runtime_target_agent_identities i INNER JOIN runtime_target_agent_generations g ON g.identity_id = i.id WHERE i.runtime_target_id = $1 AND i.agent_id = $2 AND i.deleted_at = 0 AND g.deleted_at = 0 ORDER BY g.generation DESC LIMIT 1`, targetID, agentID)
	return scanAgentTrustGeneration(row)
}

type agentTrustGenerationScanner interface {
	Scan(...any) error
}

func scanAgentTrustGeneration(row agentTrustGenerationScanner) (AgentTrustGeneration, error) {
	var generation AgentTrustGeneration
	err := row.Scan(&generation.Identity.ID, &generation.Identity.IdentityID, &generation.Identity.TargetID, &generation.Identity.AgentID, &generation.Identity.ProviderID, &generation.Identity.BuilderScope, &generation.Identity.CapabilityProfile, &generation.Identity.CapabilityVersion, &generation.Identity.ImageDigest, &generation.Identity.AgentVersion, &generation.ID, &generation.Generation, &generation.EnrollmentRef, &generation.TrustBundleRef, &generation.TrustBundleVersion, &generation.CertificateIssuer, &generation.CertificateSerial, &generation.PublicKeyFingerprint, &generation.ExpiresAt, &generation.Status, &generation.ActivatedAt, &generation.RetiredAt, &generation.RevokedAt, &generation.RevokedReason)
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
