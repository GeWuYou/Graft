// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cachex

import (
	"errors"
	"time"

	"graft/server/internal/cachex/backend"
)

// Item represents one cached payload plus its expiration semantics.
type Item struct {
	Value     []byte
	TTL       time.Duration
	ExpiresAt time.Time
}

// NewItem creates one cache item with a copied payload.
func NewItem(value []byte, ttl time.Duration) Item {
	return Item{
		Value: cloneBytes(value),
		TTL:   ttl,
	}
}

// Clone returns a defensive copy of the item payload and metadata.
func (i Item) Clone() Item {
	return Item{
		Value:     cloneBytes(i.Value),
		TTL:       i.TTL,
		ExpiresAt: i.ExpiresAt,
	}
}

// Validate checks whether the item carries coherent expiration metadata.
func (i Item) Validate() error {
	if i.TTL < 0 {
		return errors.New("cache item ttl must be non-negative")
	}
	if len(i.Value) == 0 {
		return errors.New("cache item value is required")
	}

	return nil
}

func itemFromEntry(entry backend.Entry) Item {
	return Item{
		Value:     cloneBytes(entry.Value),
		ExpiresAt: entry.ExpiresAt,
	}
}

func entryFromItem(item Item) backend.Entry {
	return backend.Entry{
		Value:     cloneBytes(item.Value),
		ExpiresAt: item.ExpiresAt,
	}
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}

	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}
