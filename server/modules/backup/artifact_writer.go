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
	root        string
	cfg         *config.Config
	dumpCommand func(context.Context, string, string) (*exec.Cmd, error)
}

func newFileArtifactWriter(cfg *config.Config) (*fileArtifactWriter, error) {
	if cfg == nil || cfg.Backup.ArtifactRoot == "" || cfg.Database.URL == "" {
		return nil, moduleapi.ErrBackupInvalidInput
	}
	return &fileArtifactWriter{root: cfg.Backup.ArtifactRoot, cfg: cfg, dumpCommand: databaseDumpCommand}, nil
}

// backupConfigSnapshot 只保留恢复审计所需的非敏感部署元数据，禁止将运行时完整配置写入工件。
type backupConfigSnapshot struct {
	SchemaVersion  int                   `json:"schema_version"`
	Application    backupApplicationInfo `json:"application"`
	EnabledModules []string              `json:"enabled_modules"`
	Database       backupDatabaseInfo    `json:"database"`
}

type backupApplicationInfo struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

type backupDatabaseInfo struct {
	Driver string `json:"driver"`
}

func backupConfigMetadata(cfg *config.Config) backupConfigSnapshot {
	return backupConfigSnapshot{
		SchemaVersion: 1,
		Application: backupApplicationInfo{
			Name:        cfg.App.Name,
			Environment: cfg.App.Env,
		},
		EnabledModules: append([]string(nil), cfg.Modules.Enabled...),
		Database:       backupDatabaseInfo{Driver: cfg.Database.Driver},
	}
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
	if err := w.ensureConfigSnapshot(configRef); err != nil {
		return moduleapi.CreateBackupInput{}, err
	}
	if _, err := os.Stat(dumpRef); os.IsNotExist(err) {
		if err := w.createDatabaseDump(ctx, dumpRef); err != nil {
			return moduleapi.CreateBackupInput{}, err
		}
	} else if err != nil {
		return moduleapi.CreateBackupInput{}, fmt.Errorf("stat database dump: %w", err)
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

// Verify 只读取冻结工件进行校验，绝不创建、修复或发布文件。
func (w *fileArtifactWriter) Verify(input backupTaskInput) (moduleapi.CreateBackupInput, error) {
	if w == nil || w.cfg == nil || !backupOperationID.MatchString(input.OperationID) {
		return moduleapi.CreateBackupInput{}, moduleapi.ErrBackupInvalidInput
	}
	directory := filepath.Join(w.root, input.OperationID)
	configRef := filepath.Join(directory, "config.snapshot")
	dumpRef := filepath.Join(directory, "database.dump")
	if err := validateConfigSnapshot(configRef); err != nil {
		return moduleapi.CreateBackupInput{}, err
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

func (w *fileArtifactWriter) ensureConfigSnapshot(target string) error {
	// #nosec G304 -- target is derived only from the configured Backup root and validated operation identity.
	if contents, err := os.ReadFile(target); err == nil {
		var snapshot backupConfigSnapshot
		if json.Unmarshal(contents, &snapshot) == nil {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read backup config snapshot: %w", err)
	}
	contents, err := json.Marshal(backupConfigMetadata(w.cfg))
	if err != nil {
		return fmt.Errorf("marshal backup config snapshot: %w", err)
	}
	return writeBackupArtifact(target, contents)
}

func validateConfigSnapshot(target string) error {
	// #nosec G304 -- target is derived only from the configured Backup root and validated operation identity.
	contents, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read backup config snapshot: %w", err)
	}
	var snapshot backupConfigSnapshot
	if err := json.Unmarshal(contents, &snapshot); err != nil {
		return fmt.Errorf("validate backup config snapshot: %w", err)
	}
	return nil
}

func writeBackupArtifact(target string, contents []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(target), ".config.snapshot-*")
	if err != nil {
		return fmt.Errorf("create temporary backup config snapshot: %w", err)
	}
	temporaryRef := temporary.Name()
	// 写入或发布失败时清理临时文件，避免下次重试误用未完成工件。
	defer func() { _ = os.Remove(temporaryRef) }()
	if err := temporary.Chmod(backupFilePermission); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set backup config snapshot permissions: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write backup config snapshot: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close backup config snapshot: %w", err)
	}
	if err := os.Rename(temporaryRef, target); err != nil {
		return fmt.Errorf("publish backup config snapshot: %w", err)
	}
	return nil
}

func (w *fileArtifactWriter) createDatabaseDump(ctx context.Context, target string) error {
	if w == nil || w.dumpCommand == nil {
		return moduleapi.ErrBackupInvalidInput
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".database.dump-*")
	if err != nil {
		return fmt.Errorf("create temporary database dump: %w", err)
	}
	temporaryRef := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryRef)
		return fmt.Errorf("close temporary database dump: %w", err)
	}
	// pg_dump 失败或发布失败时绝不留下可被下一次重试当作完成工件的半成品。
	defer func() { _ = os.Remove(temporaryRef) }()
	command, err := w.dumpCommand(ctx, temporaryRef, w.cfg.Database.URL)
	if err != nil {
		return err
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("run database dump: %w", err)
	}
	if err := os.Chmod(temporaryRef, backupFilePermission); err != nil {
		return fmt.Errorf("set database dump permissions: %w", err)
	}
	if err := os.Rename(temporaryRef, target); err != nil {
		return fmt.Errorf("publish database dump: %w", err)
	}
	return nil
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
