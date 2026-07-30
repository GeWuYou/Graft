package update_test

import (
	"os"
	"strings"
	"testing"
)

// TestUpdateOperationMigrationConvergesDeploymentStrategy 验证保留的历史列迁移与前向重命名迁移形成可回放链。
func TestUpdateOperationMigrationConvergesDeploymentStrategy(t *testing.T) {
	modeContents, err := os.ReadFile("migrations/202607300001_update_operation_mode.sql")
	if err != nil {
		t.Fatalf("read historical update mode migration: %v", err)
	}
	strategyContents, err := os.ReadFile("migrations/202607300002_rename_update_operation_deployment_strategy.sql")
	if err != nil {
		t.Fatalf("read deployment strategy migration: %v", err)
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
