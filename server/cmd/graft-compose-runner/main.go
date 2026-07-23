// Package main provides the short-lived, no-HTTP Compose update runner entrypoint.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
	"graft/server/modules/update"
)

const (
	directoryPermission   os.FileMode = 0o700
	privateFilePermission os.FileMode = 0o600
)

// main 只执行一次性 Compose runner 协议，不启动 HTTP、数据库连接或业务状态。
func main() {
	path := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_RUNNER_INPUT"))
	// #nosec G304,G703 -- the server supplies a validated, host-mounted runner input path.
	contents, err := os.ReadFile(path)
	if path == "" || err != nil {
		fatal(fmt.Errorf("read runner input: %w", err))
	}
	var input update.RunnerInput
	if err := json.Unmarshal(contents, &input); err != nil {
		fatal(fmt.Errorf("decode runner input: %w", err))
	}
	if _, err := update.ExecuteComposeRunner(context.Background(), input, &actions{}); err != nil {
		fatal(err)
	}
}
func fatal(err error) { _, _ = fmt.Fprintln(os.Stderr, err); os.Exit(1) }

type actions struct {
	backup moduleapi.CompleteBackupRunnerHandoffInput
}

func (a *actions) Backup(ctx context.Context, in update.RunnerInput) error {
	root := filepath.Join(in.Preflight.ComposeRoot, ".graft-update", "backups", in.OperationID)
	if err := os.MkdirAll(root, directoryPermission); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(in.Preflight.ComposeRoot, ".env"), filepath.Join(root, "config.snapshot")); err != nil {
		return err
	}
	// #nosec G304 -- root derives from a preflight-validated host compose root and operation ID.
	dump, err := os.OpenFile(filepath.Join(root, "database.dump"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, privateFilePermission)
	if err != nil {
		return err
	}
	// #nosec G204 -- this fixed command has no caller-provided executable or arguments.
	command := exec.CommandContext(ctx, "docker", "compose", "--env-file", ".env", "-f", "compose.yml", "exec", "-T", "postgres", "sh", "-ec", "pg_dump -U \"$POSTGRES_USER\" \"$POSTGRES_DB\"")
	command.Dir, command.Stdout, command.Stderr = in.Preflight.ComposeRoot, dump, os.Stderr
	err = command.Run()
	closeErr := dump.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	configHash, configSize, err := digest(filepath.Join(root, "config.snapshot"))
	if err != nil {
		return err
	}
	dumpHash, dumpSize, err := digest(filepath.Join(root, "database.dump"))
	if err != nil {
		return err
	}
	a.backup = moduleapi.CompleteBackupRunnerHandoffInput{OperationID: in.OperationID, TaskID: in.TaskID, ConfigSnapshotSHA256: configHash, ConfigSnapshotBytes: configSize, DatabaseDumpSHA256: dumpHash, DatabaseDumpBytes: dumpSize}
	return nil
}
func (a *actions) BackupReceipt() moduleapi.CompleteBackupRunnerHandoffInput { return a.backup }
func (a *actions) Pull(ctx context.Context, in update.RunnerInput) error {
	if err := replaceRefs(filepath.Join(in.Preflight.ComposeRoot, ".env"), in.Preflight.ServerReference, in.Preflight.WebReference); err != nil {
		return err
	}
	return compose(ctx, in, "pull")
}
func (a *actions) BootstrapMigrate(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "run", "--rm", "bootstrap")
}
func (a *actions) Recreate(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "up", "-d", "--no-deps", "--force-recreate", "server", "web")
}
func (a *actions) DockerHealth(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "ps", "--status", "running", "--services", "server", "web")
}
func (a *actions) Healthz(ctx context.Context, in update.RunnerInput) error {
	return compose(ctx, in, "exec", "-T", "server", "curl", "--fail", "--silent", "http://127.0.0.1:8080/healthz")
}
func (a *actions) RecoverPreMigration(ctx context.Context, in update.RunnerInput) error {
	root := filepath.Join(in.Preflight.ComposeRoot, ".graft-update", "backups", in.OperationID)
	if err := copyFile(filepath.Join(root, "config.snapshot"), filepath.Join(in.Preflight.ComposeRoot, ".env")); err != nil {
		return err
	}
	return compose(ctx, in, "up", "-d", "--no-deps", "--force-recreate", "server", "web")
}
func compose(ctx context.Context, in update.RunnerInput, args ...string) error {
	// #nosec G204 -- callers select only fixed runner lifecycle argument sets.
	command := exec.CommandContext(ctx, "docker", append([]string{"compose", "--env-file", ".env", "-f", "compose.yml"}, args...)...)
	command.Dir, command.Stdout, command.Stderr = in.Preflight.ComposeRoot, os.Stdout, os.Stderr
	return command.Run()
}
func copyFile(source, target string) error {
	// #nosec G304 -- source and target are derived from the preflight-validated compose root.
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	// #nosec G304 -- source and target are derived from the preflight-validated compose root.
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, privateFilePermission)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
func digest(path string) (string, int64, error) {
	// #nosec G304 -- path is derived from the preflight-validated compose root.
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

//nolint:cyclop // 四个镜像坐标必须作为一组原子替换，避免 Compose 在任一服务上回退到旧引用。
func replaceRefs(path, server, web string) error {
	if !immutableReference(server) || !immutableReference(web) {
		return errors.New("compose runner image references must be immutable digests")
	}
	// #nosec G304 -- .env path is derived from the preflight-validated compose root.
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	serverRepository, serverDigest, err := splitImmutableReference(server)
	if err != nil {
		return err
	}
	webRepository, webDigest, err := splitImmutableReference(web)
	if err != nil {
		return err
	}
	values := map[string]string{"GRAFT_SERVER_IMAGE_REPOSITORY": serverRepository, "GRAFT_SERVER_IMAGE_DIGEST": serverDigest, "GRAFT_WEB_IMAGE_REPOSITORY": webRepository, "GRAFT_WEB_IMAGE_DIGEST": webDigest}
	lines := strings.Split(string(contents), "\n")
	for index, line := range lines {
		for key, value := range values {
			if strings.HasPrefix(line, key+"=") {
				lines[index] = key + "=" + value
				delete(values, key)
			}
		}
	}
	if len(values) != 0 {
		return errors.New("official compose environment lacks immutable image references")
	}
	temporary := path + ".graft-update-tmp"
	// #nosec G703 -- temporary is a fixed suffix under the preflight-validated compose root.
	if err := os.WriteFile(temporary, []byte(strings.Join(lines, "\n")), privateFilePermission); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func splitImmutableReference(reference string) (string, string, error) {
	repository, digest, ok := strings.Cut(strings.TrimSpace(reference), "@")
	if repository == "" || !ok || !strings.HasPrefix(digest, "sha256:") {
		return "", "", errors.New("image reference must use an immutable sha256 digest")
	}
	return repository, digest, nil
}

func immutableReference(value string) bool {
	repository, digest, ok := strings.Cut(strings.TrimSpace(value), "@")
	if !ok || repository == "" || !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	return err == nil
}
