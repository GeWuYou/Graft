//go:build conformance

package main

import (
	"path/filepath"
	"testing"

	runtimetarget "graft/server/modules/runtime-target"
)

func TestParseArgumentsRequiresPhaseSpecificEvidence(t *testing.T) {
	_, _, _, err := parseArguments([]string{"--phase", "verify-restart", "--agent-id", "builder", "--target-id", "1"})
	if err == nil {
		t.Fatal("verify-restart accepted without bootstrap evidence")
	}
}

func TestVerificationBaselineRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conformance-evidence.json")
	want := runtimetarget.DockerBuilderAgentConformanceEvidence{TargetID: 7, IdentityID: "identity", AgentID: "builder", Generation: 1, ProviderID: "docker", BuilderScope: "scope", CapabilityProfile: "oci-build", CapabilityVersion: "docker/v1", LedgerProvenance: "runtime-target-controlled-execution-ledger", LedgerReceiptCount: 2}
	if err := writeVerificationBaseline(path, want); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	got, err := readVerificationBaseline(path)
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	if got.targetID != want.TargetID || got.identityID != want.IdentityID || got.agentID != want.AgentID || got.generation != want.Generation || got.receiptCount != want.LedgerReceiptCount {
		t.Fatalf("baseline = %#v", got)
	}
}

func TestVerificationBaselineRejectsUnboundLedgerEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conformance-evidence.json")
	evidence := runtimetarget.DockerBuilderAgentConformanceEvidence{TargetID: 7, IdentityID: "identity", AgentID: "builder", Generation: 1, ProviderID: "docker", BuilderScope: "scope", CapabilityProfile: "oci-build", CapabilityVersion: "docker/v1", LedgerReceiptCount: 1}
	if err := writeVerificationBaseline(path, evidence); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if _, err := readVerificationBaseline(path); err == nil {
		t.Fatal("baseline without controlled-ledger provenance was accepted")
	}
}

func TestParseArgumentsRetainsExplicitAgentID(t *testing.T) {
	_, _, baseline, err := parseArguments([]string{"--phase", "verify-bootstrap", "--agent-id", "builder"})
	if err != nil {
		t.Fatalf("parse verification arguments: %v", err)
	}
	if !baseline.agentIDExplicit || baseline.requestedAgentID != "builder" {
		t.Fatalf("agent ID input = %#v", baseline)
	}
}

func TestParseArgumentsAcceptsPrepareScenario(t *testing.T) {
	phase, scenario, _, err := parseArguments([]string{"--phase", "prepare", "--agent-id", "builder", "--image-digest", "sha256:fixture", "--agent-version", "fixture", "--enrollment-ref", "fixture", "--automation-id", "fixture", "--docker-installation-ref", "docker:fixture", "--docker-secret-ref", "secret:fixture", "--bootstrap-material-file", "/run/fixture/bootstrap-token", "--agent-config-file", "/etc/graft/config/agent.json", "--bootstrap-url", "https://backend:8443", "--agent-url", "https://backend:8444", "--bootstrap-ca-file", "/run/trust/ca.pem", "--trust-bundle-file", "/run/trust/ca.pem", "--agent-secret-file", "/run/bootstrap/token"})
	if err != nil {
		t.Fatalf("parse prepare arguments: %v", err)
	}
	if phase != "prepare" || scenario.AgentID != "builder" {
		t.Fatalf("parsed prepare = %q %#v", phase, scenario)
	}
}
