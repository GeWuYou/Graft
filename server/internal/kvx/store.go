package kvx

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrKeyRequired indicates one KV operation was called without a stable key.
	ErrKeyRequired = errors.New("kv key is required")
	// ErrNegativeTTL indicates one KV write attempted to use a negative TTL.
	ErrNegativeTTL = errors.New("kv ttl must not be negative")
)

// Item carries one stored value and its recovered expiry timestamp when known.
type Item struct {
	Value     []byte
	ExpiresAt time.Time
}

// Store defines the mechanical infra-KV contract used by runtime services.
type Store interface {
	// Put writes one value with the given TTL. A zero TTL means no expiration.
	Put(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Get 读取一个值；未命中返回 false。
	Get(ctx context.Context, key string) (Item, bool, error)
	// Delete 删除一个值；键不存在时仍视为成功。
	Delete(ctx context.Context, key string) error
	// CompareAndSwap 仅在当前字节值一致时替换值，具体实现必须保持原子性。
	CompareAndSwap(ctx context.Context, key string, oldValue []byte, newValue []byte, ttl time.Duration) (bool, error)
	// CompareAndDelete 仅在当前字节值一致时删除值，具体实现必须保持原子性。
	CompareAndDelete(ctx context.Context, key string, oldValue []byte) (bool, error)
}

// Clock provides the current wall time for TTL-backed stores.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// validateKey 验证 key 不为空。
func validateKey(key string) error {
	if key == "" {
		return ErrKeyRequired
	}
	return nil
}

// validateTTL 验证 TTL 不为负数。
func validateTTL(ttl time.Duration) error {
	if ttl < 0 {
		return ErrNegativeTTL
	}
	return nil
}

// cloneBytes 返回字节切片的深拷贝；nil 保持为 nil，空切片仍保持为空切片。
func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}
