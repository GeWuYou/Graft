package update

import (
	"os"
	"strings"
)

const updatePolicyEnv = "GRAFT_UPDATE_POLICY"

// UpdatePolicy 描述官方 Compose 部署写入 .env 的更新选择策略。
//
//revive:disable:exported // UpdatePolicy 与 OpenAPI 和部署配置的术语保持一致。
type UpdatePolicy string

const (
	// UpdatePolicyStable 只允许选择已验证 stable release。
	UpdatePolicyStable UpdatePolicy = "stable"
	// UpdatePolicyBeta 只允许选择已验证 beta release。
	UpdatePolicyBeta UpdatePolicy = "beta"
	// UpdatePolicyFixed 允许管理员选择一个已验证的具体 release。
	UpdatePolicyFixed UpdatePolicy = "fixed"
	// UpdatePolicyManual 禁止自动更新执行。
	UpdatePolicyManual UpdatePolicy = "manual"
)

//revive:enable:exported

func parseUpdatePolicy(value string) (UpdatePolicy, bool) {
	switch UpdatePolicy(strings.ToLower(strings.TrimSpace(value))) {
	case UpdatePolicyStable, UpdatePolicyBeta, UpdatePolicyFixed, UpdatePolicyManual:
		return UpdatePolicy(strings.ToLower(strings.TrimSpace(value))), true
	default:
		return "", false
	}
}

func configuredUpdatePolicy() (UpdatePolicy, bool) {
	return parseUpdatePolicy(os.Getenv(updatePolicyEnv))
}
