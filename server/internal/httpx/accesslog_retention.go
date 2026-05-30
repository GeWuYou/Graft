package httpx

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
	accessLogRetentionCleanupJobName     = "httpx.access-log-retention-cleanup"
	accessLogRetentionCleanupJobPlugin   = "core.httpx"
	accessLogRetentionCleanupJobSchedule = "0 0 17 * * *"
)

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

func (p accessLogRetentionPolicy) cutoff(now time.Time) (time.Time, error) {
	if p.retention <= 0 {
		return time.Time{}, errors.New("access log retention must be greater than zero")
	}
	if now.IsZero() {
		return time.Time{}, errors.New("cutoff calculation requires a non-zero current time")
	}

	cutoff := now.UTC().Add(-p.retention)
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

func (c *accessLogRetentionCleaner) cleanup(ctx context.Context) (int64, error) {
	if c == nil {
		return 0, errors.New("access log retention cleaner is required")
	}

	cutoff, err := c.policy.cutoff(c.now())
	if err != nil {
		return 0, err
	}
	if cutoff.IsZero() {
		return 0, errors.New("access log retention cutoff must be non-zero")
	}

	logger := c.logger()
	logger.Info("access log retention cleanup started",
		zap.String("job", accessLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
	)

	deleted, err := c.repo.DeleteAccessLogsBefore(ctx, cutoff)
	if err != nil {
		logger.Error("access log retention cleanup failed",
			zap.String("job", accessLogRetentionCleanupJobName),
			zap.Duration("retention", c.policy.retention),
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		return 0, fmt.Errorf("delete access logs before cutoff: %w", err)
	}

	logger.Info("access log retention cleanup completed",
		zap.String("job", accessLogRetentionCleanupJobName),
		zap.Duration("retention", c.policy.retention),
		zap.Time("cutoff", cutoff),
		zap.Int64("deletedRows", deleted),
	)

	return deleted, nil
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
		Name:     accessLogRetentionCleanupJobName,
		Schedule: accessLogRetentionCleanupJobSchedule,
		Plugin:   accessLogRetentionCleanupJobPlugin,
		Run: func(ctx context.Context) error {
			_, runErr := cleaner.cleanup(ctx)
			return runErr
		},
	})

	return nil
}
