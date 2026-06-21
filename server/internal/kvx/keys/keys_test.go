package keys

import "testing"

func TestJoinBuildsStableKey(t *testing.T) {
	t.Parallel()

	key := Join(" graft ", "", "tickets", "abc123")
	if key != "graft:tickets:abc123" {
		t.Fatalf("unexpected key %q", key)
	}
}
