package buildinfo

import "testing"

func TestCurrentUsesDefaultsWithoutInjectedMetadata(t *testing.T) {
	originalVersion := version
	originalGitCommit := gitCommit
	originalBuildTimeUTC := buildTimeUTC
	originalGitTreeState := gitTreeState
	defer func() {
		version = originalVersion
		gitCommit = originalGitCommit
		buildTimeUTC = originalBuildTimeUTC
		gitTreeState = originalGitTreeState
	}()

	version = ""
	gitCommit = ""
	buildTimeUTC = ""
	gitTreeState = ""

	info := Current()
	if info.Version != defaultVersion {
		t.Fatalf("expected default version %q, got %q", defaultVersion, info.Version)
	}
	if info.GitCommit != defaultGitCommit {
		t.Fatalf("expected default git commit %q, got %q", defaultGitCommit, info.GitCommit)
	}
	if info.BuildTimeUTC != defaultBuildTimeUTC {
		t.Fatalf("expected default build time %q, got %q", defaultBuildTimeUTC, info.BuildTimeUTC)
	}
	if info.GitTreeState != defaultGitTreeState {
		t.Fatalf("expected default git tree state %q, got %q", defaultGitTreeState, info.GitTreeState)
	}
	if info.IsOfficialRelease() {
		t.Fatal("default build metadata must not be treated as an official release")
	}
	if info.IsDirty() {
		t.Fatal("unknown tree state must not be treated as a dirty build")
	}
}

func TestCurrentPreservesInjectedBuildMetadata(t *testing.T) {
	originalVersion := version
	originalGitCommit := gitCommit
	originalBuildTimeUTC := buildTimeUTC
	originalGitTreeState := gitTreeState
	defer func() {
		version = originalVersion
		gitCommit = originalGitCommit
		buildTimeUTC = originalBuildTimeUTC
		gitTreeState = originalGitTreeState
	}()

	version = "0.1.0"
	gitCommit = "abc1234"
	buildTimeUTC = "2026-06-22T10:00:00Z"
	gitTreeState = "dirty"

	info := Current()
	if info.Version != "0.1.0" {
		t.Fatalf("expected injected version, got %q", info.Version)
	}
	if info.GitCommit != "abc1234" {
		t.Fatalf("expected injected git commit, got %q", info.GitCommit)
	}
	if info.BuildTimeUTC != "2026-06-22T10:00:00Z" {
		t.Fatalf("expected injected build time, got %q", info.BuildTimeUTC)
	}
	if info.GitTreeState != "dirty" {
		t.Fatalf("expected injected tree state, got %q", info.GitTreeState)
	}
	if !info.IsOfficialRelease() {
		t.Fatal("non-dev version should be treated as an official release candidate")
	}
	if !info.IsDirty() {
		t.Fatal("dirty tree state should be reported as dirty")
	}
}

func TestNormalizeAppliesFallbacksToArbitrarySnapshot(t *testing.T) {
	info := Normalize(Info{})

	if info.Version != defaultVersion {
		t.Fatalf("expected normalized version %q, got %q", defaultVersion, info.Version)
	}
	if info.GitCommit != defaultGitCommit {
		t.Fatalf("expected normalized commit %q, got %q", defaultGitCommit, info.GitCommit)
	}
	if info.BuildTimeUTC != defaultBuildTimeUTC {
		t.Fatalf("expected normalized build time %q, got %q", defaultBuildTimeUTC, info.BuildTimeUTC)
	}
	if info.GitTreeState != defaultGitTreeState {
		t.Fatalf("expected normalized tree state %q, got %q", defaultGitTreeState, info.GitTreeState)
	}
}

func TestCurrentNormalizesUnknownTreeStates(t *testing.T) {
	originalGitTreeState := gitTreeState
	defer func() {
		gitTreeState = originalGitTreeState
	}()

	gitTreeState = "MIXED"

	info := Current()
	if info.GitTreeState != defaultGitTreeState {
		t.Fatalf("expected unknown fallback, got %q", info.GitTreeState)
	}
}
