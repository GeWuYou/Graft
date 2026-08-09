package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const agentLedgerReceiptMaxLifetime = 5 * time.Minute
const agentLedgerImplementationVersionMaxLength = 255
const agentLedgerDiagnosticMaxLength = 1024

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
		generation, err := r.ReadActiveAgentTrustGeneration(txCtx, identity.TargetID, identity.AgentID, identity.Generation, now)
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
		generation, err := r.ReadActiveAgentTrustGeneration(txCtx, identity.TargetID, identity.AgentID, identity.Generation, now)
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
		err = r.executor(txCtx).QueryRowContext(txCtx, `SELECT receipt_fingerprint FROM runtime_target_agent_ledger_snapshots WHERE snapshot_id = $1 AND snapshot_digest = $2 AND generation_id = $3 AND consumed_at IS NOT NULL AND deleted_at = 0`, strings.TrimSpace(receipt.SnapshotID), strings.TrimSpace(receipt.SnapshotDigest), generation.ID).Scan(&existing)
		if err != nil || existing != fingerprint {
			return ErrAgentTrustNotActive
		}
		return nil
	})
}

func validAgentLedgerIdentity(identity AgentLedgerIdentity) bool {
	return strings.TrimSpace(identity.IdentityID) != "" && identity.TargetID > 0 && strings.TrimSpace(identity.AgentID) != "" && identity.Generation > 0 && strings.TrimSpace(identity.CertificateSerial) != "" && strings.TrimSpace(identity.PublicKeyFingerprint) != ""
}
func validAgentLedgerSnapshotID(value string) bool { return validSHA256Hex(value) }
func validAgentTelemetryReceipt(receipt AgentTelemetryReceiptInput, now time.Time) bool {
	return validAgentLedgerSnapshotID(receipt.SnapshotID) && validSHA256Hex(receipt.SnapshotDigest) && !receipt.ObservedAt.IsZero() && receipt.ExpiresAt.After(receipt.ObservedAt) && receipt.ExpiresAt.After(now) && !receipt.ExpiresAt.After(now.Add(agentLedgerReceiptMaxLifetime)) && len(strings.TrimSpace(receipt.ImplementationVersion)) <= agentLedgerImplementationVersionMaxLength && len(strings.TrimSpace(receipt.Diagnostic)) <= agentLedgerDiagnosticMaxLength
}
func sameAgentLedgerGeneration(identity AgentLedgerIdentity, generation AgentTrustGeneration) bool {
	return identity.IdentityID == generation.Identity.IdentityID && identity.CertificateSerial == generation.CertificateSerial && identity.PublicKeyFingerprint == generation.PublicKeyFingerprint
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
