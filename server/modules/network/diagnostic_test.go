package network

import (
	"context"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestDiagnosticRegistryRejectsDuplicateTargetAndReturnsSortedSnapshot(t *testing.T) {
	registry := NewDiagnosticRegistry()
	for _, name := range []string{"update", "marketplace"} {
		if err := registry.RegisterOutboundDiagnosticTarget(diagnosticTargetStub{name: name}); err != nil {
			t.Fatalf("register target %s: %v", name, err)
		}
	}
	if err := registry.RegisterOutboundDiagnosticTarget(diagnosticTargetStub{name: "update"}); err == nil {
		t.Fatal("expected duplicate target to fail")
	}
	targets := registry.OutboundDiagnosticTargets()
	if len(targets) != 2 || targets[0].Name() != "marketplace" || targets[1].Name() != "update" {
		t.Fatalf("expected sorted targets, got %#v", targets)
	}
}

func TestConsumerRegistryZeroValueReturnsEmptySnapshot(t *testing.T) {
	var registry ConsumerRegistry
	if consumers := registry.OutboundNetworkConsumers(); consumers != nil {
		t.Fatalf("expected zero-value registry to return nil snapshot, got %#v", consumers)
	}
}

type diagnosticTargetStub struct{ name string }

func (s diagnosticTargetStub) Name() string        { return s.name }
func (s diagnosticTargetStub) DisplayName() string { return s.name }
func (diagnosticTargetStub) ExecuteOutboundDiagnostic(context.Context) (moduleapi.OutboundDiagnosticResult, error) {
	return moduleapi.OutboundDiagnosticResult{}, nil
}
