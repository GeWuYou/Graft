package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"graft/server/internal/buildinfo"
)

// runValidateServerLocaleOwnership 校验服务器本地化所有权守卫。
// 
// @return 成功时返回 nil；否则返回带上下文的错误。
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

// runValidateRelease 校验仓库是否满足发布要求。
// 它要求构建信息为正式发布版本，且 Git 提交、构建时间和 Git 工作区状态均满足发布约束；随后检查仓库根目录、嵌入式 OpenAPI bundle 的新鲜度以及默认 Atlas migration 链的完整性。成功时返回 nil，任一校验失败时返回错误。
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
