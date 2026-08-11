package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestValidateReadOnlyPreflightCheckRejectsMutation(t *testing.T) {
	err := validateReadOnlyPreflightCheck(migrationPreflightCheck{
		Name:  "unsafe",
		Query: "SELECT 1; DELETE FROM registry_connections",
	})
	if err == nil || !strings.Contains(err.Error(), "read-only SELECT") {
		t.Fatalf("expected read-only query rejection, got %v", err)
	}
}

func TestRunMigrationPreflightChecksUsesReadOnlyTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer func() { _ = db.Close() }()

	query := "WITH deleted AS (\nDELETE FROM registry_connections RETURNING id\n) SELECT 0"
	mock.ExpectBegin()
	mock.ExpectQuery("WITH deleted AS").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectRollback()

	err = runMigrationPreflightChecks(context.Background(), db, []migrationPreflightCheck{{
		Name:      "data-modifying-cte",
		Query:     query,
		MustEqual: 0,
	}})
	if err != nil {
		t.Fatalf("run preflight checks: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("preflight transaction expectations: %v", err)
	}
}

func TestLoadMigrationPreflightManifestParsesChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.preflight.yaml")
	content := "migration:\n  path: server/modules/demo/migrations/202608120001_demo.sql\n  version: '202608120001'\npreflight_checks:\n  - name: duplicate-groups\n    query: SELECT 0\n    must_equal: 0\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	manifest, err := loadMigrationPreflightManifest(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if manifest.Migration.Version != "202608120001" || len(manifest.PreflightChecks) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}
