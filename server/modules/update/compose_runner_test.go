package update

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestExecuteComposeRunnerUsesFixedOrderAndWritesReceipt(t *testing.T) {
	root := t.TempDir()
	input := fixtureRunnerInput(root)
	actions := &tracingRunnerActions{}
	receipt, err := ExecuteComposeRunner(context.Background(), input, actions)
	if err != nil {
		t.Fatalf("execute compose runner: %v", err)
	}
	if !receipt.Succeeded || !receipt.MigrationStarted {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
	want := []string{"backup", "compose pull", "bootstrap migrate up", "compose recreate server web", "docker health", "healthz"}
	if !reflect.DeepEqual(actions.trace, want) {
		t.Fatalf("runner trace = %#v, want %#v", actions.trace, want)
	}
	persisted := readFixtureReceipt(t, input)
	if !reflect.DeepEqual(persisted, receipt) {
		t.Fatalf("persisted receipt = %#v, want %#v", persisted, receipt)
	}
}

func TestExecuteComposeRunnerMigrationFailureNeverRestoresDatabase(t *testing.T) {
	input := fixtureRunnerInput(t.TempDir())
	actions := &tracingRunnerActions{failAt: "bootstrap migrate up"}
	receipt, err := ExecuteComposeRunner(context.Background(), input, actions)
	if err == nil {
		t.Fatal("expected migration failure")
	}
	if receipt.FailureCode != runnerFailureMigration || !receipt.MigrationStarted || ClassifyRunnerReceipt(receipt) != ExecutionOutcomeNeedsAttention {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
	want := []string{"backup", "compose pull", "bootstrap migrate up"}
	if !reflect.DeepEqual(actions.trace, want) {
		t.Fatalf("runner trace = %#v, want %#v", actions.trace, want)
	}
}

func TestExecuteComposeRunnerRecoversOnlyBeforeMigration(t *testing.T) {
	input := fixtureRunnerInput(t.TempDir())
	actions := &tracingRunnerActions{failAt: "compose pull", recover: true}
	receipt, err := ExecuteComposeRunner(context.Background(), input, actions)
	if err == nil {
		t.Fatal("expected pull failure")
	}
	if receipt.MigrationStarted || !receipt.RecoveryCompleted || ClassifyRunnerReceipt(receipt) != ExecutionOutcomeRecovered {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
}

func TestExecuteComposeRunnerRejectsDigestMismatchBeforeBackup(t *testing.T) {
	input := fixtureRunnerInput(t.TempDir())
	input.Preflight.WebReference = "ghcr.io/gewuyou/graft-web:latest"
	actions := &tracingRunnerActions{}
	receipt, err := ExecuteComposeRunner(context.Background(), input, actions)
	if err == nil {
		t.Fatal("expected digest authority rejection")
	}
	if receipt.FailureCode != runnerFailureInvalidInput || len(actions.trace) != 0 {
		t.Fatalf("receipt = %#v, trace = %#v", receipt, actions.trace)
	}
}

func TestExecuteComposeRunnerRejectsMissingBackupCompletion(t *testing.T) {
	input := fixtureRunnerInput(t.TempDir())
	actions := &tracingRunnerActions{omitBackupReceipt: true}
	receipt, err := ExecuteComposeRunner(context.Background(), input, actions)
	if err == nil || receipt.FailureCode != runnerFailureBackup || receipt.BackupCompletion != nil {
		t.Fatalf("missing backup completion must produce a failed receipt: %#v / %v", receipt, err)
	}
	if len(actions.trace) != 1 || actions.trace[0] != "backup" {
		t.Fatalf("runner continued after missing backup completion: %#v", actions.trace)
	}
}

func TestPersistRunnerReceiptReplacesReceiptWithoutTemporaryFiles(t *testing.T) {
	input := fixtureRunnerInput(t.TempDir())
	if err := persistRunnerReceipt(input, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID, FailureCode: "first"}); err != nil {
		t.Fatalf("persist first receipt: %v", err)
	}
	if err := persistRunnerReceipt(input, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: input.OperationID, Succeeded: true}); err != nil {
		t.Fatalf("replace receipt: %v", err)
	}
	if receipt := readFixtureReceipt(t, input); !receipt.Succeeded || receipt.FailureCode != "" {
		t.Fatalf("receipt was not atomically replaced: %#v", receipt)
	}
	entries, err := os.ReadDir(filepath.Join(input.Preflight.ComposeRoot, runnerReceiptDirectory))
	if err != nil {
		t.Fatalf("list receipt directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != input.OperationID+".json" {
		t.Fatalf("temporary receipt files remain: %#v", entries)
	}
}

func TestComposeRunnerFixtureHasLocalSourceAndTargetVersions(t *testing.T) {
	contents, err := os.ReadFile("testdata/compose-runner/fixture.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture struct {
		Source struct {
			Server string `json:"server"`
			Web    string `json:"web"`
		} `json:"source"`
		Target struct {
			Server string `json:"server"`
			Web    string `json:"web"`
			Tag    string `json:"tag"`
		} `json:"target"`
		ExpectedTrace []string `json:"expected_trace"`
	}
	if err := json.Unmarshal(contents, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if fixture.Source.Server == fixture.Target.Server || fixture.Source.Web == fixture.Target.Web || fixture.Target.Tag == "" {
		t.Fatalf("fixture does not describe two versions: %#v", fixture)
	}
	if !strings.HasSuffix(fixture.Target.Server, ":"+fixture.Target.Tag) || !strings.HasSuffix(fixture.Target.Web, ":"+fixture.Target.Tag) {
		t.Fatalf("fixture target server/web do not share tag %q: %#v", fixture.Target.Tag, fixture.Target)
	}
	if !reflect.DeepEqual(fixture.ExpectedTrace, RunnerSequence()[:len(fixture.ExpectedTrace)]) {
		t.Fatalf("fixture trace = %#v, runner sequence = %#v", fixture.ExpectedTrace, RunnerSequence())
	}
}

func fixtureRunnerInput(root string) RunnerInput {
	serverDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	webDigest := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	runnerDigest := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	return RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: "fixture-operation-1", TaskID: 7, Preflight: ComposePreflight{
		DeclaredMode: "compose", DetectedMode: "compose", ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: []string{filepath.Join(root, "compose.yml")}, BundledPostgres: true,
		OfficialServerImage: "ghcr.io/gewuyou/graft-server", OfficialWebImage: "ghcr.io/gewuyou/graft-web", OfficialRunnerImage: "ghcr.io/gewuyou/graft-compose-runner",
		ServerDigest: serverDigest, WebDigest: webDigest, RunnerDigest: runnerDigest,
		ServerReference: "ghcr.io/gewuyou/graft-server@" + serverDigest, WebReference: "ghcr.io/gewuyou/graft-web@" + webDigest, RunnerReference: "ghcr.io/gewuyou/graft-compose-runner@" + runnerDigest,
	}}
}

func readFixtureReceipt(t *testing.T, input RunnerInput) RunnerReceipt {
	t.Helper()
	path, err := runnerReceiptPath(input)
	if err != nil {
		t.Fatalf("receipt path: %v", err)
	}
	// #nosec G304 -- path 由受限 operation ID 与当前测试的临时 Compose root 构造。
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read receipt: %v", err)
	}
	var receipt RunnerReceipt
	if err := json.Unmarshal(contents, &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	return receipt
}

type tracingRunnerActions struct {
	trace             []string
	failAt            string
	recover           bool
	omitBackupReceipt bool
	backup            moduleapi.CompleteBackupRunnerHandoffInput
}

func (actions *tracingRunnerActions) RecoverPreMigration(context.Context, RunnerInput) error {
	if !actions.recover {
		return errors.New("fixture recovery unavailable")
	}
	return nil
}

func (actions *tracingRunnerActions) Backup(context.Context, RunnerInput) error {
	return actions.run("backup")
}

func (actions *tracingRunnerActions) BackupReceipt() moduleapi.CompleteBackupRunnerHandoffInput {
	if actions.omitBackupReceipt {
		return moduleapi.CompleteBackupRunnerHandoffInput{}
	}
	if actions.backup.OperationID == "" {
		actions.backup = moduleapi.CompleteBackupRunnerHandoffInput{OperationID: "fixture-operation-1", TaskID: 7, ConfigSnapshotSHA256: testDigest('a'), ConfigSnapshotBytes: 3, DatabaseDumpSHA256: testDigest('b'), DatabaseDumpBytes: 5}
	}
	return actions.backup
}

func (actions *tracingRunnerActions) Pull(context.Context, RunnerInput) error {
	return actions.run("compose pull")
}

func (actions *tracingRunnerActions) BootstrapMigrate(context.Context, RunnerInput) error {
	return actions.run("bootstrap migrate up")
}

func (actions *tracingRunnerActions) Recreate(context.Context, RunnerInput) error {
	return actions.run("compose recreate server web")
}

func (actions *tracingRunnerActions) DockerHealth(context.Context, RunnerInput) error {
	return actions.run("docker health")
}

func (actions *tracingRunnerActions) Healthz(context.Context, RunnerInput) error {
	return actions.run("healthz")
}

func (actions *tracingRunnerActions) run(stage string) error {
	actions.trace = append(actions.trace, stage)
	if actions.failAt == stage {
		return errors.New("fixture stage failure")
	}
	return nil
}
