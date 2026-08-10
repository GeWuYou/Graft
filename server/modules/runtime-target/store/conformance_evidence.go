//go:build conformance

package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CountAgentLedgerReceipts 返回指定活动 generation 已消费 ledger snapshot 的数量，仅供 conformance 读取证据。
func (r *SQLRepository) CountAgentLedgerReceipts(ctx context.Context, targetID int64, agentID string, generation int64) (int64, error) {
	if r == nil || r.db == nil || targetID < 1 || generation < 1 || strings.TrimSpace(agentID) == "" {
		return 0, ErrAgentTrustNotFound
	}
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)
FROM runtime_target_agent_ledger_snapshots snapshots
INNER JOIN runtime_target_agent_generations generations ON generations.id = snapshots.generation_id
INNER JOIN runtime_target_agent_identities identities ON identities.id = generations.identity_id
WHERE identities.runtime_target_id = $1 AND identities.agent_id = $2 AND generations.generation = $3
  AND snapshots.consumed_at IS NOT NULL AND snapshots.deleted_at = 0
  AND generations.deleted_at = 0 AND identities.deleted_at = 0`, targetID, strings.TrimSpace(agentID), generation).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrAgentTrustNotFound
		}
		return 0, fmt.Errorf("count agent ledger receipts: %w", err)
	}
	return count, nil
}
