// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package storeent

import (
	"context"
	"fmt"
	"strings"

	auditstore "graft/server/modules/audit/store"
)

const defaultPolicyRuleCapacity = 16

// ListAuditPolicyRules returns enabled and disabled rules sorted by runtime priority.
func (r *repository) ListAuditPolicyRules(ctx context.Context) ([]auditstore.AuditPolicyRule, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("audit repository is unavailable")
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT
			id,
			name,
			description,
			source,
			enabled,
			priority,
			effect,
			match_type,
			method,
			path_pattern,
			event_type,
			risk_level,
			target_type,
			condition_expr,
			created_at,
			updated_at
		FROM audit_policy_rules
		ORDER BY priority ASC, length(path_pattern) DESC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit policy rules: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rules := make([]auditstore.AuditPolicyRule, 0, defaultPolicyRuleCapacity)
	for rows.Next() {
		var rule auditstore.AuditPolicyRule
		if err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&rule.Source,
			&rule.Enabled,
			&rule.Priority,
			&rule.Effect,
			&rule.MatchType,
			&rule.Method,
			&rule.PathPattern,
			&rule.EventType,
			&rule.RiskLevel,
			&rule.TargetType,
			&rule.ConditionExpr,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit policy rule: %w", err)
		}

		rule.Source = auditstore.AuditSource(strings.ToUpper(strings.TrimSpace(string(rule.Source))))
		rule.Method = strings.ToUpper(strings.TrimSpace(rule.Method))
		rule.PathPattern = strings.TrimSpace(rule.PathPattern)
		rule.EventType = strings.TrimSpace(rule.EventType)
		rule.TargetType = strings.TrimSpace(rule.TargetType)
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit policy rules: %w", err)
	}

	return rules, nil
}
