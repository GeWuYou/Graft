package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const agentLedgerReceiptMaxLifetime = 5 * time.Minute
const agentLedgerImplementationVersionMaxLength = 255
const agentLedgerDiagnosticCodeMaxLength = 64

// Agent 遥测回执的稳定 Diagnostic 分类；claim admission 三态与既有健康语义并存，
// 且会被 Runtime Target 的就绪投影消费，故保持导出。
const (
	agentLedgerDiagnosticCodeUnavailable = "unavailable"
	agentLedgerDiagnosticCodeDegraded    = "degraded"

	// AgentLedgerDiagnosticNoWork 表示 Agent 存活但当前没有可领取的外部执行工作。
	AgentLedgerDiagnosticNoWork = "no_work"
	// AgentLedgerDiagnosticIdentityRejected 表示最近一次领取被目标绑定或身份校验拒绝。
	AgentLedgerDiagnosticIdentityRejected = "identity_rejected"
	// AgentLedgerDiagnosticTransportFailed 表示最近一次领取因网络、超时或服务端故障失败。
	AgentLedgerDiagnosticTransportFailed = "transport_failed"
)

// AgentLedgerIdentity 是由 mTLS listener 提供并由 Runtime Target 再次核验的身份与证书证据。
type AgentLedgerIdentity struct {
	IdentityID           string
	TargetID             int64
	AgentID              string
	Generation           int64
	CertificateSerial    string
	PublicKeyFingerprint string
}

// AgentLedgerSnapshot 是 Runtime Target 保存的单次账本快照。
type AgentLedgerSnapshot struct {
	IdentityID, AgentID, SnapshotID, SnapshotDigest, BuilderScope, ProviderID, CapabilityProfile, CapabilityVersion, AffinityKey string
	TargetID, Generation, Sequence                                                                                               int64
	Available                                                                                                                    bool
	Running, Queued, AllocatableSlots                                                                                            int
	ObservedAt, ExpiresAt, IssuedAt                                                                                              time.Time
}

// AgentTelemetryReceiptInput 是与既有快照绑定的受限遥测回执。
type AgentTelemetryReceiptInput struct {
	SnapshotID, SnapshotDigest, ImplementationVersion, Diagnostic string
	ObservedAt, ExpiresAt                                         time.Time
	Available                                                     bool
}

// IssueAgentLedgerSnapshot 为精确的活动 Docker 证书身份持久化一次性账本快照。
//
//nolint:cyclop // 证书世代、序列推进和快照持久化必须在一个事务中失败关闭。
func (r *SQLRepository) IssueAgentLedgerSnapshot(ctx context.Context, identity AgentLedgerIdentity, snapshotID string, now, expiresAt time.Time) (AgentLedgerSnapshot, error) {
	if r == nil || r.db == nil || !validAgentLedgerIdentity(identity) || !validAgentLedgerSnapshotID(snapshotID) || now.IsZero() || !expiresAt.After(now) {
		return AgentLedgerSnapshot{}, ErrAgentTrustNotActive
	}
	var snapshot AgentLedgerSnapshot
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		generation, err := r.ReadActiveAgentTrustGenerationForLedgerMutation(txCtx, identity.TargetID, identity.AgentID, identity.Generation, now)
		if err != nil || !sameAgentLedgerGeneration(identity, generation) || generation.Identity.ProviderID != "docker" {
			return ErrAgentTrustNotActive
		}
		state, err := r.AdvanceBuilderAgentTelemetry(txCtx, identity.TargetID, identity.AgentID)
		if err != nil {
			return err
		}
		snapshot = AgentLedgerSnapshot{IdentityID: generation.Identity.IdentityID, TargetID: identity.TargetID, AgentID: identity.AgentID, Generation: identity.Generation, Sequence: state.TelemetrySequence, SnapshotID: strings.TrimSpace(snapshotID), BuilderScope: generation.Identity.BuilderScope, ProviderID: generation.Identity.ProviderID, CapabilityProfile: generation.Identity.CapabilityProfile, CapabilityVersion: generation.Identity.CapabilityVersion, AffinityKey: "docker-agent:" + identity.AgentID, Available: state.Running < state.SlotBudget, Running: state.Running, Queued: state.Queued, AllocatableSlots: state.SlotBudget, ObservedAt: now.UTC(), ExpiresAt: expiresAt.UTC(), IssuedAt: now.UTC()}
		snapshot.SnapshotDigest = agentLedgerSnapshotDigest(snapshot)
		_, err = r.executor(txCtx).ExecContext(txCtx, `INSERT INTO runtime_target_agent_ledger_snapshots (generation_id, snapshot_id, snapshot_digest, sequence, builder_scope, provider_id, capability_profile, capability_version, affinity_key, available, running, queued, allocatable_slots, observed_at, expires_at, issued_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, generation.ID, snapshot.SnapshotID, snapshot.SnapshotDigest, snapshot.Sequence, snapshot.BuilderScope, snapshot.ProviderID, snapshot.CapabilityProfile, snapshot.CapabilityVersion, snapshot.AffinityKey, snapshot.Available, snapshot.Running, snapshot.Queued, snapshot.AllocatableSlots, snapshot.ObservedAt, snapshot.ExpiresAt, snapshot.IssuedAt)
		if err != nil {
			return fmt.Errorf("persist agent ledger snapshot: %w", err)
		}
		return nil
	})
	return snapshot, err
}

// RecordAgentTelemetryReceipt 消费匹配快照；相同回执允许重试，变化内容或过期快照失败关闭。
//
//nolint:cyclop // 首次消费、等价重试和变化重放必须共享同一个持久化判断边界。
func (r *SQLRepository) RecordAgentTelemetryReceipt(ctx context.Context, identity AgentLedgerIdentity, receipt AgentTelemetryReceiptInput, now time.Time) error {
	if r == nil || r.db == nil || !validAgentLedgerIdentity(identity) || !validAgentTelemetryReceipt(receipt, now) || now.IsZero() {
		return ErrAgentTrustNotActive
	}
	fingerprint := agentTelemetryReceiptDigest(receipt)
	return r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		generation, err := r.ReadActiveAgentTrustGenerationForLedgerMutation(txCtx, identity.TargetID, identity.AgentID, identity.Generation, now)
		if err != nil || !sameAgentLedgerGeneration(identity, generation) || generation.Identity.ProviderID != "docker" {
			return ErrAgentTrustNotActive
		}
		result, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_ledger_snapshots SET consumed_at = $1, receipt_fingerprint = $2, receipt_observed_at = $3, receipt_expires_at = $4, receipt_available = $5, receipt_implementation_version = $6, receipt_diagnostic = $7, updated_at = $1 WHERE snapshot_id = $8 AND snapshot_digest = $9 AND generation_id = $10 AND consumed_at IS NULL AND expires_at > $1 AND deleted_at = 0`, now.UTC(), fingerprint, receipt.ObservedAt.UTC(), receipt.ExpiresAt.UTC(), receipt.Available, strings.TrimSpace(receipt.ImplementationVersion), strings.TrimSpace(receipt.Diagnostic), strings.TrimSpace(receipt.SnapshotID), strings.TrimSpace(receipt.SnapshotDigest), generation.ID)
		if err != nil {
			return fmt.Errorf("consume agent ledger snapshot: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read agent ledger receipt result: %w", err)
		}
		if affected == 1 {
			return nil
		}
		var existing string
		err = r.executor(txCtx).QueryRowContext(txCtx, `SELECT receipt_fingerprint FROM runtime_target_agent_ledger_snapshots WHERE snapshot_id = $1 AND snapshot_digest = $2 AND generation_id = $3 AND consumed_at IS NOT NULL AND expires_at > $4 AND deleted_at = 0`, strings.TrimSpace(receipt.SnapshotID), strings.TrimSpace(receipt.SnapshotDigest), generation.ID, now.UTC()).Scan(&existing)
		if err != nil || existing != fingerprint {
			return ErrAgentTrustNotActive
		}
		return nil
	})
}

// ListActiveDockerAgentLedgerSnapshots projects only fresh, consumed ledger receipts
// from the current active Docker Agent generation. It is the sole scheduling read
// path for Builder telemetry; legacy telemetry rows and host/provider metrics stay
// outside this boundary. An idle agent reports the bounded no_work admission class
// and remains schedulable. Eligibility is decided by each target's LATEST snapshot:
// the query ranks snapshots per target by sequence first, then applies the receipt
// diagnostic/expiry filter to that latest row, so a stale healthy receipt can never
// mask a newer degraded or expired one.
//
//nolint:cyclop // 动态账本投影必须在一个边界内限制目标集合、活动世代和每目标最新回执。
func (r *SQLRepository) ListActiveDockerAgentLedgerSnapshots(ctx context.Context, targetIDs []int64, now time.Time) ([]AgentLedgerSnapshot, error) {
	if r == nil || r.db == nil || len(targetIDs) == 0 || now.IsZero() {
		return []AgentLedgerSnapshot{}, nil
	}
	unique := make(map[int64]struct{}, len(targetIDs))
	args := make([]any, 0, len(targetIDs)+1)
	placeholders := make([]string, 0, len(targetIDs))
	for _, targetID := range targetIDs {
		if targetID < 1 {
			return []AgentLedgerSnapshot{}, nil
		}
		if _, found := unique[targetID]; found {
			continue
		}
		unique[targetID] = struct{}{}
		args = append(args, targetID)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	args = append(args, now.UTC())
	nowIndex := fmt.Sprint(len(args))
	query := `SELECT identity_id, runtime_target_id, agent_id, generation,
  sequence, snapshot_id, snapshot_digest, builder_scope, provider_id,
  capability_profile, capability_version, affinity_key, available,
  running, queued, allocatable_slots, receipt_observed_at,
  receipt_expires_at, issued_at
FROM (
  SELECT i.identity_id, i.runtime_target_id, i.agent_id, g.generation,
    s.sequence, s.snapshot_id, s.snapshot_digest, s.builder_scope, s.provider_id,
    s.capability_profile, s.capability_version, s.affinity_key, s.available,
    s.running, s.queued, s.allocatable_slots, s.receipt_observed_at,
    s.receipt_expires_at, s.issued_at, s.receipt_diagnostic,
    ROW_NUMBER() OVER (PARTITION BY i.runtime_target_id ORDER BY s.sequence DESC, s.id DESC) AS ledger_rank
  FROM runtime_target_agent_ledger_snapshots s
  INNER JOIN runtime_target_agent_generations g ON g.id = s.generation_id
  INNER JOIN runtime_target_agent_identities i ON i.id = g.identity_id
  WHERE i.runtime_target_id IN (` + strings.Join(placeholders, ",") + `)
    AND i.provider_id = 'docker'
    AND g.status = 'active' AND g.expires_at > $` + nowIndex + `
    AND s.consumed_at IS NOT NULL AND s.receipt_available = true
    AND s.deleted_at = 0 AND g.deleted_at = 0 AND i.deleted_at = 0
) latest
WHERE latest.ledger_rank = 1
  AND latest.receipt_diagnostic IN ('', '` + AgentLedgerDiagnosticNoWork + `') AND latest.receipt_expires_at > $` + nowIndex + `
ORDER BY latest.runtime_target_id ASC`
	rows, err := r.executor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active Docker agent ledger snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]AgentLedgerSnapshot, 0, len(unique))
	seen := make(map[int64]struct{}, len(unique))
	for rows.Next() {
		var snapshot AgentLedgerSnapshot
		if err := rows.Scan(&snapshot.IdentityID, &snapshot.TargetID, &snapshot.AgentID, &snapshot.Generation,
			&snapshot.Sequence, &snapshot.SnapshotID, &snapshot.SnapshotDigest, &snapshot.BuilderScope, &snapshot.ProviderID,
			&snapshot.CapabilityProfile, &snapshot.CapabilityVersion, &snapshot.AffinityKey, &snapshot.Available,
			&snapshot.Running, &snapshot.Queued, &snapshot.AllocatableSlots, &snapshot.ObservedAt, &snapshot.ExpiresAt,
			&snapshot.IssuedAt); err != nil {
			return nil, fmt.Errorf("scan active Docker agent ledger snapshot: %w", err)
		}
		if _, found := seen[snapshot.TargetID]; found {
			continue
		}
		seen[snapshot.TargetID] = struct{}{}
		result = append(result, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active Docker agent ledger snapshots: %w", err)
	}
	return result, nil
}

func validAgentLedgerIdentity(identity AgentLedgerIdentity) bool {
	return identity.TargetID > 0 && strings.TrimSpace(identity.AgentID) != "" && identity.Generation > 0 && strings.TrimSpace(identity.CertificateSerial) != "" && strings.TrimSpace(identity.PublicKeyFingerprint) != ""
}

// AgentLedgerReceiptDiagnostic 是最新新鲜回执的受限 claim admission 证据投影。
type AgentLedgerReceiptDiagnostic struct {
	Diagnostic string
	ObservedAt time.Time
}

// ReadLatestAgentLedgerReceiptDiagnostic returns the most recent consumed receipt
// diagnostic of the active Docker Agent generation for one target. The second
// return value is false when no fresh receipt exists yet.
func (r *SQLRepository) ReadLatestAgentLedgerReceiptDiagnostic(ctx context.Context, targetID int64, now time.Time) (AgentLedgerReceiptDiagnostic, bool, error) {
	if r == nil || r.db == nil || targetID < 1 || now.IsZero() {
		return AgentLedgerReceiptDiagnostic{}, false, nil
	}
	var receipt AgentLedgerReceiptDiagnostic
	err := r.db.QueryRowContext(ctx, `SELECT s.receipt_diagnostic, s.receipt_observed_at
FROM runtime_target_agent_ledger_snapshots s
INNER JOIN runtime_target_agent_generations g ON g.id = s.generation_id
INNER JOIN runtime_target_agent_identities i ON i.id = g.identity_id
WHERE i.runtime_target_id = $1 AND i.provider_id = 'docker' AND g.status = 'active'
  AND s.consumed_at IS NOT NULL AND s.receipt_expires_at > $2
  AND s.deleted_at = 0 AND g.deleted_at = 0 AND i.deleted_at = 0
ORDER BY s.sequence DESC LIMIT 1`, targetID, now.UTC()).Scan(&receipt.Diagnostic, &receipt.ObservedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentLedgerReceiptDiagnostic{}, false, nil
	}
	if err != nil {
		return AgentLedgerReceiptDiagnostic{}, false, fmt.Errorf("read latest agent ledger receipt diagnostic: %w", err)
	}
	return receipt, true, nil
}
func validAgentLedgerSnapshotID(value string) bool { return validSHA256Hex(value) }
func validAgentTelemetryReceipt(receipt AgentTelemetryReceiptInput, now time.Time) bool {
	return validAgentLedgerSnapshotID(receipt.SnapshotID) && validSHA256Hex(receipt.SnapshotDigest) && !receipt.ObservedAt.IsZero() && receipt.ExpiresAt.After(receipt.ObservedAt) && receipt.ExpiresAt.After(now) && !receipt.ExpiresAt.After(now.Add(agentLedgerReceiptMaxLifetime)) && len(strings.TrimSpace(receipt.ImplementationVersion)) <= agentLedgerImplementationVersionMaxLength && validAgentLedgerDiagnosticCode(receipt.Diagnostic)
}

func validAgentLedgerDiagnosticCode(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) > agentLedgerDiagnosticCodeMaxLength {
		return false
	}
	switch value {
	case "", agentLedgerDiagnosticCodeUnavailable, agentLedgerDiagnosticCodeDegraded,
		AgentLedgerDiagnosticNoWork, AgentLedgerDiagnosticIdentityRejected, AgentLedgerDiagnosticTransportFailed:
		return true
	default:
		return false
	}
}
func sameAgentLedgerGeneration(identity AgentLedgerIdentity, generation AgentTrustGeneration) bool {
	return (identity.IdentityID == "" || identity.IdentityID == generation.Identity.IdentityID) && identity.CertificateSerial == generation.CertificateSerial && identity.PublicKeyFingerprint == generation.PublicKeyFingerprint
}
func agentLedgerSnapshotDigest(snapshot AgentLedgerSnapshot) string {
	return sha256Hex(struct {
		IdentityID                                                                                       string
		TargetID, Generation, Sequence                                                                   int64
		AgentID, SnapshotID, BuilderScope, ProviderID, CapabilityProfile, CapabilityVersion, AffinityKey string
		Available                                                                                        bool
		Running, Queued, AllocatableSlots                                                                int
		ObservedAt, ExpiresAt, IssuedAt                                                                  string
	}{snapshot.IdentityID, snapshot.TargetID, snapshot.Generation, snapshot.Sequence, snapshot.AgentID, snapshot.SnapshotID, snapshot.BuilderScope, snapshot.ProviderID, snapshot.CapabilityProfile, snapshot.CapabilityVersion, snapshot.AffinityKey, snapshot.Available, snapshot.Running, snapshot.Queued, snapshot.AllocatableSlots, snapshot.ObservedAt.Format(time.RFC3339Nano), snapshot.ExpiresAt.Format(time.RFC3339Nano), snapshot.IssuedAt.Format(time.RFC3339Nano)})
}
func agentTelemetryReceiptDigest(receipt AgentTelemetryReceiptInput) string {
	return sha256Hex(struct {
		SnapshotID, SnapshotDigest, ImplementationVersion, Diagnostic, ObservedAt, ExpiresAt string
		Available                                                                            bool
	}{strings.TrimSpace(receipt.SnapshotID), strings.TrimSpace(receipt.SnapshotDigest), strings.TrimSpace(receipt.ImplementationVersion), strings.TrimSpace(receipt.Diagnostic), receipt.ObservedAt.UTC().Format(time.RFC3339Nano), receipt.ExpiresAt.UTC().Format(time.RFC3339Nano), receipt.Available})
}
func sha256Hex(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
