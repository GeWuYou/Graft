package audit

import (
	"os"
	"strings"
	"testing"
)

// TestAuditPolicyMigrationSeedIsIdempotent 验证审计策略迁移具备幂等性：
// 唯一索引约束规则名、upsert 允许重复执行、并且更新语义会刷新 updated_at。
func TestAuditPolicyMigrationSeedIsIdempotent(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("migrations/202605190003_audit_module_schema.sql")
	if err != nil {
		t.Fatalf("read policy migration: %v", err)
	}

	sql := string(content)
	if !strings.Contains(sql, `CREATE UNIQUE INDEX IF NOT EXISTS "audit_policy_rules_name"`) {
		t.Fatal("expected policy migration to enforce unique rule names")
	}
	if !strings.Contains(sql, `ON CONFLICT ("name") DO UPDATE SET`) {
		t.Fatal("expected policy migration seed to upsert by rule name")
	}
	if !strings.Contains(sql, `"updated_at" = NOW()`) {
		t.Fatal("expected policy migration seed upsert to refresh updated_at")
	}
}

func TestContainerDangerousActionPolicyUpgradeSeedExists(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("migrations/202606250001_audit_container_dangerous_action_policies.sql")
	if err != nil {
		t.Fatalf("read container policy migration: %v", err)
	}

	sql := string(content)
	for _, action := range []string{
		"ops.container.action.start",
		"ops.container.action.stop",
		"ops.container.action.restart",
		"ops.container.action.remove",
		"ops.container.action.batch.start",
		"ops.container.action.batch.stop",
		"ops.container.action.batch.restart",
		"ops.container.action.batch.remove",
	} {
		if !strings.Contains(sql, action) {
			t.Fatalf("expected container dangerous action %q in upgrade migration", action)
		}
	}
	if !strings.Contains(sql, `ON CONFLICT ("name") DO UPDATE SET`) {
		t.Fatal("expected container policy upgrade migration to upsert by rule name")
	}
}
