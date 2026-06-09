// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

import "time"

const (
	// TrendRangeQueryKey identifies the canonical trend range query parameter.
	TrendRangeQueryKey      = "trend_range"
	tenMinuteTrendWindow    = 10 * time.Minute
	thirtyMinuteTrendWindow = 30 * time.Minute
)

// TrendRange identifies the stable monitor trend range contract.
type TrendRange string

// String returns the wire-format trend range value.
func (r TrendRange) String() string {
	return string(r)
}

const (
	// TrendRange10Minutes identifies the default 10-minute trend window.
	TrendRange10Minutes TrendRange = "10m"
	// TrendRange30Minutes identifies the 30-minute trend window.
	TrendRange30Minutes TrendRange = "30m"
	// TrendRange1Hour identifies the 1-hour trend window.
	TrendRange1Hour TrendRange = "1h"
)

// Duration returns the canonical duration represented by the trend range.
func (r TrendRange) Duration() time.Duration {
	switch r {
	case TrendRange30Minutes:
		return thirtyMinuteTrendWindow
	case TrendRange1Hour:
		return time.Hour
	default:
		return tenMinuteTrendWindow
	}
}
