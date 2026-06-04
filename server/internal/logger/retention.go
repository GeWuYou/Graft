package logger

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
	appLogRetentionCleanupJobName     = "logger.app-log-retention-cleanup"
	appLogRetentionCleanupJobModule   = "core.logger"
	appLogRetentionCleanupJobSchedule = "0 15 17 * * *"
)

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
	repo   AppLogRepository
	policy appLogRetentionPolicy
	now    func() time.Time
}

func newAppLogRetentionCleaner(
	logger *zap.Logger,
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
		repo:   repo,
		policy: policy,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

func (c *appLogRetentionCleaner) cleanup(ctx context.Context) (int64, error) {
	if c == nil {
		return 0, errors.New("app log retention cleaner is required")
	}

	cutoff, err := c.policy.cutoff(c.now())
	if err != nil {
		return 0, err
	}
	if cutoff.IsZero() {
		return 0, errors.New("app log retention cutoff must be non-zero")
	}

	logger := c.logger()
	logger.Info("app log retention cleanup started",
		zap.String("job", appLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
	)

	deleted, err := c.repo.DeleteAppLogsBefore(ctx, cutoff)
	if err != nil {
		logger.Error("app log retention cleanup failed",
			zap.String("job", appLogRetentionCleanupJobName),
			zap.Duration("retention", c.policy.retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		return 0, fmt.Errorf("delete app logs before cutoff: %w", err)
	}

	logger.Info("app log retention cleanup completed",
		zap.String("job", appLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)

	return deleted, nil
}

// RegisterAppLogRetentionCleanupJob registers the logger-owned app-log cleanup job.
func RegisterAppLogRetentionCleanupJob(
	registry *cronx.Registry,
	logger *zap.Logger,
	repo AppLogRepository,
	cfg config.LogConfig,
) error {
	if registry == nil {
		return errors.New("cron registry is required")
	}

	cleaner, err := newAppLogRetentionCleaner(logger, repo, cfg)
	if err != nil {
		return err
	}

	registry.Register(cronx.Job{
		Name:     appLogRetentionCleanupJobName,
		Schedule: appLogRetentionCleanupJobSchedule,
		Module:   appLogRetentionCleanupJobModule,
		Run: func(ctx context.Context) error {
			_, runErr := cleaner.cleanup(ctx)
			return runErr
		},
	})

	return nil
}
