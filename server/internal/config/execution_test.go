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
