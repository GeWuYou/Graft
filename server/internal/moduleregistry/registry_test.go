// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package moduleregistry

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestMigrationDirsUsesOwnerAlignedBaseline 验证默认迁移链不再包含历史共享目录，
// 而是消费 live core-owned + module-owned 目录。
func TestMigrationDirsUsesOwnerAlignedBaseline(t *testing.T) {
	dirs, err := MigrationDirs()
	if err != nil {
		t.Fatalf("migration dirs: %v", err)
	}

	expected := []string{
		"internal/httpx/migrations",
		"internal/logger/migrations",
		"internal/drilldown/migrations",
		"modules/scheduler/migrations",
		"modules/user/migrations",
		"modules/auth/migrations",
		"modules/rbac/migrations",
		"modules/audit/migrations",
		"modules/notification/migrations",
		"modules/system-config/migrations",
	}
	if !reflect.DeepEqual(dirs, expected) {
		t.Fatalf("expected %v, got %v", expected, dirs)
	}
}

// TestDescriptorsStayAlignedWithModuleDirectories verifies the generated registry
// still includes every module directory that declares a runtime descriptor.
func TestDescriptorsStayAlignedWithModuleDirectories(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "modules"))
	if err != nil {
		t.Fatalf("read module directories: %v", err)
	}

	want := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join("..", "..", "modules", entry.Name())
		if !fileExists(filepath.Join(dir, "descriptor.go")) {
			continue
		}

		want = append(want, entry.Name())
	}
	sort.Strings(want)

	got := make([]string, 0, len(ModuleSpecs()))
	for _, descriptor := range ModuleSpecs() {
		got = append(got, descriptor.Name())
	}
	sort.Strings(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected descriptors %v, got %v", want, got)
	}
}

func TestFilteredOrderedModuleSpecsReturnsAllWhenEnabledUnset(t *testing.T) {
	got, err := FilteredOrderedModuleSpecs(nil)
	if err != nil {
		t.Fatalf("filtered ordered module specs: %v", err)
	}

	want, err := OrderedModuleSpecs()
	if err != nil {
		t.Fatalf("ordered module specs: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d modules, got %d", len(want), len(got))
	}
	for index := range want {
		if got[index].Name() != want[index].Name() {
			t.Fatalf("expected module %s at index %d, got %s", want[index].Name(), index, got[index].Name())
		}
		if !reflect.DeepEqual(got[index].DependsOn(), want[index].DependsOn()) {
			t.Fatalf("expected dependencies %v for module %s, got %v", want[index].DependsOn(), want[index].Name(), got[index].DependsOn())
		}
		if !reflect.DeepEqual(got[index].MigrationDirs(), want[index].MigrationDirs()) {
			t.Fatalf("expected migration dirs %v for module %s, got %v", want[index].MigrationDirs(), want[index].Name(), got[index].MigrationDirs())
		}
	}
}

func TestFilteredOrderedModuleSpecsRejectsUnknownModule(t *testing.T) {
	_, err := FilteredOrderedModuleSpecs([]string{"unknown"})
	if err == nil {
		t.Fatal("expected unknown enabled module error")
	}
}

func TestFilteredOrderedModuleSpecsRejectsDisabledDependency(t *testing.T) {
	_, err := FilteredOrderedModuleSpecs([]string{"rbac"})
	if err == nil {
		t.Fatal("expected disabled dependency error")
	}
}

func TestFilteredOrderedModuleSpecsFiltersEnabledSet(t *testing.T) {
	got, err := FilteredOrderedModuleSpecs([]string{"user", "auth"})
	if err != nil {
		t.Fatalf("filtered ordered module specs: %v", err)
	}

	want := []string{"user", "auth"}
	if len(got) != len(want) {
		t.Fatalf("expected %d modules, got %d", len(want), len(got))
	}
	for index, moduleSpec := range got {
		if moduleSpec.Name() != want[index] {
			t.Fatalf("expected module %s at index %d, got %s", want[index], index, moduleSpec.Name())
		}
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
