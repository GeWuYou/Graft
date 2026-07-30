package update

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestInstallationProfileMapsAvailableDeploymentContext(t *testing.T) {
	context := moduleapi.NewDeploymentContext("compose", "docker_discovered", false, []moduleapi.DeploymentComposeCandidate{
		moduleapi.NewDeploymentComposeCandidate("compose-a", "/srv/graft", []string{"/srv/graft/compose.yml"}, "graft", "high", nil),
	}, nil)
	profile := installationProfile(context)
	if profile.Capability != "compose_upgrade_available" || len(profile.ComposeCandidates) != 1 || profile.ComposeCandidates[0].Root != "/srv/graft" {
		t.Fatalf("unexpected deployment profile: %#v", profile)
	}
}

func TestInstallationProfileMapsDeploymentDiagnostic(t *testing.T) {
	profile := installationProfile(moduleapi.NewDeploymentContext("binary", "unavailable", false, nil, []moduleapi.DeploymentDiagnostic{{Code: "deployment_mode_unsupported", Message: "binary runtime is not supported"}}))
	if profile.Capability != "manual_guidance_blocked" || profile.BlockingReason != "binary runtime is not supported" {
		t.Fatalf("unexpected unavailable deployment profile: %#v", profile)
	}
}
