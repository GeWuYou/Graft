// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/cronx"
)

type retentionRepoRecorder struct {
	cutoffs     []time.Time
	limits      []int
	listQueries []AccessLogListQuery
	deleted     int64
	matched     int64
	total       int64
	err         error
	listErr     error
}

func (r *retentionRepoRecorder) CreateAccessLog(context.Context, CreateAccessLogInput) (AccessLog, error) {
	return AccessLog{}, nil
}

func (r *retentionRepoRecorder) CreateAccessLogs(context.Context, []CreateAccessLogInput) ([]AccessLog, error) {
	return nil, nil
}

func (r *retentionRepoRecorder) DeleteAccessLogsBefore(_ context.Context, occurredBefore time.Time) (int64, error) {
	r.cutoffs = append(r.cutoffs, occurredBefore)
	if r.err != nil {
		return 0, r.err
	}
	return r.deleted, nil
}

func (r *retentionRepoRecorder) DeleteAccessLogsBeforeLimit(_ context.Context, occurredBefore time.Time, limit int) (int64, error) {
	r.cutoffs = append(r.cutoffs, occurredBefore)
	r.limits = append(r.limits, limit)
	if r.err != nil {
		return 0, r.err
	}
	return r.deleted, nil
}

func (r *retentionRepoRecorder) ListAccessLogs(_ context.Context, query AccessLogListQuery) (AccessLogListResult, error) {
	r.listQueries = append(r.listQueries, query)
	if r.listErr != nil {
		return AccessLogListResult{}, r.listErr
	}
	total := r.total
	if query.OccurredTo != nil {
		total = r.matched
	}
	return AccessLogListResult{Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *retentionRepoRecorder) GetAccessLogByID(context.Context, uint64) (AccessLog, error) {
	return AccessLog{}, ErrAccessLogNotFound
}

func TestNewAccessLogRetentionPolicyRejectsNonPositiveRetention(t *testing.T) {
	_, err := newAccessLogRetentionPolicy(config.HTTPXConfig{})
	if err == nil {
		t.Fatal("expected invalid retention policy error")
	}
}

func TestAccessLogRetentionPolicyCutoff(t *testing.T) {
	policy, err := newAccessLogRetentionPolicy(config.HTTPXConfig{AccessLogRetention: 7 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}

	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	cutoff, err := accessLogRetentionCutoff(now, policy.retention)
	if err != nil {
		t.Fatalf("cutoff: %v", err)
	}

	want := now.Add(-7 * 24 * time.Hour)
	if !cutoff.Equal(want) {
		t.Fatalf("expected cutoff %s, got %s", want, cutoff)
	}
}

func TestAccessLogRetentionCleanerInvokesRepositoryWithCutoff(t *testing.T) {
	repo := &retentionRepoRecorder{deleted: 5}
	cleaner, err := newAccessLogRetentionCleaner(
		zap.NewNop(),
		repo,
		config.HTTPXConfig{AccessLogRetention: 3 * 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}

	now := time.Date(2026, 5, 30, 8, 30, 0, 0, time.UTC)
	cleaner.now = func() time.Time { return now }

	result, err := cleaner.cleanup(context.Background(), accessLogRetentionJobConfig{RetentionDays: 9, BatchSize: 1000})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if result.Metrics["deletedCount"] != int64(5) {
		t.Fatalf("expected deleted rows 5, got %#v", result)
	}
	if len(repo.cutoffs) != 1 {
		t.Fatalf("expected one cleanup invocation, got %d", len(repo.cutoffs))
	}
	if len(repo.limits) != 1 || repo.limits[0] != 1000 {
		t.Fatalf("expected cleanup limit 1000, got %#v", repo.limits)
	}

	wantCutoff := now.Add(-9 * 24 * time.Hour)
	if !repo.cutoffs[0].Equal(wantCutoff) {
		t.Fatalf("expected cutoff %s, got %s", wantCutoff, repo.cutoffs[0])
	}
	if result.Details["retentionDays"] != 9 {
		t.Fatalf("expected configured retention days in result, got %#v", result.Details)
	}
	if _, ok := result.Details["dryRun"]; ok {
		t.Fatalf("did not expect dryRun in cleanup details: %#v", result.Details)
	}
}

func TestAccessLogRetentionCleanerEstimateDoesNotDelete(t *testing.T) {
	repo := &retentionRepoRecorder{matched: 12, total: 40}
	cleaner, err := newAccessLogRetentionCleaner(
		zap.NewNop(),
		repo,
		config.HTTPXConfig{AccessLogRetention: 30 * 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}

	now := time.Date(2026, 5, 30, 8, 30, 0, 0, time.UTC)
	cleaner.now = func() time.Time { return now }

	result, err := cleaner.estimate(context.Background(), accessLogRetentionJobConfig{RetentionDays: 7, BatchSize: 500})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if len(repo.cutoffs) != 0 {
		t.Fatalf("expected estimate not to delete access logs, got %d delete calls", len(repo.cutoffs))
	}
	if len(repo.listQueries) != 2 {
		t.Fatalf("expected two estimate queries, got %d", len(repo.listQueries))
	}
	if repo.listQueries[0].OccurredTo == nil {
		t.Fatalf("expected first estimate query to filter by cutoff: %#v", repo.listQueries[0])
	}
	wantCutoff := now.Add(-7 * 24 * time.Hour)
	if !repo.listQueries[0].OccurredTo.Equal(wantCutoff) {
		t.Fatalf("expected cutoff %s, got %s", wantCutoff, *repo.listQueries[0].OccurredTo)
	}
	if repo.listQueries[1].OccurredTo != nil {
		t.Fatalf("expected total estimate query without cutoff: %#v", repo.listQueries[1])
	}
	if result.Stage != "estimated" || result.Metrics["estimatedScanCount"] != int64(12) ||
		result.Metrics["estimatedDeleteCount"] != int64(12) ||
		result.Metrics["estimatedRetainCount"] != int64(28) {
		t.Fatalf("unexpected estimate result: %#v", result)
	}
	if _, ok := result.Details["dryRun"]; ok {
		t.Fatalf("did not expect dryRun in estimate details: %#v", result.Details)
	}
}

func TestAccessLogRetentionCleanerLogsFailure(t *testing.T) {
	repo := &retentionRepoRecorder{err: errors.New("boom")}
	buffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(buffer),
		zap.InfoLevel,
	))

	cleaner, err := newAccessLogRetentionCleaner(
		logger,
		repo,
		config.HTTPXConfig{AccessLogRetention: 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 5, 30, 9, 0, 0, 0, time.UTC)
	}

	if result, err := cleaner.cleanup(context.Background(), accessLogRetentionJobConfig{BatchSize: 1000}); err == nil {
		t.Fatal("expected cleanup failure")
	} else if result.Stage != "failed" || len(result.Warnings) == 0 {
		t.Fatalf("expected failed structured result, got %#v", result)
	}

	output := buffer.String()
	for _, want := range []string{
		"access log retention cleanup started",
		"access log retention cleanup failed",
		"deletedRows",
	} {
		if want == "deletedRows" {
			if strings.Contains(output, want) {
				t.Fatalf("did not expect deletedRows on failure log, got %s", output)
			}
			continue
		}
		if !strings.Contains(output, want) {
			t.Fatalf("expected log output to contain %q, got %s", want, output)
		}
	}
}

func TestRegisterAccessLogRetentionCleanupJobMetadata(t *testing.T) {
	items, _ := registeredAccessLogRetentionCleanupJobForTest(t)

	if len(items) != 1 {
		t.Fatalf("expected one registered job, got %d", len(items))
	}
	item := items[0]
	if item.Name != accessLogRetentionCleanupJobName {
		t.Fatalf("expected job name %q, got %q", accessLogRetentionCleanupJobName, item.Name)
	}
	if item.Module != accessLogRetentionCleanupJobModule {
		t.Fatalf("expected job module %q, got %q", accessLogRetentionCleanupJobModule, item.Module)
	}
	if item.Schedule != accessLogRetentionCleanupJobSchedule {
		t.Fatalf("expected job schedule %q, got %q", accessLogRetentionCleanupJobSchedule, item.Schedule)
	}
	if item.DefaultConfig != accessLogRetentionCleanupDefaultConfig {
		t.Fatalf("expected default config %s, got %s", accessLogRetentionCleanupDefaultConfig, item.DefaultConfig)
	}
	if strings.Contains(item.DefaultConfig, "dryRun") || strings.Contains(item.ConfigSchema, "dryRun") {
		t.Fatalf("did not expect dryRun in persistent access log job config: default=%s schema=%s", item.DefaultConfig, item.ConfigSchema)
	}
	if len(item.Actions) != 1 {
		t.Fatalf("expected one access log retention action, got %#v", item.Actions)
	}
	action := item.Actions[0]
	if action.Key != accessLogRetentionDryRunActionKey ||
		action.TitleKey != accessLogRetentionDryRunActionTitleKey ||
		action.DescriptionKey != accessLogRetentionDryRunActionDescKey ||
		action.Handler == nil {
		t.Fatalf("unexpected access log retention action: %#v", action)
	}
}

func TestRegisterAccessLogRetentionConfigDefinition(t *testing.T) {
	registry := configregistry.NewRegistry()

	if err := RegisterAccessLogRetentionConfigDefinition(registry); err != nil {
		t.Fatalf("register config definition: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one config definition, got %d", len(items))
	}
	definition := items[0]
	if definition.Key != accessLogRetentionCleanupJobName ||
		definition.Module != accessLogRetentionCleanupJobModule ||
		definition.Type != configregistry.ValueTypeObject {
		t.Fatalf("unexpected access log config definition: %#v", definition)
	}
	if definition.GroupKey != accessLogRetentionConfigGroupKey ||
		definition.TitleKey != accessLogRetentionConfigTitleKey ||
		definition.DescriptionKey != accessLogRetentionConfigDescriptionKey {
		t.Fatalf("expected localized access log config metadata, got %#v", definition)
	}
	if string(definition.DefaultValue) != accessLogRetentionCleanupDefaultConfig {
		t.Fatalf("expected default config %s, got %s", accessLogRetentionCleanupDefaultConfig, definition.DefaultValue)
	}
	if !strings.Contains(string(definition.Schema), `"x-i18n"`) ||
		!strings.Contains(string(definition.Schema), `"unitKey":"systemConfig.units.days"`) ||
		!strings.Contains(string(definition.Schema), `"batchSize":{"type":"integer","minimum":1,"maximum":10000`) {
		t.Fatalf("expected x-i18n schema metadata, got %s", string(definition.Schema))
	}
}

func TestRegisterAccessLogRetentionCleanupJobHandlers(t *testing.T) {
	items, repo := registeredAccessLogRetentionCleanupJobForTest(t)
	item := items[0]

	result, err := item.Handler(context.Background(), item.RuntimeDefaultConfig())
	if err != nil {
		t.Fatalf("run registered job: %v", err)
	}
	if result.AffectedResource != "access_log" || result.Metrics["deletedCount"] != int64(2) {
		t.Fatalf("expected structured access log cleanup result, got %#v", result)
	}
	actionResult, err := item.Actions[0].Handler(context.Background(), `{"retentionDays":7,"batchSize":500}`)
	if err != nil {
		t.Fatalf("run registered action: %v", err)
	}
	if actionResult.Stage != "estimated" || len(repo.cutoffs) != 1 {
		t.Fatalf("expected action to estimate without deleting, got result=%#v deleteCalls=%d", actionResult, len(repo.cutoffs))
	}
}

func TestDecodeAccessLogRetentionJobConfigClampsRetentionDays(t *testing.T) {
	config := decodeAccessLogRetentionJobConfig(`{"retentionDays":366,"batchSize":500}`)

	if config.RetentionDays != accessLogRetentionMaxDays {
		t.Fatalf("expected retention days clamped to %d, got %d", accessLogRetentionMaxDays, config.RetentionDays)
	}
	if config.BatchSize != 500 {
		t.Fatalf("expected configured batch size, got %d", config.BatchSize)
	}
}

func registeredAccessLogRetentionCleanupJobForTest(t *testing.T) ([]cronx.Job, *retentionRepoRecorder) {
	t.Helper()
	registry := cronx.NewRegistry()
	repo := &retentionRepoRecorder{deleted: 2}

	if err := RegisterAccessLogRetentionCleanupJob(
		registry,
		zap.NewNop(),
		repo,
		config.HTTPXConfig{AccessLogRetention: 30 * 24 * time.Hour},
	); err != nil {
		t.Fatalf("register retention job: %v", err)
	}
	return registry.Items(), repo
}
