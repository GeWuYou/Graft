package contract

import (
	"strconv"
	"strings"
)

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

// NormalizeLifecycleActionArgs 校验指定 Compose 动作的受控 argv 模板。
func NormalizeLifecycleActionArgs(action string, values []string) ([]string, bool) {
	normalized, valid := NormalizeLifecycleAdditionalArgs(values)
	if !valid {
		return nil, false
	}
	allowed := lifecycleActionOptions(action)
	if allowed == nil {
		return nil, false
	}
	for index := 0; index < len(normalized); index++ {
		nextIndex, valid := validateLifecycleActionArgument(normalized, index, allowed)
		if !valid {
			return nil, false
		}
		index = nextIndex
	}
	return normalized, true
}

func validateLifecycleActionArgument(values []string, index int, allowed map[string]bool) (int, bool) {
	option, value, hasInlineValue := splitLifecycleOption(values[index])
	requiresValue, exists := allowed[option]
	if !exists || (hasInlineValue && !requiresValue) {
		return index, false
	}
	if !requiresValue {
		return index, true
	}
	if !hasInlineValue {
		index++
		if index >= len(values) || strings.HasPrefix(values[index], "-") {
			return index, false
		}
		value = values[index]
	}
	return index, validLifecycleOptionValue(option, value)
}

func splitLifecycleOption(argument string) (string, string, bool) {
	option, value, found := strings.Cut(argument, "=")
	return option, value, found
}

func validLifecycleOptionValue(option string, value string) bool {
	if value == "" {
		return false
	}
	switch option {
	case "--policy":
		return value == "always" || value == "missing"
	case "--timeout", "-t":
		seconds, err := strconv.Atoi(value)
		return err == nil && seconds >= 0 && seconds <= 3600
	default:
		return false
	}
}

func lifecycleActionOptions(action string) map[string]bool {
	switch action {
	case "stop":
		return map[string]bool{"--timeout": true, "-t": true}
	case "restart":
		return map[string]bool{"--no-deps": false, "--timeout": true, "-t": true}
	case "pull":
		return map[string]bool{
			"--ignore-buildable": false, "--ignore-pull-failures": false,
			"--include-deps": false, "--policy": true, "--quiet": false,
		}
	default:
		return nil
	}
}
