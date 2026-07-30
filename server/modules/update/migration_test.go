package update_test

import (
	"os"
	"strings"
	"testing"

	"graft/server/internal/moduleregistry"
)

// TestUpdateOperationMigrationConvergesDeploymentStrategy 验证保留的历史列迁移、前向重命名迁移及其运行时嵌入副本形成可回放链。
func TestUpdateOperationMigrationConvergesDeploymentStrategy(t *testing.T) {
	modeContents, err := os.ReadFile("migrations/202607300001_update_operation_mode.sql")
	if err != nil {
		t.Fatalf("read historical update mode migration: %v", err)
	}
	strategyContents, err := os.ReadFile("migrations/202607300002_rename_update_operation_deployment_strategy.sql")
	if err != nil {
		t.Fatalf("read deployment strategy migration: %v", err)
	}
	atlasSumContents, err := os.ReadFile("migrations/atlas.sum")
	if err != nil {
		t.Fatalf("read update migration atlas sum: %v", err)
	}

	dir, ok := moduleregistry.EmbeddedMigrationDirByPath("modules/update/migrations")
	if !ok {
		t.Fatal("expected compile-time embedded update migration dir")
	}
	embeddedFiles := make(map[string][]byte, len(dir.Files))
	for _, file := range dir.Files {
		embeddedFiles[file.Name] = file.Contents
	}
	for _, sourceFile := range []struct {
		name     string
		contents []byte
	}{
		{name: "202607300001_update_operation_mode.sql", contents: modeContents},
		{name: "202607300002_rename_update_operation_deployment_strategy.sql", contents: strategyContents},
		{name: "atlas.sum", contents: atlasSumContents},
	} {
		embeddedContents, ok := embeddedFiles[sourceFile.name]
		if !ok {
			t.Fatalf("expected embedded update migration file %s", sourceFile.name)
		}
		if string(embeddedContents) != string(sourceFile.contents) {
			t.Fatalf("expected embedded update migration %s to stay aligned with live source content", sourceFile.name)
		}
	}

	for _, want := range []string{
		"ADD COLUMN update_mode VARCHAR(32) NOT NULL DEFAULT 'unknown'",
		"ADD CONSTRAINT update_operations_update_mode_check",
		"CHECK (update_mode IN ('stable_tracking', 'beta_tracking', 'pinned_stable', 'pinned_beta', 'unknown')) NOT VALID",
		"VALIDATE CONSTRAINT update_operations_update_mode_check",
	} {
		if !strings.Contains(string(modeContents), want) {
			t.Fatalf("historical migration must contain %q", want)
		}
	}
	for _, want := range []string{
		"RENAME COLUMN update_mode TO deployment_strategy",
		"RENAME CONSTRAINT update_operations_update_mode_check",
		"TO update_operations_deployment_strategy_check",
		"COMMENT ON COLUMN update_operations.deployment_strategy IS '创建更新操作时冻结的部署升级策略；历史记录无法推导时标记为未知'",
	} {
		if !strings.Contains(string(strategyContents), want) {
			t.Fatalf("forward migration must contain %q", want)
		}
	}
}
