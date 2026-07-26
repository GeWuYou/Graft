package backup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/config"
)

func TestFileArtifactWriterWritesSanitizedConfigSnapshot(t *testing.T) {
	root := t.TempDir()
	writer := newTestArtifactWriter(root)
	writer.cfg.Auth.JWTSecret = "must-not-be-snapshotted"
	writer.cfg.Auth.SigningKey = "must-not-be-snapshotted"
	writer.cfg.Database.URL = "database-url-from-test"
	writer.dumpCommand = successfulDumpCommand("database dump")

	if _, err := writer.Create(t.Context(), testBackupTaskInput()); err != nil {
		t.Fatalf("create backup artifacts: %v", err)
	}
	// #nosec G304 -- path is composed from this test's t.TempDir and a fixed operation ID.
	contents, err := os.ReadFile(filepath.Join(root, "backup-42", "config.snapshot"))
	if err != nil {
		t.Fatalf("read config snapshot: %v", err)
	}
	for _, secret := range []string{"must-not-be-snapshotted", "database-url-from-test"} {
		if strings.Contains(string(contents), secret) {
			t.Fatalf("config snapshot leaked secret %q: %s", secret, contents)
		}
	}
	var snapshot backupConfigSnapshot
	if err := json.Unmarshal(contents, &snapshot); err != nil {
		t.Fatalf("decode config snapshot: %v", err)
	}
	if snapshot.SchemaVersion != 1 || snapshot.Application.Name != "graft" || snapshot.Application.Environment != "test" || snapshot.Database.Driver != "postgres" {
		t.Fatalf("unexpected sanitized snapshot: %#v", snapshot)
	}
}

func TestFileArtifactWriterReplacesCorruptSnapshotOnRetry(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "backup-42")
	if err := os.MkdirAll(directory, backupDirectoryPermission); err != nil {
		t.Fatalf("create backup artifact directory: %v", err)
	}
	configRef := filepath.Join(directory, "config.snapshot")
	if err := os.WriteFile(configRef, []byte(`{"schema_version":`), backupFilePermission); err != nil {
		t.Fatalf("write corrupt config snapshot: %v", err)
	}
	writer := newTestArtifactWriter(root)
	writer.dumpCommand = successfulDumpCommand("complete dump")
	if _, err := writer.Create(t.Context(), testBackupTaskInput()); err != nil {
		t.Fatalf("retry backup artifact creation: %v", err)
	}
	// #nosec G304 -- path is composed from this test's t.TempDir and a fixed operation ID.
	contents, err := os.ReadFile(configRef)
	if err != nil {
		t.Fatalf("read repaired config snapshot: %v", err)
	}
	var snapshot backupConfigSnapshot
	if err := json.Unmarshal(contents, &snapshot); err != nil {
		t.Fatalf("expected repaired config snapshot to be valid JSON: %v", err)
	}
	if snapshot.SchemaVersion != 1 {
		t.Fatalf("unexpected repaired schema version %d", snapshot.SchemaVersion)
	}
}

func TestFileArtifactWriterVerifyDoesNotRepairOrWriteArtifacts(t *testing.T) {
	root := t.TempDir()
	writer := newTestArtifactWriter(root)
	writer.dumpCommand = func(context.Context, string, string) (*exec.Cmd, error) {
		t.Fatal("Verify must not invoke pg_dump")
		return nil, nil
	}
	directory := filepath.Join(root, "backup-42")
	if err := os.MkdirAll(directory, backupDirectoryPermission); err != nil {
		t.Fatalf("create artifact directory: %v", err)
	}
	configRef := filepath.Join(directory, "config.snapshot")
	if err := os.WriteFile(configRef, []byte(`{"schema_version":`), backupFilePermission); err != nil {
		t.Fatalf("write corrupt config snapshot: %v", err)
	}
	// #nosec G304 -- path is composed from this test's t.TempDir and a fixed operation ID.
	before, err := os.ReadFile(configRef)
	if err != nil {
		t.Fatalf("read corrupt config snapshot: %v", err)
	}
	if _, err := writer.Verify(testBackupTaskInput()); err == nil {
		t.Fatal("expected frozen artifact verification failure")
	}
	// #nosec G304 -- path is composed from this test's t.TempDir and a fixed operation ID.
	after, err := os.ReadFile(configRef)
	if err != nil || string(after) != string(before) {
		t.Fatalf("Verify mutated config snapshot: before=%q after=%q err=%v", before, after, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "database.dump")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Verify created database dump: %v", err)
	}
}

func TestFileArtifactWriterPublishesOnlySuccessfulDatabaseDump(t *testing.T) {
	root := t.TempDir()
	writer := newTestArtifactWriter(root)
	writer.dumpCommand = failingDumpCommand
	if _, err := writer.Create(t.Context(), testBackupTaskInput()); err == nil {
		t.Fatal("expected dump failure")
	}
	directory := filepath.Join(root, "backup-42")
	if _, err := os.Stat(filepath.Join(directory, "database.dump")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no published dump after failure, got %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read artifact directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".database.dump-") {
			t.Fatalf("temporary dump was not cleaned up: %s", entry.Name())
		}
	}

	writer.dumpCommand = successfulDumpCommand("complete dump")
	if _, err := writer.Create(t.Context(), testBackupTaskInput()); err != nil {
		t.Fatalf("retry backup artifacts: %v", err)
	}
	// #nosec G304 -- directory is composed from this test's t.TempDir and a fixed operation ID.
	contents, err := os.ReadFile(filepath.Join(directory, "database.dump"))
	if err != nil || string(contents) != "complete dump" {
		t.Fatalf("expected published complete dump, contents=%q err=%v", contents, err)
	}
}

func newTestArtifactWriter(root string) *fileArtifactWriter {
	return &fileArtifactWriter{root: root, cfg: &config.Config{
		App:      config.AppConfig{Name: "graft", Env: "test"},
		Modules:  config.ModulesConfig{Enabled: []string{"task", "platform-backup"}},
		Database: config.DatabaseConfig{Driver: "postgres", URL: "database-url-from-test"},
	}}
}

func testBackupTaskInput() backupTaskInput {
	return backupTaskInput{OperationID: "backup-42", RequestedBy: 42, RetainUntil: time.Now().UTC().Add(time.Hour)}
}

func successfulDumpCommand(contents string) func(context.Context, string, string) (*exec.Cmd, error) {
	return func(ctx context.Context, target string, _ string) (*exec.Cmd, error) {
		// #nosec G204 -- test command and shell program are fixed; values are positional arguments, not shell source.
		return exec.CommandContext(ctx, "sh", "-c", "printf %s \"$1\" > \"$2\"", "backup-test", contents, target), nil
	}
}

func failingDumpCommand(ctx context.Context, target string, _ string) (*exec.Cmd, error) {
	// #nosec G204 -- test command and shell program are fixed; target is passed as a quoted positional argument.
	return exec.CommandContext(ctx, "sh", "-c", "printf partial > \"$1\"; exit 1", "backup-test", target), nil
}
