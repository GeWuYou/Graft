package dockerexecution

import (
	"errors"
	"strings"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestComposeLoadFailureSummarySelectsAllowedLinesAndRedactsPaths(t *testing.T) {
	summary := composeLoadFailureSummary(errors.New("failed loading /srv/graft/workspace/compose.yaml: services.web Additional property runtime is not allowed\nopen /srv/graft/workspace/compose.yaml: no such file"))
	if summary == "" {
		t.Fatal("allowlisted compose failure produced no summary")
	}
	if strings.Contains(summary, "/srv") {
		t.Fatalf("absolute path survived in summary = %q", summary)
	}
	if !strings.Contains(summary, "[path]") {
		t.Fatalf("path was not redacted in summary = %q", summary)
	}
	if !strings.Contains(summary, "services.web") || !strings.Contains(summary, "not allowed") {
		t.Fatalf("structured compose error missing from summary = %q", summary)
	}
}

func TestComposeLoadFailureSummaryDropsSensitiveLines(t *testing.T) {
	summary := composeLoadFailureSummary(errors.New("services.web invalid credential token in environment\nyaml: line 9: unknown key 'registry_password'"))
	if summary != "" {
		t.Fatalf("sensitive fragments leaked into summary = %q", summary)
	}
}

func TestComposeLoadFailureSummaryRejectsArbitraryOutput(t *testing.T) {
	if summary := composeLoadFailureSummary(errors.New("panic: runtime error: index out of range")); summary != "" {
		t.Fatalf("non-allowlisted provider output leaked into summary = %q", summary)
	}
	if composeLoadFailureSummary(nil) != "" {
		t.Fatal("nil error produced a summary")
	}
}

func TestComposeLoadFailureSummaryIsBounded(t *testing.T) {
	line := "yaml: line 1: " + strings.Repeat("长", moduleapi.ExecutionDiagnosticSummaryMaxRunes)
	summary := composeLoadFailureSummary(errors.New(line))
	if len([]rune(summary)) > moduleapi.ExecutionDiagnosticSummaryMaxRunes {
		t.Fatalf("summary length = %d, want <= %d", len([]rune(summary)), moduleapi.ExecutionDiagnosticSummaryMaxRunes)
	}
}

func TestProviderFailureClassMapping(t *testing.T) {
	if result := failedExecution(failureRuntimeUnavailable); result.FailureClass != moduleapi.ExecutionFailureClassInfrastructure {
		t.Fatalf("runtime unavailable class = %s", result.FailureClass)
	}
	if result := failedExecutionAt(failureProviderOperation, "compose_apply"); result.FailureClass != moduleapi.ExecutionFailureClassProvider || result.FailureStage != "compose_apply" {
		t.Fatalf("stage-attributed result = %#v", result)
	}
	if result := failedExecution("unrecognized_code"); result.FailureClass != moduleapi.ExecutionFailureClassUnknown {
		t.Fatalf("unknown code class = %s", result.FailureClass)
	}
}
