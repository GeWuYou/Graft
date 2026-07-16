// Package buildinfo 提供服务端构建身份信息的规范化读取边界。
package buildinfo

import "strings"

const (
	defaultVersion      = "dev"
	defaultGitCommit    = "unknown"
	defaultBuildTimeUTC = "unknown"
	defaultGitTreeState = "unknown"
)

var (
	version      = defaultVersion
	gitCommit    = defaultGitCommit
	buildTimeUTC = defaultBuildTimeUTC
	gitTreeState = defaultGitTreeState
)

// Info 是服务端构建产物使用的规范化构建元数据快照。
type Info struct {
	Version      string
	GitCommit    string
	BuildTimeUTC string
	GitTreeState string
}

// Current 从包级构建变量生成当前构建身份，并为空字段应用默认回退值。
func Current() Info {
	return Normalize(Info{
		Version:      version,
		GitCommit:    gitCommit,
		BuildTimeUTC: buildTimeUTC,
		GitTreeState: gitTreeState,
	})
}

// Normalize 对任意构建信息快照应用规范化规则和默认值回退。
func Normalize(info Info) Info {
	return normalize(info)
}

// IsOfficialRelease 根据版本字段是否仍为开发默认值判断当前构建是否像带标签的正式发布版本。
func (i Info) IsOfficialRelease() bool {
	normalized := normalize(i)
	return normalized.Version != defaultVersion
}

// IsDirty 仅在注入的 Git 工作树状态明确为 dirty 时报告构建包含未提交变更。
func (i Info) IsDirty() bool {
	return strings.EqualFold(normalize(i).GitTreeState, "dirty")
}

// normalize 返回规范化副本；空字段和无法识别的字段会回退到稳定默认值。
func normalize(info Info) Info {
	info.Version = normalizeField(info.Version, defaultVersion)
	info.GitCommit = normalizeField(info.GitCommit, defaultGitCommit)
	info.BuildTimeUTC = normalizeField(info.BuildTimeUTC, defaultBuildTimeUTC)
	info.GitTreeState = normalizeTreeState(info.GitTreeState)
	return info
}

// normalizeField 规范化一个字符串，移除前后空白，如果结果为空则返回回退值。
func normalizeField(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// normalizeTreeState 将 Git 工作树状态规范化为 clean、dirty 或默认的 unknown。
func normalizeTreeState(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	switch trimmed {
	case "", "unknown":
		return defaultGitTreeState
	case "clean", "dirty":
		return trimmed
	default:
		return defaultGitTreeState
	}
}
