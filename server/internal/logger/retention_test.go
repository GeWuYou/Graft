package logger

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/cronx"
)

type appLogRetentionRepoRecorder struct {
	mu      sync.Mutex
	created []CreateAppLogInput
	cutoffs []time.Time
	deleted int64
	err     error
}

func (r *appLogRetentionRepoRecorder) CreateAppLog(_ context.Context, input CreateAppLogInput) (AppLogRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, input)
	return AppLogRecord{}, nil
}

func (r *appLogRetentionRepoRecorder) DeleteAppLogsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
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
		nil,
		repo,
		config.LogConfig{AppLogRetention: 72 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}

	now := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	cleaner.now = func() time.Time { return now }

	result, err := cleaner.cleanup(context.Background(), appLogRetentionJobConfig{BatchSize: 1000})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if result.Metrics["deletedCount"] != int64(3) {
		t.Fatalf("expected deleted rows 3, got %#v", result)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.cutoffs) != 1 {
		t.Fatalf("expected one cutoff, got %d", len(repo.cutoffs))
	}
	wantCutoff := now.Add(-72 * time.Hour)
	if !repo.cutoffs[0].Equal(wantCutoff) {
		t.Fatalf("expected cutoff %s, got %s", wantCutoff, repo.cutoffs[0])
	}
}

func TestAppLogRetentionCleanerWritesCompletedAppLog(t *testing.T) {
	repo := &appLogRetentionRepoRecorder{deleted: 7}
	appLog := NewAppLogger(zap.NewNop(), WithAppLogRepository(repo))
	cleaner, err := newAppLogRetentionCleaner(
		zap.NewNop(),
		appLog,
		repo,
		config.LogConfig{AppLogRetention: 72 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	}

	if _, err := cleaner.cleanup(context.Background(), appLogRetentionJobConfig{BatchSize: 1000}); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	record := waitRetentionAppLogRecord(t, repo, "scheduler job completed")
	if record.Component != "internal.logger.retention" {
		t.Fatalf("expected retention component, got %#v", record)
	}
	if record.Operation != "app_log_retention_cleanup" || record.Fields["deleted_rows"] != "7" {
		t.Fatalf("expected deletion count app log fields, got %#v", record)
	}
}

func TestAppLogRetentionCleanerReturnsDeleteError(t *testing.T) {
	repo := &appLogRetentionRepoRecorder{err: errors.New("boom")}
	appLog := NewAppLogger(zap.NewNop(), WithAppLogRepository(repo))
	cleaner, err := newAppLogRetentionCleaner(
		zap.NewNop(),
		appLog,
		repo,
		config.LogConfig{AppLogRetention: 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	}

	if result, err := cleaner.cleanup(context.Background(), appLogRetentionJobConfig{BatchSize: 1000}); err == nil {
		t.Fatal("expected cleanup error")
	} else if result.Stage != "failed" || len(result.Warnings) == 0 {
		t.Fatalf("expected failed structured result, got %#v", result)
	}

	record := waitRetentionAppLogRecord(t, repo, "scheduler job failed")
	if record.Severity != AppLogSeverityError || record.Error == "" {
		t.Fatalf("expected failed scheduler app log, got %#v", record)
	}
}

func TestRegisterAppLogRetentionCleanupJob(t *testing.T) {
	registry := cronx.NewRegistry()

	if err := RegisterAppLogRetentionCleanupJob(
		registry,
		zap.NewNop(),
		nil,
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
	if items[0].ConfigSchema == "" || items[0].DefaultConfig == "" {
		t.Fatalf("expected registered job config schema/default config, got %#v", items[0])
	}
}

func waitRetentionAppLogRecord(t *testing.T, repo *appLogRetentionRepoRecorder, message string) CreateAppLogInput {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		for _, record := range repo.created {
			if record.Message == message {
				repo.mu.Unlock()
				return record
			}
		}
		repo.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	t.Fatalf("expected retention app log %q, got %#v", message, repo.created)
	return CreateAppLogInput{}
}
