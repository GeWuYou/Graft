// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package statex

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// TimeSeriesStore stores timestamp-scored serialized state entries.
type TimeSeriesStore interface {
	Append(context.Context, string, TimeSeriesSample, RetentionPolicy) error
	Range(context.Context, string, TimeSeriesQuery) ([]TimeSeriesSample, error)
}

// TimeSeriesSample is one serialized authority-state sample.
type TimeSeriesSample struct {
	ObservedAt time.Time
	Payload    []byte
}

// TimeSeriesQuery describes one score/time window query.
type TimeSeriesQuery struct {
	StartAt time.Time
	EndAt   time.Time
}

// RetentionPolicy trims old samples and sets TTL for the whole series key.
type RetentionPolicy struct {
	TrimBefore   time.Time
	ExpiresAfter time.Duration
}

type redisTimeSeriesStore struct {
	client redis.Cmdable
}

// NewRedisTimeSeriesStore adapts a Redis client to the mechanical time-series contract.
func NewRedisTimeSeriesStore(client redis.Cmdable) (TimeSeriesStore, error) {
	if client == nil {
		return nil, errors.New("statex redis client is required")
	}

	return &redisTimeSeriesStore{client: client}, nil
}

// Append stores one serialized sample under the given key using the sample time as the score.
func (s *redisTimeSeriesStore) Append(
	ctx context.Context,
	key string,
	sample TimeSeriesSample,
	policy RetentionPolicy,
) error {
	if err := validateKey(key); err != nil {
		return err
	}

	observedAt := sample.ObservedAt.UTC()
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(observedAt.UnixMilli()),
		Member: string(sample.Payload),
	})
	if !policy.TrimBefore.IsZero() {
		pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(policy.TrimBefore.UTC().UnixMilli(), 10))
	}
	if policy.ExpiresAfter > 0 {
		pipe.Expire(ctx, key, policy.ExpiresAfter)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("append time-series sample: %w", err)
	}

	return nil
}

// Range returns serialized samples ordered by score within the requested time window.
func (s *redisTimeSeriesStore) Range(ctx context.Context, key string, query TimeSeriesQuery) ([]TimeSeriesSample, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	members, err := s.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   strconv.FormatInt(query.StartAt.UTC().UnixMilli(), 10),
		Stop:    strconv.FormatInt(query.EndAt.UTC().UnixMilli(), 10),
		ByScore: true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("range time-series samples: %w", err)
	}

	samples := make([]TimeSeriesSample, 0, len(members))
	for _, member := range members {
		samples = append(samples, TimeSeriesSample{Payload: []byte(member)})
	}

	return samples, nil
}

// Validate ensures the time query is bounded and ordered.
func (q TimeSeriesQuery) Validate() error {
	if q.StartAt.IsZero() {
		return errors.New("statex time-series query start time is required")
	}
	if q.EndAt.IsZero() {
		return errors.New("statex time-series query end time is required")
	}
	if q.EndAt.Before(q.StartAt) {
		return errors.New("statex time-series query end time must not be before start time")
	}
	return nil
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("statex key is required")
	}
	return nil
}
