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
	"graft/server/internal/cronx"
)

type retentionRepoRecorder struct {
	cutoffs []time.Time
	deleted int64
	err     error
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

func (r *retentionRepoRecorder) ListAccessLogs(context.Context, AccessLogListQuery) (AccessLogListResult, error) {
	return AccessLogListResult{}, nil
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
	cutoff, err := policy.cutoff(now)
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

	deleted, err := cleaner.cleanup(context.Background())
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if deleted != 5 {
		t.Fatalf("expected deleted rows 5, got %d", deleted)
	}
	if len(repo.cutoffs) != 1 {
		t.Fatalf("expected one cleanup invocation, got %d", len(repo.cutoffs))
	}

	wantCutoff := now.Add(-3 * 24 * time.Hour)
	if !repo.cutoffs[0].Equal(wantCutoff) {
		t.Fatalf("expected cutoff %s, got %s", wantCutoff, repo.cutoffs[0])
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

	if _, err := cleaner.cleanup(context.Background()); err == nil {
		t.Fatal("expected cleanup failure")
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

func TestRegisterAccessLogRetentionCleanupJob(t *testing.T) {
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

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one registered job, got %d", len(items))
	}
	if items[0].Name != accessLogRetentionCleanupJobName {
		t.Fatalf("expected job name %q, got %q", accessLogRetentionCleanupJobName, items[0].Name)
	}
	if items[0].Plugin != accessLogRetentionCleanupJobPlugin {
		t.Fatalf("expected job plugin %q, got %q", accessLogRetentionCleanupJobPlugin, items[0].Plugin)
	}
	if items[0].Schedule != accessLogRetentionCleanupJobSchedule {
		t.Fatalf("expected job schedule %q, got %q", accessLogRetentionCleanupJobSchedule, items[0].Schedule)
	}
}
