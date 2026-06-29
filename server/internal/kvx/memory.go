package kvx

import (
	"bytes"
	"context"
	"sync"
	"time"
)

// MemoryOptions configures one in-process KV store for tests and local runtime use.
type MemoryOptions struct {
	Clock Clock
}

type memoryEntry struct {
	value     []byte
	expiresAt time.Time
}

// Memory is one in-process KV store with TTL and compare-and-update semantics.
type Memory struct {
	clock Clock
	mu    sync.Mutex
	store map[string]memoryEntry
}

// NewMemory initializes a new in-process key-value store with the given clock, or the system clock if none is specified in options.
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

// Put writes one value into the memory store.
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

// Get reads one value from the memory store.
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

// Delete removes one value from the memory store.
func (m *Memory) Delete(_ context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)

	return nil
}

// CompareAndSwap updates one value only when the current bytes still match.
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

// CompareAndDelete removes one value only when the current bytes still match.
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
