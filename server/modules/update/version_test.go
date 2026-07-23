package update

import (
	"errors"
	"testing"
)

func TestSelectLatestHonorsStableAndBetaChannels(t *testing.T) {
	beta, _ := ParseVersion("0.9.1-beta.1")
	stable, _ := ParseVersion("0.9.1")
	releases := []Release{{Version: "0.9.1-beta.2", Channel: "beta"}, {Version: "0.10.0-beta.1", Channel: "beta"}, {Version: "0.10.0", Channel: "stable"}}
	selected, ok := SelectLatest(beta, releases)
	if !ok || selected.Version != "0.10.0" {
		t.Fatalf("beta expected later stable release, got %#v", selected)
	}
	selected, ok = SelectLatest(stable, releases)
	if !ok || selected.Version != "0.10.0" {
		t.Fatalf("stable must not select beta release, got %#v", selected)
	}
}

func TestDetectInstallationProfileRequiresDeclaredAndDetectedCompose(t *testing.T) {
	profile := DetectInstallationProfile(func(key string) string {
		if key == "GRAFT_UPDATE_COMPOSE_ROOT" {
			return "/opt/graft"
		}
		return "binary"
	}, func() (string, error) { return "/usr/local/bin/graft", nil })
	if profile.Capability != "manual_guidance" {
		t.Fatalf("expected conservative capability, got %s", profile.Capability)
	}
}

func TestDetectInstallationProfileTreatsRelativeComposeRootAsBinary(t *testing.T) {
	profile := DetectInstallationProfile(func(key string) string {
		if key == declaredDeploymentModeEnv {
			return "compose"
		}
		if key == "GRAFT_UPDATE_COMPOSE_ROOT" {
			return "deploy/graft"
		}
		return ""
	}, func() (string, error) { return "/usr/local/bin/graft", nil })
	if profile.DetectedMode != "binary" || profile.Capability != "manual_guidance_blocked" {
		t.Fatalf("expected relative compose root to remain binary guidance, got %#v", profile)
	}
}

func TestDetectInstallationProfile(t *testing.T) {
	testCases := []struct {
		name       string
		env        map[string]string
		executable func() (string, error)
		capability string
		guidance   string
	}{
		{
			name:       "confirmed compose",
			env:        map[string]string{declaredDeploymentModeEnv: "compose", "GRAFT_UPDATE_COMPOSE_ROOT": "/opt/graft"},
			executable: func() (string, error) { return "/usr/local/bin/graft", nil },
			capability: "compose_upgrade_available",
			guidance:   "官方 Compose 安装已通过声明与挂载路径预检；升级前仍会执行执行器预检。",
		},
		{
			name:       "binary with complete manual guidance",
			env:        map[string]string{declaredDeploymentModeEnv: "binary", binaryWebRootEnv: "/var/www/graft", serviceManagerEnv: "manual"},
			executable: func() (string, error) { return "/usr/local/bin/graft", nil },
			capability: "manual_guidance",
			guidance:   "二进制部署只能按此受控人工步骤升级；server 与 web 必须使用同一 release tag。",
		},
		{
			name:       "binary without complete manual guidance",
			env:        map[string]string{},
			executable: func() (string, error) { return "", errors.New("not available") },
			capability: "manual_guidance_blocked",
			guidance:   "完整二进制升级指引缺少 GRAFT_UPDATE_BINARY_PATH, GRAFT_UPDATE_WEB_ROOT, GRAFT_UPDATE_SERVICE_MANAGER。不会自动替换正在运行的 binary。",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			profile := DetectInstallationProfile(func(key string) string { return testCase.env[key] }, testCase.executable)
			if profile.Capability != testCase.capability || profile.Guidance != testCase.guidance {
				t.Fatalf("expected capability %q and guidance %q, got %#v", testCase.capability, testCase.guidance, profile)
			}
		})
	}
}
