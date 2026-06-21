// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package keys

import "testing"

func TestJoinBuildsStableKey(t *testing.T) {
	t.Parallel()

	key := Join(" graft ", "", "tickets", "abc123")
	if key != "graft:tickets:abc123" {
		t.Fatalf("unexpected key %q", key)
	}
}
