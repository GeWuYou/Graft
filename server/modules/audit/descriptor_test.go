// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package audit

import "testing"

func TestDescriptorDeclaresCanonicalDependencies(t *testing.T) {
	t.Parallel()

	descriptor := NewModuleSpec()
	got := descriptor.DependsOn()
	if len(got) != 2 || got[0] != "user" || got[1] != "rbac" {
		t.Fatalf("descriptor dependencies = %v, want [user rbac]", got)
	}
}
