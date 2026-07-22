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
	if persisted != receipt {
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
	trace  []string
	failAt string
}

func (actions *tracingRunnerActions) Backup(context.Context, RunnerInput) error {
	return actions.run("backup")
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
