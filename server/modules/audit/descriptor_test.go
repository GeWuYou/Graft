package audit

import "testing"

func TestDescriptorDeclaresCanonicalDependencies(t *testing.T) {
	t.Parallel()

	descriptor := NewModuleSpec()
	got := descriptor.DependsOn()
	if len(got) != 3 || got[0] != "user" || got[1] != "rbac" || got[2] != "saved-view" {
		t.Fatalf("descriptor dependencies = %v, want [user rbac saved-view]", got)
	}
}
