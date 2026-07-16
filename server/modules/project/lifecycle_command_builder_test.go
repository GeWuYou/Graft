package project

import (
	"testing"

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
