package runtimetarget

import "testing"

func TestNewModuleSpecDependsOnSavedView(t *testing.T) {
	for _, dependency := range NewModuleSpec().Dependencies {
		if dependency == "saved-view" {
			return
		}
	}
	t.Fatal("runtime-target must depend on saved-view before resolving SavedViewService")
}
