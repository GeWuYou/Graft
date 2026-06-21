// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package keys

import "testing"

func TestSegmentNormalizesFallback(t *testing.T) {
	t.Parallel()

	got := Segment("   ", " App/Host:One ")
	if got != "app-host-one" {
		t.Fatalf("expected normalized fallback, got %q", got)
	}
}
