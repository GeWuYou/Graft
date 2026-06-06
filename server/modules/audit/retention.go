package audit

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
	auditLogRetentionCleanupJobName           = "audit.audit-log-retention-cleanup"
	auditLogRetentionCleanupJobSchedule       = "0 30 17 * * *"
	auditLogRetentionCleanupJobDisplayKey     = "scheduledTask.auditLogRetention.title"
	auditLogRetentionCleanupJobDescriptionKey = "scheduledTask.auditLogRetention.description"
	auditLogRetentionDefaultBatchSize         = 1000
	hoursPerDay                               = 24
)

const auditLogRetentionCleanupConfigSchema = `{"type":"object","properties":{"dryRun":{"type":"boolean","title":"Dry run","description":"Preview cleanup without deleting audit logs.","x-title-key":"scheduledTask.auditLogRetention.config.dryRun.title","x-description-key":"scheduledTask.auditLogRetention.config.dryRun.description"},"batchSize":{"type":"integer","minimum":1,"maximum":10000,"title":"Batch size","description":"Maximum audit log rows to delete in one cleanup batch.","x-title-key":"scheduledTask.auditLogRetention.config.batchSize.title","x-description-key":"scheduledTask.auditLogRetention.config.batchSize.description"}},"additionalProperties":false}`
const auditLogRetentionCleanupDefaultConfig = `{"dryRun":false,"batchSize":1000}`

type retentionJobConfig struct {
	DryRun    bool `json:"dryRun"`
	BatchSize int  `json:"batchSize"`
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

func (p auditLogRetentionPolicy) cutoff(now time.Time) (time.Time, error) {
	if p.retention <= 0 {
		return time.Time{}, errors.New("audit log retention must be greater than zero")
	}
	if now.IsZero() {
		return time.Time{}, errors.New("cutoff calculation requires a non-zero current time")
	}

	cutoff := now.UTC().Add(-p.retention)
	if !cutoff.Before(now.UTC()) {
		return time.Time{}, errors.New("audit log retention cutoff must be earlier than current time")
	}

	return cutoff, nil
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

	cutoff, err := c.policy.cutoff(c.now())
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
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
	)

	var deleted int64
	if !config.DryRun {
		deleted, err = c.service.DeleteBefore(ctx, cutoff)
	}
	if err != nil {
		logger.Error("audit log retention cleanup failed",
			zap.String("job", auditLogRetentionCleanupJobName),
			zap.Duration("retention", c.policy.retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		wrapped := fmt.Errorf("delete audit logs before cutoff: %w", err)
		return cleanupFailureResult("audit_log_retention_cleanup", wrapped, cutoff, config), wrapped
	}

	logger.Info("audit log retention cleanup completed",
		zap.String("job", auditLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)

	return cleanupSuccessResult(cleanupSuccessInput{
		operation: "audit_log_retention_cleanup",
		resource:  "audit_log",
		deleted:   deleted,
		retention: c.policy.retention,
		cutoff:    cutoff,
		config:    config,
		started:   started,
	}), nil
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
			"dryRun":        input.config.DryRun,
			"durationMs":    durationMS,
		},
		Details: map[string]any{
			"operation":     input.operation,
			"retentionDays": retentionDays,
			"cutoffTime":    input.cutoff.UTC().Format(time.RFC3339Nano),
			"batchSize":     input.config.BatchSize,
			"dryRun":        input.config.DryRun,
			"durationMs":    durationMS,
		},
	}
}

func cleanupFailureResult(operation string, err error, cutoff time.Time, config retentionJobConfig) cronx.JobRunResult {
	details := map[string]any{
		"operation": operation,
		"batchSize": config.BatchSize,
		"dryRun":    config.DryRun,
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
	config := retentionJobConfig{DryRun: false, BatchSize: auditLogRetentionDefaultBatchSize}
	_ = json.Unmarshal([]byte(configJSON), &config)
	if config.BatchSize <= 0 {
		config.BatchSize = auditLogRetentionDefaultBatchSize
	}
	return config
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
				Key:             "dry-run",
				Title:           "Dry run",
				Description:     "Preview audit log retention cleanup without deleting audit logs.",
				ConfigOverrides: `{"dryRun":true}`,
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
