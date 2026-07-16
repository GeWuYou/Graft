package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreManagedCreateWorkspaceUsesRootRelativePaths(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "compose.yaml")
	if err := os.WriteFile(path, []byte("original"), managedCreateFileMode); err != nil {
		t.Fatalf("write original file: %v", err)
	}

	root, err := openManagedRootFS(workspace)
	if err != nil {
		t.Fatalf("open workspace root: %v", err)
	}
	defer func() { _ = closeManagedRootFS(root) }()
	state := &managedCreateRestoreState{
		root:  root,
		items: []managedCreateRestoreItem{{path: "compose.yaml", content: []byte("original"), exists: true}},
	}
	if err := root.root.WriteFile("compose.yaml", []byte("replacement"), managedCreateFileMode); err != nil {
		t.Fatalf("write replacement file: %v", err)
	}

	if err := restoreManagedCreateWorkspace(state); err != nil {
		t.Fatalf("restore workspace: %v", err)
	}
	// #nosec G304 -- 测试路径来自 t.TempDir()，不受外部输入控制。
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != "original" {
		t.Fatalf("expected original content, got %q", content)
	}
}

func TestRestoreManagedCreateWorkspaceRejectsPathEscape(t *testing.T) {
	base := t.TempDir()
	workspace := filepath.Join(base, "workspace")
	if err := os.Mkdir(workspace, managedCreateDirMode); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	outside := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), managedCreateFileMode); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	root, err := openManagedRootFS(workspace)
	if err != nil {
		t.Fatalf("open workspace root: %v", err)
	}
	defer func() { _ = closeManagedRootFS(root) }()

	err = restoreManagedCreateWorkspace(&managedCreateRestoreState{
		root:  root,
		items: []managedCreateRestoreItem{{path: "../outside.txt", content: []byte("changed"), exists: true}},
	})
	if err == nil {
		t.Fatal("expected path escape error")
	}
	// #nosec G304 -- 测试路径来自 t.TempDir()，不受外部输入控制。
	content, readErr := os.ReadFile(outside)
	if readErr != nil {
		t.Fatalf("read outside file: %v", readErr)
	}
	if string(content) != "outside" {
		t.Fatalf("outside file was modified: %q", content)
	}
}
