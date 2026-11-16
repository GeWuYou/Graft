package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthenticationFileUsesBcryptAndRestrictedPermissions(t *testing.T) {
	root := t.TempDir()
	password := filepath.Join(root, "password")
	output := filepath.Join(root, "htpasswd")
	if err := os.WriteFile(password, []byte("synthetic-fixture-password\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAuthenticationFile("fixture", password, output); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.TrimSpace(strings.TrimPrefix(string(content), "fixture:"))
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("synthetic-fixture-password")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("authentication file permissions = %v / %v", info, err)
	}
}

func TestInvalidCredentialsDoNotCreateAnAuthenticationFile(t *testing.T) {
	root := t.TempDir()
	password := filepath.Join(root, "password")
	output := filepath.Join(root, "htpasswd")
	if err := os.WriteFile(password, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAuthenticationFile("fixture", password, output); err == nil {
		t.Fatal("empty password accepted")
	}
	if err := writeAuthenticationFile("user:injected", password, output); err == nil {
		t.Fatal("invalid username accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("invalid credentials created output: %v", err)
	}
}
