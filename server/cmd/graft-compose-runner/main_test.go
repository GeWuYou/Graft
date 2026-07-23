package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceRefsReplacesMutableComposeImageReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_SERVER_IMAGE=ghcr.io/gewuyou/graft-server:latest\nGRAFT_WEB_IMAGE=ghcr.io/gewuyou/graft-web:latest\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("a", 64)
	web := "ghcr.io/gewuyou/graft-web@sha256:" + strings.Repeat("b", 64)
	if err := replaceRefs(path, server, web); err != nil {
		t.Fatalf("replace mutable image references: %v", err)
	}
	// #nosec G304 -- path is a test-owned file under this test's temporary directory.
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated compose environment: %v", err)
	}
	if strings.Contains(string(contents), ":latest") || !strings.Contains(string(contents), "GRAFT_SERVER_IMAGE="+server) || !strings.Contains(string(contents), "GRAFT_WEB_IMAGE="+web) {
		t.Fatalf("compose environment does not contain frozen references: %s", contents)
	}
}

func TestReplaceRefsRejectsMissingComposeImageReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_SERVER_IMAGE=ghcr.io/gewuyou/graft-server:latest\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("a", 64)
	web := "ghcr.io/gewuyou/graft-web@sha256:" + strings.Repeat("b", 64)
	if err := replaceRefs(path, server, web); err == nil {
		t.Fatal("expected missing web image reference to reject update")
	}
}

func TestReplaceRefsRejectsMutableImageReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_SERVER_IMAGE=ghcr.io/gewuyou/graft-server:latest\nGRAFT_WEB_IMAGE=ghcr.io/gewuyou/graft-web:latest\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	if err := replaceRefs(path, "ghcr.io/gewuyou/graft-server:latest", "ghcr.io/gewuyou/graft-web:latest"); err == nil {
		t.Fatal("expected mutable image references to be rejected")
	}
}
