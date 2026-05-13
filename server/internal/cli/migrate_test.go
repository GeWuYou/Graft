package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveMigrationDirFindsServerRelativePathFromRepoRoot 验证仓库根目录下
// 的默认迁移目录会被解析为 `server` 相对路径。
func TestResolveMigrationDirFindsServerRelativePathFromRepoRoot(t *testing.T) {
	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", defaultMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o755); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dir: %v", err)
	}

	if resolved != migrationDir {
		t.Fatalf("expected %s, got %s", migrationDir, resolved)
	}
}

// TestResolveMigrationDirFindsPathFromServerModuleRoot 验证迁移目录解析器也支持
// 以 `server` 模块根目录作为当前工作目录。
func TestResolveMigrationDirFindsPathFromServerModuleRoot(t *testing.T) {
	root := t.TempDir()
	serverRoot := filepath.Join(root, "server")
	migrationDir := filepath.Join(serverRoot, defaultMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o755); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(serverRoot, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dir: %v", err)
	}

	if resolved != migrationDir {
		t.Fatalf("expected %s, got %s", migrationDir, resolved)
	}
}

// TestResolveMigrationDirRejectsMissingPath 验证当两个受支持的迁移目录都不
// 存在时，解析器会返回错误。
func TestResolveMigrationDirRejectsMissingPath(t *testing.T) {
	root := t.TempDir()

	_, err := resolveMigrationDir(root, defaultMigrationDir)
	if err == nil {
		t.Fatal("expected missing migration dir error")
	}
}

// TestFindAtlasCLIReportsDevGuidance 验证缺少 Atlas 时会返回可执行的开发提示。
func TestFindAtlasCLIReportsDevGuidance(t *testing.T) {
	originalLookPath := migrateLookPath
	defer func() {
		migrateLookPath = originalLookPath
	}()

	migrateLookPath = func(file string) (string, error) {
		return "", errors.New("executable file not found")
	}

	_, err := findAtlasCLI()
	if err == nil {
		t.Fatal("expected atlas lookup error")
	}

	message := err.Error()
	if !strings.Contains(message, "graft dev") {
		t.Fatalf("expected dev guidance, got %q", message)
	}
	if !strings.Contains(message, "graft serve") {
		t.Fatalf("expected serve fallback guidance, got %q", message)
	}
}
