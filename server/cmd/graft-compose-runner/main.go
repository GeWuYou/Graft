// Package main provides the short-lived, no-HTTP Compose update runner entrypoint.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	directoryPermission                   os.FileMode = 0o700
	privateFilePermission                 os.FileMode = 0o600
	composeFileArgumentCapacityMultiplier             = 2
)

// main 只执行一次性 Compose runner 协议，不启动 HTTP、数据库连接或业务状态。
func main() {
	input, err := readRunnerInput()
	if err != nil {
		fatal(err)
	}
	receipt, executionErr := update.ExecuteComposeRunner(context.Background(), input, &actions{})
	if err := writeRunnerReceiptLog(os.Stdout, receipt); err != nil {
		fatal(fmt.Errorf("write runner receipt log: %w", err))
	}
	if executionErr != nil {
		fatal(executionErr)
	}
}

func readRunnerInput() (update.RunnerInput, error) {
	encoded := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_RUNNER_INPUT_B64"))
	if encoded != "" {
		contents, err := base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return update.RunnerInput{}, fmt.Errorf("decode inline runner input: %w", err)
		}
		return decodeRunnerInput(contents)
	}

	path := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_RUNNER_INPUT"))
	if path == "" {
		return update.RunnerInput{}, errors.New("runner input is missing")
	}
	// #nosec G304,G703 -- the server supplies a validated, host-mounted runner input path.
	contents, err := os.ReadFile(path)
	if err != nil {
		return update.RunnerInput{}, fmt.Errorf("read runner input: %w", err)
	}
	return decodeRunnerInput(contents)
}

func decodeRunnerInput(contents []byte) (update.RunnerInput, error) {
	var input update.RunnerInput
	if err := json.Unmarshal(contents, &input); err != nil {
		return update.RunnerInput{}, fmt.Errorf("decode runner input: %w", err)
	}
	return input, nil
}

func writeRunnerReceiptLog(writer io.Writer, receipt update.RunnerReceipt) error {
	contents, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("encode runner receipt: %w", err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(contents)
	if _, err := fmt.Fprintln(writer, update.RunnerReceiptLogMarker+encoded); err != nil {
		return err
	}
	return nil
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
	args := append([]string{"compose", "--env-file", ".env"}, composeFileArgs(in.Preflight.ComposeFiles)...)
	args = append(args, "exec", "-T", "postgres", "sh", "-ec", "pg_dump -U \"$POSTGRES_USER\" \"$POSTGRES_DB\"")
	// #nosec G204 -- this fixed command has no caller-provided executable or arguments.
	command := exec.CommandContext(ctx, "docker", args...)
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
	if err := replaceRefs(filepath.Join(in.Preflight.ComposeRoot, ".env"), in.Preflight.ServerReference, in.Preflight.WebReference, in.Preflight.UpdatePolicy); err != nil {
		return err
	}
	return compose(ctx, in, "pull")
}
func (a *actions) VerifyImages(ctx context.Context, in update.RunnerInput) error {
	if err := verifyImageDigest(ctx, in.Preflight.ServerReference, in.Preflight.ServerDigest); err != nil {
		return fmt.Errorf("verify server image: %w", err)
	}
	if err := verifyImageDigest(ctx, in.Preflight.WebReference, in.Preflight.WebDigest); err != nil {
		return fmt.Errorf("verify web image: %w", err)
	}
	return nil
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
	commandArgs := append([]string{"compose", "--env-file", ".env"}, composeFileArgs(in.Preflight.ComposeFiles)...)
	// #nosec G204 -- callers select only fixed runner lifecycle argument sets.
	command := exec.CommandContext(ctx, "docker", append(commandArgs, args...)...)
	command.Dir, command.Stdout, command.Stderr = in.Preflight.ComposeRoot, os.Stdout, os.Stderr
	return command.Run()
}

func composeFileArgs(files []string) []string {
	args := make([]string, 0, len(files)*composeFileArgumentCapacityMultiplier)
	for _, file := range files {
		args = append(args, "-f", file)
	}
	return args
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

// replaceRefs 以一次原子文件替换更新官方 Compose 的两个完整镜像引用，避免服务分别回退到不同 release。
//
//nolint:cyclop // 三个键必须共同校验、替换并验证完整性，拆分会破坏原子写入边界。
func replaceRefs(path, server, web string, policy update.UpdatePolicy) error {
	if !taggedReference(server) || !taggedReference(web) {
		return errors.New("compose runner image references must use explicit version tags")
	}
	if policy != update.UpdatePolicyStable && policy != update.UpdatePolicyBeta && policy != update.UpdatePolicyFixed {
		return errors.New("compose runner update policy is invalid")
	}
	// #nosec G304 -- .env path is derived from the preflight-validated compose root.
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	values := map[string]string{"GRAFT_SERVER_IMAGE": server, "GRAFT_WEB_IMAGE": web, "GRAFT_UPDATE_POLICY": string(policy)}
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
		return errors.New("official compose environment lacks complete image references")
	}
	temporary := path + ".graft-update-tmp"
	// #nosec G703 -- temporary is a fixed suffix under the preflight-validated compose root.
	if err := os.WriteFile(temporary, []byte(strings.Join(lines, "\n")), privateFilePermission); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func taggedReference(value string) bool {
	reference := value
	if reference == "" || strings.TrimSpace(reference) != reference || strings.Contains(reference, "@") || strings.ContainsAny(reference, " \t\r\n") {
		return false
	}
	lastSlash := strings.LastIndex(reference, "/")
	lastColon := strings.LastIndex(reference, ":")
	if lastColon <= lastSlash || lastColon >= len(reference)-1 {
		return false
	}
	return reference[lastColon+1:] != "latest"
}

func verifyImageDigest(ctx context.Context, reference, wantDigest string) error {
	if !taggedReference(reference) || !immutableDigest(wantDigest) {
		return errors.New("image verification input is invalid")
	}
	// #nosec G204 -- 引用来自经 manifest 校验的官方明确镜像标签。
	command := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", "{{json .RepoDigests}}", reference)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	contents, err := command.Output()
	if err != nil {
		if summary := strings.TrimSpace(stderr.String()); summary != "" {
			return fmt.Errorf("inspect pulled image: %w: %.256s", err, summary)
		}
		return err
	}
	var repoDigests []string
	if err := json.Unmarshal(contents, &repoDigests); err != nil {
		return fmt.Errorf("decode pulled image digests: %w", err)
	}
	if containsVerifiedRepoDigest(repoDigests, reference, wantDigest) {
		return nil
	}
	repository := reference[:strings.LastIndex(reference, ":")]
	if summary := strings.TrimSpace(stderr.String()); summary != "" {
		return fmt.Errorf("pulled image does not include verified digest %q: %.256s", repository+"@"+wantDigest, summary)
	}
	return fmt.Errorf("pulled image does not include verified digest %q", repository+"@"+wantDigest)
}

func containsVerifiedRepoDigest(repoDigests []string, reference, wantDigest string) bool {
	repository := reference[:strings.LastIndex(reference, ":")]
	want := repository + "@" + wantDigest
	for _, actual := range repoDigests {
		if actual == want {
			return true
		}
	}
	return false
}

func immutableDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
