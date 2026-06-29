package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

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
