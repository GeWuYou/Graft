package pluginregistry

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestMigrationDirsUsesOwnerAlignedBaseline 验证默认迁移链不再包含历史共享目录，
// 而是消费 live core-owned + plugin-owned 目录。
func TestMigrationDirsUsesOwnerAlignedBaseline(t *testing.T) {
	dirs, err := MigrationDirs()
	if err != nil {
		t.Fatalf("migration dirs: %v", err)
	}

	expected := []string{
		"internal/httpx/migrations",
		"plugins/user/migrations",
		"plugins/auth/migrations",
		"plugins/rbac/migrations",
		"plugins/audit/migrations",
	}
	if !reflect.DeepEqual(dirs, expected) {
		t.Fatalf("expected %v, got %v", expected, dirs)
	}
}

// TestDescriptorsStayAlignedWithPluginDirectories verifies the generated registry
// still includes every plugin directory that declares a runtime descriptor.
func TestDescriptorsStayAlignedWithPluginDirectories(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "plugins"))
	if err != nil {
		t.Fatalf("read plugin directories: %v", err)
	}

	want := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join("..", "..", "plugins", entry.Name())
		if !fileExists(filepath.Join(dir, "descriptor.go")) {
			continue
		}

		want = append(want, entry.Name())
	}
	sort.Strings(want)

	got := make([]string, 0, len(Descriptors()))
	for _, descriptor := range Descriptors() {
		got = append(got, descriptor.Name())
	}
	sort.Strings(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected descriptors %v, got %v", want, got)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
