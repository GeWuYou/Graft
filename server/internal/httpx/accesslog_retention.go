// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package httpx

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
	accessLogRetentionCleanupJobName           = "httpx.access-log-retention-cleanup"
	accessLogRetentionCleanupJobModule         = "core.httpx"
	accessLogRetentionCleanupJobSchedule       = "0 0 17 * * *"
	accessLogRetentionCleanupJobDisplayKey     = "scheduledTask.accessLogRetention.title"
	accessLogRetentionCleanupJobDescriptionKey = "scheduledTask.accessLogRetention.description"
	accessLogRetentionDryRunActionKey          = "dryRun"
	accessLogRetentionDryRunActionTitleKey     = "scheduledTask.action.dryRun.title"
	accessLogRetentionDryRunActionDescKey      = "scheduledTask.action.dryRun.description"
	accessLogRetentionDefaultDays              = 30
	accessLogRetentionMaxDays                  = 365
	accessLogRetentionDefaultBatchSize         = 1000
	accessLogRetentionConfigDefinitionOrder    = 210
	hoursPerDay                                = 24
)

const (
	accessLogRetentionConfigDomain         = "logs"
	accessLogRetentionConfigDomainKey      = "systemConfig.domains.logs"
	accessLogRetentionConfigGroupKey       = "systemConfig.groups.coreHttpxLogRetention"
	accessLogRetentionConfigGroupDescKey   = "systemConfig.groupDescriptions.coreHttpxLogRetention"
	accessLogRetentionConfigTitleKey       = "systemConfig.items.accessLogRetentionCleanup.title"
	accessLogRetentionConfigDescriptionKey = "systemConfig.items.accessLogRetentionCleanup.description"
)

const accessLogRetentionCleanupConfigSchema = `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":365,"default":30,"title":"Log retention days","description":"Delete logs older than this many days.","x-i18n":{"titleKey":"systemConfig.fields.retentionDays.title","descriptionKey":"systemConfig.fields.retentionDays.description","unitKey":"systemConfig.units.days"}},"batchSize":{"type":"integer","minimum":1,"maximum":10000,"default":1000,"title":"Batch size","description":"Maximum rows deleted per cleanup batch.","x-i18n":{"titleKey":"systemConfig.fields.batchSize.title","descriptionKey":"systemConfig.fields.batchSize.description","unitKey":"systemConfig.units.rows"}}},"additionalProperties":false}`
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

	deleted, err := c.repo.DeleteAccessLogsBeforeLimit(ctx, cutoff, normalizedAccessLogRetentionBatchSize(config))
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

func normalizedAccessLogRetentionBatchSize(config accessLogRetentionJobConfig) int {
	if config.BatchSize > 0 {
		return config.BatchSize
	}
	return accessLogRetentionDefaultBatchSize
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
			"batchSize":     normalizedAccessLogRetentionBatchSize(config),
			"durationMs":    durationMS,
		},
		Details: map[string]any{
			"operation":     "access_log_retention_cleanup",
			"retentionDays": retentionDays,
			"cutoffTime":    cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     normalizedAccessLogRetentionBatchSize(config),
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
			"batchSize":            normalizedAccessLogRetentionBatchSize(config),
			"durationMs":           durationMS,
		},
		Details: map[string]any{
			"operation":     "access_log_retention_cleanup_estimate",
			"retentionDays": retentionDays,
			"cutoffTime":    cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     normalizedAccessLogRetentionBatchSize(config),
			"durationMs":    durationMS,
		},
	}
}

func accessLogRetentionFailureResult(err error, cutoff time.Time, config accessLogRetentionJobConfig) cronx.JobRunResult {
	details := map[string]any{"operation": "access_log_retention_cleanup", "batchSize": normalizedAccessLogRetentionBatchSize(config)}
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
	if config.RetentionDays > accessLogRetentionMaxDays {
		config.RetentionDays = accessLogRetentionMaxDays
	}
	if config.BatchSize <= 0 {
		config.BatchSize = accessLogRetentionDefaultBatchSize
	}
	return config
}

// RegisterAccessLogRetentionConfigDefinition exposes the built-in cleanup defaults as registry authority.
func RegisterAccessLogRetentionConfigDefinition(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}

	return registry.Register(configregistry.Definition{
		Key:                 accessLogRetentionCleanupJobName,
		Module:              accessLogRetentionCleanupJobModule,
		Domain:              accessLogRetentionConfigDomain,
		DomainKey:           accessLogRetentionConfigDomainKey,
		DomainLabel:         "Logs",
		Group:               "log.retention",
		GroupKey:            accessLogRetentionConfigGroupKey,
		GroupLabel:          "Access log retention",
		GroupDescription:    "Manage access log cleanup retention and batch policy.",
		GroupDescriptionKey: accessLogRetentionConfigGroupDescKey,
		Title:               "Access log retention cleanup",
		TitleKey:            accessLogRetentionConfigTitleKey,
		Description:         "Default cleanup configuration for access-log retention jobs.",
		DescriptionKey:      accessLogRetentionConfigDescriptionKey,
		Tags:                []string{"httpx", "log.retention"},
		Type:                configregistry.ValueTypeObject,
		Schema:              json.RawMessage(accessLogRetentionCleanupConfigSchema),
		DefaultValue:        json.RawMessage(accessLogRetentionCleanupDefaultConfig),
		Order:               accessLogRetentionConfigDefinitionOrder,
	})
}

// RegisterAccessLogRetentionConfigMessages registers system-config display metadata for the httpx-owned cleanup config.
func RegisterAccessLogRetentionConfigMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is required")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(accessLogRetentionConfigGroupKey), Text: "访问日志保留"},
				{Key: i18n.MessageKey(accessLogRetentionConfigGroupDescKey), Text: "管理访问日志清理的保留周期与批量策略。"},
				{Key: i18n.MessageKey(accessLogRetentionConfigTitleKey), Text: "访问日志保留清理"},
				{Key: i18n.MessageKey(accessLogRetentionConfigDescriptionKey), Text: "访问日志保留清理任务的默认配置。"},
			},
		},
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(accessLogRetentionConfigGroupKey), Text: "Access Log Retention"},
				{Key: i18n.MessageKey(accessLogRetentionConfigGroupDescKey), Text: "Manage access log cleanup retention and batch policy."},
				{Key: i18n.MessageKey(accessLogRetentionConfigTitleKey), Text: "Access log retention cleanup"},
				{Key: i18n.MessageKey(accessLogRetentionConfigDescriptionKey), Text: "Default cleanup configuration for access-log retention jobs."},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register access-log retention config messages: %w", err)
		}
	}
	return nil
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
		DefaultConfigKey:      accessLogRetentionCleanupJobName,
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
