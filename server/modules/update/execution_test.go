package update

import "testing"

func TestValidateRunnerInputRejectsNonOfficialComposeProfiles(t *testing.T) {
	valid := RunnerInput{ProtocolVersion: runnerProtocolVersion, OperationID: "operation-1", TaskID: 1, Preflight: ComposePreflight{
		DeclaredMode: "compose", DetectedMode: "compose", ComposeRoot: "/opt/graft", Platform: "linux/amd64", DockerSocket: "/var/run/docker.sock", ComposeFiles: []string{"/opt/graft/compose.yml"}, BundledPostgres: true,
		OfficialServerImage: "ghcr.io/gewuyou/graft-server", OfficialWebImage: "ghcr.io/gewuyou/graft-web",
		ServerDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WebDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ServerReference: "ghcr.io/gewuyou/graft-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WebReference: "ghcr.io/gewuyou/graft-web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}}
	if err := ValidateRunnerInput(valid); err != nil {
		t.Fatalf("valid official profile rejected: %v", err)
	}
	valid.Preflight.ComposeFiles = append(valid.Preflight.ComposeFiles, "/opt/graft/compose.override.yml")
	if err := ValidateRunnerInput(valid); err == nil {
		t.Fatal("expected compose override rejection")
	}
}

func TestClassifyRunnerReceiptNeverRollsBackAfterMigration(t *testing.T) {
	if got := ClassifyRunnerReceipt(RunnerReceipt{MigrationStarted: true, FailureCode: "healthz_failed"}); got != ExecutionOutcomeNeedsAttention {
		t.Fatalf("got %s", got)
	}
	if got := ClassifyRunnerReceipt(RunnerReceipt{MigrationStarted: false, FailureCode: "pull_failed"}); got != ExecutionOutcomeRecovered {
		t.Fatalf("got %s", got)
	}
}
