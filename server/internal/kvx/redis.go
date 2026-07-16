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

// RedisOptions 配置基于 Redis 的 KV 适配器。
type RedisOptions struct {
	Prefix string
	Now    func() time.Time
}

// Redis 将 go-redis 适配为无业务语义的 KV 契约。
type Redis struct {
	client redis.Cmdable
	prefix string
	now    func() time.Time
}

// NewRedis 使用给定客户端和选项创建 Redis 适配器；客户端为空时返回错误，未提供时钟时使用 UTC 系统时钟。
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

// Put 向 Redis 写入一个值；零 TTL 表示不过期。
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

// Get 读取 Redis 中的值；未命中不视为错误，并返回 false。
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

// Delete 删除 Redis 中的值；键不存在时仍视为成功。
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

// CompareAndSwap 通过 Redis 脚本仅在当前字节值一致时原子更新值。
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

// CompareAndDelete 通过 Redis 脚本仅在当前字节值一致时原子删除值。
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

// ttlMilliseconds 将 TTL 转为 Redis 脚本使用的毫秒数；正数但不足 1 毫秒时取 1，非正数取 0 表示不过期。
func ttlMilliseconds(ttl time.Duration) int64 {
	if ttl <= 0 {
		return 0
	}
	if ttl < time.Millisecond {
		return 1
	}
	return ttl.Milliseconds()
}
