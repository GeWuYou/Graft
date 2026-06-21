// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package backend provides cachex backend adapters.
package backend

import (
	"context"
	"time"
)

// Entry is the backend-neutral stored cache payload.
type Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

// Backend defines the minimal mechanical storage operations required by cachex.
type Backend interface {
	Name() string
	Get(context.Context, string) (Entry, bool, error)
	Set(context.Context, string, Entry) error
	Delete(context.Context, string) error
}
