package project

import (
	"encoding/json"
	"testing"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func TestLifecycleUpArgsIncludesStructuredLifecycleFlags(t *testing.T) {
	t.Parallel()

	aggregate := projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ApplicationRecordID: 7,
			ComposeProjectName:  "compose-demo",
			WorkspacePath:       "/srv/compose-demo",
			LifecycleConfig: projectstore.LifecycleConfig{
				Profiles:           []string{"app"},
				BuildBeforeUp:      true,
				ForceRecreate:      true,
				RemoveOrphans:      true,
				WaitAfterUp:        true,
				WaitTimeoutSeconds: 180,
				RenewAnonVolumes:   true,
				AdditionalArgs:     []string{"--progress", "plain"},
			},
		},
		Files: []projectstore.ApplicationFile{
			{
				Kind:         projectcontract.FileKindCompose.String(),
				Role:         projectcontract.FileRolePrimary.String(),
				AbsolutePath: "/srv/compose-demo/compose.yaml",
				DisplayPath:  "compose.yaml",
			},
		},
	}

	args, err := lifecycleUpArgs(aggregate, lifecycleConfigurationFromAggregate(aggregate))
	if err != nil {
		t.Fatalf("build lifecycle up args: %v", err)
	}

	expected := []string{
		"compose",
		"-f", "/srv/compose-demo/compose.yaml",
		"--profile", "app",
		"-p", "compose-demo",
		"up", "-d",
		"--build",
		"--force-recreate",
		"--remove-orphans",
		"--renew-anon-volumes",
		"--wait",
		"--wait-timeout", "180",
		"--progress", "plain",
	}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %#v", len(expected), len(args), args)
	}
	for index := range expected {
		if args[index] != expected[index] {
			t.Fatalf("expected argv[%d]=%q, got %q (full=%#v)", index, expected[index], args[index], args)
		}
	}
}

func TestBuildLifecycleUpArgvSkipsWaitTimeoutWhenWaitDisabled(t *testing.T) {
	t.Parallel()

	argv := buildLifecycleUpArgv(
		[]string{"docker", "compose", "-f", "compose.yaml", "-p", "compose-demo"},
		LifecycleStandardConfig{
			RemoveOrphans:      true,
			RenewAnonVolumes:   true,
			WaitAfterUp:        false,
			WaitTimeoutSeconds: 300,
		},
	)

	expected := []string{
		"docker", "compose", "-f", "compose.yaml", "-p", "compose-demo",
		"up", "-d",
		"--remove-orphans",
		"--renew-anon-volumes",
	}
	if len(argv) != len(expected) {
		t.Fatalf("expected %d argv entries, got %d: %#v", len(expected), len(argv), argv)
	}
	for index := range expected {
		if argv[index] != expected[index] {
			t.Fatalf("expected argv[%d]=%q, got %q (full=%#v)", index, expected[index], argv[index], argv)
		}
	}
}

func TestLifecycleRestartArgsRecoverMissingProjectWithUp(t *testing.T) {
	t.Parallel()

	aggregate := projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ComposeProjectName: "compose-demo",
			WorkspacePath:      "/srv/compose-demo",
		},
		Files: []projectstore.ApplicationFile{{
			Kind:         projectcontract.FileKindCompose.String(),
			AbsolutePath: "/srv/compose-demo/compose.yaml",
		}},
	}
	missing := generated.ApplicationRuntimeStatusMissing
	stopped := generated.ApplicationRuntimeStatusStopped

	missingPlan, err := lifecycleTaskPlan(
		aggregate,
		generated.ApplicationActionResponseActionApplicationActionRestart,
		&missing,
	)
	if err != nil {
		t.Fatalf("build missing restart plan: %v", err)
	}
	missingStage := onlyLifecycleStage(t, missingPlan)
	if missingStage.Key != "up" {
		t.Fatalf("missing restart stage = %q, want up", missingStage.Key)
	}
	missingArgs := lifecycleStageArgs(t, missingStage)
	if got, want := missingArgs[len(missingArgs)-2:], []string{"up", "-d"}; !equalStrings(got, want) {
		t.Fatalf("missing restart args suffix = %#v, want %#v", got, want)
	}

	stoppedPlan, err := lifecycleTaskPlan(
		aggregate,
		generated.ApplicationActionResponseActionApplicationActionRestart,
		&stopped,
	)
	if err != nil {
		t.Fatalf("build stopped restart plan: %v", err)
	}
	stoppedStage := onlyLifecycleStage(t, stoppedPlan)
	if stoppedStage.Key != "restart" {
		t.Fatalf("stopped restart stage = %q, want restart", stoppedStage.Key)
	}
	if args := lifecycleStageArgs(t, stoppedStage); args[len(args)-1] != "restart" {
		t.Fatalf("stopped restart args = %#v", args)
	}
}

func onlyLifecycleStage(t *testing.T, plan moduleapi.TaskPlan) moduleapi.StagePlan {
	t.Helper()
	if len(plan.Stages) != 1 {
		t.Fatalf("lifecycle stages = %#v, want one stage", plan.Stages)
	}
	return plan.Stages[0]
}

func lifecycleStageArgs(t *testing.T, stage moduleapi.StagePlan) []string {
	t.Helper()
	var input composeStageInput
	if err := json.Unmarshal(stage.Input, &input); err != nil {
		t.Fatalf("decode lifecycle stage: %v", err)
	}
	return input.Args
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
