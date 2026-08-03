package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigValidateCommandReturnsValidationErrorAndWritesTextReport(t *testing.T) {
	workingDirectory := t.TempDir()
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousDirectory) })

	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"config", "validate", "--set", "GRAFT_CONFIG_SCHEMA_VERSION=1", "--set", "GRAFT_AUTH_JWT_SECRET=secret"})
	if err := command.Execute(); err != nil {
		t.Fatalf("expected valid configuration, got %v", err)
	}
	if got := output.String(); got != "Configuration valid.\n" {
		t.Fatalf("unexpected output: %q", got)
	}

	command = NewRootCommand()
	output.Reset()
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"config", "validate"})
	if err := command.Execute(); err == nil {
		t.Fatal("expected missing signing material error")
	}
	if !strings.Contains(output.String(), "Configuration Validation Failed") {
		t.Fatalf("expected report output, got %s", output.String())
	}
}

func TestConfigValidateCommandRejectsInvalidComposeFile(t *testing.T) {
	composeFile := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(composeFile, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("write compose fixture: %v", err)
	}
	command := NewRootCommand()
	command.SetArgs([]string{"config", "validate", "--compose-file", composeFile, "--set", "GRAFT_CONFIG_SCHEMA_VERSION=1", "--set", "GRAFT_AUTH_JWT_SECRET=secret"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "services mapping") {
		t.Fatalf("expected compose validation error, got %v", err)
	}
}

func TestConfigValidateCommandFormatsComposeFailure(t *testing.T) {
	composeFile := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(composeFile, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("write compose fixture: %v", err)
	}
	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"config", "validate", "--format", "json", "--compose-file", composeFile})
	if err := command.Execute(); err == nil {
		t.Fatal("expected compose validation error")
	}
	if !strings.Contains(output.String(), "\"code\": \"compose\"") {
		t.Fatalf("expected JSON compose report, got %s", output.String())
	}
}

func TestConfigValidateCommandWritesJSONWithoutSecrets(t *testing.T) {
	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"config", "validate", "--format", "json", "--set", "GRAFT_CONFIG_SCHEMA_VERSION=1", "--set", "GRAFT_AUTH_JWT_SECRET=secret"})
	if err := command.Execute(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	if !strings.Contains(output.String(), "\"schema_version\": 1") || strings.Contains(output.String(), "secret") {
		t.Fatalf("unexpected JSON output: %s", output.String())
	}
}

func TestConfigValidateCommandWritesPatch(t *testing.T) {
	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"config", "validate", "--format", "patch", "--set", "GRAFT_CONFIG_SCHEMA_VERSION=1", "--set", "GRAFT_AUTH_JWT_SECRET=test", "--set", "GRAFT_OLD_DATABASE_MODE=legacy"})
	if err := command.Execute(); err == nil {
		t.Fatal("expected missing signing material")
	}
	if !strings.Contains(output.String(), "Configuration migration suggestions") {
		t.Fatalf("expected patch output, got %s", output.String())
	}
}
