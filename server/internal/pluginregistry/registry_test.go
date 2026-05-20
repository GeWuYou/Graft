package pluginregistry

import (
	"reflect"
	"testing"
)

// TestMigrationDirsUsesPluginOwnedBaseline 验证默认迁移链不再包含历史共享目录，
// 而是只消费 compile-time registry 声明的 plugin-owned 目录。
func TestMigrationDirsUsesPluginOwnedBaseline(t *testing.T) {
	dirs, err := MigrationDirs()
	if err != nil {
		t.Fatalf("migration dirs: %v", err)
	}

	expected := []string{
		"plugins/audit/migrations",
		"plugins/user/migrations",
		"plugins/rbac/migrations",
	}
	if !reflect.DeepEqual(dirs, expected) {
		t.Fatalf("expected %v, got %v", expected, dirs)
	}
}
