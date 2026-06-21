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
	// Get returns one stored value when present.
	Get(ctx context.Context, key string) (Item, bool, error)
	// Delete removes one stored value.
	Delete(ctx context.Context, key string) error
	// CompareAndSwap replaces one value only when the current bytes still match.
	CompareAndSwap(ctx context.Context, key string, oldValue []byte, newValue []byte, ttl time.Duration) (bool, error)
	// CompareAndDelete removes one value only when the current bytes still match.
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

// validateTTL validates that ttl is not negative.
func validateTTL(ttl time.Duration) error {
	if ttl < 0 {
		return ErrNegativeTTL
	}
	return nil
}

// cloneBytes returns a deep copy of the provided byte slice, or nil if the input is nil.
// When the input is an empty slice, it returns an empty copy rather than nil.
func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}
