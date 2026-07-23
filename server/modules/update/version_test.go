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
			name:       "binary with executable",
			env:        map[string]string{declaredDeploymentModeEnv: "binary"},
			executable: func() (string, error) { return "/usr/local/bin/graft", nil },
			capability: "manual_guidance",
			guidance:   "当前为二进制安装。下载同版本 server 与 web 发行包，校验 SHA-256 后按部署管理器重启服务。",
		},
		{
			name:       "binary without executable",
			env:        map[string]string{},
			executable: func() (string, error) { return "", errors.New("not available") },
			capability: "manual_guidance",
			guidance:   "无法确认可执行文件路径。请按二进制发行说明手动校验并替换 server 与 web 发行包。",
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
