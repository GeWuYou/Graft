package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"graft/server/modules/update"
)

func TestReadRunnerInputPrefersInlinePayload(t *testing.T) {
	input := update.RunnerInput{ProtocolVersion: 2, OperationID: "inline-operation"}
	contents, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal runner input: %v", err)
	}
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT_B64", base64.RawStdEncoding.EncodeToString(contents))
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT", filepath.Join(t.TempDir(), "missing.json"))
	got, err := readRunnerInput()
	if err != nil {
		t.Fatalf("read inline runner input: %v", err)
	}
	if got.OperationID != input.OperationID {
		t.Fatalf("operation ID = %q, want %q", got.OperationID, input.OperationID)
	}
}

func TestReadRunnerInputRejectsInvalidInlinePayloadWithoutFileFallback(t *testing.T) {
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT_B64", "not-base64")
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`{"operation_id":"legacy-operation"}`), privateFilePermission); err != nil {
		t.Fatalf("write legacy runner input: %v", err)
	}
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT", path)
	if _, err := readRunnerInput(); err == nil || !strings.Contains(err.Error(), "decode inline runner input") {
		t.Fatalf("invalid inline input error = %v", err)
	}
}

func TestWriteRunnerReceiptLogUsesFixedMarkerAndBase64JSON(t *testing.T) {
	receipt := update.RunnerReceipt{ProtocolVersion: 2, OperationID: "receipt-operation", Succeeded: true}
	var output bytes.Buffer
	if err := writeRunnerReceiptLog(&output, receipt); err != nil {
		t.Fatalf("write runner receipt log: %v", err)
	}
	line := strings.TrimSpace(output.String())
	if !strings.HasPrefix(line, update.RunnerReceiptLogMarker) {
		t.Fatalf("receipt marker missing: %q", line)
	}
	encoded := strings.TrimPrefix(line, update.RunnerReceiptLogMarker)
	contents, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode receipt log: %v", err)
	}
	var got update.RunnerReceipt
	if err := json.Unmarshal(contents, &got); err != nil {
		t.Fatalf("decode receipt JSON: %v", err)
	}
	if got != receipt {
		t.Fatalf("receipt = %#v, want %#v", got, receipt)
	}
}

func TestRecoverTerminatedRunnerWritesARecoveryRunnerIdentity(t *testing.T) {
	store, err := update.NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new runner state store: %v", err)
	}
	input := update.RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: "update-recovery-identity", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Preflight: update.ComposePreflight{DeploymentStrategy: update.DeploymentStrategyBetaTracking}}
	persisted := update.NewRunnerState(input, "runner-original", update.RunnerPhaseReady, 0, "runner_accepted", "", update.RunnerState{})
	persisted.LeaseHeartbeatAt = time.Now().UTC().Add(-10 * time.Minute)
	persisted.LeaseExpiresAt = time.Now().UTC().Add(-5 * time.Minute)
	if err := store.Write(persisted); err != nil {
		t.Fatalf("write persisted runner state: %v", err)
	}
	persisted, err = store.Read()
	if err != nil {
		t.Fatalf("read persisted runner state: %v", err)
	}
	contents, err := json.Marshal(update.RunnerRecoveryInput{OperationID: persisted.OperationID, RunnerID: persisted.RunnerID, SourceVersion: persisted.SourceVersion, TargetVersion: persisted.TargetVersion, Strategy: persisted.Strategy, State: &persisted})
	if err != nil {
		t.Fatalf("marshal persisted runner state: %v", err)
	}
	var output bytes.Buffer
	if err := recoverTerminatedRunnerWithStore(base64.RawStdEncoding.EncodeToString(contents), store, &output); err != nil {
		t.Fatalf("recover terminated runner: %v", err)
	}
	recovered, err := store.Read()
	if err != nil {
		t.Fatalf("read recovered runner state: %v", err)
	}
	if recovered.RunnerID == persisted.RunnerID || recovered.Receipt == nil || recovered.Receipt.RunnerID != recovered.RunnerID {
		t.Fatalf("recovery identity was not persisted: state=%#v", recovered)
	}
	if !strings.Contains(output.String(), update.RunnerReceiptLogMarker) {
		t.Fatalf("recovery receipt was not emitted: %q", output.String())
	}
}

func TestRecoverTerminatedRunnerWritesTerminalStateWhenInitialSnapshotIsMissing(t *testing.T) {
	store, err := update.NewFileRunnerStateStore(t.TempDir())
	if err != nil {
		t.Fatalf("new runner state store: %v", err)
	}
	recovery := update.RunnerRecoveryInput{OperationID: "update-recovery-missing", RunnerID: "runner-recovery-missing", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Strategy: string(update.DeploymentStrategyBetaTracking)}
	contents, err := json.Marshal(recovery)
	if err != nil {
		t.Fatalf("marshal recovery input: %v", err)
	}
	if err := recoverTerminatedRunnerWithStore(base64.RawStdEncoding.EncodeToString(contents), store, io.Discard); err != nil {
		t.Fatalf("recover missing initial state: %v", err)
	}
	state, err := store.Read()
	if err != nil || !isTerminalPhase(state.Phase) || state.Receipt == nil {
		t.Fatalf("terminal recovery state = %#v, %v", state, err)
	}
}

func TestHealthzArgsBoundsCurlExecution(t *testing.T) {
	args := healthzArgs()
	maxTimeIndex := slices.Index(args, "--max-time")
	if maxTimeIndex < 0 || maxTimeIndex+1 >= len(args) || args[maxTimeIndex+1] != healthzCurlTimeoutSeconds {
		t.Fatalf("healthz args must bound curl execution: %#v", args)
	}
}

func TestBackupFailureDoesNotExposeFilesystemDetails(t *testing.T) {
	err := backupFailure(update.RunnerBackupFailureStageConfigSnapshot, &os.PathError{Op: "open", Path: "/opt/graft/.env", Err: os.ErrPermission})
	if got := err.Error(); got != "env_snapshot: permission_denied" {
		t.Fatalf("safe backup error = %q", got)
	}
}

func TestCleanupBackupStagingRemovesOperationDirectoryAfterSuccessOrCopyFailure(t *testing.T) {
	root := t.TempDir()
	input := update.RunnerInput{
		ProtocolVersion:    2,
		OperationID:        "operation-cleanup",
		TaskID:             1,
		BackupArtifactRoot: "/var/lib/graft/backups/operation-cleanup",
		Preflight: update.ComposePreflight{
			DeclaredMode: "compose", DeploymentStrategy: update.DeploymentStrategyBetaTracking, ImageTag: "beta", DetectedMode: "compose", ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: []string{filepath.Join(root, "compose.yml")}, BundledPostgres: true,
			OfficialServerImage: "ghcr.io/gewuyou/graft-server", OfficialWebImage: "ghcr.io/gewuyou/graft-web", OfficialRunnerImage: "ghcr.io/gewuyou/graft-compose-runner",
			ServerReference: "ghcr.io/gewuyou/graft-server:beta", WebReference: "ghcr.io/gewuyou/graft-web:beta", RunnerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ServerDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", WebDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", RunnerDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	stagingRoot := filepath.Join(root, ".graft-update", "backups", input.OperationID)
	if err := os.MkdirAll(stagingRoot, directoryPermission); err != nil {
		t.Fatalf("create staging root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stagingRoot, "database.dump"), []byte("backup"), privateFilePermission); err != nil {
		t.Fatalf("write staging artifact: %v", err)
	}
	retained := filepath.Join(root, ".graft-update", "backups", "other-operation")
	if err := os.MkdirAll(retained, directoryPermission); err != nil {
		t.Fatalf("create retained root: %v", err)
	}

	if err := cleanupBackupStaging(input); err != nil {
		t.Fatalf("cleanup staging root: %v", err)
	}
	if _, err := os.Stat(stagingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staging root stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(retained); err != nil {
		t.Fatalf("retained backup root stat: %v", err)
	}

	if err := os.MkdirAll(stagingRoot, directoryPermission); err != nil {
		t.Fatalf("recreate staging root for partial copy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stagingRoot, "config.snapshot"), []byte("partial"), privateFilePermission); err != nil {
		t.Fatalf("write partial staging artifact: %v", err)
	}
	if err := cleanupBackupStaging(input); err != nil {
		t.Fatalf("cleanup partial-copy staging root: %v", err)
	}
	if _, err := os.Stat(stagingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial-copy staging root stat error = %v, want not exist", err)
	}
}

func TestComposeFileArgsPreservesEveryPreflightFileInOrder(t *testing.T) {
	files := []string{"/opt/graft/compose.yaml", "/opt/graft/overrides/web.yml"}
	want := []string{"-f", files[0], "-f", files[1]}
	if got := composeFileArgs(files); !reflect.DeepEqual(got, want) {
		t.Fatalf("compose file args = %#v, want %#v", got, want)
	}
}

func TestReplaceRefsReplacesSharedComposeImageTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_IMAGE_TAG=v1.2.2-beta.1\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server:1.2.3-beta.1"
	web := "ghcr.io/gewuyou/graft-web:1.2.3-beta.1"
	if err := replaceRefs(path, pinnedPreflight(server, web)); err != nil {
		t.Fatalf("replace compose image tag: %v", err)
	}
	// #nosec G304 -- path is a test-owned file under this test's temporary directory.
	contents, err := os.ReadFile(path) // #nosec G304 -- test-controlled temporary file.
	if err != nil {
		t.Fatalf("read updated compose environment: %v", err)
	}
	if strings.Contains(string(contents), "v1.2.2-beta.1") || !strings.Contains(string(contents), "GRAFT_IMAGE_TAG=1.2.3-beta.1") {
		t.Fatalf("compose environment does not contain the shared release tag: %s", contents)
	}
}

func TestReplaceRefsRejectsMissingComposeImageTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_APP_NAME=graft\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server:1.2.3-beta.1"
	web := "ghcr.io/gewuyou/graft-web:1.2.3-beta.1"
	if err := replaceRefs(path, pinnedPreflight(server, web)); err == nil {
		t.Fatal("expected missing image tag to reject update")
	}
}

func TestReplaceRefsRejectsInvalidTargetReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_IMAGE_TAG=latest\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	for _, target := range []struct {
		server string
		web    string
	}{
		{server: "ghcr.io/gewuyou/graft-server:latest", web: "ghcr.io/gewuyou/graft-web:latest"},
		{server: "ghcr.io/gewuyou/graft-server:1.2.3", web: "ghcr.io/gewuyou/graft-web:1.2.4"},
		{server: "registry.example/graft-server:1.2.3", web: "ghcr.io/gewuyou/graft-web:1.2.3"},
		{server: "ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("a", 64), web: "ghcr.io/gewuyou/graft-web:1.2.3"},
	} {
		if err := replaceRefs(path, pinnedPreflight(target.server, target.web)); err == nil {
			t.Fatalf("invalid target references accepted: server=%q web=%q", target.server, target.web)
		}
	}
}

func TestReplaceRefsKeepsTrackingTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_IMAGE_TAG=beta\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	preflight := pinnedPreflight("registry.example/graft-server:1.2.3-beta.1", "registry.example/graft-web:1.2.3-beta.1")
	preflight.DeploymentStrategy, preflight.ImageTag = update.DeploymentStrategyBetaTracking, "beta"
	if err := replaceRefs(path, preflight); err != nil {
		t.Fatalf("retain tracking tag: %v", err)
	}
	// #nosec G304 -- 测试仅读取本例刚写入的临时文件。
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "GRAFT_IMAGE_TAG=beta\n" {
		t.Fatalf("tracking tag changed or tracking references were validated: %q, %v", contents, err)
	}
}

func pinnedPreflight(server, web string) update.ComposePreflight {
	return update.ComposePreflight{ServerReference: server, WebReference: web, OfficialServerImage: "ghcr.io/gewuyou/graft-server", OfficialWebImage: "ghcr.io/gewuyou/graft-web", DeploymentStrategy: update.DeploymentStrategyPinnedBeta, ImageTag: "v1.2.2-beta.1"}
}

func TestReferenceTagRejectsInvalidTags(t *testing.T) {
	for _, reference := range []string{
		"ghcr.io/gewuyou/graft-server:latest",
		"ghcr.io/gewuyou/graft-server:1.2.3 beta.1",
		"ghcr.io/gewuyou/graft-server:1.2.3\n",
		"ghcr.io/gewuyou/graft-server:1:2.3",
		"ghcr.io/gewuyou/graft-server:release/1.2.3",
		"ghcr.io/gewuyou/graft-server:.1.2.3",
		"ghcr.io/gewuyou/graft-server:-1.2.3",
		"ghcr.io/gewuyou/graft-server:" + strings.Repeat("a", 129),
	} {
		if _, ok := referenceTag(reference, "ghcr.io/gewuyou/graft-server"); ok {
			t.Fatalf("invalid image tag accepted: %q", reference)
		}
	}
	if tag, ok := referenceTag("ghcr.io/gewuyou/graft-server:1.2.3-beta.1", "ghcr.io/gewuyou/graft-server"); !ok || tag != "1.2.3-beta.1" {
		t.Fatal("explicit version tag rejected")
	}
}

func TestContainsVerifiedRepoDigestSearchesEveryRepositoryDigest(t *testing.T) {
	wantDigest := "sha256:" + strings.Repeat("a", 64)
	repoDigests := []string{
		"ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("b", 64),
		"ghcr.io/gewuyou/graft-server@" + wantDigest,
	}
	if !containsVerifiedRepoDigest(repoDigests, "ghcr.io/gewuyou/graft-server:1.2.3-beta.1", wantDigest) {
		t.Fatal("verified digest after the first repository digest was not accepted")
	}
	if containsVerifiedRepoDigest(repoDigests, "ghcr.io/gewuyou/graft-web:1.2.3-beta.1", wantDigest) {
		t.Fatal("server repository digest was accepted for the web reference")
	}
}
