package route

import "testing"

func TestAPIPathCombinesCanonicalRootAndModuleFragment(t *testing.T) {
	if got := APIPath("/audit/policies/visibility"); got != "/api/audit/policies/visibility" {
		t.Fatalf("APIPath() = %q, want canonical full path", got)
	}
}
