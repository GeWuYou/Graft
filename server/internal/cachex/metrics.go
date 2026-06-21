// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cachex

import "time"

// Event describes one mechanical cache operation.
type Event struct {
	Cache     string
	Backend   string
	Operation string
	Hit       bool
	Shared    bool
	Duration  time.Duration
	Err       error
}

// Metrics consumes cache operation events.
type Metrics interface {
	Observe(Event)
}

type nopMetrics struct{}

// Observe discards metrics events.
func (nopMetrics) Observe(Event) {}

// NopMetrics returns a metrics sink that drops all events.
func NopMetrics() Metrics {
	return nopMetrics{}
}
