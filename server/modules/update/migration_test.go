package update_test

import (
	"strings"
	"testing"

	"graft/server/internal/moduleregistry"
)

// TestUpdateOperationMigrationPersistsUpdateMode 验证运行时嵌入迁移与更新模式的完整持久化约束保持一致。
func TestUpdateOperationMigrationPersistsUpdateMode(t *testing.T) {
	dir, ok := moduleregistry.EmbeddedMigrationDirByPath("modules/update/migrations")
	if !ok {
		t.Fatal("expected compile-time embedded update migration dir")
	}

	var contents string
	for _, file := range dir.Files {
		if file.Name == "202607300001_update_operation_mode.sql" {
			contents = string(file.Contents)
			break
		}
	}
	if contents == "" {
		t.Fatal("expected embedded update mode migration")
	}

	for _, want := range []string{
		"ADD COLUMN update_mode VARCHAR(32) NOT NULL DEFAULT 'unknown'",
		"ALTER COLUMN update_mode DROP DEFAULT",
		"update_operations_update_mode_check",
		"CHECK (update_mode IN ('stable_tracking', 'beta_tracking', 'pinned_stable', 'pinned_beta', 'unknown')) NOT VALID",
		"VALIDATE CONSTRAINT update_operations_update_mode_check",
	} {
		if !strings.Contains(contents, want) {
			t.Fatalf("embedded update mode migration must contain %q", want)
		}
	}
}
