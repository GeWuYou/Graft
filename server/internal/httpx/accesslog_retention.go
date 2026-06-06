package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/cronx"
)

const (
	accessLogRetentionCleanupJobName           = "httpx.access-log-retention-cleanup"
	accessLogRetentionCleanupJobModule         = "core.httpx"
	accessLogRetentionCleanupJobSchedule       = "0 0 17 * * *"
	accessLogRetentionCleanupJobDisplayKey     = "scheduledTask.accessLogRetention.title"
	accessLogRetentionCleanupJobDescriptionKey = "scheduledTask.accessLogRetention.description"
	accessLogRetentionDryRunActionKey          = "dryRun"
	accessLogRetentionDryRunActionTitleKey     = "scheduledTask.action.dryRun.title"
	accessLogRetentionDryRunActionDescKey      = "scheduledTask.action.dryRun.description"
	accessLogRetentionDefaultDays              = 30
	accessLogRetentionDefaultBatchSize         = 1000
	hoursPerDay                                = 24
)

const accessLogRetentionCleanupConfigSchema = `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650,"default":30,"title":"Retention days","description":"Delete access logs older than this number of days.","x-title-key":"scheduledTask.accessLogRetention.config.retentionDays.title","x-description-key":"scheduledTask.accessLogRetention.config.retentionDays.description"},"batchSize":{"type":"integer","minimum":1,"maximum":10000,"default":1000,"title":"Batch size","description":"Maximum access log rows to delete in one cleanup batch.","x-title-key":"scheduledTask.accessLogRetention.config.batchSize.title","x-description-key":"scheduledTask.accessLogRetention.config.batchSize.description"}},"additionalProperties":false}`
const accessLogRetentionCleanupDefaultConfig = `{"retentionDays":30,"batchSize":1000}`

type accessLogRetentionJobConfig struct {
	RetentionDays int `json:"retentionDays"`
	BatchSize     int `json:"batchSize"`
}

type accessLogRetentionPolicy struct {
	retention time.Duration
}

func newAccessLogRetentionPolicy(cfg config.HTTPXConfig) (accessLogRetentionPolicy, error) {
	retention := cfg.AccessLogRetention
	if retention <= 0 {
		return accessLogRetentionPolicy{}, errors.New("access log retention must be greater than zero")
	}

	return accessLogRetentionPolicy{retention: retention}, nil
}

func accessLogRetentionCutoff(now time.Time, retention time.Duration) (time.Time, error) {
	if retention <= 0 {
		return time.Time{}, errors.New("access log retention must be greater than zero")
	}
	if now.IsZero() {
		return time.Time{}, errors.New("cutoff calculation requires a non-zero current time")
	}

	cutoff := now.UTC().Add(-retention)
	if cutoff.After(now.UTC()) || cutoff.Equal(now.UTC()) {
		return time.Time{}, errors.New("access log retention cutoff must be earlier than current time")
	}

	return cutoff, nil
}

type accessLogRetentionCleaner struct {
	logger func() *zap.Logger
	repo   AccessLogRepository
	policy accessLogRetentionPolicy
	now    func() time.Time
}

func newAccessLogRetentionCleaner(
	logger *zap.Logger,
	repo AccessLogRepository,
	cfg config.HTTPXConfig,
) (*accessLogRetentionCleaner, error) {
	policy, err := newAccessLogRetentionPolicy(cfg)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, errors.New("access log retention cleaner requires a repository")
	}

	return &accessLogRetentionCleaner{
		logger: func() *zap.Logger {
			if logger == nil {
				return zap.NewNop()
			}
			return logger
		},
		repo:   repo,
		policy: policy,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

func (c *accessLogRetentionCleaner) cleanup(ctx context.Context, config accessLogRetentionJobConfig) (cronx.JobRunResult, error) {
	if c == nil {
		err := errors.New("access log retention cleaner is required")
		return accessLogRetentionFailureResult(err, time.Time{}, config), err
	}
	started := time.Now()
	retention := c.retentionDuration(config)

	cutoff, err := accessLogRetentionCutoff(c.now(), retention)
	if err != nil {
		return accessLogRetentionFailureResult(err, time.Time{}, config), err
	}
	if cutoff.IsZero() {
		err := errors.New("access log retention cutoff must be non-zero")
		return accessLogRetentionFailureResult(err, cutoff, config), err
	}

	logger := c.logger()
	logger.Info("access log retention cleanup started",
		zap.String("job", accessLogRetentionCleanupJobName),
		zap.Duration("retention", retention),
		zap.Time("cutoff", cutoff),
	)

	deleted, err := c.repo.DeleteAccessLogsBefore(ctx, cutoff)
	if err != nil {
		logger.Error("access log retention cleanup failed",
			zap.String("job", accessLogRetentionCleanupJobName),
			zap.Duration("retention", retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		wrapped := fmt.Errorf("delete access logs before cutoff: %w", err)
		return accessLogRetentionFailureResult(wrapped, cutoff, config), wrapped
	}

	logger.Info("access log retention cleanup completed",
		zap.String("job", accessLogRetentionCleanupJobName),
		zap.Duration("retention", retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)

	return accessLogRetentionSuccessResult(deleted, retention, cutoff, config, started), nil
}

func (c *accessLogRetentionCleaner) estimate(ctx context.Context, config accessLogRetentionJobConfig) (cronx.JobRunResult, error) {
	if c == nil {
		err := errors.New("access log retention cleaner is required")
		return accessLogRetentionFailureResult(err, time.Time{}, config), err
	}
	started := time.Now()
	retention := c.retentionDuration(config)

	cutoff, err := accessLogRetentionCutoff(c.now(), retention)
	if err != nil {
		return accessLogRetentionFailureResult(err, time.Time{}, config), err
	}
	if cutoff.IsZero() {
		err := errors.New("access log retention cutoff must be non-zero")
		return accessLogRetentionFailureResult(err, cutoff, config), err
	}

	matched, total, err := c.estimateCounts(ctx, cutoff)
	if err != nil {
		wrapped := fmt.Errorf("estimate access logs before cutoff: %w", err)
		return accessLogRetentionFailureResult(wrapped, cutoff, config), wrapped
	}
	retained := total - matched
	if retained < 0 {
		retained = 0
	}

	return accessLogRetentionEstimateResult(matched, retained, retention, cutoff, config, started), nil
}

func (c *accessLogRetentionCleaner) estimateCounts(ctx context.Context, cutoff time.Time) (int64, int64, error) {
	matched, err := c.repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:       1,
		PageSize:   1,
		OccurredTo: &cutoff,
	})
	if err != nil {
		return 0, 0, err
	}
	total, err := c.repo.ListAccessLogs(ctx, AccessLogListQuery{Page: 1, PageSize: 1})
	if err != nil {
		return 0, 0, err
	}
	return matched.Total, total.Total, nil
}

func (c *accessLogRetentionCleaner) retentionDuration(config accessLogRetentionJobConfig) time.Duration {
	if config.RetentionDays > 0 {
		return time.Duration(config.RetentionDays) * hoursPerDay * time.Hour
	}
	return c.policy.retention
}

func accessLogRetentionSuccessResult(deleted int64, retention time.Duration, cutoff time.Time, config accessLogRetentionJobConfig, started time.Time) cronx.JobRunResult {
	durationMS := time.Since(started).Milliseconds()
	retentionDays := int(retention.Hours() / hoursPerDay)
	return cronx.JobRunResult{
		Summary:          fmt.Sprintf("deleted %d rows", deleted),
		Stage:            "completed",
		AffectedResource: "access_log",
		Metrics: map[string]any{
			"deletedCount":  deleted,
			"retentionDays": retentionDays,
			"batchSize":     config.BatchSize,
			"durationMs":    durationMS,
		},
		Details: map[string]any{
			"operation":     "access_log_retention_cleanup",
			"retentionDays": retentionDays,
			"cutoffTime":    cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     config.BatchSize,
			"durationMs":    durationMS,
		},
	}
}

func accessLogRetentionEstimateResult(matched int64, retained int64, retention time.Duration, cutoff time.Time, config accessLogRetentionJobConfig, started time.Time) cronx.JobRunResult {
	durationMS := time.Since(started).Milliseconds()
	retentionDays := int(retention.Hours() / hoursPerDay)
	return cronx.JobRunResult{
		Summary:          fmt.Sprintf("estimated %d rows eligible for deletion", matched),
		Stage:            "estimated",
		AffectedResource: "access_log",
		Metrics: map[string]any{
			"estimatedScanCount":   matched,
			"estimatedDeleteCount": matched,
			"estimatedRetainCount": retained,
			"retentionDays":        retentionDays,
			"batchSize":            config.BatchSize,
			"durationMs":           durationMS,
		},
		Details: map[string]any{
			"operation":     "access_log_retention_cleanup_estimate",
			"retentionDays": retentionDays,
			"cutoffTime":    cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     config.BatchSize,
			"durationMs":    durationMS,
		},
	}
}

func accessLogRetentionFailureResult(err error, cutoff time.Time, config accessLogRetentionJobConfig) cronx.JobRunResult {
	details := map[string]any{"operation": "access_log_retention_cleanup", "batchSize": config.BatchSize}
	if !cutoff.IsZero() {
		details["cutoffTime"] = cutoff.UTC().Format(time.RFC3339Nano)
	}
	return cronx.JobRunResult{Summary: err.Error(), Stage: "failed", AffectedResource: "access_log", Details: details, Warnings: []string{err.Error()}}
}

func decodeAccessLogRetentionJobConfig(configJSON string) accessLogRetentionJobConfig {
	config := accessLogRetentionJobConfig{RetentionDays: accessLogRetentionDefaultDays, BatchSize: accessLogRetentionDefaultBatchSize}
	_ = json.Unmarshal([]byte(configJSON), &config)
	if config.RetentionDays <= 0 {
		config.RetentionDays = accessLogRetentionDefaultDays
	}
	if config.BatchSize <= 0 {
		config.BatchSize = accessLogRetentionDefaultBatchSize
	}
	return config
}

// RegisterAccessLogRetentionCleanupJob registers the bounded httpx-owned access-log cleanup job.
func RegisterAccessLogRetentionCleanupJob(
	registry *cronx.Registry,
	logger *zap.Logger,
	repo AccessLogRepository,
	cfg config.HTTPXConfig,
) error {
	if registry == nil {
		return errors.New("cron registry is required")
	}

	cleaner, err := newAccessLogRetentionCleaner(logger, repo, cfg)
	if err != nil {
		return err
	}

	registry.Register(cronx.Job{
		Name:                  accessLogRetentionCleanupJobName,
		Key:                   accessLogRetentionCleanupJobName,
		Owner:                 accessLogRetentionCleanupJobModule,
		Title:                 "Access log retention cleanup",
		Description:           "Deletes access logs beyond the configured retention window.",
		DisplayMessageKey:     accessLogRetentionCleanupJobDisplayKey,
		DescriptionMessageKey: accessLogRetentionCleanupJobDescriptionKey,
		ConfigSchema:          accessLogRetentionCleanupConfigSchema,
		DefaultConfig:         accessLogRetentionCleanupDefaultConfig,
		Actions: []cronx.JobAction{
			{
				Key:            accessLogRetentionDryRunActionKey,
				TitleKey:       accessLogRetentionDryRunActionTitleKey,
				Title:          "试运行",
				DescriptionKey: accessLogRetentionDryRunActionDescKey,
				Description:    "预览本次执行结果",
				Handler: func(ctx context.Context, configJSON string) (cronx.JobRunResult, error) {
					return cleaner.estimate(ctx, decodeAccessLogRetentionJobConfig(configJSON))
				},
			},
		},
		Schedule:       accessLogRetentionCleanupJobSchedule,
		DefaultEnabled: true,
		Module:         accessLogRetentionCleanupJobModule,
		Handler: func(ctx context.Context, configJSON string) (cronx.JobRunResult, error) {
			return cleaner.cleanup(ctx, decodeAccessLogRetentionJobConfig(configJSON))
		},
	})

	return nil
}
