package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

var workingDirectoryMu sync.Mutex

func TestResolveAndValidateUsesDocumentedPrecedence(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GRAFT_AUTH_JWT_SECRET=file-secret\nGRAFT_LOG_LEVEL=warn\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	report, err := ResolveAndValidate(ResolveOptions{
		EnvFile: envFile,
		Set:     []string{"GRAFT_CONFIG_SCHEMA_VERSION=1", "GRAFT_LOG_LEVEL=debug"},
		Environment: map[string]string{
			"GRAFT_AUTH_JWT_SECRET": "environment-secret",
			"GRAFT_LOG_LEVEL":       "error",
		},
	})
	if err != nil {
		t.Fatalf("resolve configuration: %v", err)
	}
	if value := report.Values["GRAFT_LOG_LEVEL"]; value.Source != SourceCLI || value.Value != "debug" {
		t.Fatalf("expected CLI value, got %#v", value)
	}
	if value := report.Values["GRAFT_AUTH_JWT_SECRET"]; value.Source != SourceEnvironment || value.Value != "environment-secret" {
		t.Fatalf("expected environment value, got %#v", value)
	}
	if value := report.Values["GRAFT_HTTP_ADDR"]; value.Source != SourceDefault || value.Value != ":8080" {
		t.Fatalf("expected schema default, got %#v", value)
	}
}

func TestResolveAndValidateDiscoversEnvFileIndependentlyFromEnvironment(t *testing.T) {
	workingDirectoryMu.Lock()
	t.Cleanup(workingDirectoryMu.Unlock)
	workingDirectory := t.TempDir()
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousDirectory) })
	if err := os.WriteFile(".env", []byte("GRAFT_CONFIG_SCHEMA_VERSION=1\nGRAFT_AUTH_JWT_SECRET=dotenv-secret\n"), 0o600); err != nil {
		t.Fatalf("write dotenv fixture: %v", err)
	}

	report, err := ResolveAndValidate(ResolveOptions{
		Environment:     map[string]string{},
		DiscoverEnvFile: true,
	})
	if err != nil {
		t.Fatalf("resolve discovered dotenv: %v", err)
	}
	if value := report.Values["GRAFT_AUTH_JWT_SECRET"]; value.Source != SourceEnvFile {
		t.Fatalf("expected discovered dotenv source, got %#v", value)
	}
}

func TestSchemaValidatesMigrationDeclarations(t *testing.T) {
	invalid := `schema_version: 1
environment:
  - name: GRAFT_OLD_VALUE
    type: string
    default: null
    deprecated: true
`
	var schema Schema
	if err := yaml.Unmarshal([]byte(invalid), &schema); err != nil {
		t.Fatalf("decode invalid schema fixture: %v", err)
	}
	if err := schema.Validate(); err == nil || !strings.Contains(err.Error(), "replacement declaration") {
		t.Fatalf("expected replacement declaration error, got %v", err)
	}

	valid := `schema_version: 1
environment:
  - name: GRAFT_REMOVED_VALUE
    type: string
    default: null
    removed: true
    replacement: null
    severity: warning
changes: []
`
	if err := yaml.Unmarshal([]byte(valid), &schema); err != nil {
		t.Fatalf("decode valid schema fixture: %v", err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("validate schema fixture: %v", err)
	}
	if schema.Changes == nil {
		t.Fatal("expected changes declaration to be preserved")
	}
}

func TestSchemaRejectsMissingNullableDeclarations(t *testing.T) {
	missingDefault := `schema_version: 1
environment:
  - name: GRAFT_VALUE
    type: string
changes: []
`
	var schema Schema
	if err := yaml.Unmarshal([]byte(missingDefault), &schema); err != nil {
		t.Fatalf("decode missing default fixture: %v", err)
	}
	if err := schema.Validate(); err == nil || !strings.Contains(err.Error(), "default declaration") {
		t.Fatalf("expected default declaration error, got %v", err)
	}

	missingChanges := `schema_version: 1
environment:
  - name: GRAFT_VALUE
    type: string
    default: null
`
	if err := yaml.Unmarshal([]byte(missingChanges), &schema); err != nil {
		t.Fatalf("decode missing changes fixture: %v", err)
	}
	if err := schema.Validate(); err == nil || !strings.Contains(err.Error(), "changes declaration") {
		t.Fatalf("expected changes declaration error, got %v", err)
	}
}

func TestResolveAndValidateReportsMissingSigningMaterial(t *testing.T) {
	report, err := ResolveAndValidate(ResolveOptions{Environment: map[string]string{"GRAFT_CONFIG_SCHEMA_VERSION": "1"}})
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if report.ErrorCount() != 1 || report.Findings[0].Code != "required_any_of" {
		t.Fatalf("unexpected report: %#v", report)
	}
	if !strings.Contains(report.FormatText(), "GRAFT_AUTH_JWT_SECRET or GRAFT_AUTH_SIGNING_KEY") {
		t.Fatalf("expected missing signing material in text report: %s", report.FormatText())
	}
}

func TestResolveAndValidateReportsDeprecatedAndRemovedValues(t *testing.T) {
	report, err := ResolveAndValidate(ResolveOptions{Environment: map[string]string{
		"GRAFT_CONFIG_SCHEMA_VERSION": "1",
		"GRAFT_AUTH_JWT_SECRET":       "secret",
		"GRAFT_OLD_STORAGE_PATH":      "/old",
		"GRAFT_OLD_DATABASE_MODE":     "legacy",
	}})
	if err == nil {
		t.Fatal("expected removed configuration error")
	}
	if report.ErrorCount() != 1 || len(report.Findings) != 2 {
		t.Fatalf("unexpected findings: %#v", report.Findings)
	}
	if output := report.FormatText(); !strings.Contains(output, "GRAFT_STORAGE_ROOT") || strings.Contains(output, "legacy") || strings.Contains(output, "/old") {
		t.Fatalf("unexpected safe text report: %s", output)
	}
	encoded, err := report.FormatJSON()
	if err != nil {
		t.Fatalf("format json: %v", err)
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "legacy") {
		t.Fatalf("JSON report leaked a value: %s", encoded)
	}
}

func TestResolveAndValidateRejectsInvalidSetValue(t *testing.T) {
	_, err := ResolveAndValidate(ResolveOptions{Set: []string{"GRAFT_LOG_LEVEL"}})
	if err == nil || !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("expected invalid --set error, got %v", err)
	}
}

func TestFormatPatchDoesNotExposeSensitiveValues(t *testing.T) {
	report, err := ResolveAndValidate(ResolveOptions{Environment: map[string]string{"GRAFT_CONFIG_SCHEMA_VERSION": "1"}})
	if err == nil {
		t.Fatal("expected missing signing material")
	}
	patch := report.FormatPatch()
	if !strings.Contains(patch, "Set one of: GRAFT_AUTH_JWT_SECRET or GRAFT_AUTH_SIGNING_KEY") {
		t.Fatalf("expected signing material suggestion, got %s", patch)
	}
	if strings.Contains(patch, "Value") {
		t.Fatalf("patch must not include resolved values: %s", patch)
	}
}

func TestValidateComposeFile(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "compose.yml")
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "compose.yml"))
	if err != nil {
		t.Fatalf("read official compose fixture: %v", err)
	}
	// #nosec G703 -- valid 是 t.TempDir() 下固定的 compose.yml 文件名，不接受外部路径输入。
	if err := os.WriteFile(valid, contents, 0o600); err != nil {
		t.Fatalf("write compose fixture: %v", err)
	}
	if err := ValidateComposeFile(valid); err != nil {
		t.Fatalf("validate compose file: %v", err)
	}

	invalid := filepath.Join(dir, "invalid.yml")
	if err := os.WriteFile(invalid, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("write invalid compose fixture: %v", err)
	}
	if err := ValidateComposeFile(invalid); err == nil || !strings.Contains(err.Error(), "services mapping") {
		t.Fatalf("expected services mapping error, got %v", err)
	}
}

func TestValidateComposeFileReportsFieldContractsWithoutValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.yml")
	contents := `x-graft-server-runtime-env: &graft-server-runtime-env
  GRAFT_AUTH_JWT_SECRET: ${GRAFT_AUTH_JWT_SECRET}
services:
  postgres:
    environment:
      POSTGRES_DB: graft
      POSTGRES_USER: graft
    volumes:
      - postgres:/var/lib/postgresql/data
    restart: unless-stopped
  redis:
    command: [redis-server]
    restart: unless-stopped
  config-validate:
    user: "0:0"
    command: [config, validate]
    restart: "no"
    volumes:
      - compose:/opt/graft/deployment/compose.yml:ro
      - env:/opt/graft/deployment/.env:ro
  bootstrap:
    environment:
      <<: *graft-server-runtime-env
    command: [migrate, up]
    restart: "no"
    depends_on:
      config-validate:
        condition: service_started
  application-root-init:
    user: "0:0"
    restart: "no"
    volumes: [apps:/opt/graft/apps]
    depends_on:
      config-validate:
        condition: service_completed_successfully
  backup-root-init:
    user: "0:0"
    restart: "no"
    volumes: [backups:/var/lib/graft/backups]
    depends_on:
      config-validate:
        condition: service_completed_successfully
  server:
    user: "0:0"
    restart: unless-stopped
    entrypoint: [/app/graft serve]
    environment:
      <<: *graft-server-runtime-env
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - state:/var/lib/graft/update-state:ro
    depends_on:
      bootstrap:
        condition: service_completed_successfully
      application-root-init:
        condition: service_completed_successfully
      backup-root-init:
        condition: service_completed_successfully
  web:
    restart: unless-stopped
    ports: ["3000:81"]
    depends_on:
      server:
        condition: service_healthy
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write compose fixture: %v", err)
	}
	err := ValidateComposeFile(path)
	if err == nil {
		t.Fatal("expected compose contract failure")
	}
	message := err.Error()
	for _, expected := range []string{
		"postgres missing required environment POSTGRES_PASSWORD",
		"bootstrap missing required dependency config-validate",
		"web missing required container port 80",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected finding %q, got %s", expected, message)
		}
	}
	if strings.Contains(message, "graft") {
		t.Fatalf("compose report must not leak field values: %s", message)
	}
}

func TestComposeFieldHelpersSupportLongSyntax(t *testing.T) {
	var ports, secrets yaml.Node
	if err := yaml.Unmarshal([]byte("- target: 80\n"), &ports); err != nil {
		t.Fatalf("parse ports: %v", err)
	}
	if err := yaml.Unmarshal([]byte("- source: signing-key\n"), &secrets); err != nil {
		t.Fatalf("parse secrets: %v", err)
	}
	if !hasPort(ports, "80") {
		t.Fatal("expected long port syntax to match container port")
	}
	if !hasSecret(secrets, "signing-key") {
		t.Fatal("expected long secret syntax to match source")
	}
}

func TestFindVolumeEntryRequiresExactScalarTarget(t *testing.T) {
	volume := yaml.Node{Kind: yaml.ScalarNode, Value: "data:/opt/userdata:ro"}
	if _, found := findVolumeEntry(volume, "/data"); found {
		t.Fatal("expected path substring not to match volume target")
	}
}
