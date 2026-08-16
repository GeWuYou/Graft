package cronx

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
)

// TestRegistryIsDeclarationOnly 验证 Registry 只保存 Job Definition 声明，不执行
// handler、启动 scheduler 或取得长期资源；调度器仍由独立 runtime owner 装配。
func TestRegistryIsDeclarationOnly(t *testing.T) {
	var runs atomic.Int32
	job := validTestJob()
	job.Run = func(context.Context) error {
		runs.Add(1)
		return nil
	}

	registry := NewRegistry()
	registry.Register(job)
	if got := runs.Load(); got != 0 {
		t.Fatalf("expected registration not to execute job, got %d runs", got)
	}

	items := registry.Items()
	if len(items) != 1 || items[0].RuntimeKey() != job.RuntimeKey() {
		t.Fatalf("expected registered declaration snapshot, got %#v", items)
	}
	items[0].Key = "mutated"
	if got := registry.Items()[0].RuntimeKey(); got != job.RuntimeKey() {
		t.Fatalf("expected registry declaration to remain isolated from item mutation, got %q", got)
	}

	if _, err := job.Invoke(context.Background(), "{}"); err != nil {
		t.Fatalf("invoke job explicitly: %v", err)
	}
	if got := runs.Load(); got != 1 {
		t.Fatalf("expected explicit invocation to execute job once, got %d runs", got)
	}
	if !reflect.DeepEqual(registry.Items()[0].Actions, job.Actions) {
		t.Fatal("expected registry to preserve declaration actions")
	}
}

func TestJobRuntimeTitleDoesNotUseMessageKeyAsDisplayText(t *testing.T) {
	job := Job{
		Key:            "audit.audit-log-retention-cleanup",
		TitleKey:       "scheduledTask.auditLogRetention.title",
		DescriptionKey: "scheduledTask.auditLogRetention.description",
	}

	if got := job.RuntimeTitle(); got != job.Key {
		t.Fatalf("expected runtime title to fall back to job key, got %q", got)
	}
	if got := job.RuntimeDescription(); got != job.Key {
		t.Fatalf("expected runtime description to fall back to display title, got %q", got)
	}
}

func TestJobRuntimeTitleUsesExplicitDisplayText(t *testing.T) {
	job := Job{
		Key:         "audit.audit-log-retention-cleanup",
		Title:       "Audit log retention cleanup",
		Description: "Deletes audit logs beyond the configured retention window.",
	}

	if got := job.RuntimeTitle(); got != job.Title {
		t.Fatalf("expected explicit runtime title, got %q", got)
	}
	if got := job.RuntimeDescription(); got != job.Description {
		t.Fatalf("expected explicit runtime description, got %q", got)
	}
}

func TestJobValidateRejectsUnsupportedNonEmptyCategory(t *testing.T) {
	job := validTestJob()
	job.Category = JobCategory("unknown")

	if err := job.Validate(); err == nil {
		t.Fatal("expected unsupported category to fail validation")
	}
}

func TestJobValidateAllowsEmptyCategoryAsCustomDefault(t *testing.T) {
	job := validTestJob()

	if err := job.Validate(); err != nil {
		t.Fatalf("expected empty category to validate as default custom category: %v", err)
	}
	if got := job.RuntimeCategory(); got != JobCategoryCustom {
		t.Fatalf("expected empty category to default to custom, got %q", got)
	}
}

func validTestJob() Job {
	return Job{
		Key:       "audit.audit-log-retention-cleanup",
		ModuleKey: "audit",
		Schedule:  "0 0 * * * *",
		Run:       func(context.Context) error { return nil },
	}
}
