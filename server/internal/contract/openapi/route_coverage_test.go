package openapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestRequiredLiveRoutesStayCoveredByRootSpec(t *testing.T) {
	t.Parallel()

	repoRoot, err := findRepositoryRoot()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	specPath := filepath.Join(repoRoot, "openapi", "openapi.yaml")
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromFile(specPath)
	if err != nil {
		t.Fatalf("load root openapi spec: %v", err)
	}
	if err := spec.Validate(loader.Context); err != nil {
		t.Fatalf("validate root openapi spec: %v", err)
	}

	requiredPaths := []string{
		"/api/auth/sessions",
		"/api/auth/sessions/revoke-all",
		"/api/auth/sessions/revoke-others",
		"/api/auth/sessions/{sessionID}/revoke",
		"/api/auth/change-password",
		"/api/auth/complete-required-password-change",
		"/api/users/{id}",
		"/api/users/{id}/delete",
		"/api/users/{id}/sessions",
		"/api/users/{id}/sessions/{sessionID}/revoke",
		"/api/users/{id}/sessions/revoke-all",
		"/api/roles/{id}",
		"/api/roles/{id}/permissions",
		"/api/roles/{id}/permissions/replace",
		"/api/roles/{id}/permissions/add",
		"/api/roles/{id}/permissions/remove",
		"/api/roles/{id}/status",
		"/api/roles/{id}/delete",
		"/api/permissions/{id}",
		"/api/users/{id}/roles",
		"/api/users/{id}/roles/replace",
		"/api/users/{id}/roles/add",
		"/api/users/{id}/roles/remove",
		"/api/users/roles/replace",
		"/api/users/roles/add",
		"/api/users/roles/remove",
		"/api/audit/logs",
		"/api/audit/overview",
		"/api/monitor/server-status",
		"/api/modules/runtime",
		"/api/modules/runtime/{module_key}",
		"/api/dashboard/summary",
		"/api/dashboard/widgets/{widget_id}",
		"/api/scheduled-tasks",
		"/api/scheduled-tasks/{taskKey}",
		"/api/scheduled-tasks/{taskKey}/enable",
		"/api/scheduled-tasks/{taskKey}/disable",
		"/api/scheduled-tasks/{taskKey}/runs",
		"/api/scheduled-tasks/runs/{runId}",
		"/api/scheduled-tasks/{taskKey}/run",
	}

	for _, route := range requiredPaths {
		pathItem := spec.Paths.Find(route)
		if pathItem == nil {
			t.Fatalf("required live route %s is missing from root spec", route)
		}
	}

	excludedPaths := []string{
		"/healthz",
		"/docs",
		"/openapi.json",
		"/openapi.yaml",
	}

	for _, route := range excludedPaths {
		if route == "/healthz" {
			if spec.Paths.Find(route) == nil {
				t.Fatalf("expected excluded operational route %s to remain documented", route)
			}
			continue
		}
		if spec.Paths.Find(route) != nil {
			t.Fatalf("excluded non-business route %s should not appear in root business api spec", route)
		}
	}
}

func findRepositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(current, "openapi", "openapi.yaml")); statErr == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
		current = parent
	}
}

func TestNoExcludedDocsRoutesLeakIntoBusinessSpec(t *testing.T) {
	t.Parallel()

	repoRoot, err := findRepositoryRoot()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	// #nosec G304 -- repoRoot is constrained by findRepositoryRoot to the repository-owned root containing openapi/openapi.yaml.
	source, err := os.ReadFile(filepath.Join(repoRoot, "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read root openapi spec: %v", err)
	}

	for _, unexpected := range []string{"/docs", "/openapi.json", "/openapi.yaml"} {
		if strings.Contains(string(source), unexpected+":") {
			t.Fatalf("unexpected excluded route %s found in root spec source", unexpected)
		}
	}
}
