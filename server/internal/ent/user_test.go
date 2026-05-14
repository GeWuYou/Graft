package ent

import (
	"strings"
	"testing"
	"time"
)

// TestUserStringMasksPasswordHash 验证敏感口令散列不会出现在实体字符串表示中。
func TestUserStringMasksPasswordHash(t *testing.T) {
	hash := "super-secret-hash"
	now := time.Unix(1700000000, 0)

	user := &User{
		ID:                1,
		Username:          "alice",
		Display:           "Alice",
		PasswordHash:      &hash,
		PasswordChangedAt: &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	text := user.String()
	if strings.Contains(text, hash) {
		t.Fatalf("expected String() to mask password hash, got %q", text)
	}
	if !strings.Contains(text, "password_hash=<sensitive>") {
		t.Fatalf("expected String() to include sensitive marker, got %q", text)
	}
}
