package logger

import (
	"os"
	"strings"
	"testing"
)

func TestAppLogApplicationCategoryCorrectionMigration(t *testing.T) {
	sql, err := os.ReadFile("migrations/202607160002_app_log_application_category.sql")
	if err != nil {
		t.Fatalf("read category correction migration: %v", err)
	}

	contents := string(sql)
	if !strings.Contains(contents, `ALTER COLUMN "category" SET DEFAULT 'application'`) {
		t.Fatalf("expected application column default correction, got %q", contents)
	}
	if strings.Contains(contents, `SET "category" = 'application'`) {
		t.Fatalf("correction migration must not reclassify explicit categories, got %q", contents)
	}
}
