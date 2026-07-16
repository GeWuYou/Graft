package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkspaceTemplatePreservesDirectoriesAndArbitraryTextFiles(t *testing.T) {
	root := t.TempDir()
	template := filepath.Join(root, "templates", "custom")
	if err := os.MkdirAll(filepath.Join(template, "empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(template, "file-without-extension"), []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := loadWorkspaceTemplate(root, "custom")
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	if len(entries) != 2 || entries[0].Path != "empty" || entries[0].NodeType != "directory" || entries[1].Path != "file-without-extension" || entries[1].NodeType != "file" {
		t.Fatalf("unexpected template entries: %#v", entries)
	}
}

func TestNormalizeManagedWorkspaceEntriesAcceptsArbitraryTextAndEmptyDirectory(t *testing.T) {
	compose := "services: {}\n"
	content := "arbitrary text\n"
	entries, err := normalizeManagedWorkspaceEntries([]ManagedWorkspaceEntry{
		{Path: "nested", NodeType: "directory"},
		{Path: "nested/readme", NodeType: "file", Content: &content},
		{Path: "compose.yaml", NodeType: "file", Content: &compose},
	}, "compose.yaml")
	if err != nil {
		t.Fatalf("normalize entries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("entry count = %d, want 3", len(entries))
	}
}

func TestNormalizeManagedWorkspaceEntriesRejectsFileAncestorRegardlessOfOrder(t *testing.T) {
	compose := "services: {}\n"
	for _, entries := range [][]ManagedWorkspaceEntry{
		{{Path: "config/child", NodeType: "file", Content: &compose}, {Path: "config", NodeType: "file", Content: &compose}, {Path: "compose.yaml", NodeType: "file", Content: &compose}},
		{{Path: "config", NodeType: "file", Content: &compose}, {Path: "config/child", NodeType: "file", Content: &compose}, {Path: "compose.yaml", NodeType: "file", Content: &compose}},
	} {
		if _, err := normalizeManagedWorkspaceEntries(entries, "compose.yaml"); !errors.Is(err, errProjectInvalidArgument) {
			t.Fatalf("expected file ancestor conflict, got %v", err)
		}
	}
}

func TestLoadWorkspaceTemplateRejectsOversizedFileBeforeRead(t *testing.T) {
	root := t.TempDir()
	template := filepath.Join(root, "templates", "custom")
	if err := os.MkdirAll(template, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(template, "large.txt"), make([]byte, maxWorkspaceFileBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadWorkspaceTemplate(root, "custom"); err == nil {
		t.Fatal("expected oversized template file rejection")
	}
}
