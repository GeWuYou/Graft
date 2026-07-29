package update

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRunnerInputAcceptsOrderedComposeFilesUnderRoot(t *testing.T) {
	root := t.TempDir()
	overrides := filepath.Join(root, "overrides")
	if err := os.Mkdir(overrides, 0o750); err != nil {
		t.Fatalf("create overrides directory: %v", err)
	}
	for _, name := range []string{"compose.yaml", "overrides/web.yml"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("services: {}\n"), 0o600); err != nil {
			t.Fatalf("create compose file %s: %v", name, err)
		}
	}
	valid := RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: "operation-1", TaskID: 1, Preflight: ComposePreflight{
		DeclaredMode: "compose", UpdateMode: UpdateModeBetaTracking, ImageTag: "beta", DetectedMode: "compose", ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: []string{filepath.Join(root, "compose.yaml"), filepath.Join(root, "overrides/web.yml")}, BundledPostgres: true,
		OfficialServerImage: "ghcr.io/gewuyou/graft-server", OfficialWebImage: "ghcr.io/gewuyou/graft-web",
		OfficialRunnerImage: "ghcr.io/gewuyou/graft-compose-runner",
		ServerDigest:        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WebDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		RunnerDigest:    "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ServerReference: "ghcr.io/gewuyou/graft-server:1.2.3-beta.1", WebReference: "ghcr.io/gewuyou/graft-web:1.2.3-beta.1",
		RunnerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}}
	if err := ValidateRunnerInput(valid); err != nil {
		t.Fatalf("valid official profile rejected: %v", err)
	}
	valid.Preflight.ComposeFiles[1] = filepath.Join(t.TempDir(), "web.yml")
	if err := ValidateRunnerInput(valid); err == nil {
		t.Fatal("expected compose file outside root rejection")
	}
}

func TestValidateRunnerInputRejectsComposeSymlinkEscapeAndNestedFirstFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "compose.yml"), []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("create compose file: %v", err)
	}
	external := t.TempDir()
	externalFile := filepath.Join(external, "web.yml")
	if err := os.WriteFile(externalFile, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("create external compose file: %v", err)
	}
	link := filepath.Join(root, "web.yml")
	if err := os.Symlink(externalFile, link); err != nil {
		t.Fatalf("create compose symlink: %v", err)
	}

	valid := RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: "operation-1", TaskID: 1, Preflight: ComposePreflight{
		DeclaredMode: "compose", UpdateMode: UpdateModeBetaTracking, ImageTag: "beta", DetectedMode: "compose", ComposeRoot: root, Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: []string{filepath.Join(root, "compose.yml"), link}, BundledPostgres: true,
		OfficialServerImage: "ghcr.io/gewuyou/graft-server", OfficialWebImage: "ghcr.io/gewuyou/graft-web", OfficialRunnerImage: "ghcr.io/gewuyou/graft-compose-runner",
		ServerDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WebDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RunnerDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ServerReference: "ghcr.io/gewuyou/graft-server:1.2.3-beta.1", WebReference: "ghcr.io/gewuyou/graft-web:1.2.3-beta.1", RunnerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}}
	if err := ValidateRunnerInput(valid); err == nil {
		t.Fatal("expected compose symlink escaping root rejection")
	}

	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o750); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	nestedCompose := filepath.Join(nested, "compose.yml")
	if err := os.WriteFile(nestedCompose, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("create nested compose file: %v", err)
	}
	valid.Preflight.ComposeFiles = []string{nestedCompose}
	if err := ValidateRunnerInput(valid); err == nil {
		t.Fatal("expected nested first compose file rejection")
	}
}

func TestComposeRunnerContainerConfigUsesNonRootUser(t *testing.T) {
	input := RunnerInput{Preflight: ComposePreflight{ComposeRoot: "/opt/graft", DockerSocket: "/var/run/docker.sock"}}
	config, _ := composeRunnerContainerConfig(input, "runner-input")
	if config.User != "65532:65532" {
		t.Fatalf("runner user = %q, want non-root service user", config.User)
	}
}

func TestParseRunnerReceiptLogAcceptsOnlyBoundProtocolMarker(t *testing.T) {
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "update-log-1", Succeeded: true}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal receipt: %v", err)
	}
	parsed, ok := parseRunnerReceiptLog(RunnerReceiptLogMarker + base64.RawStdEncoding.EncodeToString(encoded))
	if !ok || parsed.OperationID != receipt.OperationID || !parsed.Succeeded {
		t.Fatalf("unexpected parsed receipt: %#v, %v", parsed, ok)
	}
	if _, ok := parseRunnerReceiptLog("ordinary runner log"); ok {
		t.Fatal("ordinary runner log must not be treated as a receipt")
	}
}

func TestValidateRunnerInputRejectsMissingOrMutableRunnerIdentity(t *testing.T) {
	valid := RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: "operation-1", TaskID: 1, Preflight: ComposePreflight{
		DeclaredMode: "compose", DetectedMode: "compose", ComposeRoot: "/opt/graft", Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: []string{"/opt/graft/compose.yml"}, BundledPostgres: true,
		OfficialServerImage: "ghcr.io/gewuyou/graft-server", OfficialWebImage: "ghcr.io/gewuyou/graft-web", OfficialRunnerImage: "ghcr.io/gewuyou/graft-compose-runner",
		ServerDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WebDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RunnerDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ServerReference: "ghcr.io/gewuyou/graft-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WebReference: "ghcr.io/gewuyou/graft-web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", RunnerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}}
	valid.Preflight.RunnerReference = "ghcr.io/gewuyou/graft-compose-runner:latest"
	if err := ValidateRunnerInput(valid); err == nil {
		t.Fatal("expected mutable runner reference rejection")
	}
	valid.Preflight.RunnerReference = ""
	if err := ValidateRunnerInput(valid); err == nil {
		t.Fatal("expected missing runner identity rejection")
	}
}

func TestClassifyRunnerReceiptNeverRollsBackAfterMigration(t *testing.T) {
	if got := ClassifyRunnerReceipt(RunnerReceipt{MigrationStarted: true, FailureCode: "healthz_failed"}); got != ExecutionOutcomeNeedsAttention {
		t.Fatalf("got %s", got)
	}
	if got := ClassifyRunnerReceipt(RunnerReceipt{MigrationStarted: false, FailureCode: "pull_failed", RecoveryCompleted: true}); got != ExecutionOutcomeRecovered {
		t.Fatalf("got %s", got)
	}
	if got := ClassifyRunnerReceipt(RunnerReceipt{MigrationStarted: false, FailureCode: "pull_failed"}); got != ExecutionOutcomeFailed {
		t.Fatalf("got %s", got)
	}
}
