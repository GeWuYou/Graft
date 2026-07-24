package update

import (
	"context"
	"testing"

	"graft/server/internal/moduleapi"
)

type composeReaderStub struct {
	candidates []moduleapi.UpdateComposeRuntimeCandidate
}

func (s composeReaderStub) DiscoverCurrentServerCompose(context.Context) ([]moduleapi.UpdateComposeRuntimeCandidate, error) {
	return s.candidates, nil
}

func TestDetectInstallationProfileBlocksIncompleteBinaryGuidance(t *testing.T) {
	profile := DetectInstallationProfile(func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "binary"
		}
		return ""
	}, func() (string, error) { return "/opt/graft/graft-server", nil })
	if profile.Capability != "manual_guidance_blocked" || profile.BlockingReason == "" {
		t.Fatalf("expected explicit binary guidance block, got %#v", profile)
	}
}

func TestDetectInstallationProfileBuildsSystemdManualSteps(t *testing.T) {
	values := map[string]string{declaredDeploymentModeEnv: "binary", binaryPathEnv: "/opt/graft/graft-server", binaryWebRootEnv: "/srv/graft/web", serviceManagerEnv: "systemd", serviceNameEnv: "graft.service"}
	profile := DetectInstallationProfile(func(key string) string { return values[key] }, func() (string, error) { return "", nil })
	if profile.Capability != "manual_guidance" || len(profile.ManualSteps) != 5 || profile.ManualSteps[4].Key != "restartSystemd" || profile.ManualSteps[4].Params["service_name"] != "graft.service" {
		t.Fatalf("expected exact systemd guidance, got %#v", profile)
	}
}

func TestDetectInstallationProfileUsesDockerCandidateWhenExplicitRootIsMissing(t *testing.T) {
	profile := DetectInstallationProfileWithComposeReader(func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "compose"
		}
		return ""
	}, func() (string, error) { return "", nil }, composeReaderStub{candidates: []moduleapi.UpdateComposeRuntimeCandidate{{CandidateKey: "compose-a", Root: "/srv/graft", Confidence: "high"}}})
	if profile.ComposeRootSource != "docker_discovered" || profile.Capability != "compose_upgrade_available" || len(profile.ComposeCandidates) != 1 {
		t.Fatalf("expected docker-discovered compose profile, got %#v", profile)
	}
}

func TestDetectInstallationProfileExplicitRootNeverFallsBackToDocker(t *testing.T) {
	profile := DetectInstallationProfileWithComposeReader(func(key string) string {
		switch key {
		case declaredDeploymentModeEnv:
			return "compose"
		case "GRAFT_UPDATE_COMPOSE_ROOT":
			return "relative/path"
		default:
			return ""
		}
	}, func() (string, error) { return "", nil }, composeReaderStub{candidates: []moduleapi.UpdateComposeRuntimeCandidate{{CandidateKey: "compose-a", Root: "/srv/graft"}}})
	if profile.ComposeRootSource != "explicit_env" || profile.DetectedMode != "binary" || len(profile.ComposeCandidates) != 0 {
		t.Fatalf("invalid explicit root must not fall back, got %#v", profile)
	}
}
