// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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

// NewMemory creates one in-process cache backend.
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
	m.mu.RLock()
	entry, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return Entry{}, false, nil
	}

	if !entry.ExpiresAt.IsZero() && !entry.ExpiresAt.After(m.now()) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return Entry{}, false, nil
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
