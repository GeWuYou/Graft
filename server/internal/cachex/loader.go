// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cachex

import (
	"context"
	"errors"
)

// ErrLoaderRequired indicates that a read-through cache call did not receive a loader.
var ErrLoaderRequired = errors.New("cache loader is required")

// Loader builds one item when the cache misses.
type Loader func(context.Context) (Item, error)
