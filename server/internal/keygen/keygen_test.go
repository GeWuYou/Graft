package keygen

import (
	"encoding/base64"
	"strings"
	"testing"
)

// TestGenerateEnvLine 验证生成结果可直接作为 `.env` 配置行使用。
func TestGenerateEnvLine(t *testing.T) {
	line, err := GenerateEnvLine("GRAFT_AUTH_JWT_SECRET")
	if err != nil {
		t.Fatalf("generate env line: %v", err)
	}

	const prefix = "GRAFT_AUTH_JWT_SECRET="
	if !strings.HasPrefix(line, prefix) {
		t.Fatalf("expected prefix %q, got %q", prefix, line)
	}

	secret := strings.TrimPrefix(line, prefix)
	if len(secret) == 0 {
		t.Fatal("expected non-empty secret")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		t.Fatalf("decode generated secret: %v", err)
	}
	if len(decoded) != randomSecretBytes {
		t.Fatalf("expected %d random bytes, got %d", randomSecretBytes, len(decoded))
	}
}

// TestGenerateEnvLineRejectsEmptyEnvKey 验证缺少环境变量名时会直接失败。
func TestGenerateEnvLineRejectsEmptyEnvKey(t *testing.T) {
	if _, err := GenerateEnvLine("   "); err == nil {
		t.Fatal("expected empty env key error")
	}
}
