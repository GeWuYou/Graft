package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/moduleregistry"
)

// defaultMigrationDir 定义 `server` 模块默认迁移链使用的 registry 选择器。
const defaultMigrationDir = moduleregistry.DefaultMigrationDir

const migrationFileMode = 0o600
const migrationVersionMatchCount = 2

var migrationVersionPattern = regexp.MustCompile(`^(\d+)_.*\.sql$`)

// 这些变量保留为可替换的命令边界，便于测试覆盖 Atlas 查找、子进程执行和
// 当前工作目录解析，而不把真实系统依赖硬编码到测试中。
var migrateLookPath = exec.LookPath
var migrateCommandContext = exec.CommandContext
var migrateGetwd = os.Getwd
var migrateStdin io.Reader = os.Stdin
var migrateRegistryMigrationDirs = moduleregistry.MigrationDirs
var migrateReadDir = os.ReadDir
var migrateReadFile = os.ReadFile
var migrateWriteFile = os.WriteFile
var migrateMkdirTemp = os.MkdirTemp
var migrateRemoveAll = os.RemoveAll

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
	command.PersistentFlags().StringVar(&migrationDir, "dir", defaultMigrationDir, "migration directory or owner-aligned default chain")

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

	absDirs, cleanup, err := prepareMigrationDirs(commandContext, atlasPath, opts.migrationDir, absDirs)
	if err != nil {
		return err
	}
	defer cleanup()

	return applyAtlasMigrations(commandContext, atlasPath, cmd, cfg.Database.URL, absDirs)
}

func synthesizeDefaultMigrationDir(commandContext context.Context, atlasPath string, sourceDirs []string) (string, func(), error) {
	tempDir, err := migrateMkdirTemp("", "graft-atlas-default-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary default migration dir: %w", err)
	}

	cleanup := func() {
		_ = migrateRemoveAll(tempDir)
	}

	if err := copyMigrationFilesIntoDir(tempDir, sourceDirs); err != nil {
		cleanup()
		return "", nil, err
	}

	var hashStderr strings.Builder
	if err := runAtlasMigrationCommand(
		commandContext,
		atlasPath,
		nil,
		&hashStderr,
		"hash",
		"--dir", "file://"+filepath.ToSlash(tempDir),
	); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("hash synthesized default migration dir %s: %w", tempDir, wrapAtlasCommandError(err, hashStderr.String()))
	}

	return tempDir, cleanup, nil
}

func prepareMigrationDirs(commandContext context.Context, atlasPath string, migrationDir string, absDirs []string) ([]string, func(), error) {
	if migrationDir != defaultMigrationDir {
		return absDirs, func() {}, nil
	}

	defaultDir, cleanup, err := synthesizeDefaultMigrationDir(commandContext, atlasPath, absDirs)
	if err != nil {
		return nil, nil, err
	}

	return []string{defaultDir}, cleanup, nil
}

func applyAtlasMigrations(commandContext context.Context, atlasPath string, cmd *cobra.Command, databaseURL string, absDirs []string) error {
	for _, absDir := range absDirs {
		if err := runAtlasMigrationCommand(
			commandContext,
			atlasPath,
			cmd,
			nil,
			"apply",
			"--dir", "file://"+filepath.ToSlash(absDir),
			"--url", databaseURL,
		); err != nil {
			return fmt.Errorf("apply atlas migrations from %s: %w", absDir, err)
		}
	}

	return nil
}

func copyMigrationFilesIntoDir(targetDir string, sourceDirs []string) error {
	copiedAny := false
	copiedNames := make(map[string]string, len(sourceDirs))
	copiedVersions := make(map[string]string, len(sourceDirs))

	for _, sourceDir := range sourceDirs {
		copied, err := copyMigrationFilesFromSource(targetDir, sourceDir, copiedNames, copiedVersions)
		if err != nil {
			return err
		}
		copiedAny = copiedAny || copied
	}

	if !copiedAny {
		return fmt.Errorf("default migration chain has no SQL migration files")
	}

	return nil
}

func copyMigrationFilesFromSource(targetDir string, sourceDir string, copiedNames map[string]string, copiedVersions map[string]string) (bool, error) {
	entries, err := migrateReadDir(sourceDir)
	if err != nil {
		return false, fmt.Errorf("read migration dir %s: %w", sourceDir, err)
	}

	copiedAny := false
	for _, entry := range entries {
		copied, err := copyMigrationFileEntry(targetDir, sourceDir, entry, copiedNames, copiedVersions)
		if err != nil {
			return false, err
		}
		copiedAny = copiedAny || copied
	}

	return copiedAny, nil
}

func copyMigrationFileEntry(targetDir string, sourceDir string, entry os.DirEntry, copiedNames map[string]string, copiedVersions map[string]string) (bool, error) {
	if entry.IsDir() {
		return false, nil
	}

	name := entry.Name()
	if filepath.Ext(name) != ".sql" {
		return false, nil
	}

	if previousSource, exists := copiedNames[name]; exists {
		return false, fmt.Errorf("duplicate migration filename %s from %s and %s", name, previousSource, sourceDir)
	}
	if version := migrationFileVersion(name); version != "" {
		if previousSource, exists := copiedVersions[version]; exists {
			return false, fmt.Errorf("duplicate migration version %s from %s and %s", version, previousSource, sourceDir)
		}
		copiedVersions[version] = sourceDir
	}

	if err := copyMigrationFile(targetDir, sourceDir, name); err != nil {
		return false, err
	}

	copiedNames[name] = sourceDir
	return true, nil
}

func copyMigrationFile(targetDir string, sourceDir string, name string) error {
	sourcePath := filepath.Join(sourceDir, name)
	content, err := migrateReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read migration file %s: %w", sourcePath, err)
	}

	targetPath := filepath.Join(targetDir, name)
	if err := migrateWriteFile(targetPath, content, migrationFileMode); err != nil {
		return fmt.Errorf("write synthesized migration file %s: %w", targetPath, err)
	}

	return nil
}

func migrationFileVersion(name string) string {
	matches := migrationVersionPattern.FindStringSubmatch(name)
	if len(matches) != migrationVersionMatchCount {
		return ""
	}

	return matches[1]
}

func runAtlasMigrationCommand(commandContext context.Context, atlasPath string, cmd *cobra.Command, stderrCapture io.Writer, args ...string) error {
	command := migrateCommandContext(commandContext, atlasPath, append([]string{"migrate"}, args...)...)

	stdout := io.Discard
	stderr := io.Discard
	if cmd != nil {
		stdout = cmd.OutOrStdout()
		stderr = cmd.ErrOrStderr()
	}
	if stderrCapture != nil {
		stderr = io.MultiWriter(stderr, stderrCapture)
	}

	command.Stdout = stdout
	command.Stderr = stderr
	command.Stdin = migrateStdin

	return command.Run()
}

func wrapAtlasCommandError(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}

	return fmt.Errorf("%w: %s", err, stderr)
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
