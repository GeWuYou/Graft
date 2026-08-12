package cli

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
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

func TestValidateReadOnlyPreflightCheckRejectsWhitespaceSeparatedMutation(t *testing.T) {
	err := validateReadOnlyPreflightCheck(migrationPreflightCheck{
		Name: "unsafe",
		Query: `WITH deleted AS (
DELETE FROM registry_connections RETURNING id
) SELECT 0`,
	})
	if err == nil || !strings.Contains(err.Error(), "mutating SQL keyword") {
		t.Fatalf("expected mutating SQL keyword rejection, got %v", err)
	}
}

func TestRunMigrationPreflightChecksRejectsMutationBeforeQuery(t *testing.T) {
	db, probe := newPreflightProbeDB(t, 0)

	err := runMigrationPreflightChecks(context.Background(), db, []migrationPreflightCheck{{
		Name: "data-modifying-cte",
		Query: `WITH deleted AS (
DELETE FROM registry_connections RETURNING id
) SELECT 0`,
		MustEqual: 0,
	}})
	if err == nil || !strings.Contains(err.Error(), "mutating SQL keyword") {
		t.Fatalf("expected mutating SQL keyword rejection, got %v", err)
	}
	if probe.queryCount != 0 {
		t.Fatalf("expected lexical rejection before query execution, got %d queries", probe.queryCount)
	}
	if probe.rollbackCount != 1 {
		t.Fatalf("expected rollback after rejected preflight transaction, got %d", probe.rollbackCount)
	}
}

func TestRunMigrationPreflightChecksUsesReadOnlyTransaction(t *testing.T) {
	db, probe := newPreflightProbeDB(t, 0)

	err := runMigrationPreflightChecks(context.Background(), db, []migrationPreflightCheck{{
		Name:      "empty-duplicates",
		Query:     "SELECT 0",
		MustEqual: 0,
	}})
	if err != nil {
		t.Fatalf("run preflight checks: %v", err)
	}
	if len(probe.beginOptions) != 1 {
		t.Fatalf("expected one transaction begin, got %d", len(probe.beginOptions))
	}
	if !probe.beginOptions[0].ReadOnly {
		t.Fatalf("expected read-only transaction options, got %#v", probe.beginOptions[0])
	}
	if probe.queryCount != 1 {
		t.Fatalf("expected one preflight query, got %d", probe.queryCount)
	}
	if probe.rollbackCount != 1 {
		t.Fatalf("expected rollback after read-only preflight, got %d", probe.rollbackCount)
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

var preflightProbeDriverID atomic.Uint64

type preflightProbeState struct {
	beginOptions  []driver.TxOptions
	queryRows     []int64
	queryCount    int
	rollbackCount int
	commitCount   int
}

func newPreflightProbeDB(t *testing.T, queryRows ...int64) (*sql.DB, *preflightProbeState) {
	t.Helper()

	state := &preflightProbeState{queryRows: queryRows}
	driverName := fmt.Sprintf("graft-preflight-probe-%d", preflightProbeDriverID.Add(1))
	sql.Register(driverName, preflightProbeDriver{state: state})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open preflight probe database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, state
}

type preflightProbeDriver struct {
	state *preflightProbeState
}

func (d preflightProbeDriver) Open(string) (driver.Conn, error) {
	return &preflightProbeConn{state: d.state}, nil
}

type preflightProbeConn struct {
	state *preflightProbeState
}

func (c *preflightProbeConn) Prepare(query string) (driver.Stmt, error) {
	return preflightProbeStmt{conn: c, query: query}, nil
}

func (c *preflightProbeConn) Close() error {
	return nil
}

func (c *preflightProbeConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *preflightProbeConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.state.beginOptions = append(c.state.beginOptions, opts)
	return preflightProbeTx{state: c.state}, nil
}

func (c *preflightProbeConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	c.state.queryCount++
	return &preflightProbeRows{values: c.state.queryRows}, nil
}

type preflightProbeTx struct {
	state *preflightProbeState
}

func (tx preflightProbeTx) Commit() error {
	tx.state.commitCount++
	return nil
}

func (tx preflightProbeTx) Rollback() error {
	tx.state.rollbackCount++
	return nil
}

type preflightProbeStmt struct {
	conn  *preflightProbeConn
	query string
}

func (s preflightProbeStmt) Close() error {
	return nil
}

func (s preflightProbeStmt) NumInput() int {
	return -1
}

func (s preflightProbeStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, fmt.Errorf("unexpected preflight exec: %s", s.query)
}

func (s preflightProbeStmt) Query([]driver.Value) (driver.Rows, error) {
	s.conn.state.queryCount++
	return &preflightProbeRows{values: s.conn.state.queryRows}, nil
}

type preflightProbeRows struct {
	values []int64
	index  int
}

func (r *preflightProbeRows) Columns() []string {
	return []string{"count"}
}

func (r *preflightProbeRows) Close() error {
	return nil
}

func (r *preflightProbeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	dest[0] = r.values[r.index]
	r.index++
	return nil
}
