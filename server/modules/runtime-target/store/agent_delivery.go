package store

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrAgentDeliveryRejected 表示投递授权或回执不满足其冻结的身份、时效或重放约束。
var ErrAgentDeliveryRejected = errors.New("runtime target agent delivery rejected")

const sha256HexLength = 64

// AgentDeliveryGrant 是 Runtime Target 保存的非秘密引导材料投递授权事实。
type AgentDeliveryGrant struct {
	ID                    int64
	GenerationID          int64
	GrantID               string
	TokenVerifier         string
	ExpectedAutomationID  string
	DockerInstallationRef string
	ExpiresAt             time.Time
	Status                string
	HandoffID             string
	HandedOffAt           *time.Time
	DeliveredAt           *time.Time
	ConsumedAt            *time.Time
	RevokedAt             *time.Time
	RevokedReason         string
}

// AgentDeliveryReceipt 是既有部署信任边界验证后的 Docker 投递证据。
// 它不包含 Docker secret 值，也不能用作 Agent 激活授权。
type AgentDeliveryReceipt struct {
	ID                    int64
	GrantID               string
	DeliveryGrantID       int64
	ReceiptID             string
	ProtocolVersion       string
	AutomationID          string
	HandoffID             string
	AssertedDeliveredAt   time.Time
	AcceptedAt            time.Time
	DockerInstallationRef string
	DockerSecretRef       string
	PayloadFingerprint    string
}

// CreatePendingAgentDeliveryGrant 创建一个与单一待激活世代绑定的投递授权。
func (r *SQLRepository) CreatePendingAgentDeliveryGrant(ctx context.Context, grant AgentDeliveryGrant, now time.Time) (AgentDeliveryGrant, error) {
	if r == nil || r.db == nil || now.IsZero() || !validPendingAgentDeliveryGrant(grant, now) {
		return AgentDeliveryGrant{}, ErrAgentDeliveryRejected
	}
	var created AgentDeliveryGrant
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		var generationStatus string
		if err := r.executor(txCtx).QueryRowContext(txCtx, `SELECT status FROM runtime_target_agent_generations WHERE id = $1 AND deleted_at = 0`, grant.GenerationID).Scan(&generationStatus); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrAgentDeliveryRejected
			}
			return fmt.Errorf("read agent generation for delivery grant: %w", err)
		}
		if generationStatus != "pending" {
			return ErrAgentDeliveryRejected
		}
		// 过期授权仍可能被历史索引视为 live；先在同一事务中撤销，保留其审计事实后再创建替代授权。
		if _, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_delivery_grants SET status = 'revoked', revoked_at = $1, revoked_reason = $2, updated_at = $1 WHERE generation_id = $3 AND status IN ('pending', 'delivered') AND expires_at <= $1 AND deleted_at = 0`, now.UTC(), "expired before replacement delivery grant", grant.GenerationID); err != nil {
			return fmt.Errorf("revoke expired agent delivery grants: %w", err)
		}
		row := r.executor(txCtx).QueryRowContext(txCtx, `INSERT INTO runtime_target_agent_delivery_grants (generation_id, grant_id, token_verifier, expected_automation_id, docker_installation_ref, expires_at, status) VALUES ($1,$2,$3,$4,$5,$6,'pending') RETURNING id, generation_id, grant_id, token_verifier, expected_automation_id, docker_installation_ref, expires_at, status, handoff_id, handed_off_at, delivered_at, consumed_at, revoked_at, revoked_reason`, grant.GenerationID, strings.TrimSpace(grant.GrantID), strings.TrimSpace(grant.TokenVerifier), strings.TrimSpace(grant.ExpectedAutomationID), strings.TrimSpace(grant.DockerInstallationRef), grant.ExpiresAt.UTC())
		return scanAgentDeliveryGrant(row, &created)
	})
	return created, err
}

// ReadLiveAgentDeliveryGrant 返回某个待激活世代仍可消费的唯一投递授权。
// 调用方只能据此恢复本地交付编排，不能绕过一次性交接再次释放令牌。
func (r *SQLRepository) ReadLiveAgentDeliveryGrant(ctx context.Context, targetID int64, agentID string, generation int64, now time.Time) (AgentDeliveryGrant, error) {
	if r == nil || r.db == nil || targetID < 1 || generation < 1 || strings.TrimSpace(agentID) == "" || now.IsZero() {
		return AgentDeliveryGrant{}, ErrAgentDeliveryRejected
	}
	var grant AgentDeliveryGrant
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT g.id, g.generation_id, g.grant_id, g.token_verifier, g.expected_automation_id, g.docker_installation_ref, g.expires_at, g.status, g.handoff_id, g.handed_off_at, g.delivered_at, g.consumed_at, g.revoked_at, g.revoked_reason FROM runtime_target_agent_delivery_grants g INNER JOIN runtime_target_agent_generations gen ON gen.id = g.generation_id INNER JOIN runtime_target_agent_identities i ON i.id = gen.identity_id WHERE i.runtime_target_id = $1 AND i.agent_id = $2 AND gen.generation = $3 AND g.status IN ('pending', 'delivered') AND g.expires_at > $4 AND g.deleted_at = 0 AND gen.deleted_at = 0 AND i.deleted_at = 0`, targetID, strings.TrimSpace(agentID), generation, now.UTC()).Scan(&grant.ID, &grant.GenerationID, &grant.GrantID, &grant.TokenVerifier, &grant.ExpectedAutomationID, &grant.DockerInstallationRef, &grant.ExpiresAt, &grant.Status, &grant.HandoffID, &grant.HandedOffAt, &grant.DeliveredAt, &grant.ConsumedAt, &grant.RevokedAt, &grant.RevokedReason)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentDeliveryGrant{}, ErrNotFound
	}
	return grant, err
}

// AcceptAgentDeliveryHandoff 固化一次既有部署信任边界已验证的交接身份。
func (r *SQLRepository) AcceptAgentDeliveryHandoff(ctx context.Context, grantID, automationID, handoffID string, now time.Time) (AgentDeliveryGrant, error) {
	if r == nil || r.db == nil || strings.TrimSpace(grantID) == "" || strings.TrimSpace(automationID) == "" || strings.TrimSpace(handoffID) == "" || now.IsZero() {
		return AgentDeliveryGrant{}, ErrAgentDeliveryRejected
	}
	var accepted AgentDeliveryGrant
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		row := r.executor(txCtx).QueryRowContext(txCtx, `UPDATE runtime_target_agent_delivery_grants SET handoff_id = $1, handed_off_at = $2, updated_at = $2 WHERE grant_id = $3 AND expected_automation_id = $4 AND status = 'pending' AND handoff_id = '' AND expires_at > $2 AND deleted_at = 0 RETURNING id, generation_id, grant_id, token_verifier, expected_automation_id, docker_installation_ref, expires_at, status, handoff_id, handed_off_at, delivered_at, consumed_at, revoked_at, revoked_reason`, strings.TrimSpace(handoffID), now.UTC(), strings.TrimSpace(grantID), strings.TrimSpace(automationID))
		err := scanAgentDeliveryGrant(row, &accepted)
		if errors.Is(err, ErrAgentTrustNotFound) {
			return ErrAgentDeliveryRejected
		}
		return err
	})
	return accepted, err
}

// RecordAgentDeliveryReceipt 保存一份已验证的 Docker 投递证据。
// 同一 receipt ID 只有在规范化内容完全一致时才可幂等重试。
//
//nolint:gocognit,cyclop // 同一事务必须区分幂等重放、首次证据写入和授权状态切换，不能拆成可观察的中间状态。
func (r *SQLRepository) RecordAgentDeliveryReceipt(ctx context.Context, receipt AgentDeliveryReceipt, now time.Time) (AgentDeliveryReceipt, bool, error) {
	if r == nil || r.db == nil || !validAgentDeliveryReceipt(receipt, now) {
		return AgentDeliveryReceipt{}, false, ErrAgentDeliveryRejected
	}
	var recorded AgentDeliveryReceipt
	var replay bool
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		existing, found, err := r.readAgentDeliveryReceiptByID(txCtx, receipt.ReceiptID)
		if err != nil {
			return err
		}
		if found {
			if !sameAgentDeliveryReceipt(existing, receipt) {
				return ErrAgentDeliveryRejected
			}
			recorded, replay = existing, true
			return nil
		}
		row := r.executor(txCtx).QueryRowContext(txCtx, `INSERT INTO runtime_target_agent_delivery_receipts (delivery_grant_id, receipt_id, protocol_version, automation_id, handoff_id, asserted_delivered_at, accepted_at, docker_installation_ref, docker_secret_ref, payload_fingerprint) SELECT id, $1, $2, CAST($3 AS TEXT), CAST($4 AS TEXT), $5, $6, CAST($7 AS TEXT), $8, $9 FROM runtime_target_agent_delivery_grants WHERE grant_id = $10 AND expected_automation_id = CAST($3 AS TEXT) AND handoff_id = CAST($4 AS TEXT) AND docker_installation_ref = CAST($7 AS TEXT) AND status = 'pending' AND expires_at > $6 AND deleted_at = 0 RETURNING id, delivery_grant_id, receipt_id, protocol_version, automation_id, handoff_id, asserted_delivered_at, accepted_at, docker_installation_ref, docker_secret_ref, payload_fingerprint`, strings.TrimSpace(receipt.ReceiptID), strings.TrimSpace(receipt.ProtocolVersion), strings.TrimSpace(receipt.AutomationID), strings.TrimSpace(receipt.HandoffID), receipt.AssertedDeliveredAt.UTC(), now.UTC(), strings.TrimSpace(receipt.DockerInstallationRef), strings.TrimSpace(receipt.DockerSecretRef), strings.TrimSpace(receipt.PayloadFingerprint), strings.TrimSpace(receipt.GrantID))
		if err := scanAgentDeliveryReceipt(row, &recorded); err != nil {
			if errors.Is(err, ErrAgentTrustNotFound) {
				return ErrAgentDeliveryRejected
			}
			return err
		}
		recorded.GrantID = strings.TrimSpace(receipt.GrantID)
		result, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_agent_delivery_grants SET status = 'delivered', delivered_at = $1, updated_at = $1 WHERE id = $2 AND status = 'pending' AND deleted_at = 0`, now.UTC(), recorded.DeliveryGrantID)
		if err != nil {
			return fmt.Errorf("mark agent delivery grant delivered: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return ErrAgentDeliveryRejected
		}
		return nil
	})
	return recorded, replay, err
}

func (r *SQLRepository) readAgentDeliveryReceiptByID(ctx context.Context, receiptID string) (AgentDeliveryReceipt, bool, error) {
	var receipt AgentDeliveryReceipt
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT r.id, r.delivery_grant_id, r.receipt_id, r.protocol_version, r.automation_id, r.handoff_id, r.asserted_delivered_at, r.accepted_at, r.docker_installation_ref, r.docker_secret_ref, r.payload_fingerprint, g.grant_id FROM runtime_target_agent_delivery_receipts r INNER JOIN runtime_target_agent_delivery_grants g ON g.id = r.delivery_grant_id WHERE r.receipt_id = $1 AND r.deleted_at = 0 AND g.deleted_at = 0`, strings.TrimSpace(receiptID)).Scan(&receipt.ID, &receipt.DeliveryGrantID, &receipt.ReceiptID, &receipt.ProtocolVersion, &receipt.AutomationID, &receipt.HandoffID, &receipt.AssertedDeliveredAt, &receipt.AcceptedAt, &receipt.DockerInstallationRef, &receipt.DockerSecretRef, &receipt.PayloadFingerprint, &receipt.GrantID)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrAgentTrustNotFound
	}
	if errors.Is(err, ErrAgentTrustNotFound) {
		return AgentDeliveryReceipt{}, false, nil
	}
	return receipt, err == nil, err
}

type agentDeliveryScanner interface{ Scan(...any) error }

func scanAgentDeliveryGrant(row agentDeliveryScanner, grant *AgentDeliveryGrant) error {
	err := row.Scan(&grant.ID, &grant.GenerationID, &grant.GrantID, &grant.TokenVerifier, &grant.ExpectedAutomationID, &grant.DockerInstallationRef, &grant.ExpiresAt, &grant.Status, &grant.HandoffID, &grant.HandedOffAt, &grant.DeliveredAt, &grant.ConsumedAt, &grant.RevokedAt, &grant.RevokedReason)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAgentTrustNotFound
	}
	return err
}

func scanAgentDeliveryReceipt(row agentDeliveryScanner, receipt *AgentDeliveryReceipt) error {
	err := row.Scan(&receipt.ID, &receipt.DeliveryGrantID, &receipt.ReceiptID, &receipt.ProtocolVersion, &receipt.AutomationID, &receipt.HandoffID, &receipt.AssertedDeliveredAt, &receipt.AcceptedAt, &receipt.DockerInstallationRef, &receipt.DockerSecretRef, &receipt.PayloadFingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAgentTrustNotFound
	}
	return err
}

func validPendingAgentDeliveryGrant(grant AgentDeliveryGrant, now time.Time) bool {
	return grant.GenerationID > 0 && strings.TrimSpace(grant.GrantID) != "" && validSHA256Hex(grant.TokenVerifier) && strings.TrimSpace(grant.ExpectedAutomationID) != "" && strings.TrimSpace(grant.DockerInstallationRef) != "" && grant.ExpiresAt.After(now.UTC())
}

func validAgentDeliveryReceipt(receipt AgentDeliveryReceipt, now time.Time) bool {
	return strings.TrimSpace(receipt.GrantID) != "" && strings.TrimSpace(receipt.ReceiptID) != "" && receipt.ProtocolVersion == "graft.delivery-receipt.v1" && strings.TrimSpace(receipt.AutomationID) != "" && strings.TrimSpace(receipt.HandoffID) != "" && !receipt.AssertedDeliveredAt.IsZero() && !receipt.AssertedDeliveredAt.After(now) && strings.TrimSpace(receipt.DockerInstallationRef) != "" && strings.TrimSpace(receipt.DockerSecretRef) != "" && validSHA256Hex(receipt.PayloadFingerprint)
}

func sameAgentDeliveryReceipt(left, right AgentDeliveryReceipt) bool {
	return left.GrantID == strings.TrimSpace(right.GrantID) && left.ProtocolVersion == strings.TrimSpace(right.ProtocolVersion) && left.AutomationID == strings.TrimSpace(right.AutomationID) && left.HandoffID == strings.TrimSpace(right.HandoffID) && left.AssertedDeliveredAt.Equal(right.AssertedDeliveredAt.UTC()) && left.DockerInstallationRef == strings.TrimSpace(right.DockerInstallationRef) && left.DockerSecretRef == strings.TrimSpace(right.DockerSecretRef) && left.PayloadFingerprint == strings.TrimSpace(right.PayloadFingerprint)
}

func validSHA256Hex(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != sha256HexLength {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}
