package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

var backupOperationID = regexp.MustCompile(`^[a-zA-Z0-9-]{1,128}$`)

const (
	backupDirectoryPermission = 0o700
	backupFilePermission      = 0o600
)

type fileArtifactWriter struct {
	root string
	cfg  *config.Config
}

func newFileArtifactWriter(cfg *config.Config) (*fileArtifactWriter, error) {
	if cfg == nil || cfg.Backup.ArtifactRoot == "" || cfg.Database.URL == "" {
		return nil, moduleapi.ErrBackupInvalidInput
	}
	return &fileArtifactWriter{root: cfg.Backup.ArtifactRoot, cfg: cfg}, nil
}

//nolint:cyclop // Artifact creation keeps every storage boundary visible in one owner method.
func (w *fileArtifactWriter) Create(ctx context.Context, input backupTaskInput) (moduleapi.CreateBackupInput, error) {
	if w == nil || w.cfg == nil || !backupOperationID.MatchString(input.OperationID) {
		return moduleapi.CreateBackupInput{}, moduleapi.ErrBackupInvalidInput
	}
	directory := filepath.Join(w.root, input.OperationID)
	if err := os.MkdirAll(directory, backupDirectoryPermission); err != nil {
		return moduleapi.CreateBackupInput{}, fmt.Errorf("create backup artifact directory: %w", err)
	}
	configRef := filepath.Join(directory, "config.snapshot")
	dumpRef := filepath.Join(directory, "database.dump")
	if _, err := os.Stat(configRef); os.IsNotExist(err) {
		contents, marshalErr := json.Marshal(w.cfg)
		if marshalErr != nil {
			return moduleapi.CreateBackupInput{}, fmt.Errorf("marshal backup config snapshot: %w", marshalErr)
		}
		if writeErr := os.WriteFile(configRef, contents, backupFilePermission); writeErr != nil {
			return moduleapi.CreateBackupInput{}, fmt.Errorf("write backup config snapshot: %w", writeErr)
		}
	}
	if _, err := os.Stat(dumpRef); os.IsNotExist(err) {
		command, commandErr := databaseDumpCommand(ctx, dumpRef, w.cfg.Database.URL)
		if commandErr != nil {
			return moduleapi.CreateBackupInput{}, commandErr
		}
		if runErr := command.Run(); runErr != nil {
			return moduleapi.CreateBackupInput{}, fmt.Errorf("run database dump: %w", runErr)
		}
	}
	configArtifact, err := digestBackupArtifact(configRef)
	if err != nil {
		return moduleapi.CreateBackupInput{}, err
	}
	dumpArtifact, err := digestBackupArtifact(dumpRef)
	if err != nil {
		return moduleapi.CreateBackupInput{}, err
	}
	actor := input.RequestedBy
	return moduleapi.CreateBackupInput{Purpose: backupTaskPurpose, ConfigSnapshot: configArtifact, DatabaseDump: dumpArtifact, RetainUntil: input.RetainUntil, CreatedBy: &actor}, nil
}

// databaseDumpCommand keeps database credentials in the child environment and
// never includes them in Task input, logs, or process arguments.
func databaseDumpCommand(ctx context.Context, target string, databaseURL string) (*exec.Cmd, error) {
	connection, err := pgconn.ParseConfig(databaseURL)
	if err != nil || connection.Host == "" || connection.Database == "" || connection.User == "" {
		return nil, moduleapi.ErrBackupInvalidInput
	}
	// #nosec G204 -- executable and option set are fixed; parsed connection values never enter Task input or logs.
	command := exec.CommandContext(ctx, "pg_dump", "--file", target, "--host", connection.Host, "--port", strconv.FormatUint(uint64(connection.Port), 10), "--username", connection.User, "--dbname", connection.Database)
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "PGPASSWORD=" + connection.Password}
	if connection.TLSConfig == nil {
		command.Env = append(command.Env, "PGSSLMODE=disable")
	}
	return command, nil
}

func digestBackupArtifact(ref string) (moduleapi.BackupArtifact, error) {
	// #nosec G304 -- ref is derived only from the configured Backup root and validated operation identity.
	file, err := os.Open(ref)
	if err != nil {
		return moduleapi.BackupArtifact{}, fmt.Errorf("open backup artifact: %w", err)
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return moduleapi.BackupArtifact{}, fmt.Errorf("digest backup artifact: %w", err)
	}
	return moduleapi.BackupArtifact{StorageRef: ref, SHA256: hex.EncodeToString(hash.Sum(nil)), SizeBytes: size}, nil
}
