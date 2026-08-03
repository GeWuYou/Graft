package cli

import (
	"context"
	"testing"
)

func TestNewRootCommandRegistersUpdateCutover(t *testing.T) {
	command := NewRootCommand()
	updateCommand, _, err := command.Find([]string{"update"})
	if err != nil || updateCommand == nil {
		t.Fatalf("find update command: %v", err)
	}
	cutover, _, err := updateCommand.Find([]string{"cutover-v1"})
	if err != nil || cutover == nil {
		t.Fatalf("find cutover command: %v", err)
	}
}

func TestUpdateCutoverCommandUsesInjectedRunner(t *testing.T) {
	original := updateCutover
	defer func() { updateCutover = original }()
	called := false
	updateCutover = func(context.Context) error { called = true; return nil }
	command := NewRootCommand()
	command.SetArgs([]string{"update", "cutover-v1"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute cutover: %v", err)
	}
	if !called {
		t.Fatal("cutover runner was not called")
	}
}
