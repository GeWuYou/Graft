package audit

import (
	"context"
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
)

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

func (c *auditLogRetentionCleaner) cleanup(ctx context.Context) (int64, error) {
	if c == nil {
		return 0, errors.New("audit log retention cleaner is required")
	}

	cutoff, err := c.policy.cutoff(c.now())
	if err != nil {
		return 0, err
	}
	if cutoff.IsZero() {
		return 0, errors.New("audit log retention cutoff must be non-zero")
	}

	logger := c.logger()
	logger.Info("audit log retention cleanup started",
		zap.String("job", auditLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
	)

	deleted, err := c.service.DeleteBefore(ctx, cutoff)
	if err != nil {
		logger.Error("audit log retention cleanup failed",
			zap.String("job", auditLogRetentionCleanupJobName),
			zap.Duration("retention", c.policy.retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		return 0, fmt.Errorf("delete audit logs before cutoff: %w", err)
	}

	logger.Info("audit log retention cleanup completed",
		zap.String("job", auditLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)

	return deleted, nil
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
		Schedule:              auditLogRetentionCleanupJobSchedule,
		DefaultEnabled:        true,
		Module:                moduleID,
		Run: func(ctx context.Context) error {
			_, runErr := cleaner.cleanup(ctx)
			return runErr
		},
	})

	return nil
}
