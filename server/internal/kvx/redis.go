package kvx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var compareAndSwapScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current then
	return 0
end
if current ~= ARGV[1] then
	return 0
end
local ttl = tonumber(ARGV[3])
if ttl > 0 then
	redis.call("PSETEX", KEYS[1], ttl, ARGV[2])
else
	redis.call("SET", KEYS[1], ARGV[2])
end
return 1
`)

var compareAndDeleteScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current then
	return 0
end
if current ~= ARGV[1] then
	return 0
end
redis.call("DEL", KEYS[1])
return 1
`)

// RedisOptions configures the Redis-backed KV adapter.
type RedisOptions struct {
	Prefix string
	Now    func() time.Time
}

// Redis adapts go-redis to the mechanical KV contract.
type Redis struct {
	client redis.Cmdable
	prefix string
	now    func() time.Time
}

// NewRedis initializes a Redis-backed KV store with the provided client and options.
// It returns an error if client is nil.
// NewRedis creates a new Redis adapter using the provided client and options.
// It returns an error if the client is nil. If options.Now is nil, it defaults
// to time.Now().UTC().
func NewRedis(client redis.Cmdable, options RedisOptions) (*Redis, error) {
	if client == nil {
		return nil, errors.New("kv redis client is required")
	}

	now := options.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	return &Redis{
		client: client,
		prefix: strings.TrimSpace(options.Prefix),
		now:    now,
	}, nil
}

// Put writes one value into Redis with the given TTL.
func (r *Redis) Put(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := validateTTL(ttl); err != nil {
		return err
	}

	prefixedKey := r.prefixed(key)
	if err := r.client.Set(ctx, prefixedKey, cloneBytes(value), ttl).Err(); err != nil {
		return fmt.Errorf("kv redis put %q: %w", prefixedKey, err)
	}
	return nil
}

// Get reads one value from Redis when present.
func (r *Redis) Get(ctx context.Context, key string) (Item, bool, error) {
	if err := validateKey(key); err != nil {
		return Item{}, false, err
	}

	prefixedKey := r.prefixed(key)
	value, err := r.client.Get(ctx, prefixedKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return Item{}, false, nil
	}
	if err != nil {
		return Item{}, false, fmt.Errorf("kv redis get %q: %w", prefixedKey, err)
	}

	item := Item{Value: cloneBytes(value)}
	ttl, ttlErr := r.client.PTTL(ctx, prefixedKey).Result()
	if ttlErr != nil && !errors.Is(ttlErr, redis.Nil) {
		return Item{}, false, fmt.Errorf("kv redis pttl %q: %w", prefixedKey, ttlErr)
	}
	if ttl > 0 {
		item.ExpiresAt = r.now().Add(ttl)
	}

	return item, true, nil
}

// Delete removes one Redis value.
func (r *Redis) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	prefixedKey := r.prefixed(key)
	if err := r.client.Del(ctx, prefixedKey).Err(); err != nil {
		return fmt.Errorf("kv redis delete %q: %w", prefixedKey, err)
	}
	return nil
}

// CompareAndSwap updates one Redis value only when the current bytes still match.
func (r *Redis) CompareAndSwap(ctx context.Context, key string, oldValue []byte, newValue []byte, ttl time.Duration) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}
	if err := validateTTL(ttl); err != nil {
		return false, err
	}

	prefixedKey := r.prefixed(key)
	result, err := compareAndSwapScript.Run(
		ctx,
		r.client,
		[]string{prefixedKey},
		string(oldValue),
		string(newValue),
		ttlMilliseconds(ttl),
	).Int64()
	if err != nil {
		return false, fmt.Errorf("kv redis compare-and-swap %q: %w", prefixedKey, err)
	}

	return result == 1, nil
}

// CompareAndDelete removes one Redis value only when the current bytes still match.
func (r *Redis) CompareAndDelete(ctx context.Context, key string, oldValue []byte) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	prefixedKey := r.prefixed(key)
	result, err := compareAndDeleteScript.Run(
		ctx,
		r.client,
		[]string{prefixedKey},
		string(oldValue),
	).Int64()
	if err != nil {
		return false, fmt.Errorf("kv redis compare-and-delete %q: %w", prefixedKey, err)
	}

	return result == 1, nil
}

func (r *Redis) prefixed(key string) string {
	if r.prefix == "" {
		return key
	}
	return r.prefix + ":" + key
}

// ttlMilliseconds converts a duration to milliseconds. It returns 0 for durations of 0 or less, 1 for positive durations less than a millisecond, and otherwise returns the duration in milliseconds.
func ttlMilliseconds(ttl time.Duration) int64 {
	if ttl <= 0 {
		return 0
	}
	if ttl < time.Millisecond {
		return 1
	}
	return ttl.Milliseconds()
}
