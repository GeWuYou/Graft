package moduleapi

import (
	"strings"
	"testing"
)

func TestValidExecutionStage(t *testing.T) {
	for _, stage := range []string{string(ExecutionStageMaterialResolve), string(ExecutionStageComposeLoad), string(ExecutionStageComposeApply), string(ExecutionStageContainerApply), string(ExecutionStageControllerPull)} {
		if !ValidExecutionStage(stage) {
			t.Fatalf("bounded stage %q was rejected", stage)
		}
	}
	if ValidExecutionStage("deploy") || ValidExecutionStage("material resolve") {
		t.Fatal("unbounded stage value was accepted")
	}
	for _, class := range []string{string(ExecutionFailureClassTransient), string(ExecutionFailureClassPermanent), string(ExecutionFailureClassConfiguration), string(ExecutionFailureClassAuthorization), string(ExecutionFailureClassProvider), string(ExecutionFailureClassInfrastructure), string(ExecutionFailureClassInternal), string(ExecutionFailureClassUnknown)} {
		if !ValidExecutionFailureClass(class) {
			t.Fatalf("bounded failure class %q was rejected", class)
		}
	}
	if ValidExecutionFailureClass("fatal") {
		t.Fatal("unbounded failure class was accepted")
	}
}

func TestSanitizeExecutionDiagnosticSummary(t *testing.T) {
	if summary := SanitizeExecutionDiagnosticSummary("yaml: line 3:\tcould not find expected ':'"); summary != "yaml: line 3: could not find expected ':'" {
		t.Fatalf("sanitized summary = %q", summary)
	}
	if summary := SanitizeExecutionDiagnosticSummary("line\x00one\x07second"); summary != "lineonesecond" {
		t.Fatalf("control characters were not dropped: %q", summary)
	}
	long := strings.Repeat("长", ExecutionDiagnosticSummaryMaxRunes+10)
	if summary := SanitizeExecutionDiagnosticSummary(long); len([]rune(summary)) != ExecutionDiagnosticSummaryMaxRunes {
		t.Fatalf("summary length = %d, want %d", len([]rune(summary)), ExecutionDiagnosticSummaryMaxRunes)
	}
	if summary := SanitizeExecutionDiagnosticSummary("  padded   text  "); summary != "padded text" {
		t.Fatalf("collapsed summary = %q", summary)
	}
	if SanitizeExecutionDiagnosticSummary("") != "" {
		t.Fatal("empty summary was not preserved")
	}
}
