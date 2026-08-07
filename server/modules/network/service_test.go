package network

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

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
	service := NewService(&moduleConfigManagerStub{value: moduleapi.ModuleConfigValue{EffectiveValue: json.RawMessage(`{"enabled":false,"http_proxy":"","https_proxy":"","no_proxy":[]}`), HasOverride: true, Version: 7, UpdatedAt: &updatedAt, UpdatedByName: "Graft Admin"}}, diagnostics, consumers, &diagnosticHistoryStoreStub{}, nil)
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
	if overview.Version != 7 {
		t.Fatalf("expected module config version, got %#v", overview)
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
	core, observed := observer.New(zap.ErrorLevel)
	service := NewService(nil, diagnostics, NewConsumerRegistry(), history, zap.New(core))
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
	if observed.Len() != 1 || observed.All()[0].Message != "outbound diagnostic execution failed" || observed.All()[0].ContextMap()["target_id"] != "target" {
		t.Fatalf("expected raw execution failure to be logged, got %#v", observed.All())
	}
}

func TestServiceDiagnoseRejectsUnregisteredTarget(t *testing.T) {
	service := NewService(nil, NewDiagnosticRegistry(), NewConsumerRegistry(), &diagnosticHistoryStoreStub{}, nil)
	if _, err := service.Diagnose(context.Background(), "missing"); !errors.Is(err, errDiagnosticTargetNotFound) {
		t.Fatalf("expected unregistered target to be rejected, got %v", err)
	}
}

func TestServiceDiagnosticHistoryRejectsUnregisteredTargetBeforeStoreRead(t *testing.T) {
	history := &diagnosticHistoryStoreStub{}
	service := NewService(nil, NewDiagnosticRegistry(), NewConsumerRegistry(), history, nil)
	if _, err := service.DiagnosticHistory(context.Background(), "missing", 20); !errors.Is(err, errDiagnosticTargetNotFound) {
		t.Fatalf("expected unregistered target to be rejected, got %v", err)
	}
	if history.listCalled {
		t.Fatal("expected history store not to be read for unregistered target")
	}
}

func TestServiceUpdateRejectsInvalidPolicyBeforeWrite(t *testing.T) {
	configs := &moduleConfigManagerStub{}
	service := NewService(configs, NewDiagnosticRegistry(), NewConsumerRegistry(), &diagnosticHistoryStoreStub{}, nil)
	if _, err := service.Update(context.Background(), moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPProxy: "socks5://proxy.example:1080", NoProxy: []string{}}, nil, 0); !errors.Is(err, errInvalidOutboundPolicy) {
		t.Fatalf("expected invalid policy to be rejected, got %v", err)
	}
	if configs.updateCalled {
		t.Fatal("expected invalid policy not to be persisted")
	}
}

func TestServiceDiagnoseReturnsCompletedResultWhenHistoryPersistenceFails(t *testing.T) {
	diagnostics := NewDiagnosticRegistry()
	if err := diagnostics.RegisterOutboundDiagnosticTarget(diagnosticTargetStub{name: "target"}); err != nil {
		t.Fatalf("register diagnostic target: %v", err)
	}
	history := &diagnosticHistoryStoreStub{appendErr: errors.New("database unavailable")}
	service := NewService(nil, diagnostics, NewConsumerRegistry(), history, nil)
	result, err := service.Diagnose(context.Background(), "target")
	if err != nil {
		t.Fatalf("expected persistence failure to preserve the completed result, got %v", err)
	}
	if result.TestedAt.IsZero() {
		t.Fatalf("expected completed diagnostic result, got %#v", result)
	}
}

func TestServiceObserveCapabilityProjectsConnectivityAggregate(t *testing.T) {
	completedAt := time.Date(2026, 8, 7, 10, 20, 30, 0, time.UTC)
	tests := []struct {
		name      string
		aggregate ConnectivityAggregate
		want      moduleapi.CapabilityStatus
	}{
		{name: "no completed checks", aggregate: ConnectivityAggregate{}, want: moduleapi.CapabilityStatusUnknown},
		{name: "healthy", aggregate: ConnectivityAggregate{LastRunAt: &completedAt, TargetCount: 2, HealthyCount: 2}, want: moduleapi.CapabilityStatusHealthy},
		{name: "degraded", aggregate: ConnectivityAggregate{LastRunAt: &completedAt, TargetCount: 2, HealthyCount: 1, DegradedCount: 1}, want: moduleapi.CapabilityStatusDegraded},
		{name: "failed dominates degraded", aggregate: ConnectivityAggregate{LastRunAt: &completedAt, TargetCount: 2, DegradedCount: 1, FailedCount: 1}, want: moduleapi.CapabilityStatusUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &Service{connectivity: &connectivityStoreStub{aggregate: test.aggregate}}
			got, err := service.ObserveCapability(context.Background())
			if err != nil {
				t.Fatalf("observe capability: %v", err)
			}
			if got.Status != test.want {
				t.Fatalf("status = %q, want %q", got.Status, test.want)
			}
			if test.aggregate.LastRunAt != nil && !got.ObservedAt.Equal(completedAt) {
				t.Fatalf("observed at = %s, want %s", got.ObservedAt, completedAt)
			}
		})
	}
}

func TestServiceObserveCapabilityReturnsAggregateError(t *testing.T) {
	wantErr := errors.New("aggregate read failed")
	service := &Service{connectivity: &connectivityStoreStub{aggregateErr: wantErr}}
	if _, err := service.ObserveCapability(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected aggregate error, got %v", err)
	}
}

type moduleConfigManagerStub struct {
	value        moduleapi.ModuleConfigValue
	updateCalled bool
}

func (s moduleConfigManagerStub) GetModuleConfig(context.Context, string, string) (moduleapi.ModuleConfigValue, error) {
	return s.value, nil
}
func (s *moduleConfigManagerStub) UpdateModuleConfig(context.Context, string, string, json.RawMessage, *uint64, int64) (moduleapi.ModuleConfigValue, error) {
	s.updateCalled = true
	return s.value, nil
}
func (moduleConfigManagerStub) ResetModuleConfig(context.Context, string, string, *uint64, int64) (moduleapi.ModuleConfigValue, error) {
	return moduleapi.ModuleConfigValue{}, nil
}

type diagnosticHistoryStoreStub struct {
	appendedTarget string
	appended       moduleapi.OutboundDiagnosticResult
	appendErr      error
	listCalled     bool
}

type connectivityStoreStub struct {
	aggregate    ConnectivityAggregate
	aggregateErr error
}

func (s *connectivityStoreStub) Append(context.Context, moduleapi.ConnectivityReport) (ConnectivityCheck, error) {
	return ConnectivityCheck{}, nil
}

func (s *connectivityStoreStub) Latest(context.Context) ([]ConnectivityCheck, error) {
	return nil, nil
}

func (s *connectivityStoreStub) History(context.Context, moduleapi.ConnectivityTargetID, int) ([]ConnectivityCheck, error) {
	return nil, nil
}

func (s *connectivityStoreStub) Report(context.Context, moduleapi.ConnectivityTargetID, int64) (moduleapi.ConnectivityReport, error) {
	return moduleapi.ConnectivityReport{}, nil
}

func (s *connectivityStoreStub) Aggregate(context.Context) (ConnectivityAggregate, error) {
	return s.aggregate, s.aggregateErr
}

func (s *diagnosticHistoryStoreStub) Append(_ context.Context, target string, result moduleapi.OutboundDiagnosticResult) error {
	s.appendedTarget, s.appended = target, result
	return s.appendErr
}
func (s *diagnosticHistoryStoreStub) List(context.Context, string, int) ([]moduleapi.OutboundDiagnosticResult, error) {
	s.listCalled = true
	return nil, nil
}

type failingDiagnosticTarget struct{}

func (failingDiagnosticTarget) Name() string        { return "target" }
func (failingDiagnosticTarget) DisplayName() string { return "target" }
func (failingDiagnosticTarget) ExecuteOutboundDiagnostic(context.Context) (moduleapi.OutboundDiagnosticResult, error) {
	return moduleapi.OutboundDiagnosticResult{}, context.DeadlineExceeded
}
