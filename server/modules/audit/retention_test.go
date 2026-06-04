package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/config"
	"graft/server/internal/cronx"
)

func TestNewAuditLogRetentionPolicyRejectsNonPositiveRetention(t *testing.T) {
	_, err := newAuditLogRetentionPolicy(config.AuditConfig{})
	if err == nil {
		t.Fatal("expected invalid retention policy error")
	}
}

func TestAuditLogRetentionPolicyCutoff(t *testing.T) {
	policy, err := newAuditLogRetentionPolicy(config.AuditConfig{LogRetention: 30 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}

	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	cutoff, err := policy.cutoff(now)
	if err != nil {
		t.Fatalf("cutoff: %v", err)
	}

	want := now.Add(-30 * 24 * time.Hour)
	if !cutoff.Equal(want) {
		t.Fatalf("expected cutoff %s, got %s", want, cutoff)
	}
}

func TestAuditLogRetentionCleanerInvokesServiceWithCutoff(t *testing.T) {
	repo := &stubAuditRepository{deletedRows: 5}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cleaner, err := newAuditLogRetentionCleaner(
		zap.NewNop(),
		service,
		config.AuditConfig{LogRetention: 7 * 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	}

	deleted, err := cleaner.cleanup(context.Background())
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	wantCutoff := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	if deleted != 5 {
		t.Fatalf("expected deleted row count 5, got %d", deleted)
	}
	if !repo.deletedBefore.Equal(wantCutoff) {
		t.Fatalf("expected cutoff %s, got %s", wantCutoff, repo.deletedBefore)
	}
}

func TestAuditLogRetentionCleanerLogsFailure(t *testing.T) {
	repo := &stubAuditRepository{deleteErr: errors.New("boom")}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	core, logs := observer.New(zap.InfoLevel)
	cleaner, err := newAuditLogRetentionCleaner(
		zap.New(core),
		service,
		config.AuditConfig{LogRetention: 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	}

	if _, err := cleaner.cleanup(context.Background()); err == nil {
		t.Fatal("expected cleanup failure")
	}

	if logs.FilterMessage("audit log retention cleanup started").Len() != 1 {
		t.Fatalf("expected start log, got %#v", logs.All())
	}
	if logs.FilterMessage("audit log retention cleanup failed").Len() != 1 {
		t.Fatalf("expected failure log, got %#v", logs.All())
	}
}

func TestRegisterAuditLogRetentionCleanupJob(t *testing.T) {
	registry := cronx.NewRegistry()
	repo := &stubAuditRepository{deletedRows: 2}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if err := registerAuditLogRetentionCleanupJob(
		registry,
		zap.NewNop(),
		service,
		config.AuditConfig{LogRetention: 30 * 24 * time.Hour},
	); err != nil {
		t.Fatalf("register retention job: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one registered retention job, got %d", len(items))
	}
	if items[0].Name != auditLogRetentionCleanupJobName {
		t.Fatalf("expected job name %q, got %q", auditLogRetentionCleanupJobName, items[0].Name)
	}
	if items[0].Module != moduleID {
		t.Fatalf("expected job module %q, got %q", moduleID, items[0].Module)
	}
	if items[0].Schedule != auditLogRetentionCleanupJobSchedule {
		t.Fatalf("expected job schedule %q, got %q", auditLogRetentionCleanupJobSchedule, items[0].Schedule)
	}
	if err := items[0].Validate(); err != nil {
		t.Fatalf("validate registered job: %v", err)
	}
	if !repo.deletedBefore.IsZero() {
		t.Fatal("expected startup registration to avoid cleanup execution")
	}

	if err := items[0].Run(context.Background()); err != nil {
		t.Fatalf("run registered job: %v", err)
	}
	if repo.deletedBefore.IsZero() {
		t.Fatal("expected job run to invoke cleanup")
	}
}
