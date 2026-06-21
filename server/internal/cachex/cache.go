// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package cachex provides a mechanical cache layer for core-owned runtime services.
package cachex

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/cachex/backend"
	"graft/server/internal/cachex/keys"
)

// Cache performs read-through and write-through cache operations for one cache namespace.
type Cache struct {
	name    string
	keyRoot string
	backend backend.Backend
	ttl     time.Duration
	metrics Metrics
	group   *Group
}

// Name returns the stable cache name under the owning manager.
func (c *Cache) Name() string {
	if c == nil {
		return ""
	}

	return c.name
}

// Get loads one item from the configured backend.
func (c *Cache) Get(ctx context.Context, key keys.Key) (Item, bool, error) {
	if err := c.validate(); err != nil {
		return Item{}, false, err
	}

	cacheKey := c.cacheKey(key)
	started := time.Now()
	entry, ok, err := c.backend.Get(ctx, cacheKey)
	c.metrics.Observe(Event{
		Cache:     c.name,
		Backend:   c.backend.Name(),
		Operation: "get",
		Hit:       ok,
		Duration:  time.Since(started),
		Err:       err,
	})
	if err != nil {
		return Item{}, false, err
	}
	if !ok {
		return Item{}, false, nil
	}

	return itemFromEntry(entry), true, nil
}

// Set writes one item into the configured backend.
func (c *Cache) Set(ctx context.Context, key keys.Key, item Item) error {
	if err := c.validate(); err != nil {
		return err
	}

	normalized, err := c.normalizeItem(item)
	if err != nil {
		return err
	}

	started := time.Now()
	err = c.backend.Set(ctx, c.cacheKey(key), entryFromItem(normalized))
	c.metrics.Observe(Event{
		Cache:     c.name,
		Backend:   c.backend.Name(),
		Operation: "set",
		Duration:  time.Since(started),
		Err:       err,
	})
	return err
}

// Delete removes one item from the configured backend.
func (c *Cache) Delete(ctx context.Context, key keys.Key) error {
	if err := c.validate(); err != nil {
		return err
	}

	started := time.Now()
	err := c.backend.Delete(ctx, c.cacheKey(key))
	c.metrics.Observe(Event{
		Cache:     c.name,
		Backend:   c.backend.Name(),
		Operation: "delete",
		Duration:  time.Since(started),
		Err:       err,
	})
	return err
}

// GetOrLoad returns a cached item or executes the loader once per key on misses.
func (c *Cache) GetOrLoad(ctx context.Context, key keys.Key, loader Loader) (Item, error) {
	if loader == nil {
		return Item{}, ErrLoaderRequired
	}

	item, ok, err := c.Get(ctx, key)
	if err != nil {
		return Item{}, err
	}
	if ok {
		return item, nil
	}

	cacheKey := c.cacheKey(key)
	loaded, err, shared := c.group.Do(cacheKey, func() (Item, error) {
		current, found, currentErr := c.Get(ctx, key)
		if currentErr != nil {
			return Item{}, currentErr
		}
		if found {
			return current, nil
		}

		started := time.Now()
		loadedItem, loadErr := loader(ctx)
		if loadErr != nil {
			c.metrics.Observe(Event{
				Cache:     c.name,
				Backend:   c.backend.Name(),
				Operation: "load",
				Duration:  time.Since(started),
				Err:       loadErr,
			})
			return Item{}, loadErr
		}

		normalized, normalizeErr := c.normalizeItem(loadedItem)
		if normalizeErr != nil {
			c.metrics.Observe(Event{
				Cache:     c.name,
				Backend:   c.backend.Name(),
				Operation: "load",
				Duration:  time.Since(started),
				Err:       normalizeErr,
			})
			return Item{}, normalizeErr
		}
		if setErr := c.backend.Set(ctx, cacheKey, entryFromItem(normalized)); setErr != nil {
			c.metrics.Observe(Event{
				Cache:     c.name,
				Backend:   c.backend.Name(),
				Operation: "load",
				Duration:  time.Since(started),
				Err:       setErr,
			})
			return Item{}, setErr
		}

		c.metrics.Observe(Event{
			Cache:     c.name,
			Backend:   c.backend.Name(),
			Operation: "load",
			Duration:  time.Since(started),
		})
		return normalized, nil
	})
	c.metrics.Observe(Event{
		Cache:     c.name,
		Backend:   c.backend.Name(),
		Operation: "singleflight",
		Shared:    shared,
		Err:       err,
	})
	if err != nil {
		return Item{}, err
	}

	return loaded, nil
}

func (c *Cache) validate() error {
	if c == nil {
		return errors.New("cache is unavailable")
	}
	if c.backend == nil {
		return errors.New("cache backend is unavailable")
	}
	if c.group == nil {
		return errors.New("cache singleflight group is unavailable")
	}
	if strings.TrimSpace(c.name) == "" {
		return errors.New("cache name is required")
	}
	if strings.TrimSpace(c.keyRoot) == "" {
		return errors.New("cache key root is required")
	}

	return nil
}

func (c *Cache) cacheKey(key keys.Key) string {
	return fmt.Sprintf("%s:%s", c.keyRoot, key.String())
}

func (c *Cache) normalizeItem(item Item) (Item, error) {
	if err := item.Validate(); err != nil {
		return Item{}, err
	}

	normalized := item.Clone()
	if normalized.ExpiresAt.IsZero() && normalized.TTL == 0 && c.ttl > 0 {
		normalized.TTL = c.ttl
	}
	if normalized.ExpiresAt.IsZero() && normalized.TTL > 0 {
		normalized.ExpiresAt = time.Now().Add(normalized.TTL)
	}

	return normalized, nil
}
