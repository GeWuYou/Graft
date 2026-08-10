//go:build conformance

package runtimetarget

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerBuilderAgentFixtureTargetAuthorityRejectsUnavailableRepository(t *testing.T) {
	if _, err := (dockerBuilderAgentFixtureTargetAuthority{}).Resolve(context.Background()); err == nil {
		t.Fatal("fixture target authority accepted an unavailable repository")
	}
}

func TestWriteDockerBuilderAgentConformanceMaterialRestrictsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets", "bootstrap.json")
	if err := writeDockerBuilderAgentConformanceMaterial(path, DockerBuilderAgentConformanceMaterial{TargetID: 1, AgentID: "builder", BootstrapToken: "secret"}); err != nil {
		t.Fatalf("write material: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat material: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("material mode = %o", info.Mode().Perm())
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "secret\n" {
		t.Fatalf("material did not round trip")
	}
}

func TestConformancePayloadFingerprintIsStableAndScoped(t *testing.T) {
	scenario := DockerBuilderAgentFixtureScenario{AgentID: "builder", ImageDigest: "sha256:fixture", AgentVersion: "v1", EnrollmentRef: "fixture"}
	if conformancePayloadFingerprint(scenario, 1) != conformancePayloadFingerprint(scenario, 1) || conformancePayloadFingerprint(scenario, 1) == conformancePayloadFingerprint(scenario, 2) {
		t.Fatal("fixture payload fingerprint is not stable and target scoped")
	}
}
