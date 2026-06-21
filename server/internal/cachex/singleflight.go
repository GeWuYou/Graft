// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cachex

import "golang.org/x/sync/singleflight"

// Group collapses concurrent cache-miss loaders for the same cache key.
type Group struct {
	group singleflight.Group
}

// NewGroup creates one miss-collapse group.
func NewGroup() *Group {
	return &Group{}
}

// Do executes fn once for the given key and shares the result with concurrent callers.
func (g *Group) Do(key string, fn func() (Item, error)) (Item, error, bool) {
	if g == nil {
		item, err := fn()
		return item, err, false
	}

	value, err, shared := g.group.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		return Item{}, err, shared
	}

	item, ok := value.(Item)
	if !ok {
		return Item{}, nil, shared
	}

	return item.Clone(), nil, shared
}
