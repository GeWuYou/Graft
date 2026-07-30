package update

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestEvaluateReadinessReturnsOrderedUpgradeableChecks(t *testing.T) {
	status := Status{
		CurrentVersion: "v1.0.0",
		ImageTag:       "beta",
		UpdateMode:     UpdateModeBetaTracking,
		Latest:         &Release{Version: "v1.1.0-beta.1", NotesURL: "https://example.test/releases/v1.1.0-beta.1"},
		Profile: InstallationProfile{
			DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available", ComposeRootSource: "docker_discovered",
			ComposeCandidates: []ComposeRootCandidate{{CandidateKey: "compose-a", Root: "/srv/graft", ConfigFiles: []string{"/srv/graft/compose.yml"}}},
		},
	}

	readiness := EvaluateReadiness(status, true)
	if readiness.Overall != readinessOverallUpgradeReady || readiness.NextAction == nil || readiness.NextAction.ID != "start_upgrade" {
		t.Fatalf("expected upgrade-ready next action, got %#v", readiness)
	}
	if readiness.TotalCount != 5 || readiness.ReadyCount != 4 || len(readiness.Checks) != 5 {
		t.Fatalf("unexpected readiness counts: %#v", readiness)
	}
	for index, wantOrder := range []int{10, 20, 30, 40, 50} {
		if readiness.Checks[index].Order != wantOrder {
			t.Fatalf("check %d order = %d, want %d", index, readiness.Checks[index].Order, wantOrder)
		}
	}
	if readiness.Checks[3].State != moduleapi.ReadinessStateWarning || readiness.Checks[3].Severity != moduleapi.ReadinessSeverityInfo || readiness.Checks[3].Blocking {
		t.Fatalf("newer release should be informational and non-blocking: %#v", readiness.Checks[3])
	}
}

func TestEvaluateReadinessSeparatesUnknownReleaseState(t *testing.T) {
	status := Status{CheckError: releaseDiscoveryFailedMessage, CacheStale: true}

	readiness := EvaluateReadiness(status, true)
	if readiness.Overall != readinessOverallStatusUnknown || readiness.NextAction == nil || readiness.NextAction.ID != "check_updates" {
		t.Fatalf("expected unknown status with recheck action, got %#v", readiness)
	}
	check := readiness.Checks[3]
	if check.State != moduleapi.ReadinessStateUnavailable || check.Severity != moduleapi.ReadinessSeverityWarning || !check.Blocking {
		t.Fatalf("expected unavailable blocking release catalog check, got %#v", check)
	}
}

func TestEvaluateReadinessIncludesReleaseActionOnlyForMeaningfulNotesURL(t *testing.T) {
	tests := []struct {
		name     string
		notesURL string
		want     string
	}{
		{name: "missing", notesURL: "", want: ""},
		{name: "whitespace", notesURL: " \t", want: ""},
		{name: "present", notesURL: " https://example.test/releases/v1.1.0 ", want: "https://example.test/releases/v1.1.0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			readiness := EvaluateReadiness(Status{Latest: &Release{Version: "1.1.0", NotesURL: test.notesURL}}, true)
			actions := readiness.Checks[3].Actions
			if test.want == "" {
				if len(actions) != 0 {
					t.Fatalf("release actions = %#v, want none", actions)
				}
				return
			}
			if len(actions) != 1 || actions[0].ID != "view_release" || actions[0].Target != test.want {
				t.Fatalf("release actions = %#v, want view_release to %q", actions, test.want)
			}
		})
	}
}

func TestEvaluateReadinessBlocksAvailableReleaseUntilDeploymentIsSupported(t *testing.T) {
	status := Status{CurrentVersion: "v1.0.0", ImageTag: "beta", UpdateMode: UpdateModeBetaTracking, Latest: &Release{Version: "v1.1.0-beta.1"}, Profile: InstallationProfile{DeclaredMode: "binary", DetectedMode: "binary", Capability: "manual_guidance"}}

	readiness := EvaluateReadiness(status, true)
	if readiness.Overall != readinessOverallUpgradeBlocked || readiness.NextAction == nil || readiness.NextAction.ID != "view_documentation" {
		t.Fatalf("expected blocked upgrade with migration guidance, got %#v", readiness)
	}
	if readiness.Checks[0].State != moduleapi.ReadinessStateFailed || readiness.Checks[0].Severity != moduleapi.ReadinessSeverityCritical || !readiness.Checks[0].Blocking {
		t.Fatalf("expected official Compose failure to be critical and blocking: %#v", readiness.Checks[0])
	}
}

func TestEvaluateReadinessRedactsSensitiveComposeEvidence(t *testing.T) {
	status := Status{Profile: InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available", ComposeRootSource: "docker_discovered", ComposeCandidates: []ComposeRootCandidate{{Root: "/srv/graft", ConfigFiles: []string{"/srv/graft/compose.yml"}}}}}

	readiness := EvaluateReadiness(status, false)
	project := readiness.Checks[1]
	if project.Blocking {
		t.Fatalf("read-only diagnostics should retain the non-sensitive compose project result: %#v", project)
	}
	for _, evidence := range project.Evidence {
		if evidence.Sensitive || evidence.Value == "/srv/graft" || evidence.Value == "/srv/graft/compose.yml" {
			t.Fatalf("read-only readiness leaked sensitive host evidence: %#v", project.Evidence)
		}
	}
	permission := readiness.Checks[4]
	if permission.State != moduleapi.ReadinessStateFailed || !permission.Blocking || permission.Severity != moduleapi.ReadinessSeverityCritical {
		t.Fatalf("expected update manage permission to remain an explicit blocking check: %#v", permission)
	}
}

func TestStatusForUpdateViewerRecalculatesPermissionAndHostEvidence(t *testing.T) {
	status := Status{Profile: InstallationProfile{DeclaredMode: "compose", DetectedMode: "compose", Capability: "compose_upgrade_available", ComposeRootSource: "docker_discovered", ComposeCandidates: []ComposeRootCandidate{{Root: "/srv/graft", ConfigFiles: []string{"/srv/graft/compose.yml"}}}}}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})

	viewer := statusForUpdateViewer(ctx, updateAuthorizerStub{}, status)
	if viewer.Readiness.Checks[4].State != moduleapi.ReadinessStatePassed || len(viewer.Readiness.Checks[1].Evidence) < 2 {
		t.Fatalf("manage viewer should receive permission and host evidence: %#v", viewer.Readiness)
	}
	readOnly := statusForUpdateViewer(ctx, updateAuthorizerStub{err: errors.New("denied")}, status)
	if readOnly.Readiness.Checks[4].State != moduleapi.ReadinessStateFailed || len(readOnly.Profile.ComposeCandidates) != 0 || len(readOnly.Readiness.Checks[1].Evidence) != 1 {
		t.Fatalf("read-only viewer should receive redacted evidence and permission result: %#v", readOnly)
	}
}
