package storeent

import (
	"context"
	"database/sql"
	"errors"
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

// GetAuditVisibilityDefault returns the named default audit visibility strategy.
func (r *repository) GetAuditVisibilityDefault(ctx context.Context, key string) (auditstore.AuditVisibilityDefault, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("audit repository is unavailable")
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT key, strategy, updated_at, updated_by, updated_by_name
		FROM audit_visibility_defaults
		WHERE key = $1`,
		strings.TrimSpace(key),
	)

	var (
		item      auditstore.AuditVisibilityDefault
		updatedBy sql.NullInt64
	)
	if err := row.Scan(&item.Key, &item.Strategy, &item.UpdatedAt, &updatedBy, &item.UpdatedByName); err != nil {
		if err == sql.ErrNoRows {
			return auditstore.AuditVisibilityDefault{
				Key:           strings.TrimSpace(key),
				Strategy:      auditstore.AuditVisibilityStrategyVisible,
				UpdatedByName: "system",
			}, nil
		}
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("read audit visibility default: %w", err)
	}
	if updatedBy.Valid {
		value := toStoreID(updatedBy.Int64)
		item.UpdatedBy = &value
	}
	item.Strategy = normalizeStoredAuditVisibility(item.Strategy)
	item.Key = strings.TrimSpace(item.Key)
	item.UpdatedByName = strings.TrimSpace(item.UpdatedByName)
	return item, nil
}

// UpsertAuditVisibilityDefault creates or updates one global audit visibility default.
func (r *repository) UpsertAuditVisibilityDefault(
	ctx context.Context,
	key string,
	strategy auditstore.AuditVisibilityStrategy,
	userID *uint64,
	username string,
) (auditstore.AuditVisibilityDefault, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("audit repository is unavailable")
	}

	updatedBy, err := nullableUint64(userID)
	if err != nil {
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("upsert audit visibility default: %w", err)
	}

	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO audit_visibility_defaults (
			key,
			strategy,
			updated_at,
			updated_by,
			updated_by_name
		) VALUES ($1, $2, NOW(), $3, $4)
		ON CONFLICT (key) DO UPDATE SET
			strategy = EXCLUDED.strategy,
			updated_at = NOW(),
			updated_by = EXCLUDED.updated_by,
			updated_by_name = EXCLUDED.updated_by_name
		RETURNING key, strategy, updated_at, updated_by, updated_by_name`,
		strings.TrimSpace(key),
		string(normalizeStoredAuditVisibility(strategy)),
		updatedBy,
		strings.TrimSpace(username),
	)

	var (
		item              auditstore.AuditVisibilityDefault
		returnedUpdatedBy sql.NullInt64
	)
	if err := row.Scan(&item.Key, &item.Strategy, &item.UpdatedAt, &returnedUpdatedBy, &item.UpdatedByName); err != nil {
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("upsert audit visibility default: %w", err)
	}
	if returnedUpdatedBy.Valid {
		value := toStoreID(returnedUpdatedBy.Int64)
		item.UpdatedBy = &value
	}
	item.Strategy = normalizeStoredAuditVisibility(item.Strategy)
	item.Key = strings.TrimSpace(item.Key)
	item.UpdatedByName = strings.TrimSpace(item.UpdatedByName)
	return item, nil
}

// ListAuditVisibilityOverrides returns all source+action visibility overrides.
func (r *repository) ListAuditVisibilityOverrides(ctx context.Context) ([]auditstore.AuditVisibilityOverride, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("audit repository is unavailable")
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT
			id,
			source,
			action_key,
			strategy,
			description,
			created_at,
			created_by,
			created_by_name,
			updated_at,
			updated_by,
			updated_by_name
		FROM audit_visibility_overrides
		ORDER BY source ASC, action_key ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit visibility overrides: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]auditstore.AuditVisibilityOverride, 0, defaultPolicyRuleCapacity)
	for rows.Next() {
		item, scanErr := scanAuditVisibilityOverride(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit visibility overrides: %w", err)
	}
	return items, nil
}

// FindAuditVisibilityOverride returns one exact source+action visibility override when it exists.
func (r *repository) FindAuditVisibilityOverride(
	ctx context.Context,
	source auditstore.AuditSource,
	actionKey string,
) (auditstore.AuditVisibilityOverride, bool, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditVisibilityOverride{}, false, fmt.Errorf("audit repository is unavailable")
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT
			id,
			source,
			action_key,
			strategy,
			description,
			created_at,
			created_by,
			created_by_name,
			updated_at,
			updated_by,
			updated_by_name
		FROM audit_visibility_overrides
		WHERE source = $1 AND action_key = $2`,
		string(source),
		strings.TrimSpace(actionKey),
	)

	item, err := scanAuditVisibilityOverride(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auditstore.AuditVisibilityOverride{}, false, nil
		}
		return auditstore.AuditVisibilityOverride{}, false, fmt.Errorf("find audit visibility override: %w", err)
	}
	return item, true, nil
}

// UpsertAuditVisibilityOverride creates or updates one source+action visibility override.
func (r *repository) UpsertAuditVisibilityOverride(
	ctx context.Context,
	input auditstore.UpsertAuditVisibilityOverrideInput,
) (auditstore.AuditVisibilityOverride, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("audit repository is unavailable")
	}

	who, err := nullableUint64(input.Actor.UserID)
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("upsert audit visibility override: %w", err)
	}

	row := r.db.QueryRowContext(
		ctx,
		`INSERT INTO audit_visibility_overrides (
			source,
			action_key,
			strategy,
			description,
			created_at,
			created_by,
			created_by_name,
			updated_at,
			updated_by,
			updated_by_name
		) VALUES ($1, $2, $3, $4, NOW(), $5, $6, NOW(), $5, $6)
		ON CONFLICT (source, action_key) DO UPDATE SET
			strategy = EXCLUDED.strategy,
			description = EXCLUDED.description,
			updated_at = NOW(),
			updated_by = EXCLUDED.updated_by,
			updated_by_name = EXCLUDED.updated_by_name
		RETURNING
			id,
			source,
			action_key,
			strategy,
			description,
			created_at,
			created_by,
			created_by_name,
			updated_at,
			updated_by,
			updated_by_name`,
		string(input.Source),
		strings.TrimSpace(input.ActionKey),
		string(normalizeStoredAuditVisibility(input.Strategy)),
		strings.TrimSpace(input.Description),
		who,
		strings.TrimSpace(input.Actor.Username),
	)

	item, err := scanAuditVisibilityOverride(row)
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("upsert audit visibility override: %w", err)
	}
	return item, nil
}

// DeleteAuditVisibilityOverride removes one source+action visibility override.
func (r *repository) DeleteAuditVisibilityOverride(ctx context.Context, source auditstore.AuditSource, actionKey string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("audit repository is unavailable")
	}

	if _, err := r.db.ExecContext(
		ctx,
		`DELETE FROM audit_visibility_overrides WHERE source = $1 AND action_key = $2`,
		string(source),
		strings.TrimSpace(actionKey),
	); err != nil {
		return fmt.Errorf("delete audit visibility override: %w", err)
	}
	return nil
}

func scanAuditVisibilityOverride(scanner interface {
	Scan(dest ...any) error
}) (auditstore.AuditVisibilityOverride, error) {
	var (
		item      auditstore.AuditVisibilityOverride
		createdBy sql.NullInt64
		updatedBy sql.NullInt64
	)
	if err := scanner.Scan(
		&item.ID,
		&item.Source,
		&item.ActionKey,
		&item.Strategy,
		&item.Description,
		&item.CreatedAt,
		&createdBy,
		&item.CreatedByName,
		&item.UpdatedAt,
		&updatedBy,
		&item.UpdatedByName,
	); err != nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("scan audit visibility override: %w", err)
	}
	if createdBy.Valid {
		value := toStoreID(createdBy.Int64)
		item.CreatedBy = &value
	}
	if updatedBy.Valid {
		value := toStoreID(updatedBy.Int64)
		item.UpdatedBy = &value
	}
	item.Source = normalizeAuditSource(strings.TrimSpace(string(item.Source)))
	item.ActionKey = strings.TrimSpace(item.ActionKey)
	item.Strategy = normalizeStoredAuditVisibility(item.Strategy)
	item.Description = strings.TrimSpace(item.Description)
	item.CreatedByName = strings.TrimSpace(item.CreatedByName)
	item.UpdatedByName = strings.TrimSpace(item.UpdatedByName)
	return item, nil
}
