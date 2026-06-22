package cli

import (
	"bytes"
	"strings"
	"testing"

	"graft/server/internal/buildinfo"
)

func TestRunVersionPrintsCanonicalBuildInfo(t *testing.T) {
	originalProvider := versionInfoProvider
	defer func() {
		versionInfoProvider = originalProvider
	}()

	versionInfoProvider = func() buildinfo.Info {
		return buildinfo.Info{
			Version:      "0.1.0",
			GitCommit:    "abc1234",
			BuildTimeUTC: "2026-06-22T10:00:00Z",
			GitTreeState: "clean",
		}
	}

	command := newVersionCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetArgs(nil)

	if err := command.Execute(); err != nil {
		t.Fatalf("execute version command: %v", err)
	}

	output := stdout.String()
	for _, expected := range []string{
		"version: 0.1.0",
		"git_commit: abc1234",
		"build_time_utc: 2026-06-22T10:00:00Z",
		"git_tree_state: clean",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestWriteVersionOutputUsesExplicitFallbackFields(t *testing.T) {
	var stdout bytes.Buffer

	if err := writeVersionOutput(&stdout, buildinfo.Info{
		Version:      "dev",
		GitCommit:    "unknown",
		BuildTimeUTC: "unknown",
		GitTreeState: "unknown",
	}); err != nil {
		t.Fatalf("write version output: %v", err)
	}

	expected := "version: dev\ngit_commit: unknown\nbuild_time_utc: unknown\ngit_tree_state: unknown\n"
	if stdout.String() != expected {
		t.Fatalf("expected exact fallback output:\n%s\ngot:\n%s", expected, stdout.String())
	}
}

func TestNewRootCommandRegistersVersionCommand(t *testing.T) {
	command := NewRootCommand()

	found, _, err := command.Find([]string{"version"})
	if err != nil {
		t.Fatalf("find version command: %v", err)
	}
	if found == nil || found.Name() != "version" {
		t.Fatalf("expected version command, got %#v", found)
	}
}
