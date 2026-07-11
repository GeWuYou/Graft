package project

import "testing"

func TestIsProjectLogDebugEnabled(t *testing.T) {
	t.Setenv(projectLogDebugEnvironmentKey, "true")
	if !isProjectLogDebugEnabled() {
		t.Fatal("expected project log debug to be enabled")
	}

	t.Setenv(projectLogDebugEnvironmentKey, "false")
	if isProjectLogDebugEnabled() {
		t.Fatal("expected project log debug to be disabled")
	}
}
