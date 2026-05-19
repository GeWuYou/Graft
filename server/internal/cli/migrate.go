package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/pluginregistry"
)

// defaultMigrationDir 定义 server 模块内 Atlas 版本化迁移目录的默认相对路径。
const defaultMigrationDir = pluginregistry.DefaultCoreMigrationDir

// 这些变量保留为可替换的命令边界，便于测试覆盖 Atlas 查找、子进程执行和
// 当前工作目录解析，而不把真实系统依赖硬编码到测试中。
var migrateLookPath = exec.LookPath
var migrateCommandContext = exec.CommandContext
var migrateGetwd = os.Getwd
var migrateStdin io.Reader = os.Stdin
var migrateRegistryMigrationDirs = pluginregistry.MigrationDirs
var migrateReadDir = os.ReadDir

// migrateUpOptions 封装一次显式迁移执行所需的输入。
type migrateUpOptions struct {
	migrationDir string
	workingDir   string
}

// newMigrateCommand 创建显式数据库迁移命令树。
//
// 迁移能力保持在独立的 `graft migrate` 子树下，避免普通运行时启动路径
// 隐式修改数据库 schema。
func newMigrateCommand() *cobra.Command {
	var migrationDir string

	command := &cobra.Command{
		Use:   "migrate",
		Short: "Run explicit database migration commands",
	}
	command.PersistentFlags().StringVar(&migrationDir, "dir", defaultMigrationDir, "migration directory")

	command.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply pending Atlas versioned migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrateUp(cmd, migrateUpOptions{migrationDir: migrationDir})
		},
	})

	return command
}

// runMigrateUp 执行一次 Atlas 版本化迁移应用。
//
// 参数：
//   - cmd: 当前 Cobra 命令，用于继承上下文和标准输入输出。
//   - opts: 迁移目录与工作目录等显式执行选项。
//
// 返回值：
//   - error: 当配置加载、迁移目录解析、Atlas 查找或迁移执行失败时返回错误。
func runMigrateUp(cmd *cobra.Command, opts migrateUpOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	workingDir := opts.workingDir
	if strings.TrimSpace(workingDir) == "" {
		workingDir, err = migrateGetwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}
	}

	absDirs, err := resolveMigrationDirs(workingDir, opts.migrationDir)
	if err != nil {
		return fmt.Errorf("resolve migration dir: %w", err)
	}

	atlasPath, err := findAtlasCLI()
	if err != nil {
		return err
	}

	commandContext := cmd.Context()
	if commandContext == nil {
		commandContext = context.Background()
	}

	for _, absDir := range absDirs {
		command := migrateCommandContext(
			commandContext,
			atlasPath,
			"migrate",
			"apply",
			"--dir", "file://"+filepath.ToSlash(absDir),
			"--url", cfg.Database.URL,
		)
		command.Stdout = cmd.OutOrStdout()
		command.Stderr = cmd.ErrOrStderr()
		command.Stdin = migrateStdin

		if err := command.Run(); err != nil {
			return fmt.Errorf("apply atlas migrations from %s: %w", absDir, err)
		}
	}

	return nil
}

// findAtlasCLI 解析本地可执行的 Atlas CLI 路径。
//
// 如果 Atlas 不存在，这里直接返回面向开发者的下一步提示，明确哪些命令
// 依赖迁移工具，哪些命令只适用于 schema 已经同步的场景。
func findAtlasCLI() (string, error) {
	atlasPath, err := migrateLookPath("atlas")
	if err == nil {
		return atlasPath, nil
	}

	return "", fmt.Errorf(
		"atlas CLI is required for `graft migrate up` and `graft dev`; install Atlas first, or run `graft serve` only after the database schema is already up to date: %w",
		err,
	)
}

// resolveMigrationDirs 从当前目录向上搜索可用的迁移目录集合。
//
// 默认目录不再直接等同于单个 core 迁移路径；它会先回到 compile-time
// registry 读取当前进程声明的完整目录集合，再逐一解析为绝对路径。
func resolveMigrationDirs(baseDir string, migrationDir string) ([]string, error) {
	if strings.TrimSpace(migrationDir) == "" {
		return nil, fmt.Errorf("migration dir is required")
	}

	includeAllResolvedDirs := true
	searchDirs := []string{migrationDir}
	if migrationDir == defaultMigrationDir {
		var err error
		searchDirs, err = migrateRegistryMigrationDirs()
		if err != nil {
			return nil, fmt.Errorf("load compile-time migration registry: %w", err)
		}
		includeAllResolvedDirs = false
	}

	resolved := make([]string, 0, len(searchDirs))
	for _, current := range searchDirs {
		absDir, err := resolveMigrationDir(baseDir, current)
		if err != nil {
			if shouldSkipMissingMigrationDir(includeAllResolvedDirs, err) {
				continue
			}
			return nil, err
		}

		resolved, err = appendResolvedMigrationDir(resolved, absDir, includeAllResolvedDirs)
		if err != nil {
			return nil, err
		}
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("no migration directories with atlas state found in compile-time registry")
	}

	return resolved, nil
}

func shouldSkipMissingMigrationDir(includeAllResolvedDirs bool, err error) bool {
	return !includeAllResolvedDirs && errors.Is(err, os.ErrNotExist)
}

func appendResolvedMigrationDir(resolved []string, absDir string, includeAllResolvedDirs bool) ([]string, error) {
	if includeAllResolvedDirs {
		return append(resolved, absDir), nil
	}

	hasAtlasState, err := directoryContainsAtlasState(absDir)
	if err != nil {
		return nil, err
	}
	if !hasAtlasState {
		return resolved, nil
	}

	return append(resolved, absDir), nil
}

// resolveMigrationDir 从当前目录向上搜索可用的单个迁移目录。
//
// 默认目录同时支持仓库根目录和 `server` 模块根目录两种工作目录，减少 IDE、
// Shell 和测试环境切换时对单一 cwd 约定的依赖。
func resolveMigrationDir(baseDir string, migrationDir string) (string, error) {
	if strings.TrimSpace(migrationDir) == "" {
		return "", fmt.Errorf("migration dir is required")
	}

	searchDirs := []string{migrationDir}
	if !strings.HasPrefix(filepath.ToSlash(migrationDir), "server/") {
		searchDirs = append(searchDirs, filepath.Join("server", migrationDir))
	}

	current := baseDir
	for {
		for _, relativeDir := range searchDirs {
			candidate := filepath.Join(current, relativeDir)
			info, err := os.Stat(candidate)
			if err == nil {
				if !info.IsDir() {
					return "", fmt.Errorf("migration dir %s is not a directory", candidate)
				}

				absDir, err := filepath.Abs(candidate)
				if err != nil {
					return "", fmt.Errorf("resolve migration dir %s: %w", candidate, err)
				}

				return absDir, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("cannot find migration dir %q from %s: %w", migrationDir, baseDir, os.ErrNotExist)
}

func directoryContainsAtlasState(absDir string) (bool, error) {
	entries, err := migrateReadDir(absDir)
	if err != nil {
		return false, fmt.Errorf("read migration dir %s: %w", absDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == "atlas.sum" {
			return true, nil
		}
	}

	return false, nil
}
