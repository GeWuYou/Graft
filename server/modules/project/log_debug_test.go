package project

import (
	"testing"

	"graft/server/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogProjectLogDiagnosticUsesInjectedConfig(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	service := &Service{logger: zap.New(core), debugConfig: config.ProjectConfig{LogDebug: true}}

	service.logProjectLogDiagnostic("snapshot-started", zap.Uint64("project_id", 7))
	if entries := observed.All(); len(entries) != 1 {
		t.Fatalf("expected one diagnostic entry, got %#v", entries)
	}

	service.debugConfig.LogDebug = false
	service.logProjectLogDiagnostic("snapshot-stopped")
	if entries := observed.All(); len(entries) != 1 {
		t.Fatalf("expected disabled diagnostics to emit no entry, got %#v", entries)
	}
}

func TestLogProjectLogDiagnosticAddsStructuredEvent(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	service := &Service{logger: zap.New(core), debugConfig: config.ProjectConfig{LogDebug: true}}

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
