package backend

import (
	"context"
	"sync"
	"time"
)

// Memory stores cache entries in-process.
type Memory struct {
	mu    sync.RWMutex
	items map[string]Entry
	now   func() time.Time
}

// NewMemory returns a new in-process cache backend.
func NewMemory() *Memory {
	return &Memory{
		items: make(map[string]Entry),
		now:   time.Now,
	}
}

// Name returns the backend name.
func (m *Memory) Name() string {
	return "memory"
}

// Get returns one stored entry when present and not expired.
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

// Set stores one entry.
func (m *Memory) Set(_ context.Context, key string, entry Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = cloneEntry(entry)
	return nil
}

// Delete removes one entry.
func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
	return nil
}

// cloneEntry returns a deep copy of entry to prevent external modification of cached contents.
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
