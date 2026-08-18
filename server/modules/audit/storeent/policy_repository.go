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

// ListAuditPolicyRules 返回按运行时优先级排序的启用和停用策略规则。
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

// GetAuditVisibilityDefault 返回指定名称的审计可见性默认策略。
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

// UpsertAuditVisibilityDefault 创建或更新一个全局审计可见性默认策略。
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

// ListAuditVisibilityOverrides 返回全部来源加动作可见性覆盖规则。
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

// FindAuditVisibilityOverride 按来源和动作精确查找存在的覆盖规则。
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

// UpsertAuditVisibilityOverride 创建或更新一个来源加动作可见性覆盖规则。
func (r *repository) UpsertAuditVisibilityOverride(
	ctx context.Context,
	input auditstore.UpsertAuditVisibilityOverrideInput,
) (auditstore.AuditVisibilityOverride, error) {
	if r == nil || r.db == nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("audit repository is unavailable")
	}

	return upsertAuditVisibilityOverride(ctx, r.db, input)
}

// UpsertAuditVisibilityOverrides 在一个数据库事务内按输入顺序写入全部覆盖规则。
func (r *repository) UpsertAuditVisibilityOverrides(
	ctx context.Context,
	inputs []auditstore.UpsertAuditVisibilityOverrideInput,
) ([]auditstore.AuditVisibilityOverride, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("audit repository is unavailable")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin audit visibility override transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	items := make([]auditstore.AuditVisibilityOverride, 0, len(inputs))
	for _, input := range inputs {
		item, upsertErr := upsertAuditVisibilityOverride(ctx, tx, input)
		if upsertErr != nil {
			return nil, upsertErr
		}
		items = append(items, item)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit audit visibility override transaction: %w", err)
	}
	return items, nil
}

type auditVisibilityOverrideQueryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func upsertAuditVisibilityOverride(
	ctx context.Context,
	queryer auditVisibilityOverrideQueryRower,
	input auditstore.UpsertAuditVisibilityOverrideInput,
) (auditstore.AuditVisibilityOverride, error) {
	who, err := nullableUint64(input.Actor.UserID)
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("upsert audit visibility override: %w", err)
	}

	row := queryer.QueryRowContext(
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
		) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, $5, $6, CURRENT_TIMESTAMP, $5, $6)
		ON CONFLICT (source, action_key) DO UPDATE SET
			strategy = EXCLUDED.strategy,
			description = EXCLUDED.description,
			updated_at = CURRENT_TIMESTAMP,
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

// DeleteAuditVisibilityOverride 删除一条来源加动作可见性覆盖规则。
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

// scanAuditVisibilityOverride 将审计可见性覆盖规则的查询结果扫描并规范化为结构体。
// 字符串字段会被去除首尾空白，来源和策略会被标准化，可空的创建者和更新者字段会被映射为对应 ID。
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
