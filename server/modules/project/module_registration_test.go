package project

import (
	"strings"
	"testing"
)

// TestPermissionItemsDeclareLocalizedMetadata 验证项目模块注册的内置权限都声明了本地化元数据。
func TestPermissionItemsDeclareLocalizedMetadata(t *testing.T) {
	for _, item := range permissionItems("project") {
		if item.Code == "" {
			t.Fatal("permission code must not be empty")
		}
		if strings.HasPrefix(item.Code, "ops.") {
			t.Fatalf("permission %s retains the legacy ops prefix", item.Code)
		}
		if !strings.HasPrefix(item.DisplayKey, "rbac.permissionCatalog.") {
			t.Fatalf("permission %s has invalid display key %q", item.Code, item.DisplayKey)
		}
		if !strings.HasPrefix(item.DescriptionKey, "rbac.permissionCatalog.") {
			t.Fatalf("permission %s has invalid description key %q", item.Code, item.DescriptionKey)
		}
	}
}
