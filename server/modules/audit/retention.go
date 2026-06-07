package audit

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
)

const (
	auditLogRetentionCleanupJobName           = "audit.audit-log-retention-cleanup"
	auditLogRetentionCleanupJobSchedule       = "0 30 17 * * *"
	auditLogRetentionCleanupJobDisplayKey     = "scheduledTask.auditLogRetention.title"
	auditLogRetentionCleanupJobDescriptionKey = "scheduledTask.auditLogRetention.description"
	auditLogRetentionDryRunActionKey          = "dryRun"
	auditLogRetentionDryRunActionTitleKey     = "scheduledTask.action.dryRun.title"
	auditLogRetentionDryRunActionDescKey      = "scheduledTask.action.dryRun.description"
	auditLogRetentionDefaultDays              = 30
	auditLogRetentionDefaultBatchSize         = 1000
	auditLogRetentionConfigDefinitionOrder    = 230
	hoursPerDay                               = 24
)

const auditLogRetentionCleanupConfigSchema = `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650,"default":30,"title":"Retention days","description":"Delete audit logs older than this number of days.","x-title-key":"scheduledTask.auditLogRetention.config.retentionDays.title","x-description-key":"scheduledTask.auditLogRetention.config.retentionDays.description"},"batchSize":{"type":"integer","minimum":1,"maximum":10000,"default":1000,"title":"Batch size","description":"Maximum audit log rows to delete in one cleanup batch.","x-title-key":"scheduledTask.auditLogRetention.config.batchSize.title","x-description-key":"scheduledTask.auditLogRetention.config.batchSize.description"}},"additionalProperties":false}`
const auditLogRetentionCleanupDefaultConfig = `{"retentionDays":30,"batchSize":1000}`

type retentionJobConfig struct {
	RetentionDays int  `json:"retentionDays"`
	DryRun        bool `json:"-"`
	BatchSize     int  `json:"batchSize"`
}

type auditLogRetentionPolicy struct {
	retention time.Duration
}

func newAuditLogRetentionPolicy(cfg config.AuditConfig) (auditLogRetentionPolicy, error) {
	retention := cfg.LogRetention
	if retention <= 0 {
		return auditLogRetentionPolicy{}, errors.New("audit log retention must be greater than zero")
	}

	return auditLogRetentionPolicy{retention: retention}, nil
}

type auditLogRetentionCleaner struct {
	logger  func() *zap.Logger
	service *Service
	policy  auditLogRetentionPolicy
	now     func() time.Time
}

func newAuditLogRetentionCleaner(
	logger *zap.Logger,
	service *Service,
	cfg config.AuditConfig,
) (*auditLogRetentionCleaner, error) {
	policy, err := newAuditLogRetentionPolicy(cfg)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, errors.New("audit log retention cleaner requires a service")
	}

	return &auditLogRetentionCleaner{
		logger: func() *zap.Logger {
			if logger == nil {
				return zap.NewNop()
			}
			return logger
		},
		service: service,
		policy:  policy,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

func (c *auditLogRetentionCleaner) cleanup(ctx context.Context, config retentionJobConfig) (cronx.JobRunResult, error) {
	if c == nil {
		err := errors.New("audit log retention cleaner is required")
		return cleanupFailureResult("audit_log_retention_cleanup", err, time.Time{}, retentionJobConfig{}), err
	}
	started := time.Now()
	retention := c.retentionDuration(config)

	cutoff, err := auditLogRetentionCutoff(c.now(), retention)
	if err != nil {
		return cleanupFailureResult("audit_log_retention_cleanup", err, time.Time{}, retentionJobConfig{}), err
	}
	if cutoff.IsZero() {
		err := errors.New("audit log retention cutoff must be non-zero")
		return cleanupFailureResult("audit_log_retention_cleanup", err, cutoff, retentionJobConfig{}), err
	}
	logger := c.logger()
	logger.Info("audit log retention cleanup started",
		zap.String("job", auditLogRetentionCleanupJobName),
		zap.Duration("retention", retention),
		zap.Time("cutoff", cutoff),
	)

	var deleted int64
	if !config.DryRun {
		deleted, err = c.service.DeleteBefore(ctx, cutoff)
	}
	if err != nil {
		logger.Error("audit log retention cleanup failed",
			zap.String("job", auditLogRetentionCleanupJobName),
			zap.Duration("retention", retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		wrapped := fmt.Errorf("delete audit logs before cutoff: %w", err)
		return cleanupFailureResult("audit_log_retention_cleanup", wrapped, cutoff, config), wrapped
	}

	logger.Info("audit log retention cleanup completed",
		zap.String("job", auditLogRetentionCleanupJobName),
		zap.Duration("retention", retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)

	return cleanupSuccessResult(cleanupSuccessInput{
		operation: "audit_log_retention_cleanup",
		resource:  "audit_log",
		deleted:   deleted,
		retention: retention,
		cutoff:    cutoff,
		config:    config,
		started:   started,
	}), nil
}

func auditLogRetentionCutoff(now time.Time, retention time.Duration) (time.Time, error) {
	if retention <= 0 {
		return time.Time{}, errors.New("audit log retention must be greater than zero")
	}
	if now.IsZero() {
		return time.Time{}, errors.New("cutoff calculation requires a non-zero current time")
	}

	cutoff := now.UTC().Add(-retention)
	if !cutoff.Before(now.UTC()) {
		return time.Time{}, errors.New("audit log retention cutoff must be earlier than current time")
	}

	return cutoff, nil
}

func (c *auditLogRetentionCleaner) estimate(ctx context.Context, config retentionJobConfig) (cronx.JobRunResult, error) {
	config.DryRun = true
	return c.cleanup(ctx, config)
}

func (c *auditLogRetentionCleaner) retentionDuration(config retentionJobConfig) time.Duration {
	if config.RetentionDays > 0 {
		return time.Duration(config.RetentionDays) * hoursPerDay * time.Hour
	}
	return c.policy.retention
}

type cleanupSuccessInput struct {
	operation string
	resource  string
	deleted   int64
	retention time.Duration
	cutoff    time.Time
	config    retentionJobConfig
	started   time.Time
}

func cleanupSuccessResult(input cleanupSuccessInput) cronx.JobRunResult {
	durationMS := time.Since(input.started).Milliseconds()
	retentionDays := int(input.retention.Hours() / hoursPerDay)
	return cronx.JobRunResult{
		Summary:          fmt.Sprintf("deleted %d rows", input.deleted),
		Stage:            "completed",
		AffectedResource: input.resource,
		Metrics: map[string]any{
			"deletedCount":  input.deleted,
			"retentionDays": retentionDays,
			"batchSize":     input.config.BatchSize,
			"durationMs":    durationMS,
		},
		Details: map[string]any{
			"operation":     input.operation,
			"retentionDays": retentionDays,
			"cutoffTime":    input.cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     input.config.BatchSize,
			"durationMs":    durationMS,
		},
	}
}

func cleanupFailureResult(operation string, err error, cutoff time.Time, config retentionJobConfig) cronx.JobRunResult {
	details := map[string]any{
		"operation": operation,
		"batchSize": config.BatchSize,
	}
	if !cutoff.IsZero() {
		details["cutoffTime"] = cutoff.UTC().Format(time.RFC3339Nano)
	}
	return cronx.JobRunResult{
		Summary: err.Error(),
		Stage:   "failed",
		Details: details,
		Warnings: []string{
			err.Error(),
		},
	}
}

func decodeRetentionJobConfig(configJSON string) retentionJobConfig {
	config := retentionJobConfig{RetentionDays: auditLogRetentionDefaultDays, BatchSize: auditLogRetentionDefaultBatchSize}
	_ = json.Unmarshal([]byte(configJSON), &config)
	if config.RetentionDays <= 0 {
		config.RetentionDays = auditLogRetentionDefaultDays
	}
	if config.BatchSize <= 0 {
		config.BatchSize = auditLogRetentionDefaultBatchSize
	}
	return config
}

func registerAuditLogRetentionConfigDefinition(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}

	return registry.Register(configregistry.Definition{
		Key:          auditLogRetentionCleanupJobName,
		Module:       moduleID,
		Group:        "log.retention",
		Title:        "Audit log retention cleanup",
		Description:  "Default cleanup configuration for audit-log retention jobs.",
		Type:         configregistry.ValueTypeObject,
		Schema:       json.RawMessage(auditLogRetentionCleanupConfigSchema),
		DefaultValue: json.RawMessage(auditLogRetentionCleanupDefaultConfig),
		Order:        auditLogRetentionConfigDefinitionOrder,
	})
}

func registerAuditLogRetentionCleanupJob(
	registry *cronx.Registry,
	logger *zap.Logger,
	service *Service,
	cfg config.AuditConfig,
) error {
	if registry == nil {
		return errors.New("cron registry is required")
	}

	cleaner, err := newAuditLogRetentionCleaner(logger, service, cfg)
	if err != nil {
		return err
	}

	registry.Register(cronx.Job{
		Name:                  auditLogRetentionCleanupJobName,
		Key:                   auditLogRetentionCleanupJobName,
		Owner:                 moduleID,
		Title:                 "Audit log retention cleanup",
		Description:           "Deletes audit logs beyond the configured retention window.",
		DisplayMessageKey:     auditLogRetentionCleanupJobDisplayKey,
		DescriptionMessageKey: auditLogRetentionCleanupJobDescriptionKey,
		ConfigSchema:          auditLogRetentionCleanupConfigSchema,
		DefaultConfig:         auditLogRetentionCleanupDefaultConfig,
		Actions: []cronx.JobAction{
			{
				Key:            auditLogRetentionDryRunActionKey,
				TitleKey:       auditLogRetentionDryRunActionTitleKey,
				Title:          "Dry run",
				DescriptionKey: auditLogRetentionDryRunActionDescKey,
				Description:    "Preview cleanup result",
				Handler: func(ctx context.Context, configJSON string) (cronx.JobRunResult, error) {
					return cleaner.estimate(ctx, decodeRetentionJobConfig(configJSON))
				},
			},
		},
		Schedule:       auditLogRetentionCleanupJobSchedule,
		DefaultEnabled: true,
		Module:         moduleID,
		Handler: func(ctx context.Context, configJSON string) (cronx.JobRunResult, error) {
			return cleaner.cleanup(ctx, decodeRetentionJobConfig(configJSON))
		},
	})

	return nil
}
