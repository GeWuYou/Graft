package cronx

import "testing"

func TestJobRuntimeTitleDoesNotUseMessageKeyAsDisplayText(t *testing.T) {
	job := Job{
		Key:                   "audit.audit-log-retention-cleanup",
		DisplayMessageKey:     "scheduledTask.auditLogRetention.title",
		DescriptionMessageKey: "scheduledTask.auditLogRetention.description",
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
