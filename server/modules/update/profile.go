package update

import "graft/server/internal/moduleapi"

// InstallationProfile 是 Update Center 对 Deployment Runtime 当前不可变上下文的受限投影。
// Update 不解释环境变量或 Docker inspect facts，只将 Deployment 的结论映射为既有响应契约。
type InstallationProfile struct {
	DeclaredMode                    string                 `json:"declared_mode"`
	DetectedMode                    string                 `json:"detected_mode"`
	Capability                      string                 `json:"capability"`
	Guidance                        string                 `json:"guidance"`
	BinaryPath                      string                 `json:"binary_path,omitempty"`
	WebRoot                         string                 `json:"web_root,omitempty"`
	ServiceManager                  string                 `json:"service_manager,omitempty"`
	ServiceName                     string                 `json:"service_name,omitempty"`
	ManualSteps                     []ManualStep           `json:"manual_steps,omitempty"`
	BlockingReason                  string                 `json:"blocking_reason,omitempty"`
	ComposeRootSource               string                 `json:"compose_root_source"`
	ComposeRootConfirmationRequired bool                   `json:"compose_root_confirmation_required"`
	ComposeCandidates               []ComposeRootCandidate `json:"compose_candidates"`
}

// ManualStep 保留既有二进制人工指引的响应兼容形状；Deployment Runtime 不再猜测其文件系统路径。
type ManualStep struct {
	Key    string            `json:"key"`
	Params map[string]string `json:"params,omitempty"`
}

// ComposeRootCandidate 是当前 DeploymentContext 的受控候选投影；原始路径永不接受为 HTTP 输入。
type ComposeRootCandidate struct {
	CandidateKey string   `json:"key"`
	Root         string   `json:"host_path"`
	ConfigFiles  []string `json:"compose_files"`
	ProjectName  string   `json:"project_name,omitempty"`
	Confidence   string   `json:"confidence"`
	Warning      string   `json:"warning,omitempty"`
}

func installationProfile(context moduleapi.DeploymentContext) InstallationProfile {
	profile := InstallationProfile{
		DeclaredMode:                    context.Mode(),
		DetectedMode:                    context.Mode(),
		ComposeRootSource:               context.ComposeRootSource(),
		ComposeRootConfirmationRequired: context.ComposeConfirmationRequired(),
		ComposeCandidates:               []ComposeRootCandidate{},
	}
	for _, candidate := range context.ComposeCandidates() {
		warning := ""
		if warnings := candidate.Warnings(); len(warnings) > 0 {
			warning = warnings[0]
		}
		profile.ComposeCandidates = append(profile.ComposeCandidates, ComposeRootCandidate{
			CandidateKey: candidate.Key(),
			Root:         candidate.Root(),
			ConfigFiles:  candidate.ConfigFiles(),
			ProjectName:  candidate.ProjectName(),
			Confidence:   candidate.Confidence(),
			Warning:      warning,
		})
	}
	if context.Available() {
		profile.Capability = "compose_upgrade_available"
		if profile.ComposeRootConfirmationRequired {
			profile.Guidance = "Compose 根目录存在多个或低置信度候选；执行升级前必须确认候选。"
		} else {
			profile.Guidance = "Compose 根目录已由 Deployment Runtime 唯一确认。"
		}
		return profile
	}
	profile.Capability = "manual_guidance_blocked"
	if diagnostics := context.Diagnostics(); len(diagnostics) > 0 {
		profile.BlockingReason = diagnostics[0].Message
	}
	if profile.BlockingReason == "" {
		profile.BlockingReason = "当前部署运行时不可用于受控 Compose 升级"
	}
	profile.Guidance = profile.BlockingReason
	return profile
}

func normalizeMode(value string) string {
	if value == "compose" || value == "binary" {
		return value
	}
	return "unknown"
}
