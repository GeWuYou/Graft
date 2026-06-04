package logger

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/cronx"
)

type appLogRetentionRepoRecorder struct {
	created []CreateAppLogInput
	cutoffs []time.Time
	deleted int64
	err     error
}

func (r *appLogRetentionRepoRecorder) CreateAppLog(_ context.Context, input CreateAppLogInput) (AppLogRecord, error) {
	r.created = append(r.created, input)
	return AppLogRecord{}, nil
}

func (r *appLogRetentionRepoRecorder) DeleteAppLogsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.cutoffs = append(r.cutoffs, cutoff)
	if r.err != nil {
		return 0, r.err
	}
	return r.deleted, nil
}

func (r *appLogRetentionRepoRecorder) ListAppLogs(context.Context, AppLogListQuery) (AppLogListResult, error) {
	return AppLogListResult{}, nil
}

func (r *appLogRetentionRepoRecorder) GetAppLogByID(context.Context, uint64) (AppLogRecord, error) {
	return AppLogRecord{}, ErrAppLogNotFound
}

func TestNewAppLogRetentionPolicyRejectsNonPositiveRetention(t *testing.T) {
	_, err := newAppLogRetentionPolicy(config.LogConfig{})
	if err == nil {
		t.Fatal("expected invalid retention policy error")
	}
}

func TestAppLogRetentionCleanerInvokesRepositoryWithCutoff(t *testing.T) {
	repo := &appLogRetentionRepoRecorder{deleted: 3}
	cleaner, err := newAppLogRetentionCleaner(
		zap.NewNop(),
		repo,
		config.LogConfig{AppLogRetention: 72 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}

	now := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	cleaner.now = func() time.Time { return now }

	deleted, err := cleaner.cleanup(context.Background())
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("expected deleted rows 3, got %d", deleted)
	}
	if len(repo.cutoffs) != 1 {
		t.Fatalf("expected one cutoff, got %d", len(repo.cutoffs))
	}
	wantCutoff := now.Add(-72 * time.Hour)
	if !repo.cutoffs[0].Equal(wantCutoff) {
		t.Fatalf("expected cutoff %s, got %s", wantCutoff, repo.cutoffs[0])
	}
}

func TestAppLogRetentionCleanerReturnsDeleteError(t *testing.T) {
	cleaner, err := newAppLogRetentionCleaner(
		zap.NewNop(),
		&appLogRetentionRepoRecorder{err: errors.New("boom")},
		config.LogConfig{AppLogRetention: 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	}

	if _, err := cleaner.cleanup(context.Background()); err == nil {
		t.Fatal("expected cleanup error")
	}
}

func TestRegisterAppLogRetentionCleanupJob(t *testing.T) {
	registry := cronx.NewRegistry()

	if err := RegisterAppLogRetentionCleanupJob(
		registry,
		zap.NewNop(),
		&appLogRetentionRepoRecorder{},
		config.LogConfig{AppLogRetention: 7 * 24 * time.Hour},
	); err != nil {
		t.Fatalf("register retention job: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one registered job, got %d", len(items))
	}
	if items[0].Name != appLogRetentionCleanupJobName {
		t.Fatalf("expected job name %q, got %q", appLogRetentionCleanupJobName, items[0].Name)
	}
	if items[0].Module != appLogRetentionCleanupJobModule {
		t.Fatalf("expected job module %q, got %q", appLogRetentionCleanupJobModule, items[0].Module)
	}
	if items[0].Schedule != appLogRetentionCleanupJobSchedule {
		t.Fatalf("expected job schedule %q, got %q", appLogRetentionCleanupJobSchedule, items[0].Schedule)
	}
}
