package update

import (
	"context"
	"testing"

	"graft/server/internal/moduleapi"
)

type composeReaderStub struct {
	candidates  []moduleapi.UpdateComposeRuntimeCandidate
	seenContext context.Context
}

func (s *composeReaderStub) DiscoverCurrentServerCompose(ctx context.Context) ([]moduleapi.UpdateComposeRuntimeCandidate, error) {
	s.seenContext = ctx
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
	profile := DetectInstallationProfileWithComposeReader(context.Background(), func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "compose"
		}
		return ""
	}, func(_ string) (string, bool) { return "", false }, func() (string, error) { return "", nil }, &composeReaderStub{candidates: []moduleapi.UpdateComposeRuntimeCandidate{{CandidateKey: "compose-a", Root: "/srv/graft", Confidence: "high"}}})
	if profile.ComposeRootSource != "docker_discovered" || profile.Capability != "compose_upgrade_available" || len(profile.ComposeCandidates) != 1 {
		t.Fatalf("expected docker-discovered compose profile, got %#v", profile)
	}
	if profile.ComposeRootConfirmationRequired {
		t.Fatalf("expected unique high-confidence candidate to skip confirmation, got %#v", profile)
	}
}

func TestDetectInstallationProfileRequiresConfirmationForFallbackCandidate(t *testing.T) {
	profile := DetectInstallationProfileWithComposeReader(context.Background(), func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "compose"
		}
		return ""
	}, func(_ string) (string, bool) { return "", false }, func() (string, error) { return "", nil }, &composeReaderStub{candidates: []moduleapi.UpdateComposeRuntimeCandidate{{CandidateKey: "compose-a", Root: "/srv/graft", Confidence: "medium"}}})
	if !profile.ComposeRootConfirmationRequired {
		t.Fatalf("expected fallback candidate to require confirmation, got %#v", profile)
	}
}

func TestDetectInstallationProfileExplicitRootNeverFallsBackToDocker(t *testing.T) {
	reader := &composeReaderStub{candidates: []moduleapi.UpdateComposeRuntimeCandidate{{CandidateKey: "compose-a", Root: "/srv/graft"}}}
	profile := DetectInstallationProfileWithComposeReader(context.Background(), func(key string) string {
		switch key {
		case declaredDeploymentModeEnv:
			return "compose"
		case "GRAFT_UPDATE_COMPOSE_ROOT":
			return "relative/path"
		default:
			return ""
		}
	}, func(key string) (string, bool) {
		if key == "GRAFT_UPDATE_COMPOSE_ROOT" {
			return "relative/path", true
		}
		return "", false
	}, func() (string, error) { return "", nil }, reader)
	if profile.ComposeRootSource != "explicit_env" || profile.DetectedMode != "binary" || len(profile.ComposeCandidates) != 0 || reader.seenContext != nil {
		t.Fatalf("invalid explicit root must not fall back, got %#v", profile)
	}
}

func TestDetectInstallationProfilePropagatesContextToDockerDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := &composeReaderStub{}
	DetectInstallationProfileWithComposeReader(ctx, func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "compose"
		}
		return ""
	}, func(_ string) (string, bool) { return "", false }, func() (string, error) { return "", nil }, reader)
	if reader.seenContext != ctx || reader.seenContext.Err() != context.Canceled {
		t.Fatalf("expected canceled context to reach Docker discovery, got %v", reader.seenContext)
	}
}

func TestDetectInstallationProfileBlocksBlankComposeRootWithoutReader(t *testing.T) {
	reader := &composeReaderStub{candidates: []moduleapi.UpdateComposeRuntimeCandidate{{CandidateKey: "compose-a", Root: "/srv/graft"}}}
	profile := DetectInstallationProfileWithComposeReader(context.Background(), func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "compose"
		}
		return ""
	}, func(key string) (string, bool) {
		if key == "GRAFT_UPDATE_COMPOSE_ROOT" {
			return "", true
		}
		return "", false
	}, func() (string, error) { return "/usr/local/bin/graft", nil }, reader)
	if profile.Capability != "manual_guidance_blocked" || profile.ComposeRootSource != "explicit_env" || profile.BlockingReason == "" || reader.seenContext != nil {
		t.Fatalf("expected blank root without discovery to fail closed, got %#v", profile)
	}
}

func TestDetectInstallationProfileMapsNoComposeFilesToEmptyArray(t *testing.T) {
	reader := &composeReaderStub{candidates: []moduleapi.UpdateComposeRuntimeCandidate{{CandidateKey: "compose-a", Root: "/srv/graft"}}}
	profile := DetectInstallationProfileWithComposeReader(context.Background(), func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "compose"
		}
		return ""
	}, func(_ string) (string, bool) { return "", false }, func() (string, error) { return "", nil }, reader)
	if profile.ComposeCandidates[0].ConfigFiles == nil || len(profile.ComposeCandidates[0].ConfigFiles) != 0 {
		t.Fatalf("expected non-nil empty compose files, got %#v", profile.ComposeCandidates[0].ConfigFiles)
	}
}
