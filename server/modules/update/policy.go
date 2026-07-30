package update

import (
	"os"
	"strings"
)

const imageTagEnv = "GRAFT_IMAGE_TAG"

// UpdateMode 表示从注入的镜像标签推导出的部署升级意图。
//
//nolint:revive // OpenAPI and persisted operation fields use the stable update_mode term.
type UpdateMode string

const (
	// UpdateModeStableTracking 表示跟随稳定发行频道。
	UpdateModeStableTracking UpdateMode = "stable_tracking"
	// UpdateModeBetaTracking 表示跟随 Beta 发行频道。
	UpdateModeBetaTracking UpdateMode = "beta_tracking"
	// UpdateModePinnedStable 表示锁定在稳定发行频道。
	UpdateModePinnedStable UpdateMode = "pinned_stable"
	// UpdateModePinnedBeta 表示锁定在 Beta 发行频道。
	UpdateModePinnedBeta UpdateMode = "pinned_beta"
	// UpdateModeUnknown 表示镜像标签无法推导出受支持的部署意图。
	UpdateModeUnknown UpdateMode = "unknown"
)

// DeploymentStrategy 是 Update 对唯一部署配置的解释结果，不持久化为另一份策略。
type DeploymentStrategy struct {
	ImageTag string
	Mode     UpdateMode
	Channel  string
	Tracking bool
}

func configuredDeploymentStrategy() (DeploymentStrategy, bool) {
	return parseDeploymentStrategy(os.Getenv(imageTagEnv))
}

func parseDeploymentStrategy(value string) (DeploymentStrategy, bool) {
	tag := strings.TrimSpace(value)
	switch tag {
	case "latest":
		return DeploymentStrategy{ImageTag: tag, Mode: UpdateModeStableTracking, Channel: "stable", Tracking: true}, true
	case "beta":
		return DeploymentStrategy{ImageTag: tag, Mode: UpdateModeBetaTracking, Channel: "beta", Tracking: true}, true
	}
	version, err := ParseVersion(tag)
	if err != nil || !strings.HasPrefix(tag, "v") {
		return DeploymentStrategy{ImageTag: tag, Mode: UpdateModeUnknown}, false
	}
	if version.IsPrerelease() {
		return DeploymentStrategy{ImageTag: tag, Mode: UpdateModePinnedBeta, Channel: "beta"}, true
	}
	return DeploymentStrategy{ImageTag: tag, Mode: UpdateModePinnedStable, Channel: "stable"}, true
}

func validUpdateMode(value UpdateMode) bool {
	switch value {
	case UpdateModeStableTracking, UpdateModeBetaTracking, UpdateModePinnedStable, UpdateModePinnedBeta, UpdateModeUnknown:
		return true
	}
	return false
}
