package contract

import "strings"

const (
	// ProjectLifecycleMaxAdditionalArgs bounds the number of user-supplied Compose arguments.
	ProjectLifecycleMaxAdditionalArgs = 32
	// ProjectLifecycleMaxAdditionalArgLength bounds one user-supplied Compose argument.
	ProjectLifecycleMaxAdditionalArgLength = 256
)

// NormalizeLifecycleAdditionalArgs 裁剪并校验用户提供的 Compose 参数存储形式。
// 命令权威覆盖项仍由组装 Compose 命令的 project service 拥有。
func NormalizeLifecycleAdditionalArgs(values []string) ([]string, bool) {
	if len(values) > ProjectLifecycleMaxAdditionalArgs {
		return nil, false
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		argument := strings.TrimSpace(value)
		if argument == "" || len(argument) > ProjectLifecycleMaxAdditionalArgLength || strings.ContainsAny(argument, "\r\n\x00") {
			return nil, false
		}
		normalized = append(normalized, argument)
	}
	return normalized, true
}
