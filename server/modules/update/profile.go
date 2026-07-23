package update

import (
	"os"
	"path/filepath"
	"strings"
)

const declaredDeploymentModeEnv = "GRAFT_UPDATE_DEPLOYMENT_MODE"

// InstallationProfile 组合 operator 声明与运行时证据；Capability 只在二者可安全对齐时允许自动化。
type InstallationProfile struct {
	DeclaredMode string `json:"declared_mode"`
	DetectedMode string `json:"detected_mode"`
	Capability   string `json:"capability"`
	Guidance     string `json:"guidance"`
}

// DetectInstallationProfile 基于部署路径和运行时证据生成保守画像，永不把声明环境变量单独当作执行授权。
func DetectInstallationProfile(getenv func(string) string, executable func() (string, error)) InstallationProfile {
	declared := normalizeMode(getenv(declaredDeploymentModeEnv))
	composeRoot := strings.TrimSpace(getenv("GRAFT_UPDATE_COMPOSE_ROOT"))
	detected := "binary"
	if filepath.IsAbs(composeRoot) {
		detected = "compose"
	}
	profile := InstallationProfile{DeclaredMode: declared, DetectedMode: detected}
	switch {
	case detected == "compose" && declared == "compose":
		profile.Capability = "compose_upgrade_available"
		profile.Guidance = "官方 Compose 安装已通过声明与挂载路径预检；升级前仍会执行执行器预检。"
	case detected == "compose":
		profile.Capability = "manual_guidance"
		profile.Guidance = "检测到 Compose 部署证据，但声明模式未确认；请设置 GRAFT_UPDATE_DEPLOYMENT_MODE=compose 后重试。"
	default:
		profile.Capability = "manual_guidance"
		path, err := executable()
		if err == nil && strings.TrimSpace(path) != "" {
			profile.Guidance = "当前为二进制安装。下载同版本 server 与 web 发行包，校验 SHA-256 后按部署管理器重启服务。"
		} else {
			profile.Guidance = "无法确认可执行文件路径。请按二进制发行说明手动校验并替换 server 与 web 发行包。"
		}
	}
	return profile
}

func runtimeInstallationProfile() InstallationProfile {
	return DetectInstallationProfile(os.Getenv, os.Executable)
}

func normalizeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "compose", "binary":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}
