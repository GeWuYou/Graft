// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package moduleruntime

import (
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/module"
)

func TestBuildSnapshotReportsAllEnabledRegisteredModules(t *testing.T) {
	t.Parallel()

	snapshot := BuildSnapshot(&config.Config{}, []module.Spec{
		{
			ID:            "user",
			MigrationPath: []string{"modules/user/migrations"},
			Builder:       module.BuilderFunc(noopBuilder),
		},
		{
			ID:           "rbac",
			Dependencies: []string{"user"},
			Builder:      module.BuilderFunc(noopBuilder),
		},
	})

	if snapshot.Summary.TotalModules != 2 || snapshot.Summary.EnabledModules != 2 ||
		snapshot.Summary.RegisteredModules != 2 || snapshot.Summary.HealthyModules != 2 {
		t.Fatalf("unexpected summary: %#v", snapshot.Summary)
	}
	if string(snapshot.Items[0].EnablementSource) != enablementSourceAll || !snapshot.Items[0].Enabled {
		t.Fatalf("expected all-enabled source for first item: %#v", snapshot.Items[0])
	}
	if string(snapshot.Items[0].MigrationStatus.Status) != migrationStatusDeclared {
		t.Fatalf("expected migration declaration, got %#v", snapshot.Items[0].MigrationStatus)
	}
	if string(snapshot.Items[0].SchemaStatus.Status) != schemaStatusDeclared {
		t.Fatalf("expected schema declared from module-owned migration evidence, got %#v", snapshot.Items[0].SchemaStatus)
	}
	if got := snapshot.Items[1].Dependencies[0]; string(got.Status) != dependencyStatusSatisfied || !got.Enabled || !got.Present {
		t.Fatalf("expected satisfied dependency, got %#v", got)
	}
}

func TestBuildSnapshotReportsAllowlistDisabledAndDegradedDependencies(t *testing.T) {
	t.Parallel()

	snapshot := BuildSnapshot(&config.Config{
		Modules: config.ModulesConfig{Enabled: []string{"rbac"}},
	}, []module.Spec{
		{ID: "user", Builder: module.BuilderFunc(noopBuilder)},
		{ID: "rbac", Dependencies: []string{"user"}, Builder: module.BuilderFunc(noopBuilder)},
		{ID: "audit", Dependencies: []string{"missing"}, Builder: module.BuilderFunc(noopBuilder)},
	})

	userItem := snapshot.Items[0]
	if userItem.Enabled || string(userItem.RuntimeStatus) != runtimeStatusDisabled || string(userItem.Health) != healthDisabled {
		t.Fatalf("expected user disabled by allowlist, got %#v", userItem)
	}

	rbacItem := snapshot.Items[1]
	if !rbacItem.Enabled || string(rbacItem.RuntimeStatus) != runtimeStatusDegraded || string(rbacItem.Health) != healthDegraded {
		t.Fatalf("expected rbac degraded by disabled dependency, got %#v", rbacItem)
	}
	if got := rbacItem.Dependencies[0]; string(got.Status) != dependencyStatusDisabled {
		t.Fatalf("expected disabled dependency, got %#v", got)
	}

	auditItem := snapshot.Items[2]
	if auditItem.Enabled || string(auditItem.Dependencies[0].Status) != dependencyStatusMissing {
		t.Fatalf("expected missing dependency evidence even for disabled item, got %#v", auditItem)
	}
	if snapshot.Summary.EnabledModules != 1 || snapshot.Summary.DegradedModules != 1 {
		t.Fatalf("unexpected summary: %#v", snapshot.Summary)
	}
}

func noopBuilder(module.BuildContext) (module.Module, error) {
	return noopModule{}, nil
}

type noopModule struct{}

func (noopModule) Register(*module.Context) error { return nil }
func (noopModule) Boot(*module.Context) error     { return nil }
func (noopModule) Shutdown(*module.Context) error { return nil }
