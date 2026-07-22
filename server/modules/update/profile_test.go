package update

import "testing"

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
	if profile.Capability != "manual_guidance" || len(profile.ManualSteps) != 5 || profile.ManualSteps[4] != "执行 systemctl restart graft.service，然后验证 /healthz。" {
		t.Fatalf("expected exact systemd guidance, got %#v", profile)
	}
}
