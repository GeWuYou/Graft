package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"

	"graft/server/internal/app"
)

var backendOpenAPIFreshnessTargets = []string{
	"backend-monitor",
	"backend-health",
	"backend-rbac-management",
	"backend-user-write",
	"backend-auth-session",
	"backend-modules-runtime",
}

// runValidateMigrationVersions 执行迁移版本校验。它会解析仓库根目录，并运行 `scripts/check_migration_versions.py` 的 `--mode all` 校验。
func runValidateMigrationVersions(cmd *cobra.Command) error {
	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for migration version validation: %w", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "check_migration_versions.py")
	if err := backendCommandRunner(cmd, "python3", scriptPath, "--mode", "all"); err != nil {
		return fmt.Errorf("run migration version validation: %w", err)
	}

	return nil
}

// runValidateOpenAPI 校验 OpenAPI 规范并执行后续新鲜度检查。
// specPath 为空时使用默认根规范路径。
// 返回校验过程中的错误。
func runValidateOpenAPI(cmd *cobra.Command, specPath string) error {
	specPath = strings.TrimSpace(specPath)
	if specPath == "" {
		specPath = defaultOpenAPIRootSpec
	}

	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for openapi validation: %w", err)
	}

	rootSpec := filepath.Join(repoRoot, filepath.FromSlash(specPath))
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	document, err := loader.LoadFromFile(rootSpec)
	if err != nil {
		return fmt.Errorf("load openapi spec %q: %w", rootSpec, err)
	}
	if err := document.Validate(loader.Context); err != nil {
		return fmt.Errorf("validate openapi spec %q: %w", rootSpec, err)
	}

	if err := backendOpenAPIFreshnessRunner(cmd); err != nil {
		return err
	}

	return nil
}

// runValidateOpenAPIFreshness 验证嵌入式 OpenAPI 规范包以及生成的后端 OpenAPI 产物是否保持最新。
// 它会先检查运行时嵌入的 OpenAPI bundle 与源文件是否一致，再对各个后端目标执行新鲜度检查和边界审计。
func runValidateOpenAPIFreshness(cmd *cobra.Command) error {
	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for generated freshness validation: %w", err)
	}

	if err := validateEmbeddedOpenAPIBundleFreshness(repoRoot); err != nil {
		return err
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "openapi_generated_freshness_check.py")
	if cmd == nil {
		cmd = &cobra.Command{}
	}
	for _, target := range backendOpenAPIFreshnessTargets {
		if err := backendCommandRunner(cmd, "python3", scriptPath, "--target", target, "--mode", "check"); err != nil {
			return fmt.Errorf("run backend generated freshness check: %w", err)
		}
	}

	boundaryAuditPath := filepath.Join(repoRoot, "scripts", "openapi_generated_backend_boundary_audit.py")
	if err := backendCommandRunner(cmd, "python3", boundaryAuditPath); err != nil {
		return fmt.Errorf("run backend generated DTO boundary audit: %w", err)
	}

	return nil
}

// validateEmbeddedOpenAPIBundleFreshness 校验运行时嵌入的 OpenAPI bundle 是否与规范源一致。
// 它会比较两者的 SHA-256 摘要；如果不一致，则返回包含重新生成 bundle 指引的错误。
func validateEmbeddedOpenAPIBundleFreshness(repoRoot string) error {
	canonicalPath := filepath.Join(repoRoot, filepath.FromSlash(app.OpenAPIDocsBundleSourcePath()))

	canonicalBundle, err := backendReadFile(canonicalPath)
	if err != nil {
		return fmt.Errorf("read canonical bundled openapi spec %q: %w", canonicalPath, err)
	}
	canonicalDigest := sha256.Sum256(canonicalBundle)
	generatedDigest := app.OpenAPIDocsBundleSHA256()
	if hex.EncodeToString(canonicalDigest[:]) == generatedDigest {
		return nil
	}

	return fmt.Errorf(
		"runtime generated bundled openapi spec is stale: server/internal/app generated sha256=%s does not match %s (sha256=%s); run `cd server && go generate ./internal/app` to sync runtime docs asset",
		generatedDigest,
		app.OpenAPIDocsBundleSourcePath(),
		hex.EncodeToString(canonicalDigest[:]),
	)
}
