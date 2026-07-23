package migrationcontract

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCheckerReportsEnforceAndReportFindings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	expectQuery(mock, foreignKeyContractQuery, []string{"schema", "table", "name", "definition", "target_key_exists", "validated"},
		[]driver.Value{"public", "child", "fk_child_parent__parent_id", "FOREIGN KEY (parent_id) REFERENCES parent(id)", false, false})
	expectQuery(mock, invalidIndexQuery, []string{"schema", "table", "name"}, []driver.Value{"public", "child", "idx_child_broken"})
	expectQuery(mock, atlasRevisionTableQuery, []string{"exists"}, []driver.Value{true})
	expectQuery(mock, atlasRevisionPartialQuery, []string{"partial_count"}, []driver.Value{int64(1)})
	expectQuery(mock, constraintNameQuery, []string{"schema", "table", "name", "kind"}, []driver.Value{"public", "child", "child_parent_fkey", "f"})
	expectQuery(mock, indexNameQuery, []string{"schema", "table", "name"}, []driver.Value{"public", "child", "child_parent_idx"})
	expectQuery(mock, foreignKeySourceIndexQuery, []string{"schema", "table", "name"}, []driver.Value{"public", "child", "fk_child_parent__parent_id"})

	checker, err := NewChecker(db)
	if err != nil {
		t.Fatalf("new checker: %v", err)
	}
	result, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("check schema: %v", err)
	}
	if !result.HasEnforceFindings() {
		t.Fatal("expected enforce findings")
	}

	got := make(map[string]Severity, len(result.Findings))
	for _, finding := range result.Findings {
		got[finding.ID] = finding.Severity
	}
	want := map[string]Severity{
		FindingFKTargetKeyMissing:   SeverityEnforce,
		FindingFKNotValidated:       SeverityEnforce,
		FindingInvalidIndex:         SeverityEnforce,
		FindingAtlasRevisionPartial: SeverityEnforce,
		FindingConstraintName:       SeverityReport,
		FindingIndexName:            SeverityReport,
		FindingFKSourceIndexMissing: SeverityReport,
	}
	for id, severity := range want {
		if got[id] != severity {
			t.Errorf("finding %s severity = %q, want %q", id, got[id], severity)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("query expectations: %v", err)
	}
}

func TestCheckerReportsMissingAtlasRevisionTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	expectQuery(mock, foreignKeyContractQuery, []string{"schema", "table", "name", "definition", "target_key_exists", "validated"})
	expectQuery(mock, invalidIndexQuery, []string{"schema", "table", "name"})
	expectQuery(mock, atlasRevisionTableQuery, []string{"exists"}, []driver.Value{false})
	expectQuery(mock, constraintNameQuery, []string{"schema", "table", "name", "kind"})
	expectQuery(mock, indexNameQuery, []string{"schema", "table", "name"})
	expectQuery(mock, foreignKeySourceIndexQuery, []string{"schema", "table", "name"})

	checker, err := NewChecker(db)
	if err != nil {
		t.Fatalf("new checker: %v", err)
	}
	result, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("check schema: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].ID != FindingAtlasRevisionTableMissing {
		t.Fatalf("findings = %#v, want only missing revision table", result.Findings)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("query expectations: %v", err)
	}
}

func TestNewCheckerRequiresQueryer(t *testing.T) {
	if _, err := NewChecker(nil); err == nil {
		t.Fatal("expected nil queryer to fail")
	}
}

func expectQuery(mock sqlmock.Sqlmock, query string, columns []string, values ...[]driver.Value) {
	rows := sqlmock.NewRows(columns)
	for _, row := range values {
		rows.AddRow(row...)
	}
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)
}
