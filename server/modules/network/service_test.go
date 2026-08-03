package network

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestServiceOverviewIncludesRealOverrideAttributionAndRegisteredConsumers(t *testing.T) {
	updatedAt := time.Date(2026, 8, 3, 10, 20, 30, 0, time.UTC)
	diagnostics := NewDiagnosticRegistry()
	if err := diagnostics.RegisterOutboundDiagnosticTarget(diagnosticTargetStub{name: "platform-update-release"}); err != nil {
		t.Fatalf("register diagnostic target: %v", err)
	}
	consumers := NewConsumerRegistry()
	if err := consumers.RegisterOutboundNetworkConsumer(outboundConsumerStub{name: "platform-update"}); err != nil {
		t.Fatalf("register consumer: %v", err)
	}
	service := NewService(moduleConfigManagerStub{value: moduleapi.ModuleConfigValue{EffectiveValue: json.RawMessage(`{"enabled":false,"http_proxy":"","https_proxy":"","no_proxy":[]}`), HasOverride: true, UpdatedAt: &updatedAt, UpdatedByName: "Graft Admin"}}, diagnostics, consumers, &diagnosticHistoryStoreStub{})
	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if overview.UpdatedAt == nil || !overview.UpdatedAt.Equal(updatedAt) || overview.UpdatedByName != "Graft Admin" {
		t.Fatalf("unexpected attribution: %#v", overview)
	}
	if len(overview.Consumers) != 1 || overview.Consumers[0].Name() != "platform-update" {
		t.Fatalf("expected registered platform update consumer, got %#v", overview.Consumers)
	}
}

type outboundConsumerStub struct{ name string }

func (s outboundConsumerStub) Name() string        { return s.name }
func (s outboundConsumerStub) DisplayName() string { return s.name }

func TestServiceDiagnosePersistsSanitizedExecutionFailure(t *testing.T) {
	diagnostics := NewDiagnosticRegistry()
	if err := diagnostics.RegisterOutboundDiagnosticTarget(failingDiagnosticTarget{}); err != nil {
		t.Fatalf("register diagnostic target: %v", err)
	}
	history := &diagnosticHistoryStoreStub{}
	service := NewService(nil, diagnostics, NewConsumerRegistry(), history)
	result, err := service.Diagnose(context.Background(), "target")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if result.Connected || result.Message != errDiagnosticExecutionFailed.Error() || result.TestedAt.IsZero() {
		t.Fatalf("unexpected sanitized result: %#v", result)
	}
	if history.appendedTarget != "target" || history.appended.Message != result.Message {
		t.Fatalf("expected persisted sanitized diagnostic, got %#v", history)
	}
}

type moduleConfigManagerStub struct{ value moduleapi.ModuleConfigValue }

func (s moduleConfigManagerStub) GetModuleConfig(context.Context, string, string) (moduleapi.ModuleConfigValue, error) {
	return s.value, nil
}
func (moduleConfigManagerStub) UpdateModuleConfig(context.Context, string, string, json.RawMessage, *uint64) (moduleapi.ModuleConfigValue, error) {
	return moduleapi.ModuleConfigValue{}, nil
}
func (moduleConfigManagerStub) ResetModuleConfig(context.Context, string, string) (moduleapi.ModuleConfigValue, error) {
	return moduleapi.ModuleConfigValue{}, nil
}

type diagnosticHistoryStoreStub struct {
	appendedTarget string
	appended       moduleapi.OutboundDiagnosticResult
}

func (s *diagnosticHistoryStoreStub) Append(_ context.Context, target string, result moduleapi.OutboundDiagnosticResult) error {
	s.appendedTarget, s.appended = target, result
	return nil
}
func (*diagnosticHistoryStoreStub) List(context.Context, string, int) ([]moduleapi.OutboundDiagnosticResult, error) {
	return nil, nil
}

type failingDiagnosticTarget struct{}

func (failingDiagnosticTarget) Name() string        { return "target" }
func (failingDiagnosticTarget) DisplayName() string { return "target" }
func (failingDiagnosticTarget) ExecuteOutboundDiagnostic(context.Context) (moduleapi.OutboundDiagnosticResult, error) {
	return moduleapi.OutboundDiagnosticResult{}, context.DeadlineExceeded
}
