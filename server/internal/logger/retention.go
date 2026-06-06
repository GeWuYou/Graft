package logger

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
	appLogRetentionCleanupJobName           = "logger.app-log-retention-cleanup"
	appLogRetentionCleanupJobModule         = "core.logger"
	appLogRetentionCleanupJobSchedule       = "0 15 17 * * *"
	appLogRetentionCleanupJobDisplayKey     = "scheduledTask.appLogRetention.title"
	appLogRetentionCleanupJobDescriptionKey = "scheduledTask.appLogRetention.description"
	appLogRetentionDefaultBatchSize         = 1000
	hoursPerDay                             = 24
)

const appLogRetentionCleanupConfigSchema = `{"type":"object","properties":{"dryRun":{"type":"boolean","title":"Dry run","description":"Preview cleanup without deleting app logs.","x-title-key":"scheduledTask.appLogRetention.config.dryRun.title","x-description-key":"scheduledTask.appLogRetention.config.dryRun.description"},"batchSize":{"type":"integer","minimum":1,"maximum":10000,"title":"Batch size","description":"Maximum app log rows to delete in one cleanup batch.","x-title-key":"scheduledTask.appLogRetention.config.batchSize.title","x-description-key":"scheduledTask.appLogRetention.config.batchSize.description"}},"additionalProperties":false}`
const appLogRetentionCleanupDefaultConfig = `{"dryRun":false,"batchSize":1000}`

type appLogRetentionJobConfig struct {
	DryRun    bool `json:"dryRun"`
	BatchSize int  `json:"batchSize"`
}

type appLogRetentionPolicy struct {
	retention time.Duration
}

func newAppLogRetentionPolicy(cfg config.LogConfig) (appLogRetentionPolicy, error) {
	retention := cfg.AppLogRetention
	if retention <= 0 {
		return appLogRetentionPolicy{}, errors.New("app log retention must be greater than zero")
	}

	return appLogRetentionPolicy{retention: retention}, nil
}

func (p appLogRetentionPolicy) cutoff(now time.Time) (time.Time, error) {
	if p.retention <= 0 {
		return time.Time{}, errors.New("app log retention must be greater than zero")
	}
	if now.IsZero() {
		return time.Time{}, errors.New("cutoff calculation requires a non-zero current time")
	}

	cutoff := now.UTC().Add(-p.retention)
	if !cutoff.Before(now.UTC()) {
		return time.Time{}, errors.New("app log retention cutoff must be earlier than current time")
	}

	return cutoff, nil
}

type appLogRetentionCleaner struct {
	logger func() *zap.Logger
	appLog func() AppLogger
	repo   AppLogRepository
	policy appLogRetentionPolicy
	now    func() time.Time
}

func newAppLogRetentionCleaner(
	logger *zap.Logger,
	appLogger AppLogger,
	repo AppLogRepository,
	cfg config.LogConfig,
) (*appLogRetentionCleaner, error) {
	policy, err := newAppLogRetentionPolicy(cfg)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, errors.New("app log retention cleaner requires a repository")
	}

	return &appLogRetentionCleaner{
		logger: func() *zap.Logger {
			if logger == nil {
				return zap.NewNop()
			}
			return logger
		},
		appLog: func() AppLogger {
			if appLogger != nil {
				return appLogger.Named("internal.logger.retention")
			}
			return NewAppLogger(logger).Named("internal.logger.retention")
		},
		repo:   repo,
		policy: policy,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

func (c *appLogRetentionCleaner) cleanup(ctx context.Context, config appLogRetentionJobConfig) (cronx.JobRunResult, error) {
	if c == nil {
		err := errors.New("app log retention cleaner is required")
		return appLogRetentionFailureResult(err, time.Time{}, config), err
	}
	started := time.Now()

	cutoff, err := c.policy.cutoff(c.now())
	if err != nil {
		return appLogRetentionFailureResult(err, time.Time{}, config), err
	}
	if cutoff.IsZero() {
		err := errors.New("app log retention cutoff must be non-zero")
		return appLogRetentionFailureResult(err, cutoff, config), err
	}

	logger := c.logger()
	logger.Info("app log retention cleanup started",
		zap.String("job", appLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
	)

	var deleted int64
	if !config.DryRun {
		deleted, err = c.repo.DeleteAppLogsBefore(ctx, cutoff)
	}
	if err != nil {
		logger.Error("app log retention cleanup failed",
			zap.String("job", appLogRetentionCleanupJobName),
			zap.Duration("retention", c.policy.retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		c.appLog().Error(ctx, "scheduler job failed",
			StringField(FieldOperation, "app_log_retention_cleanup"),
			StringField("job", appLogRetentionCleanupJobName),
			DurationField("retention", c.policy.retention),
			TimeField("cutoff", cutoff),
			ErrorField(err),
		)
		wrapped := fmt.Errorf("delete app logs before cutoff: %w", err)
		return appLogRetentionFailureResult(wrapped, cutoff, config), wrapped
	}

	logger.Info("app log retention cleanup completed",
		zap.String("job", appLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)
	c.appLog().Info(ctx, "scheduler job completed",
		StringField(FieldOperation, "app_log_retention_cleanup"),
		StringField("job", appLogRetentionCleanupJobName),
		DurationField("retention", c.policy.retention),
		TimeField("cutoff", cutoff),
		Int64Field("deleted_rows", deleted),
	)

	return appLogRetentionSuccessResult(deleted, c.policy.retention, cutoff, config, started), nil
}

func appLogRetentionSuccessResult(deleted int64, retention time.Duration, cutoff time.Time, config appLogRetentionJobConfig, started time.Time) cronx.JobRunResult {
	durationMS := time.Since(started).Milliseconds()
	retentionDays := int(retention.Hours() / hoursPerDay)
	return cronx.JobRunResult{
		Summary:          fmt.Sprintf("deleted %d rows", deleted),
		Stage:            "completed",
		AffectedResource: "app_log",
		Metrics: map[string]any{
			"deletedCount":  deleted,
			"retentionDays": retentionDays,
			"batchSize":     config.BatchSize,
			"dryRun":        config.DryRun,
			"durationMs":    durationMS,
		},
		Details: map[string]any{
			"operation":     "app_log_retention_cleanup",
			"retentionDays": retentionDays,
			"cutoffTime":    cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     config.BatchSize,
			"dryRun":        config.DryRun,
			"durationMs":    durationMS,
		},
	}
}

func appLogRetentionFailureResult(err error, cutoff time.Time, config appLogRetentionJobConfig) cronx.JobRunResult {
	details := map[string]any{"operation": "app_log_retention_cleanup", "batchSize": config.BatchSize, "dryRun": config.DryRun}
	if !cutoff.IsZero() {
		details["cutoffTime"] = cutoff.UTC().Format(time.RFC3339Nano)
	}
	return cronx.JobRunResult{Summary: err.Error(), Stage: "failed", AffectedResource: "app_log", Details: details, Warnings: []string{err.Error()}}
}

func decodeAppLogRetentionJobConfig(configJSON string) appLogRetentionJobConfig {
	config := appLogRetentionJobConfig{BatchSize: appLogRetentionDefaultBatchSize}
	_ = json.Unmarshal([]byte(configJSON), &config)
	if config.BatchSize <= 0 {
		config.BatchSize = appLogRetentionDefaultBatchSize
	}
	return config
}

// RegisterAppLogRetentionCleanupJob registers the logger-owned app-log cleanup job.
func RegisterAppLogRetentionCleanupJob(
	registry *cronx.Registry,
	logger *zap.Logger,
	appLogger AppLogger,
	repo AppLogRepository,
	cfg config.LogConfig,
) error {
	if registry == nil {
		return errors.New("cron registry is required")
	}

	cleaner, err := newAppLogRetentionCleaner(logger, appLogger, repo, cfg)
	if err != nil {
		return err
	}

	registry.Register(cronx.Job{
		Name:                  appLogRetentionCleanupJobName,
		Key:                   appLogRetentionCleanupJobName,
		Owner:                 appLogRetentionCleanupJobModule,
		Title:                 "App log retention cleanup",
		Description:           "Deletes app logs beyond the configured retention window.",
		DisplayMessageKey:     appLogRetentionCleanupJobDisplayKey,
		DescriptionMessageKey: appLogRetentionCleanupJobDescriptionKey,
		ConfigSchema:          appLogRetentionCleanupConfigSchema,
		DefaultConfig:         appLogRetentionCleanupDefaultConfig,
		Schedule:              appLogRetentionCleanupJobSchedule,
		DefaultEnabled:        true,
		Module:                appLogRetentionCleanupJobModule,
		Handler: func(ctx context.Context, configJSON string) (cronx.JobRunResult, error) {
			return cleaner.cleanup(ctx, decodeAppLogRetentionJobConfig(configJSON))
		},
	})

	return nil
}
