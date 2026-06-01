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

	content, err := os.ReadFile("migrations/202605280004_audit_policy_rules.sql")
	if err != nil {
		t.Fatalf("read policy migration: %v", err)
	}

	sql := string(content)
	if !strings.Contains(sql, `CREATE UNIQUE INDEX "audit_policy_rules_name"`) {
		t.Fatal("expected policy migration to enforce unique rule names")
	}
	if !strings.Contains(sql, `ON CONFLICT ("name") DO UPDATE SET`) {
		t.Fatal("expected policy migration seed to upsert by rule name")
	}
	if !strings.Contains(sql, `"updated_at" = NOW()`) {
		t.Fatal("expected policy migration seed upsert to refresh updated_at")
	}
}
