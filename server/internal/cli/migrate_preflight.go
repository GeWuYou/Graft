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

	for _, check := range manifest.PreflightChecks {
		if err := validateReadOnlyPreflightCheck(check); err != nil {
			return err
		}
		var actual int64
		if err := db.QueryRowContext(ctx, check.Query).Scan(&actual); err != nil {
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
	query := strings.TrimSpace(strings.ToUpper(check.Query))
	if (!strings.HasPrefix(query, "SELECT") && !strings.HasPrefix(query, "WITH")) || strings.Contains(query, ";") {
		return fmt.Errorf("migration preflight check %q must be one read-only SELECT or WITH query", check.Name)
	}
	for _, forbidden := range []string{" INSERT ", " UPDATE ", " DELETE ", " ALTER ", " DROP ", " CREATE ", " COPY ", " CALL ", " DO "} {
		if strings.Contains(" "+query+" ", forbidden) {
			return fmt.Errorf("migration preflight check %q contains a mutating SQL keyword", check.Name)
		}
	}
	return nil
}
