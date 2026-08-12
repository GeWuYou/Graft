package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// migrationPreflightManifest 是 sidecar 供部署前检查消费的子集，刻意不进入 Atlas 执行路径。
type migrationPreflightManifest struct {
	Migration struct {
		Path    string `yaml:"path"`
		Version string `yaml:"version"`
	} `yaml:"migration"`
	PreflightChecks []migrationPreflightCheck `yaml:"preflight_checks"`
}

type migrationPreflightCheck struct {
	Name      string `yaml:"name"`
	Query     string `yaml:"query"`
	MustEqual int64  `yaml:"must_equal"`
}

type migratePreflightOptions struct {
	manifest string
}

var migratePreflightRunner = runMigrationPreflight

// runMigrationPreflight 在 operator 应用 migration 前执行 manifest 声明的只读目标数据检查。
func runMigrationPreflight(ctx context.Context, databaseURL string, manifestPath string) error {
	manifest, err := loadMigrationPreflightManifest(manifestPath)
	if err != nil {
		return err
	}
	if len(manifest.PreflightChecks) == 0 {
		return fmt.Errorf("migration preflight manifest %s has no preflight_checks", manifestPath)
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open postgres database pool: %w", err)
	}
	defer func() { _ = db.Close() }()

	return runMigrationPreflightChecks(ctx, db, manifest.PreflightChecks)
}

// runMigrationPreflightChecks 在只读事务中执行全部 preflight 查询，数据库拒绝词法检查无法识别的写入路径。
func runMigrationPreflightChecks(ctx context.Context, db *sql.DB, checks []migrationPreflightCheck) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin read-only migration preflight transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, check := range checks {
		if err := validateReadOnlyPreflightCheck(check); err != nil {
			return err
		}
		var actual int64
		if err := tx.QueryRowContext(ctx, check.Query).Scan(&actual); err != nil {
			return fmt.Errorf("run migration preflight check %q: %w", check.Name, err)
		}
		if actual != check.MustEqual {
			return fmt.Errorf("migration preflight check %q failed: expected %d, found %d", check.Name, check.MustEqual, actual)
		}
	}
	return nil
}

func loadMigrationPreflightManifest(path string) (migrationPreflightManifest, error) {
	// #nosec G304 -- manifest 由 operator 通过显式 CLI flag 选择，读取后仅执行受限的只读检查。
	content, err := os.ReadFile(path)
	if err != nil {
		return migrationPreflightManifest{}, fmt.Errorf("read migration preflight manifest: %w", err)
	}
	var manifest migrationPreflightManifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return migrationPreflightManifest{}, fmt.Errorf("parse migration preflight manifest: %w", err)
	}
	return manifest, nil
}

func validateReadOnlyPreflightCheck(check migrationPreflightCheck) error {
	if strings.TrimSpace(check.Name) == "" {
		return fmt.Errorf("migration preflight check name is required")
	}
	query := strings.TrimSpace(check.Query)
	tokens, hasStatementSeparator := scanPreflightSQLTokens(query)
	if len(tokens) == 0 || (tokens[0] != "SELECT" && tokens[0] != "WITH") || hasStatementSeparator {
		return fmt.Errorf("migration preflight check %q must be one read-only SELECT or WITH query", check.Name)
	}
	for _, token := range tokens {
		if _, ok := forbiddenPreflightSQLKeywords[token]; ok {
			return fmt.Errorf("migration preflight check %q contains a mutating SQL keyword", check.Name)
		}
	}
	return nil
}

var forbiddenPreflightSQLKeywords = map[string]struct{}{
	"INSERT": {},
	"UPDATE": {},
	"DELETE": {},
	"MERGE":  {},
	"ALTER":  {},
	"DROP":   {},
	"CREATE": {},
	"COPY":   {},
	"CALL":   {},
	"DO":     {},
}

const (
	preflightSQLTokenCapacity = 8
	sqlPairLength             = 2
)

// scanPreflightSQLTokens 只扫描 SQL 代码区的关键字，避免换行、括号或注释让写入语句绕过 preflight 词法闸门。
func scanPreflightSQLTokens(query string) ([]string, bool) {
	tokens := make([]string, 0, preflightSQLTokenCapacity)
	hasStatementSeparator := false
	for i := 0; i < len(query); {
		next, token, separator := scanNextPreflightSQLToken(query, i)
		if token != "" {
			tokens = append(tokens, token)
		}
		hasStatementSeparator = hasStatementSeparator || separator
		i = next
	}
	return tokens, hasStatementSeparator
}

func scanNextPreflightSQLToken(query string, start int) (int, string, bool) {
	ch := query[start]
	if ch == '\'' {
		return skipSingleQuotedSQLString(query, start), "", false
	}
	if ch == '"' {
		return skipDoubleQuotedSQLIdentifier(query, start), "", false
	}
	if start+1 < len(query) && ch == '-' && query[start+1] == '-' {
		return skipSQLLineComment(query, start+sqlPairLength), "", false
	}
	if start+1 < len(query) && ch == '/' && query[start+1] == '*' {
		return skipSQLBlockComment(query, start+sqlPairLength), "", false
	}
	if ch == '$' {
		if next, ok := skipDollarQuotedSQLString(query, start); ok {
			return next, "", false
		}
	}
	if ch == ';' {
		return start + 1, "", true
	}
	if !isSQLIdentifierStart(ch) {
		return start + 1, "", false
	}
	end := start + 1
	for end < len(query) && isSQLIdentifierPart(query[end]) {
		end++
	}
	return end, strings.ToUpper(query[start:end]), false
}

func skipSingleQuotedSQLString(query string, start int) int {
	for i := start + 1; i < len(query); i++ {
		if query[i] != '\'' {
			continue
		}
		if i+1 < len(query) && query[i+1] == '\'' {
			i++
			continue
		}
		return i + 1
	}
	return len(query)
}

func skipDoubleQuotedSQLIdentifier(query string, start int) int {
	for i := start + 1; i < len(query); i++ {
		if query[i] != '"' {
			continue
		}
		if i+1 < len(query) && query[i+1] == '"' {
			i++
			continue
		}
		return i + 1
	}
	return len(query)
}

func skipSQLLineComment(query string, start int) int {
	for i := start; i < len(query); i++ {
		if query[i] == '\n' || query[i] == '\r' {
			return i + 1
		}
	}
	return len(query)
}

func skipSQLBlockComment(query string, start int) int {
	for i := start; i+1 < len(query); i++ {
		if query[i] == '*' && query[i+1] == '/' {
			return i + sqlPairLength
		}
	}
	return len(query)
}

func skipDollarQuotedSQLString(query string, start int) (int, bool) {
	end := start + 1
	for end < len(query) && query[end] != '$' {
		if !isSQLDollarQuoteTagPart(query[end]) {
			return start, false
		}
		end++
	}
	if end >= len(query) || query[end] != '$' {
		return start, false
	}
	delimiter := query[start : end+1]
	if next := strings.Index(query[end+1:], delimiter); next >= 0 {
		return end + 1 + next + len(delimiter), true
	}
	return len(query), true
}

func isSQLIdentifierStart(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_'
}

func isSQLIdentifierPart(ch byte) bool {
	return isSQLIdentifierStart(ch) || (ch >= '0' && ch <= '9') || ch == '$'
}

func isSQLDollarQuoteTagPart(ch byte) bool {
	return isSQLIdentifierPart(ch) && ch != '$'
}
