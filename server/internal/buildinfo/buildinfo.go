// Package buildinfo exposes the canonical server build identity surface.
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

// Info is the canonical build metadata baseline for the server artifact.
type Info struct {
	Version      string
	GitCommit    string
	BuildTimeUTC string
	GitTreeState string
}

// Current constructs the current build identity from package-level variables, applying default fallbacks for empty fields.
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

// IsOfficialRelease reports whether the current identity looks like a tagged release build.
func (i Info) IsOfficialRelease() bool {
	normalized := normalize(i)
	return normalized.Version != defaultVersion
}

// IsDirty reports whether the injected tree state explicitly marks the build as dirty.
func (i Info) IsDirty() bool {
	return strings.EqualFold(normalize(i).GitTreeState, "dirty")
}

// normalize returns a normalized copy of info with canonical field values, applying fallback defaults to empty or invalid fields.
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

// normalizeTreeState returns a canonical representation of the given git tree state value.
// The result is either "clean", "dirty", or the default unknown state.
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
