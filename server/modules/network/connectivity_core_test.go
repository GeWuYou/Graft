package network

import (
	"context"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestConnectivityRegistryRejectsDuplicateTargetAndReturnsSortedDescriptorSnapshots(t *testing.T) {
	registry := NewDiagnosticRegistry()
	for _, id := range []moduleapi.ConnectivityTargetID{"update", "marketplace"} {
		if err := registry.RegisterConnectivityTarget(connectivityTargetStub{descriptor: connectivityDescriptor(id)}); err != nil {
			t.Fatalf("register target %s: %v", id, err)
		}
	}
	if err := registry.RegisterConnectivityTarget(connectivityTargetStub{descriptor: connectivityDescriptor("update")}); err == nil {
		t.Fatal("expected duplicate connectivity target to fail")
	}
	descriptors := registry.ConnectivityTargetDescriptors()
	if len(descriptors) != 2 || descriptors[0].ID != "marketplace" || descriptors[1].ID != "update" {
		t.Fatalf("expected sorted descriptors, got %#v", descriptors)
	}
	descriptors[0].Capabilities.ProbeKinds[0] = moduleapi.ConnectivityProbeLDAPBind
	fresh := registry.ConnectivityTargetDescriptors()
	if fresh[0].Capabilities.ProbeKinds[0] != moduleapi.ConnectivityProbeHTTP {
		t.Fatalf("expected descriptor capabilities to be immutable snapshots, got %#v", fresh[0])
	}
}

func TestDiagnosticRegistryRegistersCanonicalTargetFromLegacyAdapter(t *testing.T) {
	registry := NewDiagnosticRegistry()
	target := connectivityLegacyTargetStub{connectivityTargetStub: connectivityTargetStub{descriptor: connectivityDescriptor("platform-update")}}
	if err := registry.RegisterOutboundDiagnosticTarget(target); err != nil {
		t.Fatalf("register legacy adapter: %v", err)
	}
	if _, found := registry.ConnectivityTarget("platform-update"); !found {
		t.Fatal("expected legacy adapter to be available through canonical registry")
	}
}

func TestConnectivityReportSnapshotDoesNotRetainMutableProbeOrDisclosurePointers(t *testing.T) {
	route := &moduleapi.RouteExplanation{MatchedStrategy: "platform_default", Decision: "http_proxy", Reason: "host_not_matched_by_no_proxy"}
	exitIP := &moduleapi.ExitIPDisclosure{Masked: "***.***.45.19", Available: true}
	httpStatus := 200
	probes := []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusSucceeded, Duration: 18 * time.Millisecond, HTTPStatus: &httpStatus, Summary: "HTTP 200"}}
	report := moduleapi.NewConnectivityReport("github", moduleapi.ConnectivityReportStatusHealthy, time.Date(2026, 8, 4, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)), 18*time.Millisecond, probes, route, exitIP)
	probes[0].Summary = "mutated"
	*probes[0].HTTPStatus = 503
	route.Decision = "direct"
	exitIP.Masked = "unmasked-value-must-not-leak"
	if report.Probes[0].Summary != "HTTP 200" || report.Probes[0].HTTPStatus == nil || *report.Probes[0].HTTPStatus != 200 || report.Route.Decision != "http_proxy" || report.ExitIP.Masked != "***.***.45.19" {
		t.Fatalf("expected report construction to snapshot mutable inputs, got %#v", report)
	}
	if report.SchemaVersion != 1 || report.CheckedAt.Location() != time.UTC {
		t.Fatalf("expected versioned UTC report, got %#v", report)
	}
	snapshot := report.Snapshot()
	snapshot.Probes[0].Summary = "changed snapshot"
	*snapshot.Probes[0].HTTPStatus = 404
	snapshot.Route.Reason = "changed snapshot"
	if report.Probes[0].Summary != "HTTP 200" || report.Probes[0].HTTPStatus == nil || *report.Probes[0].HTTPStatus != 200 || report.Route.Reason != "host_not_matched_by_no_proxy" {
		t.Fatalf("expected report snapshot to avoid shared mutation, got %#v", report)
	}
}

func TestConnectivityReportSanitizesUntrustedProbeTextAndFullExitIP(t *testing.T) {
	report := moduleapi.NewConnectivityReport("github", moduleapi.ConnectivityReportStatusFailed, time.Now(), -time.Second, []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusFailed, Duration: -time.Second, Summary: "Authorization: Bearer secret", ErrorCode: "request failed!"}}, nil, &moduleapi.ExitIPDisclosure{Masked: "34.12.45.19", Available: true})
	if report.TotalLatency != 0 || report.Probes[0].Duration != 0 || report.Probes[0].Summary != "sanitized diagnostic detail" || report.Probes[0].ErrorCode != "requestfailed" {
		t.Fatalf("expected sanitized report envelope, got %#v", report)
	}
	if report.ExitIP != nil {
		t.Fatalf("expected unmasked exit IP to be excluded, got %#v", report.ExitIP)
	}
}

func connectivityDescriptor(id moduleapi.ConnectivityTargetID) moduleapi.ConnectivityTargetDescriptor {
	return moduleapi.ConnectivityTargetDescriptor{
		ID:       id,
		ModuleID: "network-test",
		Category: "general",
		TitleKey: "network.targets.test",
		Capabilities: moduleapi.ConnectivityTargetCapabilities{
			ProbeKinds: []moduleapi.ConnectivityProbeKind{moduleapi.ConnectivityProbeHTTP, moduleapi.ConnectivityProbeTLS},
			Features:   []moduleapi.ConnectivityTargetFeature{moduleapi.ConnectivityFeatureHistory, moduleapi.ConnectivityFeatureExport},
		},
	}
}

type connectivityTargetStub struct {
	descriptor moduleapi.ConnectivityTargetDescriptor
}

func (s connectivityTargetStub) ConnectivityTargetDescriptor() moduleapi.ConnectivityTargetDescriptor {
	return s.descriptor.Snapshot()
}

func (connectivityTargetStub) RunConnectivityProbes(context.Context) (moduleapi.ConnectivityReport, error) {
	return moduleapi.ConnectivityReport{}, nil
}

type connectivityLegacyTargetStub struct{ connectivityTargetStub }

func (connectivityLegacyTargetStub) Name() string { return "platform-update-release" }
func (connectivityLegacyTargetStub) DisplayName() string {
	return "network.diagnosticTargets.platformUpdate"
}
func (connectivityLegacyTargetStub) ExecuteOutboundDiagnostic(context.Context) (moduleapi.OutboundDiagnosticResult, error) {
	return moduleapi.OutboundDiagnosticResult{}, nil
}
