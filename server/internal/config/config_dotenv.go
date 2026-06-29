package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// loadDotenv 从显式指定或自动发现的 .env 文件加载环境变量。
// 当 `GRAFT_ENV_FILE` 有值时优先加载该路径；否则会在工作目录向上查找可用的 `.env` 文件并加载。
// 返回加载过程中的错误。
func loadDotenv() error {
	if explicit := strings.TrimSpace(os.Getenv("GRAFT_ENV_FILE")); explicit != "" {
		if err := godotenv.Load(explicit); err != nil {
			return fmt.Errorf("load %s: %w", explicit, err)
		}
		return nil
	}

	dotenvPath, err := findDotenvPath()
	if err != nil {
		return err
	}
	if dotenvPath != "" {
		return godotenv.Load(dotenvPath)
	}

	return nil
}

// findDotenvPath 查找可加载的 .env 文件路径。
// 它从当前工作目录向上搜索，优先匹配 `<dir>/.env`，其次匹配 `<dir>/server/.env`；
// 找到第一个存在的文件路径并返回，未找到时返回空字符串和 nil。
func findDotenvPath() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	for _, dir := range dotenvSearchDirs(workingDir) {
		for _, candidate := range []string{
			filepath.Join(dir, ".env"),
			filepath.Join(dir, "server", ".env"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("stat dotenv candidate %s: %w", candidate, err)
			}
		}
	}

	return "", nil
}

// dotenvSearchDirs 生成从起始目录向上用于查找 dotenv 文件的目录列表。
// 当起始目录为空白时，返回 nil；当到达文件系统根目录或命中边界目录时停止。
// @param start 起始目录。
// @returns 可用于搜索 dotenv 文件的目录列表。
func dotenvSearchDirs(start string) []string {
	if strings.TrimSpace(start) == "" {
		return nil
	}

	dirs := []string{}
	current := filepath.Clean(start)
	for {
		dirs = append(dirs, current)

		parent := filepath.Dir(current)
		if parent == current {
			return dirs
		}
		if isDotenvSearchBoundary(current) {
			if filepath.Base(current) == "server" {
				dirs = append(dirs, parent)
			}
			return dirs
		}
		current = parent
	}
}

// isDotenvSearchBoundary 判断目录是否为 dotenv 搜索边界。
// 当目录名为 `server`，或目录下存在 `.git` 标记，或存在名为 `server` 的子目录时，返回 true。
func isDotenvSearchBoundary(dir string) bool {
	if filepath.Base(dir) == "server" {
		return true
	}

	for _, marker := range []string{".git", "server"} {
		info, err := os.Stat(filepath.Join(dir, marker))
		if err != nil {
			continue
		}
		if marker == "server" && !info.IsDir() {
			continue
		}
		return true
	}

	return false
}
