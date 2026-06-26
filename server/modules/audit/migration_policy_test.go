package audit_test

import (
	"os"
	"strings"
	"testing"

	"graft/server/internal/moduleregistry"
)

// TestAuditPolicyMigrationSeedIsIdempotent 验证审计策略迁移具备幂等性：
// 唯一索引约束规则名、upsert 允许重复执行、并且更新语义会刷新 updated_at。
func TestAuditPolicyMigrationSeedIsIdempotent(t *testing.T) {
	t.Parallel()

	for _, sql := range auditMigrationVariants(t, "202605190003_audit_module_schema.sql") {
		if !strings.Contains(sql.contents, `CREATE UNIQUE INDEX IF NOT EXISTS "audit_policy_rules_name"`) {
			t.Fatalf("%s: expected policy migration to enforce unique rule names", sql.name)
		}
		if !strings.Contains(sql.contents, `ON CONFLICT ("name") DO UPDATE SET`) {
			t.Fatalf("%s: expected policy migration seed to upsert by rule name", sql.name)
		}
		if !strings.Contains(sql.contents, `"updated_at" = NOW()`) {
			t.Fatalf("%s: expected policy migration seed upsert to refresh updated_at", sql.name)
		}
	}
}

func TestContainerDangerousActionPolicyUpgradeSeedExists(t *testing.T) {
	t.Parallel()

	for _, sql := range auditMigrationVariants(t, "202606250001_audit_container_dangerous_action_policies.sql") {
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
			if !strings.Contains(sql.contents, action) {
				t.Fatalf("%s: expected container dangerous action %q in upgrade migration", sql.name, action)
			}
		}
		if !strings.Contains(sql.contents, `ON CONFLICT ("name") DO UPDATE SET`) {
			t.Fatalf("%s: expected container policy upgrade migration to upsert by rule name", sql.name)
		}
	}
}

type auditMigrationVariant struct {
	name     string
	contents string
}

func auditMigrationVariants(t *testing.T, fileName string) []auditMigrationVariant {
	t.Helper()

	var (
		content []byte
		err     error
	)
	switch fileName {
	case "202605190003_audit_module_schema.sql":
		content, err = os.ReadFile("migrations/202605190003_audit_module_schema.sql")
	case "202606250001_audit_container_dangerous_action_policies.sql":
		content, err = os.ReadFile("migrations/202606250001_audit_container_dangerous_action_policies.sql")
	default:
		t.Fatalf("unsupported audit migration file %s", fileName)
	}
	if err != nil {
		t.Fatalf("read audit migration source %s: %v", fileName, err)
	}

	dir, ok := moduleregistry.EmbeddedMigrationDirByPath("modules/audit/migrations")
	if !ok {
		t.Fatal("expected compile-time embedded audit migration dir")
	}

	embeddedContent := ""
	for _, file := range dir.Files {
		if file.Name == fileName {
			embeddedContent = string(file.Contents)
			break
		}
	}
	if embeddedContent == "" {
		t.Fatalf("expected embedded audit migration file %s", fileName)
	}
	if embeddedContent != string(content) {
		t.Fatalf("expected embedded audit migration %s to stay aligned with live source content", fileName)
	}

	return []auditMigrationVariant{
		{name: "live-source", contents: string(content)},
		{name: "compile-time-embedded", contents: embeddedContent},
	}
}
