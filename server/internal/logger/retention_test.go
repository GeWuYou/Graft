// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/cronx"
)

type appLogRetentionRepoRecorder struct {
	mu          sync.Mutex
	created     []CreateAppLogInput
	cutoffs     []time.Time
	limits      []int
	deleted     int64
	matched     int64
	total       int64
	err         error
	listErr     error
	listQueries []AppLogListQuery
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

func (r *appLogRetentionRepoRecorder) DeleteAppLogsBeforeLimit(_ context.Context, cutoff time.Time, limit int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cutoffs = append(r.cutoffs, cutoff)
	r.limits = append(r.limits, limit)
	if r.err != nil {
		return 0, r.err
	}
	return r.deleted, nil
}

func (r *appLogRetentionRepoRecorder) ListAppLogs(_ context.Context, query AppLogListQuery) (AppLogListResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listQueries = append(r.listQueries, query)
	if r.listErr != nil {
		return AppLogListResult{}, r.listErr
	}
	total := r.total
	if query.OccurredBefore != nil {
		total = r.matched
	}
	return AppLogListResult{Total: total, Page: query.Page, PageSize: query.PageSize}, nil
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

	result, err := cleaner.cleanup(context.Background(), appLogRetentionJobConfig{RetentionDays: 9, BatchSize: 1000})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if result.Metrics["deletedCount"] != int64(3) {
		t.Fatalf("expected deleted rows 3, got %#v", result)
	}
	if result.Details["retentionDays"] != 9 {
		t.Fatalf("expected configured retention days in result, got %#v", result)
	}
	if _, ok := result.Details["dryRun"]; ok {
		t.Fatalf("did not expect dryRun in persistent cleanup result details: %#v", result.Details)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.cutoffs) != 1 {
		t.Fatalf("expected one cutoff, got %d", len(repo.cutoffs))
	}
	if len(repo.limits) != 1 || repo.limits[0] != 1000 {
		t.Fatalf("expected cleanup limit 1000, got %#v", repo.limits)
	}
	wantCutoff := now.Add(-9 * hoursPerDay * time.Hour)
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
	} else if result.Stage != "failed" || result.Summary != appLogRetentionFailureSummary || len(result.Warnings) == 0 {
		t.Fatalf("expected failed structured result, got %#v", result)
	} else if strings.Contains(result.Summary, "boom") || strings.Contains(result.Warnings[0], "boom") {
		t.Fatalf("expected failure result to hide raw error, got %#v", result)
	}

	record := waitRetentionAppLogRecord(t, repo, "scheduler job failed")
	if record.Severity != AppLogSeverityError || record.Error == "" {
		t.Fatalf("expected failed scheduler app log, got %#v", record)
	}
}

func TestRegisterAppLogRetentionCleanupJob(t *testing.T) {
	registry := cronx.NewRegistry()
	repo := &appLogRetentionRepoRecorder{}

	if err := RegisterAppLogRetentionCleanupJob(
		registry,
		zap.NewNop(),
		nil,
		repo,
		config.LogConfig{AppLogRetention: 7 * 24 * time.Hour},
	); err != nil {
		t.Fatalf("register retention job: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one registered job, got %d", len(items))
	}
	assertAppLogRetentionJobMetadata(t, items[0])
	assertAppLogRetentionDryRunAction(t, repo, items[0])
}

func TestRegisterAppLogRetentionConfigDefinition(t *testing.T) {
	registry := configregistry.NewRegistry()

	if err := RegisterAppLogRetentionConfigDefinition(registry); err != nil {
		t.Fatalf("register config definition: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one config definition, got %d", len(items))
	}
	assertAppLogRetentionConfigDefinition(t, items[0])
}

func assertAppLogRetentionConfigDefinition(t *testing.T, definition configregistry.Definition) {
	t.Helper()

	if definition.Key != appLogRetentionCleanupJobName ||
		definition.Module != appLogRetentionCleanupJobModule ||
		definition.Type != configregistry.ValueTypeObject {
		t.Fatalf("unexpected app log config definition: %#v", definition)
	}
	if definition.GroupKey != appLogRetentionConfigGroupKey ||
		definition.DomainKey != appLogRetentionConfigDomainKey ||
		definition.GroupDescriptionKey != appLogRetentionConfigGroupDescKey ||
		definition.TitleKey != appLogRetentionConfigTitleKey ||
		definition.DescriptionKey != appLogRetentionConfigDescriptionKey {
		t.Fatalf("expected localized app log config metadata, got %#v", definition)
	}
	if definition.GroupLabel == "core.logger / log.retention" {
		t.Fatalf("group label must be product-facing fallback, got %q", definition.GroupLabel)
	}
	if string(definition.DefaultValue) != appLogRetentionCleanupDefaultConfig {
		t.Fatalf("expected default config %s, got %s", appLogRetentionCleanupDefaultConfig, definition.DefaultValue)
	}
	if !strings.Contains(string(definition.Schema), `"x-i18n"`) ||
		!strings.Contains(string(definition.Schema), `"unitKey":"systemConfig.units.rows"`) ||
		!strings.Contains(string(definition.Schema), `"batchSize":{"type":"integer","minimum":1,"maximum":10000`) {
		t.Fatalf("expected x-i18n schema metadata, got %s", string(definition.Schema))
	}
}

func assertAppLogRetentionJobMetadata(t *testing.T, job cronx.Job) {
	t.Helper()

	if job.Name != appLogRetentionCleanupJobName {
		t.Fatalf("expected job name %q, got %q", appLogRetentionCleanupJobName, job.Name)
	}
	if job.Module != appLogRetentionCleanupJobModule {
		t.Fatalf("expected job module %q, got %q", appLogRetentionCleanupJobModule, job.Module)
	}
	if job.Schedule != appLogRetentionCleanupJobSchedule {
		t.Fatalf("expected job schedule %q, got %q", appLogRetentionCleanupJobSchedule, job.Schedule)
	}
	if job.DefaultConfig != appLogRetentionCleanupDefaultConfig {
		t.Fatalf("expected default config %s, got %s", appLogRetentionCleanupDefaultConfig, job.DefaultConfig)
	}
	assertAppLogRetentionJobConfig(t, job)
	assertAppLogRetentionJobActions(t, job)
}

func assertAppLogRetentionJobConfig(t *testing.T, job cronx.Job) {
	t.Helper()

	if job.ConfigSchema == "" || job.DefaultConfig == "" {
		t.Fatalf("expected registered job config schema/default config, got %#v", job)
	}
	if strings.Contains(job.DefaultConfig, "dryRun") || strings.Contains(job.ConfigSchema, "dryRun") {
		t.Fatalf("did not expect dryRun in persistent app log job config: default=%s schema=%s", job.DefaultConfig, job.ConfigSchema)
	}
	if !strings.Contains(job.DefaultConfig, "retentionDays") || !strings.Contains(job.ConfigSchema, "retentionDays") {
		t.Fatalf("expected retentionDays in app log job config: default=%s schema=%s", job.DefaultConfig, job.ConfigSchema)
	}
}

func assertAppLogRetentionJobActions(t *testing.T, job cronx.Job) {
	t.Helper()

	if len(job.Actions) != 1 || job.Actions[0].Key != appLogRetentionDryRunActionKey {
		t.Fatalf("expected dryRun action, got %#v", job.Actions)
	}
}

func assertAppLogRetentionDryRunAction(t *testing.T, repo *appLogRetentionRepoRecorder, job cronx.Job) {
	t.Helper()

	repo.matched = 9
	repo.total = 20
	actionResult, err := job.Actions[0].Handler(context.Background(), `{"retentionDays":7,"batchSize":500}`)
	if err != nil {
		t.Fatalf("run dryRun action: %v", err)
	}
	if actionResult.Stage != "estimated" ||
		actionResult.Metrics["estimatedDeleteCount"] != int64(9) ||
		actionResult.Metrics["estimatedRetainCount"] != int64(11) {
		t.Fatalf("expected dryRun action to return estimate result, got %#v", actionResult)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.cutoffs) != 0 {
		t.Fatalf("expected dryRun action to skip deletion, got cutoffs %#v", repo.cutoffs)
	}
	if len(repo.listQueries) != 2 || repo.listQueries[0].OccurredBefore == nil || repo.listQueries[0].OccurredTo != nil {
		t.Fatalf("expected dryRun estimate to use exclusive cutoff query, got %#v", repo.listQueries)
	}
}

func TestAppLogRetentionDryRunClampsEstimatedDeleteCountToBatchSize(t *testing.T) {
	repo := &appLogRetentionRepoRecorder{matched: 12, total: 20}
	cleaner, err := newAppLogRetentionCleaner(
		zap.NewNop(),
		nil,
		repo,
		config.LogConfig{AppLogRetention: 7 * 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	}

	result, err := cleaner.estimate(context.Background(), appLogRetentionJobConfig{RetentionDays: 7, BatchSize: 5})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if result.Metrics["estimatedScanCount"] != int64(12) || result.Metrics["estimatedDeleteCount"] != int64(5) {
		t.Fatalf("expected clamped delete estimate with full scan count, got %#v", result.Metrics)
	}
}

func TestDecodeAppLogRetentionJobConfigClampsRetentionDays(t *testing.T) {
	config := decodeAppLogRetentionJobConfig(`{"retentionDays":366,"batchSize":500}`)

	if config.RetentionDays != appLogRetentionMaxDays {
		t.Fatalf("expected retention days clamped to %d, got %d", appLogRetentionMaxDays, config.RetentionDays)
	}
	if config.BatchSize != 500 {
		t.Fatalf("expected configured batch size, got %d", config.BatchSize)
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
