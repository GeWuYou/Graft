package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisOptions 配置 cachex 的 Redis 后端适配器。
type RedisOptions struct {
	Prefix string
	Now    func() time.Time
}

// Redis 将 go-redis 适配为 cachex 的机械后端契约。
type Redis struct {
	client redis.Cmdable
	prefix string
	now    func() time.Time
}

// NewRedis 创建一个 Redis 后端适配器。若提供的客户端为 nil 则返回错误；时钟函数默认为 time.Now；前缀会被去除首尾空白。
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

// Name 返回后端标识 redis，供缓存指标区分存储实现。
func (r *Redis) Name() string {
	return "redis"
}

// Get 返回存在的 Redis 缓存项，并把 Redis TTL 转换为绝对过期时间。
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

// Set 将缓存项写入 Redis；已过期项直接删除，避免写入失效值。
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

// Delete 删除 Redis 中的指定缓存项；Redis 对不存在键的处理保持为成功。
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefixed(key)).Err()
}

func (r *Redis) prefixed(key string) string {
	if r.prefix == "" {
		return key
	}

	return r.prefix + ":" + key
}

// cloneBytes 深拷贝字节切片；空输入返回 nil 以保持空载荷语义。
func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}
