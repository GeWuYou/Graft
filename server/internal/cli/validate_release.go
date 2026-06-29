package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"graft/server/internal/buildinfo"
)

func runValidateServerLocaleOwnership(cmd *cobra.Command) error {
	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for server locale ownership guard: %w", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "check_server_locale_ownership.py")
	if err := backendCommandRunner(cmd, "python3", scriptPath); err != nil {
		return fmt.Errorf("run server locale ownership guard: %w", err)
	}

	return nil
}

// RunValidateRelease validates that the repository is ready for release.
func runValidateRelease(_ *cobra.Command) error {
	info := buildinfo.Normalize(buildReleaseInfoSnapshot())
	if !info.IsOfficialRelease() || info.GitCommit == "unknown" || info.BuildTimeUTC == "unknown" {
		return fmt.Errorf(
			"`graft validate release` requires release-grade BuildInfo; current version=%q git_commit=%q build_time_utc=%q git_tree_state=%q",
			info.Version,
			info.GitCommit,
			info.BuildTimeUTC,
			info.GitTreeState,
		)
	}
	if info.GitTreeState != "clean" {
		return fmt.Errorf("`graft validate release` requires a clean release build; current git_tree_state=%q", info.GitTreeState)
	}

	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for release validation: %w", err)
	}

	if err := validateEmbeddedOpenAPIBundleFreshness(repoRoot); err != nil {
		return err
	}

	if _, err := buildAtlasMigrationDir(repoRoot, defaultMigrationDir); err != nil {
		return fmt.Errorf("validate embedded default migration chain: %w", err)
	}

	return nil
}
