package config

import "testing"

func TestExecutionModePreservesLegacyDefaultAndExplicitLocal(t *testing.T) {
	for _, mode := range []string{"", ExecutionModeAgent, ExecutionModeLocal} {
		cfg := &Config{Execution: ExecutionConfig{Mode: mode}}
		if err := validateExecutionConfig(cfg); err != nil {
			t.Fatal(err)
		}
		expected := mode
		if expected == "" {
			expected = ExecutionModeAgent
		}
		if cfg.Execution.Mode != expected {
			t.Fatalf("mode=%q want=%q", cfg.Execution.Mode, expected)
		}
	}
	if err := validateExecutionConfig(&Config{Execution: ExecutionConfig{Mode: "invalid"}}); err == nil {
		t.Fatal("unknown mode accepted")
	}
}

func TestConfigurationPreflightEnforcesDeclaredExecutionModeEnum(t *testing.T) {
	report, err := ResolveAndValidate(ResolveOptions{Set: []string{"GRAFT_CONFIG_SCHEMA_VERSION=1", "GRAFT_AUTH_JWT_SECRET=synthetic-test-secret", "GRAFT_EXECUTION_MODE=invalid"}})
	if err == nil {
		t.Fatal("preflight accepted an unsupported execution mode")
	}
	found := false
	for _, item := range report.Findings {
		if item.Key == "GRAFT_EXECUTION_MODE" && item.Code == "invalid_value" {
			found = true
		}
	}
	if !found {
		t.Fatalf("mode finding missing: %#v", report.Findings)
	}
}
