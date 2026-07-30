package update

import (
	"os"
	"strings"
)

const imageTagEnv = "GRAFT_IMAGE_TAG"

// DeploymentStrategy 表示从注入的镜像标签推导出的部署升级策略。
type DeploymentStrategy string

const (
	// DeploymentStrategyStableTracking 表示跟随稳定发行频道。
	DeploymentStrategyStableTracking DeploymentStrategy = "stable_tracking"
	// DeploymentStrategyBetaTracking 表示跟随 Beta 发行频道。
	DeploymentStrategyBetaTracking DeploymentStrategy = "beta_tracking"
	// DeploymentStrategyPinnedStable 表示锁定在稳定发行频道。
	DeploymentStrategyPinnedStable DeploymentStrategy = "pinned_stable"
	// DeploymentStrategyPinnedBeta 表示锁定在 Beta 发行频道。
	DeploymentStrategyPinnedBeta DeploymentStrategy = "pinned_beta"
	// DeploymentStrategyUnknown 表示镜像标签无法推导出受支持的部署策略。
	DeploymentStrategyUnknown DeploymentStrategy = "unknown"
)

// ResolvedDeploymentStrategy 是 Update 对唯一部署配置的解释结果，不持久化为另一份策略。
type ResolvedDeploymentStrategy struct {
	ImageTag string
	Mode     DeploymentStrategy
	Channel  string
	Tracking bool
}

func configuredDeploymentStrategy() (ResolvedDeploymentStrategy, bool) {
	return parseDeploymentStrategy(os.Getenv(imageTagEnv))
}

func parseDeploymentStrategy(value string) (ResolvedDeploymentStrategy, bool) {
	tag := strings.TrimSpace(value)
	switch tag {
	case "latest":
		return ResolvedDeploymentStrategy{ImageTag: tag, Mode: DeploymentStrategyStableTracking, Channel: "stable", Tracking: true}, true
	case "beta":
		return ResolvedDeploymentStrategy{ImageTag: tag, Mode: DeploymentStrategyBetaTracking, Channel: "beta", Tracking: true}, true
	}
	version, err := ParseVersion(tag)
	if err != nil || !strings.HasPrefix(tag, "v") {
		return ResolvedDeploymentStrategy{ImageTag: tag, Mode: DeploymentStrategyUnknown}, false
	}
	if version.IsPrerelease() {
		return ResolvedDeploymentStrategy{ImageTag: tag, Mode: DeploymentStrategyPinnedBeta, Channel: "beta"}, true
	}
	return ResolvedDeploymentStrategy{ImageTag: tag, Mode: DeploymentStrategyPinnedStable, Channel: "stable"}, true
}

func validDeploymentStrategy(value DeploymentStrategy) bool {
	switch value {
	case DeploymentStrategyStableTracking, DeploymentStrategyBetaTracking, DeploymentStrategyPinnedStable, DeploymentStrategyPinnedBeta, DeploymentStrategyUnknown:
		return true
	}
	return false
}
