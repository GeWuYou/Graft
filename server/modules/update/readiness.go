package update

import (
	"strconv"
	"strings"

	"graft/server/internal/moduleapi"
)

const (
	readinessOverallUpToDate       = "up_to_date"
	readinessOverallUpgradeReady   = "upgrade_ready"
	readinessOverallUpgradeBlocked = "upgrade_blocked"
	readinessOverallStatusUnknown  = "status_unknown"
	readinessOrderOfficialCompose  = 10
	readinessOrderComposeProject   = 20
	readinessOrderImageStrategy    = 30
	readinessOrderRelease          = 40
	readinessOrderPermission       = 50
)

// EvaluateReadiness 将 Update 所有的部署、版本和授权事实组装为可复用的平台诊断模型。
// 它不读取宿主机、不触发发布请求，也不替代受控升级启动前的严格预检。
func EvaluateReadiness(status Status, canManage bool) moduleapi.Readiness {
	checks := []moduleapi.ReadinessCheck{
		officialComposeReadiness(status.Profile),
		composeProjectReadiness(status.Profile, canManage),
		imageStrategyReadiness(status),
		releaseReadiness(status),
		managePermissionReadiness(canManage),
	}
	readyCount := 0
	for index := range checks {
		if checks[index].Actions == nil {
			checks[index].Actions = []moduleapi.ReadinessAction{}
		}
		if checks[index].Evidence == nil {
			checks[index].Evidence = []moduleapi.ReadinessEvidence{}
		}
		check := checks[index]
		if check.State == moduleapi.ReadinessStatePassed {
			readyCount++
		}
	}
	result := moduleapi.Readiness{ReadyCount: readyCount, TotalCount: len(checks), Checks: checks}
	result.Overall, result.NextAction = updateReadinessOverall(status, checks)
	return result
}

func officialComposeReadiness(profile InstallationProfile) moduleapi.ReadinessCheck {
	passed := profile.DeclaredMode == "compose" && profile.DetectedMode == "compose" && profile.Capability == "compose_upgrade_available"
	state, severity := readinessResult(passed)
	check := moduleapi.ReadinessCheck{
		ID: "official_compose", Order: readinessOrderOfficialCompose, State: state, Severity: severity, Blocking: !passed,
		TitleKey:   "platformUpdate.readiness.officialCompose.title",
		SummaryKey: "platformUpdate.readiness.officialCompose." + readinessKeySuffix(passed),
		DetailKey:  "platformUpdate.readiness.officialCompose.detail",
		Evidence: []moduleapi.ReadinessEvidence{
			{Code: "declared_deployment_mode", State: evidenceState(profile.DeclaredMode == "compose"), LabelKey: "platformUpdate.readiness.evidence.declaredDeploymentMode", Value: profile.DeclaredMode, Expected: "compose"},
			{Code: "detected_deployment_mode", State: evidenceState(profile.DetectedMode == "compose"), LabelKey: "platformUpdate.readiness.evidence.detectedDeploymentMode", Value: profile.DetectedMode, Expected: "compose"},
		},
	}
	if !passed {
		check.Actions = []moduleapi.ReadinessAction{documentationAction("platformUpdate.readiness.actions.viewComposeMigration", "/docs/official-compose-migration")}
	}
	return check
}

func composeProjectReadiness(profile InstallationProfile, canManage bool) moduleapi.ReadinessCheck {
	found := profile.ComposeRootSource == "explicit_env" || len(profile.ComposeCandidates) > 0
	state, severity := readinessResult(found)
	check := moduleapi.ReadinessCheck{
		ID: "compose_project", Order: readinessOrderComposeProject, State: state, Severity: severity, Blocking: !found,
		TitleKey:   "platformUpdate.readiness.composeProject.title",
		SummaryKey: "platformUpdate.readiness.composeProject." + readinessKeySuffix(found),
		DetailKey:  "platformUpdate.readiness.composeProject.detail",
		Params:     composeCandidateParams(profile),
		Evidence: []moduleapi.ReadinessEvidence{
			{Code: "compose_root_source", State: evidenceState(found), LabelKey: "platformUpdate.readiness.evidence.composeRootSource", Value: profile.ComposeRootSource, Expected: "explicit_env_or_discovered"},
		},
	}
	if canManage {
		for _, candidate := range profile.ComposeCandidates {
			check.Evidence = append(check.Evidence, moduleapi.ReadinessEvidence{Code: "compose_root", State: moduleapi.ReadinessEvidencePassed, LabelKey: "platformUpdate.readiness.evidence.composeRoot", Value: candidate.Root, Expected: "absolute_host_path", Sensitive: true})
			for _, configFile := range candidate.ConfigFiles {
				check.Evidence = append(check.Evidence, moduleapi.ReadinessEvidence{Code: "compose_file", State: moduleapi.ReadinessEvidencePassed, LabelKey: "platformUpdate.readiness.evidence.composeFile", Value: configFile, Expected: "compose_configuration", Sensitive: true})
			}
		}
	}
	if !found {
		check.Actions = []moduleapi.ReadinessAction{documentationAction("platformUpdate.readiness.actions.viewComposeMigration", "/docs/official-compose-migration")}
	}
	return check
}

func imageStrategyReadiness(status Status) moduleapi.ReadinessCheck {
	configured := status.DeploymentStrategy != DeploymentStrategyUnknown
	state, severity := readinessResult(configured)
	check := moduleapi.ReadinessCheck{
		ID: "image_strategy", Order: readinessOrderImageStrategy, State: state, Severity: severity, Blocking: !configured,
		TitleKey:   "platformUpdate.readiness.imageStrategy.title",
		SummaryKey: "platformUpdate.readiness.imageStrategy." + readinessKeySuffix(configured),
		DetailKey:  "platformUpdate.readiness.imageStrategy.detail",
		Evidence: []moduleapi.ReadinessEvidence{
			{Code: "image_tag", State: evidenceState(configured), LabelKey: "platformUpdate.readiness.evidence.imageTag", Value: status.ImageTag, Expected: "supported_update_tag"},
			{Code: "deployment_strategy", State: evidenceState(configured), LabelKey: "platformUpdate.readiness.evidence.deploymentStrategy", Value: string(status.DeploymentStrategy), Expected: "supported_deployment_strategy"},
		},
	}
	if !configured {
		check.Actions = []moduleapi.ReadinessAction{documentationAction("platformUpdate.readiness.actions.viewComposeMigration", "/docs/official-compose-migration")}
	}
	return check
}

func releaseReadiness(status Status) moduleapi.ReadinessCheck {
	if status.CheckError != "" || status.CacheStale {
		return moduleapi.ReadinessCheck{
			ID: "release_availability", Order: readinessOrderRelease, State: moduleapi.ReadinessStateUnavailable, Severity: moduleapi.ReadinessSeverityWarning, Blocking: true,
			TitleKey: "platformUpdate.readiness.releaseAvailability.title", SummaryKey: "platformUpdate.readiness.releaseAvailability.unavailable", DetailKey: "platformUpdate.readiness.releaseAvailability.detail",
			Evidence: []moduleapi.ReadinessEvidence{{Code: "release_catalog", State: moduleapi.ReadinessEvidenceUnavailable, LabelKey: "platformUpdate.readiness.evidence.releaseCatalog", Expected: "fresh_verified_catalog"}},
			Actions:  []moduleapi.ReadinessAction{checkUpdatesAction()},
		}
	}
	release := preferredReadinessRelease(status)
	if release == nil {
		return moduleapi.ReadinessCheck{
			ID: "release_availability", Order: readinessOrderRelease, State: moduleapi.ReadinessStatePassed, Severity: moduleapi.ReadinessSeveritySuccess,
			TitleKey: "platformUpdate.readiness.releaseAvailability.title", SummaryKey: "platformUpdate.readiness.releaseAvailability.upToDate", DetailKey: "platformUpdate.readiness.releaseAvailability.detail",
			Evidence: []moduleapi.ReadinessEvidence{{Code: "release_catalog", State: moduleapi.ReadinessEvidencePassed, LabelKey: "platformUpdate.readiness.evidence.releaseCatalog", Expected: "no_newer_release"}},
			Actions:  []moduleapi.ReadinessAction{checkUpdatesAction()},
		}
	}
	check := moduleapi.ReadinessCheck{
		ID: "release_availability", Order: readinessOrderRelease, State: moduleapi.ReadinessStateWarning, Severity: moduleapi.ReadinessSeverityInfo,
		TitleKey: "platformUpdate.readiness.releaseAvailability.title", SummaryKey: "platformUpdate.readiness.releaseAvailability.available", DetailKey: "platformUpdate.readiness.releaseAvailability.detail",
		Params:   map[string]string{"current_version": status.CurrentVersion, "latest_version": release.Version},
		Evidence: []moduleapi.ReadinessEvidence{{Code: "latest_release", State: moduleapi.ReadinessEvidencePassed, LabelKey: "platformUpdate.readiness.evidence.latestRelease", Value: release.Version, Expected: "newer_than_current"}},
	}
	if notesURL := strings.TrimSpace(release.NotesURL); notesURL != "" {
		check.Actions = []moduleapi.ReadinessAction{{ID: "view_release", Type: moduleapi.ReadinessActionNavigate, LabelKey: "platformUpdate.readiness.actions.viewRelease", Target: notesURL}}
	}
	return check
}

func preferredReadinessRelease(status Status) *Release {
	var channel string
	switch status.DeploymentStrategy {
	case DeploymentStrategyPinnedStable:
		channel = "stable"
	case DeploymentStrategyPinnedBeta:
		channel = "beta"
	default:
		return status.Latest
	}
	current, err := ParseVersion(status.CurrentVersion)
	if err != nil {
		return nil
	}
	selected, found := SelectLatestForChannel(current, channel, status.AvailableReleases)
	if !found {
		return nil
	}
	return &selected
}

func managePermissionReadiness(canManage bool) moduleapi.ReadinessCheck {
	state, severity := readinessResult(canManage)
	return moduleapi.ReadinessCheck{
		ID: "update_manage_permission", Order: readinessOrderPermission, State: state, Severity: severity, Blocking: !canManage,
		TitleKey:   "platformUpdate.readiness.updateManagePermission.title",
		SummaryKey: "platformUpdate.readiness.updateManagePermission." + readinessKeySuffix(canManage),
		DetailKey:  "platformUpdate.readiness.updateManagePermission.detail",
		Evidence:   []moduleapi.ReadinessEvidence{{Code: "permission", State: evidenceState(canManage), LabelKey: "platformUpdate.readiness.evidence.updateManagePermission", Value: "platform-update.manage", Expected: "granted"}},
	}
}

func updateReadinessOverall(status Status, checks []moduleapi.ReadinessCheck) (string, *moduleapi.ReadinessAction) {
	if status.CheckError != "" || status.CacheStale {
		action := checkUpdatesAction()
		return readinessOverallStatusUnknown, &action
	}
	if preferredReadinessRelease(status) == nil {
		action := checkUpdatesAction()
		return readinessOverallUpToDate, &action
	}
	for _, check := range checks {
		if check.Blocking {
			if len(check.Actions) > 0 {
				action := check.Actions[0]
				return readinessOverallUpgradeBlocked, &action
			}
			return readinessOverallUpgradeBlocked, nil
		}
	}
	action := moduleapi.ReadinessAction{ID: "start_upgrade", Type: moduleapi.ReadinessActionNavigate, LabelKey: "platformUpdate.readiness.actions.startUpgrade", Target: "/platform/updates?intent=start_upgrade"}
	return readinessOverallUpgradeReady, &action
}

func readinessResult(passed bool) (moduleapi.ReadinessState, moduleapi.ReadinessSeverity) {
	if passed {
		return moduleapi.ReadinessStatePassed, moduleapi.ReadinessSeveritySuccess
	}
	return moduleapi.ReadinessStateFailed, moduleapi.ReadinessSeverityCritical
}

func evidenceState(passed bool) moduleapi.ReadinessEvidenceState {
	if passed {
		return moduleapi.ReadinessEvidencePassed
	}
	return moduleapi.ReadinessEvidenceFailed
}

func readinessKeySuffix(passed bool) string {
	if passed {
		return "passed"
	}
	return "failed"
}

func checkUpdatesAction() moduleapi.ReadinessAction {
	return moduleapi.ReadinessAction{ID: "check_updates", Type: moduleapi.ReadinessActionRecheck, LabelKey: "platformUpdate.readiness.actions.checkUpdates"}
}

func documentationAction(labelKey, target string) moduleapi.ReadinessAction {
	return moduleapi.ReadinessAction{ID: "view_documentation", Type: moduleapi.ReadinessActionDocumentation, LabelKey: labelKey, Target: target}
}

func composeCandidateParams(profile InstallationProfile) map[string]string {
	return map[string]string{"candidate_count": strconv.Itoa(len(profile.ComposeCandidates))}
}
