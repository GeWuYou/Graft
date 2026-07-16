package kvx

import (
	"bytes"
	"context"
	"sync"
	"time"
)

// MemoryOptions 配置供测试和本地运行时使用的进程内 KV 存储。
type MemoryOptions struct {
	Clock Clock
}

type memoryEntry struct {
	value     []byte
	expiresAt time.Time
}

// Memory 是带 TTL 和比较更新语义的进程内 KV 存储；锁保护整个键值表及过期清理过程。
type Memory struct {
	clock Clock
	mu    sync.Mutex
	store map[string]memoryEntry
}

// NewMemory 使用指定时钟创建进程内 KV 存储；未提供时钟时使用 UTC 系统时钟。
func NewMemory(options MemoryOptions) *Memory {
	clock := options.Clock
	if clock == nil {
		clock = systemClock{}
	}

	return &Memory{
		clock: clock,
		store: make(map[string]memoryEntry),
	}
}

// Put 写入一个值；负 TTL 会被拒绝，零 TTL 表示不过期。
func (m *Memory) Put(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := validateTTL(ttl); err != nil {
		return err
	}

	now := m.clock.Now()
	entry := newMemoryEntry(value, ttl, now)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneExpiredLocked(now)
	m.store[key] = entry

	return nil
}

// Get 读取一个值；未命中不视为错误，并返回 false。
func (m *Memory) Get(_ context.Context, key string) (Item, bool, error) {
	if err := validateKey(key); err != nil {
		return Item{}, false, err
	}

	now := m.clock.Now()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneExpiredLocked(now)

	entry, ok := m.store[key]
	if !ok {
		return Item{}, false, nil
	}

	return Item{
		Value:     cloneBytes(entry.value),
		ExpiresAt: entry.expiresAt,
	}, true, nil
}

// Delete 删除一个值；键不存在时仍视为成功。
func (m *Memory) Delete(_ context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)

	return nil
}

// CompareAndSwap 仅在当前字节值仍与期望值一致时原子更新值。
func (m *Memory) CompareAndSwap(_ context.Context, key string, oldValue []byte, newValue []byte, ttl time.Duration) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}
	if err := validateTTL(ttl); err != nil {
		return false, err
	}

	now := m.clock.Now()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneExpiredLocked(now)

	if _, ok := m.matchEntryLocked(key, oldValue); !ok {
		return false, nil
	}

	m.store[key] = newMemoryEntry(newValue, ttl, now)

	return true, nil
}

// CompareAndDelete 仅在当前字节值仍与期望值一致时删除值。
func (m *Memory) CompareAndDelete(_ context.Context, key string, oldValue []byte) (bool, error) {
	if err := validateKey(key); err != nil {
		return false, err
	}

	now := m.clock.Now()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneExpiredLocked(now)

	if _, ok := m.matchEntryLocked(key, oldValue); !ok {
		return false, nil
	}

	delete(m.store, key)
	return true, nil
}

func (m *Memory) pruneExpiredLocked(now time.Time) {
	for key, entry := range m.store {
		if !entry.expiresAt.IsZero() && !entry.expiresAt.After(now) {
			delete(m.store, key)
		}
	}
}

func (m *Memory) matchEntryLocked(key string, expected []byte) (memoryEntry, bool) {
	entry, ok := m.store[key]
	if !ok || !bytes.Equal(entry.value, expected) {
		return memoryEntry{}, false
	}
	return entry, true
}

// newMemoryEntry 克隆值并按 TTL 生成 memoryEntry；当 ttl 大于 0 时设置过期时间。
func newMemoryEntry(value []byte, ttl time.Duration, now time.Time) memoryEntry {
	entry := memoryEntry{value: cloneBytes(value)}
	if ttl > 0 {
		entry.expiresAt = now.Add(ttl)
	}
	return entry
}
