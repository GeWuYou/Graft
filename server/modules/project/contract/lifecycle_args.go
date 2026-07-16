package contract

import "strings"

const (
	// ApplicationLifecycleMaxAdditionalArgs 限制调用方可追加的 Compose 参数数量，避免任务载荷无界增长。
	ApplicationLifecycleMaxAdditionalArgs = 32
	// ApplicationLifecycleMaxAdditionalArgLength 限制单个追加参数长度，避免异常参数进入任务计划。
	ApplicationLifecycleMaxAdditionalArgLength = 256
)

// NormalizeLifecycleAdditionalArgs 裁剪并校验用户提供的 Compose 参数存储形式。
// 命令权威覆盖项仍由组装 Compose 命令的 Application 服务拥有。
func NormalizeLifecycleAdditionalArgs(values []string) ([]string, bool) {
	if len(values) > ApplicationLifecycleMaxAdditionalArgs {
		return nil, false
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		argument := strings.TrimSpace(value)
		if argument == "" || len(argument) > ApplicationLifecycleMaxAdditionalArgLength || strings.ContainsAny(argument, "\r\n\x00") {
			return nil, false
		}
		normalized = append(normalized, argument)
	}
	return normalized, true
}
