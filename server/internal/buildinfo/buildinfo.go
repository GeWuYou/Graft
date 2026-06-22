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

// Current returns the current build identity with explicit local-build fallbacks.
func Current() Info {
	return normalize(Info{
		Version:      version,
		GitCommit:    gitCommit,
		BuildTimeUTC: buildTimeUTC,
		GitTreeState: gitTreeState,
	})
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

func normalize(info Info) Info {
	info.Version = normalizeField(info.Version, defaultVersion)
	info.GitCommit = normalizeField(info.GitCommit, defaultGitCommit)
	info.BuildTimeUTC = normalizeField(info.BuildTimeUTC, defaultBuildTimeUTC)
	info.GitTreeState = normalizeTreeState(info.GitTreeState)
	return info
}

func normalizeField(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

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
