package project

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestIsProjectLogDebugEnabled(t *testing.T) {
	t.Setenv(projectLogDebugEnvironmentKey, "true")
	if !isProjectLogDebugEnabled() {
		t.Fatal("expected project log debug to be enabled")
	}

	t.Setenv(projectLogDebugEnvironmentKey, "false")
	if isProjectLogDebugEnabled() {
		t.Fatal("expected project log debug to be disabled")
	}
}

func TestLogProjectLogDiagnosticAddsStructuredEvent(t *testing.T) {
	t.Setenv(projectLogDebugEnvironmentKey, "true")
	core, observed := observer.New(zapcore.InfoLevel)
	service := &Service{logger: zap.New(core)}

	service.logProjectLogDiagnostic("snapshot-started", zap.Uint64("project_id", 7))

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one diagnostic entry, got %#v", entries)
	}
	if entries[0].Message != "project log diagnostic" {
		t.Fatalf("unexpected diagnostic message %q", entries[0].Message)
	}
	context := entries[0].ContextMap()
	if context["event"] != "snapshot-started" || context["project_id"] != uint64(7) {
		t.Fatalf("expected structured diagnostic fields, got %#v", context)
	}
}
