// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisOptions configures the cachex Redis backend adapter.
type RedisOptions struct {
	Prefix string
	Now    func() time.Time
}

// Redis adapts go-redis to the cachex mechanical backend contract.
type Redis struct {
	client redis.Cmdable
	prefix string
	now    func() time.Time
}

// NewRedis creates one Redis backend adapter.
func NewRedis(client redis.Cmdable, options RedisOptions) (*Redis, error) {
	if client == nil {
		return nil, fmt.Errorf("redis backend client is required")
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &Redis{
		client: client,
		prefix: strings.TrimSpace(options.Prefix),
		now:    now,
	}, nil
}

// Name returns the backend name.
func (r *Redis) Name() string {
	return "redis"
}

// Get returns one stored Redis entry when present.
func (r *Redis) Get(ctx context.Context, key string) (Entry, bool, error) {
	value, err := r.client.Get(ctx, r.prefixed(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, err
	}

	entry := Entry{Value: cloneBytes(value)}
	ttl, ttlErr := r.client.PTTL(ctx, r.prefixed(key)).Result()
	if ttlErr != nil && !errors.Is(ttlErr, redis.Nil) {
		return Entry{}, false, ttlErr
	}
	if ttl > 0 {
		entry.ExpiresAt = r.now().Add(ttl)
	}

	return entry, true, nil
}

// Set writes one entry to Redis.
func (r *Redis) Set(ctx context.Context, key string, entry Entry) error {
	if !entry.ExpiresAt.IsZero() && !entry.ExpiresAt.After(r.now()) {
		return r.Delete(ctx, key)
	}

	ttl := time.Duration(0)
	if !entry.ExpiresAt.IsZero() {
		ttl = entry.ExpiresAt.Sub(r.now())
		if ttl < 0 {
			ttl = 0
		}
	}

	return r.client.Set(ctx, r.prefixed(key), cloneBytes(entry.Value), ttl).Err()
}

// Delete removes one entry from Redis.
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefixed(key)).Err()
}

func (r *Redis) prefixed(key string) string {
	if r.prefix == "" {
		return key
	}

	return r.prefix + ":" + key
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}
