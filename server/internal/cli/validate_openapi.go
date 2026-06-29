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

func runValidateOpenAPI(_ *cobra.Command, specPath string) error {
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

	if err := backendOpenAPIFreshnessRunner(); err != nil {
		return err
	}

	return nil
}

// runValidateOpenAPIFreshness 验证嵌入式 OpenAPI 规范包和生成的后端规范是否为最新状态。
func runValidateOpenAPIFreshness() error {
	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for generated freshness validation: %w", err)
	}

	if err := validateEmbeddedOpenAPIBundleFreshness(repoRoot); err != nil {
		return err
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "openapi_generated_freshness_check.py")
	cmd := &cobra.Command{}
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

// validateEmbeddedOpenAPIBundleFreshness checks that the runtime-embedded OpenAPI bundle matches the canonical source by comparing their SHA-256 digests. Returns an error if the digests do not match, including instructions to regenerate the bundle via `go generate`.
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
