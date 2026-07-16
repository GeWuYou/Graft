package backend

import (
	"context"
	"sync"
	"time"
)

// Memory 在进程内保存缓存项，并通过读写锁保护共享映射。
type Memory struct {
	mu    sync.RWMutex
	items map[string]Entry
	now   func() time.Time
}

// NewMemory 创建一个使用系统时钟、可并发访问的进程内缓存后端。
func NewMemory() *Memory {
	return &Memory{
		items: make(map[string]Entry),
		now:   time.Now,
	}
}

// Name 返回后端标识 memory，供缓存指标区分存储实现。
func (m *Memory) Name() string {
	return "memory"
}

// Get 返回存在且未过期的缓存项；发现过期项时会在确认期间未被更新后删除它。
func (m *Memory) Get(_ context.Context, key string) (Entry, bool, error) {
	now := m.now()
	m.mu.RLock()
	entry, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return Entry{}, false, nil
	}

	if !entry.ExpiresAt.IsZero() && !entry.ExpiresAt.After(now) {
		m.mu.Lock()
		latest, exists := m.items[key]
		if !exists {
			m.mu.Unlock()
			return Entry{}, false, nil
		}
		if !latest.ExpiresAt.IsZero() && !latest.ExpiresAt.After(now) {
			delete(m.items, key)
			m.mu.Unlock()
			return Entry{}, false, nil
		}
		m.mu.Unlock()
		return cloneEntry(latest), true, nil
	}

	return cloneEntry(entry), true, nil
}

// Set 保存缓存项的副本，避免调用方之后修改共享载荷。
func (m *Memory) Set(_ context.Context, key string, entry Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = cloneEntry(entry)
	return nil
}

// Delete 删除指定键的缓存项；键不存在时视为成功。
func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
	return nil
}

// cloneEntry 深拷贝缓存项，隔离后端内部数据与调用方持有的字节切片。
func cloneEntry(entry Entry) Entry {
	cloned := Entry{
		ExpiresAt: entry.ExpiresAt,
	}
	if len(entry.Value) > 0 {
		cloned.Value = make([]byte, len(entry.Value))
		copy(cloned.Value, entry.Value)
	}

	return cloned
}
