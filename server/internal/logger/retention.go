// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/cronx"
	"graft/server/internal/i18n"
)

const (
	appLogRetentionCleanupJobName           = "logger.app-log-retention-cleanup"
	appLogRetentionCleanupJobModule         = "core.logger"
	appLogRetentionCleanupJobSchedule       = "0 15 17 * * *"
	appLogRetentionCleanupJobDisplayKey     = "scheduler.job.appLogRetentionCleanup.title"
	appLogRetentionCleanupJobDescriptionKey = "scheduledTask.appLogRetention.description"
	appLogRetentionDryRunActionKey          = "dryRun"
	appLogRetentionDryRunActionTitleKey     = "scheduledTask.action.dryRun.title"
	appLogRetentionDryRunActionDescKey      = "scheduledTask.action.dryRun.description"
	appLogRetentionDefaultDays              = 30
	appLogRetentionMaxDays                  = 365
	appLogRetentionDefaultBatchSize         = 1000
	appLogRetentionConfigDefinitionOrder    = 220
	appLogRetentionFailureSummary           = "app log retention cleanup failed"
	hoursPerDay                             = 24
)

const (
	appLogRetentionConfigDomain         = "logs"
	appLogRetentionConfigDomainKey      = "systemConfig.domains.logs"
	appLogRetentionConfigGroupKey       = "systemConfig.groups.coreLoggerLogRetention"
	appLogRetentionConfigGroupDescKey   = "systemConfig.groupDescriptions.coreLoggerLogRetention"
	appLogRetentionConfigTitleKey       = "systemConfig.items.appLogRetentionCleanup.title"
	appLogRetentionConfigDescriptionKey = "systemConfig.items.appLogRetentionCleanup.description"
)

const appLogRetentionCleanupConfigSchema = `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":365,"default":30,"title":"Log retention days","description":"Delete logs older than this many days.","x-i18n":{"titleKey":"systemConfig.fields.retentionDays.title","descriptionKey":"systemConfig.fields.retentionDays.description","unitKey":"systemConfig.units.days"}},"batchSize":{"type":"integer","minimum":1,"maximum":10000,"default":1000,"title":"Batch size","description":"Maximum rows deleted per cleanup batch.","x-i18n":{"titleKey":"systemConfig.fields.batchSize.title","descriptionKey":"systemConfig.fields.batchSize.description","unitKey":"systemConfig.units.rows"}}},"additionalProperties":false}`
const appLogRetentionCleanupDefaultConfig = `{"retentionDays":30,"batchSize":1000}`

type appLogRetentionJobConfig struct {
	RetentionDays int  `json:"retentionDays"`
	DryRun        bool `json:"-"`
	BatchSize     int  `json:"batchSize"`
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
	retention := c.retentionDuration(config)

	cutoff, err := appLogRetentionCutoff(c.now(), retention)
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
		zap.Duration("retention", retention),
		zap.Time("cutoff", cutoff),
	)

	deleted, err := c.repo.DeleteAppLogsBeforeLimit(ctx, cutoff, normalizedAppLogRetentionBatchSize(config))
	if err != nil {
		logger.Error("app log retention cleanup failed",
			zap.String("job", appLogRetentionCleanupJobName),
			zap.Duration("retention", retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		c.appLog().Error(ctx, "scheduler job failed",
			StringField(FieldOperation, "app_log_retention_cleanup"),
			StringField("job", appLogRetentionCleanupJobName),
			DurationField("retention", retention),
			TimeField("cutoff", cutoff),
			ErrorField(err),
		)
		wrapped := fmt.Errorf("delete app logs before cutoff: %w", err)
		return appLogRetentionFailureResult(wrapped, cutoff, config), wrapped
	}

	logger.Info("app log retention cleanup completed",
		zap.String("job", appLogRetentionCleanupJobName),
		zap.Duration("retention", retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)
	c.appLog().Info(ctx, "scheduler job completed",
		StringField(FieldOperation, "app_log_retention_cleanup"),
		StringField("job", appLogRetentionCleanupJobName),
		DurationField("retention", retention),
		TimeField("cutoff", cutoff),
		Int64Field("deleted_rows", deleted),
	)

	return appLogRetentionSuccessResult(deleted, retention, cutoff, config, started), nil
}

func appLogRetentionCutoff(now time.Time, retention time.Duration) (time.Time, error) {
	if retention <= 0 {
		return time.Time{}, errors.New("app log retention must be greater than zero")
	}
	if now.IsZero() {
		return time.Time{}, errors.New("cutoff calculation requires a non-zero current time")
	}

	cutoff := now.UTC().Add(-retention)
	if !cutoff.Before(now.UTC()) {
		return time.Time{}, errors.New("app log retention cutoff must be earlier than current time")
	}

	return cutoff, nil
}

func (c *appLogRetentionCleaner) estimate(ctx context.Context, config appLogRetentionJobConfig) (cronx.JobRunResult, error) {
	if c == nil {
		err := errors.New("app log retention cleaner is required")
		return appLogRetentionFailureResult(err, time.Time{}, config), err
	}
	started := time.Now()
	retention := c.retentionDuration(config)

	cutoff, err := appLogRetentionCutoff(c.now(), retention)
	if err != nil {
		return appLogRetentionFailureResult(err, time.Time{}, config), err
	}
	if cutoff.IsZero() {
		err := errors.New("app log retention cutoff must be non-zero")
		return appLogRetentionFailureResult(err, cutoff, config), err
	}

	matched, total, err := c.estimateCounts(ctx, cutoff)
	if err != nil {
		wrapped := fmt.Errorf("estimate app logs before cutoff: %w", err)
		return appLogRetentionFailureResult(wrapped, cutoff, config), wrapped
	}
	retained := total - matched
	if retained < 0 {
		retained = 0
	}

	return appLogRetentionEstimateResult(matched, retained, retention, cutoff, config, started), nil
}

func (c *appLogRetentionCleaner) estimateCounts(ctx context.Context, cutoff time.Time) (int64, int64, error) {
	matched, err := c.repo.ListAppLogs(ctx, AppLogListQuery{
		Page:           1,
		PageSize:       1,
		OccurredBefore: &cutoff,
	})
	if err != nil {
		return 0, 0, err
	}
	total, err := c.repo.ListAppLogs(ctx, AppLogListQuery{Page: 1, PageSize: 1})
	if err != nil {
		return 0, 0, err
	}
	return matched.Total, total.Total, nil
}

func (c *appLogRetentionCleaner) retentionDuration(config appLogRetentionJobConfig) time.Duration {
	if config.RetentionDays > 0 {
		return time.Duration(config.RetentionDays) * hoursPerDay * time.Hour
	}
	return c.policy.retention
}

func normalizedAppLogRetentionBatchSize(config appLogRetentionJobConfig) int {
	if config.BatchSize > 0 {
		return config.BatchSize
	}
	return appLogRetentionDefaultBatchSize
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
			"batchSize":     normalizedAppLogRetentionBatchSize(config),
			"durationMs":    durationMS,
		},
		Details: map[string]any{
			"operation":     "app_log_retention_cleanup",
			"retentionDays": retentionDays,
			"cutoffTime":    cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     normalizedAppLogRetentionBatchSize(config),
			"durationMs":    durationMS,
		},
	}
}

func appLogRetentionEstimateResult(matched int64, retained int64, retention time.Duration, cutoff time.Time, config appLogRetentionJobConfig, started time.Time) cronx.JobRunResult {
	durationMS := time.Since(started).Milliseconds()
	retentionDays := int(retention.Hours() / hoursPerDay)
	batchSize := normalizedAppLogRetentionBatchSize(config)
	estimatedDeleteCount := matched
	if estimatedDeleteCount > int64(batchSize) {
		estimatedDeleteCount = int64(batchSize)
	}
	return cronx.JobRunResult{
		Summary:          fmt.Sprintf("estimated %d rows eligible for deletion", matched),
		Stage:            "estimated",
		AffectedResource: "app_log",
		Metrics: map[string]any{
			"estimatedScanCount":   matched,
			"estimatedDeleteCount": estimatedDeleteCount,
			"estimatedRetainCount": retained,
			"retentionDays":        retentionDays,
			"batchSize":            batchSize,
			"durationMs":           durationMS,
		},
		Details: map[string]any{
			"operation":     "app_log_retention_cleanup_estimate",
			"retentionDays": retentionDays,
			"cutoffTime":    cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     batchSize,
			"durationMs":    durationMS,
		},
	}
}

func appLogRetentionFailureResult(_ error, cutoff time.Time, config appLogRetentionJobConfig) cronx.JobRunResult {
	details := map[string]any{"operation": "app_log_retention_cleanup", "batchSize": normalizedAppLogRetentionBatchSize(config)}
	if !cutoff.IsZero() {
		details["cutoffTime"] = cutoff.UTC().Format(time.RFC3339Nano)
	}
	return cronx.JobRunResult{Summary: appLogRetentionFailureSummary, Stage: "failed", AffectedResource: "app_log", Details: details, Warnings: []string{appLogRetentionFailureSummary}}
}

func decodeAppLogRetentionJobConfig(configJSON string) appLogRetentionJobConfig {
	config := appLogRetentionJobConfig{RetentionDays: appLogRetentionDefaultDays, BatchSize: appLogRetentionDefaultBatchSize}
	_ = json.Unmarshal([]byte(configJSON), &config)
	if config.RetentionDays <= 0 {
		config.RetentionDays = appLogRetentionDefaultDays
	}
	if config.RetentionDays > appLogRetentionMaxDays {
		config.RetentionDays = appLogRetentionMaxDays
	}
	if config.BatchSize <= 0 {
		config.BatchSize = appLogRetentionDefaultBatchSize
	}
	return config
}

// RegisterAppLogRetentionConfigDefinition exposes the built-in cleanup defaults as registry authority.
func RegisterAppLogRetentionConfigDefinition(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}

	return registry.Register(configregistry.Definition{
		Key:                 appLogRetentionCleanupJobName,
		Module:              appLogRetentionCleanupJobModule,
		Domain:              appLogRetentionConfigDomain,
		DomainKey:           appLogRetentionConfigDomainKey,
		DomainLabel:         "Logs",
		Group:               "log.retention",
		GroupKey:            appLogRetentionConfigGroupKey,
		GroupLabel:          "App log retention",
		GroupDescription:    "Manage application log cleanup retention and batch policy.",
		GroupDescriptionKey: appLogRetentionConfigGroupDescKey,
		Title:               "App log retention cleanup",
		TitleKey:            appLogRetentionConfigTitleKey,
		Description:         "Default cleanup configuration for app-log retention jobs.",
		DescriptionKey:      appLogRetentionConfigDescriptionKey,
		Tags:                []string{"logger", "log.retention"},
		Type:                configregistry.ValueTypeObject,
		Schema:              json.RawMessage(appLogRetentionCleanupConfigSchema),
		DefaultValue:        json.RawMessage(appLogRetentionCleanupDefaultConfig),
		Order:               appLogRetentionConfigDefinitionOrder,
	})
}

// RegisterAppLogRetentionConfigMessages registers system-config display metadata for the logger-owned cleanup config.
func RegisterAppLogRetentionConfigMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is required")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(appLogRetentionConfigGroupKey), Text: "应用日志保留"},
				{Key: i18n.MessageKey(appLogRetentionConfigGroupDescKey), Text: "管理应用日志清理的保留周期与批量策略。"},
				{Key: i18n.MessageKey(appLogRetentionConfigTitleKey), Text: "应用日志保留清理"},
				{Key: i18n.MessageKey(appLogRetentionConfigDescriptionKey), Text: "应用日志保留清理任务的默认配置。"},
			},
		},
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(appLogRetentionConfigGroupKey), Text: "App Log Retention"},
				{Key: i18n.MessageKey(appLogRetentionConfigGroupDescKey), Text: "Manage application log cleanup retention and batch policy."},
				{Key: i18n.MessageKey(appLogRetentionConfigTitleKey), Text: "App log retention cleanup"},
				{Key: i18n.MessageKey(appLogRetentionConfigDescriptionKey), Text: "Default cleanup configuration for app-log retention jobs."},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register app-log retention config messages: %w", err)
		}
	}
	return nil
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
		Title:                 "Application log retention cleanup",
		TitleKey:              appLogRetentionCleanupJobDisplayKey,
		Description:           "Deletes app logs beyond the configured retention window.",
		DescriptionKey:        appLogRetentionCleanupJobDescriptionKey,
		DisplayMessageKey:     appLogRetentionCleanupJobDisplayKey,
		DescriptionMessageKey: appLogRetentionCleanupJobDescriptionKey,
		ConfigSchema:          appLogRetentionCleanupConfigSchema,
		DefaultConfig:         appLogRetentionCleanupDefaultConfig,
		DefaultConfigKey:      appLogRetentionCleanupJobName,
		Actions: []cronx.JobAction{
			{
				Key:            appLogRetentionDryRunActionKey,
				TitleKey:       appLogRetentionDryRunActionTitleKey,
				Title:          "Dry run",
				DescriptionKey: appLogRetentionDryRunActionDescKey,
				Description:    "Preview cleanup result",
				Handler: func(ctx context.Context, configJSON string) (cronx.JobRunResult, error) {
					return cleaner.estimate(ctx, decodeAppLogRetentionJobConfig(configJSON))
				},
			},
		},
		Schedule:       appLogRetentionCleanupJobSchedule,
		DefaultEnabled: true,
		Module:         appLogRetentionCleanupJobModule,
		Handler: func(ctx context.Context, configJSON string) (cronx.JobRunResult, error) {
			return cleaner.cleanup(ctx, decodeAppLogRetentionJobConfig(configJSON))
		},
	})

	return nil
}
