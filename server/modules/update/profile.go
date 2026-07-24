package update

import (
	"context"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
)

const declaredDeploymentModeEnv = "GRAFT_UPDATE_DEPLOYMENT_MODE"

const (
	binaryPathEnv     = "GRAFT_UPDATE_BINARY_PATH"
	binaryWebRootEnv  = "GRAFT_UPDATE_WEB_ROOT"
	serviceManagerEnv = "GRAFT_UPDATE_SERVICE_MANAGER"
	serviceNameEnv    = "GRAFT_UPDATE_SERVICE_NAME"
)

// InstallationProfile 组合 operator 声明与运行时证据；Capability 只在二者可安全对齐时允许自动化。
type InstallationProfile struct {
	DeclaredMode      string                 `json:"declared_mode"`
	DetectedMode      string                 `json:"detected_mode"`
	Capability        string                 `json:"capability"`
	Guidance          string                 `json:"guidance"`
	BinaryPath        string                 `json:"binary_path,omitempty"`
	WebRoot           string                 `json:"web_root,omitempty"`
	ServiceManager    string                 `json:"service_manager,omitempty"`
	ServiceName       string                 `json:"service_name,omitempty"`
	ManualSteps       []ManualStep           `json:"manual_steps,omitempty"`
	BlockingReason    string                 `json:"blocking_reason,omitempty"`
	ComposeRootSource string                 `json:"compose_root_source"`
	ComposeCandidates []ComposeRootCandidate `json:"compose_candidates"`
}

// ComposeRootCandidate 是一次状态读取得到的候选；Root 只用于服务端当前请求，不作为持久配置。
type ComposeRootCandidate struct {
	CandidateKey string   `json:"key"`
	Root         string   `json:"host_path"`
	WorkingDir   string   `json:"-"`
	ConfigFiles  []string `json:"compose_files"`
	ProjectName  string   `json:"project_name,omitempty"`
	Confidence   string   `json:"confidence"`
	Warning      string   `json:"warning,omitempty"`
}

// ManualStep 是 binary 安装人工升级指引的稳定本地化键与参数，不在 API 契约中固化某一种语言。
type ManualStep struct {
	Key    string            `json:"key"`
	Params map[string]string `json:"params,omitempty"`
}

// DetectInstallationProfile 基于部署路径和运行时证据生成保守画像，永不把声明环境变量单独当作执行授权。
func DetectInstallationProfile(getenv func(string) string, executable func() (string, error)) InstallationProfile {
	return DetectInstallationProfileWithComposeReader(context.Background(), getenv, func(key string) (string, bool) {
		value := getenv(key)
		return value, value != ""
	}, executable, nil)
}

// DetectInstallationProfileWithComposeReader 在未配置显式根目录时读取容器运行时候选；显式路径失败时不会回退。
//
//nolint:cyclop,gocognit,gocyclo // 显式路径、自动发现、声明不匹配和 binary 指引是互斥的安装安全门。
func DetectInstallationProfileWithComposeReader(ctx context.Context, getenv func(string) string, lookupEnv func(string) (string, bool), executable func() (string, error), reader moduleapi.UpdateComposeRuntimeReader) InstallationProfile {
	if ctx == nil {
		ctx = context.Background()
	}
	declared := normalizeMode(getenv(declaredDeploymentModeEnv))
	composeRootValue, composeRootSet := lookupEnv("GRAFT_UPDATE_COMPOSE_ROOT")
	composeRoot := strings.TrimSpace(composeRootValue)
	detected := "binary"
	if filepath.IsAbs(composeRoot) {
		detected = "compose"
	}
	profile := InstallationProfile{DeclaredMode: declared, DetectedMode: detected, ComposeRootSource: "unavailable", ComposeCandidates: []ComposeRootCandidate{}}
	if composeRootSet {
		profile.ComposeRootSource = "explicit_env"
	}
	if composeRootSet && (composeRoot == "" || detected != "compose") {
		profile.Capability = "manual_guidance_blocked"
		profile.BlockingReason = "GRAFT_UPDATE_COMPOSE_ROOT 必须是非空的宿主机绝对路径"
		profile.Guidance = profile.BlockingReason + "；不会回退到 Docker 自动发现。"
		return profile
	}
	switch {
	case detected == "compose" && declared == "compose":
		profile.ComposeRootSource = "explicit_env"
		profile.Capability = "compose_upgrade_available"
		profile.Guidance = "官方 Compose 安装已通过声明与挂载路径预检；升级前仍会执行执行器预检。"
	case declared == "compose" && !composeRootSet && reader != nil:
		candidates, err := reader.DiscoverCurrentServerCompose(ctx)
		profile.DetectedMode = "compose"
		profile.ComposeRootSource = "docker_discovered"
		for _, candidate := range candidates {
			warning := ""
			if len(candidate.Warnings) > 0 {
				warning = candidate.Warnings[0]
			}
			profile.ComposeCandidates = append(profile.ComposeCandidates, ComposeRootCandidate{CandidateKey: candidate.CandidateKey, Root: candidate.Root, WorkingDir: candidate.WorkingDir, ConfigFiles: copyStrings(candidate.ConfigFiles), ProjectName: candidate.ProjectName, Confidence: candidate.Confidence, Warning: warning})
		}
		if err != nil || len(profile.ComposeCandidates) == 0 {
			profile.Capability = "manual_guidance"
			profile.BlockingReason = "无法从 Docker 当前 server 容器发现可确认的 Compose 根目录"
			profile.Guidance = profile.BlockingReason
		} else {
			profile.Capability = "compose_upgrade_available"
			profile.Guidance = "Compose 根目录由 Docker bind mount 推导；执行升级前必须确认候选。"
		}
	case declared == "compose" && !composeRootSet:
		profile.DetectedMode = "compose"
		profile.ComposeRootSource = "docker_unavailable"
		profile.Capability = "manual_guidance_blocked"
		profile.BlockingReason = "未配置 GRAFT_UPDATE_COMPOSE_ROOT，且 Docker 运行时发现不可用"
		profile.Guidance = profile.BlockingReason + "；不会使用容器内路径或 binary 方式替代。"
	case detected == "compose":
		profile.ComposeRootSource = "explicit_env"
		profile.Capability = "manual_guidance"
		profile.Guidance = "检测到 Compose 部署证据，但声明模式未确认；请设置 GRAFT_UPDATE_DEPLOYMENT_MODE=compose 后重试。"
	default:
		profile = binaryInstallationProfile(profile, getenv, executable)
	}
	return profile
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string{}, values...)
}

// binaryInstallationProfile 只生成可复现的人工升级指引；缺失任一必要宿主信息时明确阻断完整指引。
func binaryInstallationProfile(profile InstallationProfile, getenv func(string) string, executable func() (string, error)) InstallationProfile {
	profile.Capability = "manual_guidance"
	profile.BinaryPath = strings.TrimSpace(getenv(binaryPathEnv))
	profile.WebRoot = strings.TrimSpace(getenv(binaryWebRootEnv))
	profile.ServiceManager = normalizeServiceManager(getenv(serviceManagerEnv))
	profile.ServiceName = strings.TrimSpace(getenv(serviceNameEnv))
	if profile.BinaryPath == "" {
		if path, err := executable(); err == nil {
			profile.BinaryPath = strings.TrimSpace(path)
		}
	}
	missing := make([]string, 0)
	if profile.BinaryPath == "" {
		missing = append(missing, binaryPathEnv)
	}
	if profile.WebRoot == "" {
		missing = append(missing, binaryWebRootEnv)
	}
	if profile.ServiceManager == "" {
		missing = append(missing, serviceManagerEnv)
	}
	if profile.ServiceManager == "systemd" && profile.ServiceName == "" {
		missing = append(missing, serviceNameEnv)
	}
	if len(missing) > 0 {
		profile.Capability = "manual_guidance_blocked"
		profile.BlockingReason = "完整二进制升级指引缺少 " + strings.Join(missing, ", ")
		profile.Guidance = profile.BlockingReason + "。不会自动替换正在运行的 binary。"
		return profile
	}
	profile.ManualSteps = []ManualStep{
		{Key: "download"},
		{Key: "verify"},
		{Key: "deploy", Params: map[string]string{"binary_path": profile.BinaryPath, "web_root": profile.WebRoot}},
		{Key: "migrate"},
	}
	if profile.ServiceManager == "systemd" {
		profile.ManualSteps = append(profile.ManualSteps, ManualStep{Key: "restartSystemd", Params: map[string]string{"service_name": profile.ServiceName}})
	} else {
		profile.ManualSteps = append(profile.ManualSteps, ManualStep{Key: "restartManual"})
	}
	profile.Guidance = "二进制部署只能按此受控人工步骤升级；server 与 web 必须使用同一 release tag。"
	return profile
}

func normalizeServiceManager(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "systemd", "manual":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "compose", "binary":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}
