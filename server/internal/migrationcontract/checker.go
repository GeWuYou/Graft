// Package migrationcontract 评估迁移门禁所拥有的 PostgreSQL schema 事实。
package migrationcontract

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
)

const (
	// FindingFKTargetKeyMissing 标识外键目标列没有精确匹配主键或唯一约束。
	FindingFKTargetKeyMissing = "migration.fk_target_key_missing"
	// FindingFKNotValidated 标识仍未验证的外键。
	FindingFKNotValidated = "migration.fk_not_validated"
	// FindingInvalidIndex 标识 public schema 中无法供查询或约束使用的索引。
	FindingInvalidIndex = "migration.invalid_index"
	// FindingAtlasRevisionTableMissing 标识迁移执行后缺少 Atlas revision 表。
	FindingAtlasRevisionTableMissing = "migration.atlas_revision_table_missing"
	// FindingAtlasRevisionPartial 标识 Atlas 保留了未完成或失败的 revision。
	FindingAtlasRevisionPartial = "migration.atlas_revision_partial"
	// FindingConstraintName 标识不符合 Phase 1 约定的约束名称。
	FindingConstraintName = "migration.constraint_name"
	// FindingIndexName 标识不符合 Phase 1 约定的索引名称。
	FindingIndexName = "migration.index_name"
	// FindingFKSourceIndexMissing 标识外键源列缺少可用 B-tree 左前缀索引。
	FindingFKSourceIndexMissing = "migration.fk_source_index_missing"
)

// Severity 表示 finding 是否阻断 migration gate。
type Severity string

const (
	// SeverityEnforce 表示 finding 必须阻断 gate。
	SeverityEnforce Severity = "enforce"
	// SeverityReport 表示 finding 仅作为 Phase 1 基线报告。
	SeverityReport Severity = "report"
)

// Finding 是可序列化且稳定的 catalog contract 诊断。
type Finding struct {
	ID       string         `json:"id"`
	Severity Severity       `json:"severity"`
	Object   string         `json:"object"`
	Message  string         `json:"message"`
	Details  map[string]any `json:"details,omitempty"`
}

// Result 是一次 schema contract 检查的 JSON 安全结果。
type Result struct {
	Findings []Finding `json:"findings"`
}

// HasEnforceFindings 返回结果中是否存在需要阻断的 finding。
func (r Result) HasEnforceFindings() bool {
	for _, finding := range r.Findings {
		if finding.Severity == SeverityEnforce {
			return true
		}
	}
	return false
}

// Queryer 是 checker 所需的最小 database/sql 查询能力。
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Checker 从 PostgreSQL catalog 读取最终 schema state；它不会解析 migration SQL。
type Checker struct {
	queryer Queryer
}

// NewChecker 创建一个使用给定查询连接的 schema contract checker。
func NewChecker(queryer Queryer) (*Checker, error) {
	if queryer == nil {
		return nil, fmt.Errorf("migration contract queryer is required")
	}
	return &Checker{queryer: queryer}, nil
}

// Check 评估 Phase 1 强制与报告型 PostgreSQL catalog contract。
func (c *Checker) Check(ctx context.Context) (Result, error) {
	result := Result{Findings: make([]Finding, 0)}
	checks := []func(context.Context) ([]Finding, error){
		c.checkForeignKeys,
		c.checkInvalidIndexes,
		c.checkAtlasRevisions,
		c.checkConstraintNames,
		c.checkIndexNames,
		c.checkForeignKeySourceIndexes,
	}
	for _, check := range checks {
		findings, err := check(ctx)
		if err != nil {
			return Result{}, err
		}
		result.Findings = append(result.Findings, findings...)
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].ID != result.Findings[j].ID {
			return result.Findings[i].ID < result.Findings[j].ID
		}
		return result.Findings[i].Object < result.Findings[j].Object
	})
	return result, nil
}

func (c *Checker) checkForeignKeys(ctx context.Context) ([]Finding, error) {
	rows, err := c.queryer.QueryContext(ctx, foreignKeyContractQuery)
	if err != nil {
		return nil, fmt.Errorf("query foreign key contract: %w", err)
	}
	defer func() { _ = rows.Close() }()

	findings := make([]Finding, 0)
	for rows.Next() {
		var schema, table, name, definition string
		var targetKeyExists, validated bool
		if err := rows.Scan(&schema, &table, &name, &definition, &targetKeyExists, &validated); err != nil {
			return nil, fmt.Errorf("scan foreign key contract: %w", err)
		}
		object := qualifiedConstraint(schema, table, name)
		if !targetKeyExists {
			findings = append(findings, Finding{
				ID: FindingFKTargetKeyMissing, Severity: SeverityEnforce, Object: object,
				Message: "foreign key target columns do not exactly match a primary key or unique constraint",
				Details: map[string]any{"definition": definition},
			})
		}
		if !validated {
			findings = append(findings, Finding{
				ID: FindingFKNotValidated, Severity: SeverityEnforce, Object: object,
				Message: "foreign key is not validated", Details: map[string]any{"definition": definition},
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate foreign key contract: %w", err)
	}
	return findings, nil
}

func (c *Checker) checkInvalidIndexes(ctx context.Context) ([]Finding, error) {
	rows, err := c.queryer.QueryContext(ctx, invalidIndexQuery)
	if err != nil {
		return nil, fmt.Errorf("query invalid indexes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanSimpleFindings(rows, FindingInvalidIndex, SeverityEnforce, "invalid index remains in the public schema")
}

func (c *Checker) checkAtlasRevisions(ctx context.Context) ([]Finding, error) { //nolint:cyclop // the catalog contract has distinct missing, partial, scan, and iteration outcomes.
	rows, err := c.queryer.QueryContext(ctx, atlasRevisionTableQuery)
	if err != nil {
		return nil, fmt.Errorf("query Atlas revision table: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var exists bool
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate Atlas revision table: %w", err)
		}
		return nil, fmt.Errorf("atlas revision table query returned no row")
	}
	if err := rows.Scan(&exists); err != nil {
		return nil, fmt.Errorf("scan Atlas revision table: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Atlas revision table: %w", err)
	}
	if !exists {
		return []Finding{{
			ID: FindingAtlasRevisionTableMissing, Severity: SeverityEnforce, Object: "public.atlas_schema_revisions",
			Message: "Atlas revision table is missing after migration execution",
		}}, nil
	}

	partialRows, err := c.queryer.QueryContext(ctx, atlasRevisionPartialQuery)
	if err != nil {
		return nil, fmt.Errorf("query Atlas partial revisions: %w", err)
	}
	defer func() { _ = partialRows.Close() }()
	var partialCount int64
	if !partialRows.Next() {
		if err := partialRows.Err(); err != nil {
			return nil, fmt.Errorf("iterate Atlas partial revisions: %w", err)
		}
		return nil, fmt.Errorf("atlas partial revision query returned no row")
	}
	if err := partialRows.Scan(&partialCount); err != nil {
		return nil, fmt.Errorf("scan Atlas partial revisions: %w", err)
	}
	if err := partialRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Atlas partial revisions: %w", err)
	}
	if partialCount > 0 {
		return []Finding{{
			ID: FindingAtlasRevisionPartial, Severity: SeverityEnforce, Object: "public.atlas_schema_revisions",
			Message: "Atlas revision table contains unfinished or failed revisions",
			Details: map[string]any{"partial_revision_count": partialCount},
		}}, nil
	}
	return nil, nil
}

func (c *Checker) checkConstraintNames(ctx context.Context) ([]Finding, error) {
	rows, err := c.queryer.QueryContext(ctx, constraintNameQuery)
	if err != nil {
		return nil, fmt.Errorf("query constraint names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	findings := make([]Finding, 0)
	for rows.Next() {
		var schema, table, name, kind string
		if err := rows.Scan(&schema, &table, &name, &kind); err != nil {
			return nil, fmt.Errorf("scan constraint name: %w", err)
		}
		if constraintNamePattern(kind).MatchString(name) {
			continue
		}
		findings = append(findings, Finding{
			ID: FindingConstraintName, Severity: SeverityReport, Object: qualifiedConstraint(schema, table, name),
			Message: "constraint name does not follow the Phase 1 convention",
			Details: map[string]any{"constraint_type": kind},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate constraint names: %w", err)
	}
	return findings, nil
}

func (c *Checker) checkIndexNames(ctx context.Context) ([]Finding, error) {
	rows, err := c.queryer.QueryContext(ctx, indexNameQuery)
	if err != nil {
		return nil, fmt.Errorf("query index names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	findings := make([]Finding, 0)
	for rows.Next() {
		var schema, table, name string
		if err := rows.Scan(&schema, &table, &name); err != nil {
			return nil, fmt.Errorf("scan index name: %w", err)
		}
		if indexNamePattern.MatchString(name) {
			continue
		}
		findings = append(findings, Finding{
			ID: FindingIndexName, Severity: SeverityReport, Object: qualifiedIndex(schema, table, name),
			Message: "index name does not follow the Phase 1 convention",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate index names: %w", err)
	}
	return findings, nil
}

func (c *Checker) checkForeignKeySourceIndexes(ctx context.Context) ([]Finding, error) {
	rows, err := c.queryer.QueryContext(ctx, foreignKeySourceIndexQuery)
	if err != nil {
		return nil, fmt.Errorf("query foreign key source indexes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanSimpleFindings(rows, FindingFKSourceIndexMissing, SeverityReport, "foreign key source columns do not have a valid B-tree left-prefix index")
}

func scanSimpleFindings(rows *sql.Rows, id string, severity Severity, message string) ([]Finding, error) {
	findings := make([]Finding, 0)
	for rows.Next() {
		var schema, table, name string
		if err := rows.Scan(&schema, &table, &name); err != nil {
			return nil, fmt.Errorf("scan %s: %w", id, err)
		}
		findings = append(findings, Finding{ID: id, Severity: severity, Object: qualifiedConstraint(schema, table, name), Message: message})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", id, err)
	}
	return findings, nil
}

func qualifiedConstraint(schema, table, name string) string {
	return schema + "." + table + "." + name
}

func qualifiedIndex(schema, table, name string) string {
	return schema + "." + table + "." + name
}

func constraintNamePattern(kind string) *regexp.Regexp {
	switch kind {
	case "p":
		return regexp.MustCompile(`^pk_[a-z0-9_]+$`)
	case "u":
		return regexp.MustCompile(`^uq_[a-z0-9_]+$`)
	case "f":
		return regexp.MustCompile(`^fk_[a-z0-9_]+__[a-z0-9_]+$`)
	case "c":
		return regexp.MustCompile(`^ck_[a-z0-9_]+$`)
	default:
		return regexp.MustCompile(`^$`)
	}
}

var indexNamePattern = regexp.MustCompile(`^(idx|uq)_[a-z0-9_]+$`)

const foreignKeyContractQuery = `
SELECT ns.nspname, rel.relname, con.conname, pg_get_constraintdef(con.oid),
       EXISTS (
         SELECT 1 FROM pg_constraint target
         WHERE target.conrelid = con.confrelid
           AND target.contype IN ('p', 'u')
           AND cardinality(target.conkey) = cardinality(con.confkey)
           AND target.conkey @> con.confkey
           AND target.conkey <@ con.confkey
       ) AS target_key_exists,
       con.convalidated
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
JOIN pg_namespace ns ON ns.oid = rel.relnamespace
WHERE con.contype = 'f' AND ns.nspname = 'public'
ORDER BY ns.nspname, rel.relname, con.conname`

const invalidIndexQuery = `
SELECT ns.nspname, rel.relname, idx.relname
FROM pg_index i
JOIN pg_class idx ON idx.oid = i.indexrelid
JOIN pg_class rel ON rel.oid = i.indrelid
JOIN pg_namespace ns ON ns.oid = rel.relnamespace
WHERE ns.nspname = 'public' AND NOT i.indisvalid
ORDER BY ns.nspname, rel.relname, idx.relname`

const atlasRevisionTableQuery = `
SELECT EXISTS (
         SELECT 1 FROM pg_class rel
         JOIN pg_namespace ns ON ns.oid = rel.relnamespace
         WHERE ns.nspname = 'public' AND rel.relname = 'atlas_schema_revisions' AND rel.relkind = 'r'
       ) AS exists`

const atlasRevisionPartialQuery = `
SELECT count(*) FROM public.atlas_schema_revisions
WHERE applied <> total OR error <> '' OR partial_hashes IS NOT NULL`

const constraintNameQuery = `
SELECT ns.nspname, rel.relname, con.conname, con.contype
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
JOIN pg_namespace ns ON ns.oid = rel.relnamespace
WHERE ns.nspname = 'public' AND con.contype IN ('p', 'u', 'f', 'c')
ORDER BY ns.nspname, rel.relname, con.conname`

const indexNameQuery = `
SELECT ns.nspname, rel.relname, idx.relname
FROM pg_index i
JOIN pg_class idx ON idx.oid = i.indexrelid
JOIN pg_class rel ON rel.oid = i.indrelid
JOIN pg_namespace ns ON ns.oid = rel.relnamespace
LEFT JOIN pg_constraint con ON con.conindid = i.indexrelid
WHERE ns.nspname = 'public' AND con.oid IS NULL
ORDER BY ns.nspname, rel.relname, idx.relname`

const foreignKeySourceIndexQuery = `
SELECT ns.nspname, rel.relname, con.conname
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
JOIN pg_namespace ns ON ns.oid = rel.relnamespace
WHERE con.contype = 'f' AND ns.nspname = 'public'
  AND NOT EXISTS (
    SELECT 1 FROM pg_index i
    JOIN pg_class idx ON idx.oid = i.indexrelid
    JOIN pg_am am ON am.oid = idx.relam
    WHERE i.indrelid = con.conrelid
      AND i.indisvalid
      AND i.indisready
      AND am.amname = 'btree'
      AND i.indpred IS NULL
      AND (i.indkey::smallint[])[1:array_length(con.conkey, 1)] = con.conkey
  )
ORDER BY ns.nspname, rel.relname, con.conname`
