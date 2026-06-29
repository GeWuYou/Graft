package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	atlasmigrate "ariga.io/atlas/sql/migrate"

	"graft/server/internal/moduleregistry"
)

var migrationVersionPattern = regexp.MustCompile(`^(\d+)_.*\.sql$`)

type migrationDirInputKind int

const (
	migrationDirInputKindDefault migrationDirInputKind = iota
	migrationDirInputKindRepoOwned
	migrationDirInputKindExternal
)

type migrationDirInput struct {
	kind         migrationDirInputKind
	displayValue string
	selector     string
	externalPath string
}

type migrationDirSource struct {
	path          string
	dir           atlasmigrate.Dir
	hasAtlasState bool
}

// buildAtlasMigrationDir 根据迁移目录规范构造 Atlas 迁移目录。
// 规范可指定默认链、仓库拥有目录或外部路径（"file:" 前缀）。baseDir 用于解析外部路径。
func buildAtlasMigrationDir(baseDir string, migrationDir string) (atlasmigrate.Dir, error) {
	input, err := parseMigrationDirInput(migrationDir)
	if err != nil {
		return nil, err
	}

	switch input.kind {
	case migrationDirInputKindDefault:
		return buildDefaultAtlasMigrationDir()
	case migrationDirInputKindRepoOwned:
		return loadRepoOwnedAtlasMigrationDir(input.selector)
	case migrationDirInputKindExternal:
		return loadExternalAtlasMigrationDir(baseDir, input.externalPath)
	default:
		return nil, fmt.Errorf("unsupported migration dir input %q", input.displayValue)
	}
}

// buildDefaultAtlasMigrationDir 从编译期注册表加载迁移源，筛选出包含 Atlas 状态的源目录，并将其合成为单一迁移目录。
func buildDefaultAtlasMigrationDir() (atlasmigrate.Dir, error) {
	searchDirs, err := migrateRegistryMigrationDirs()
	if err != nil {
		return nil, fmt.Errorf("load compile-time migration registry: %w", err)
	}

	sources := make([]migrationDirSource, 0, len(searchDirs))
	for _, current := range searchDirs {
		source, err := loadRepoOwnedMigrationDirSource(current)
		if err != nil {
			return nil, err
		}
		if !source.hasAtlasState {
			continue
		}
		if err := atlasmigrate.Validate(source.dir); err != nil {
			return nil, fmt.Errorf("validate migration dir %s: %w", source.path, err)
		}
		sources = append(sources, source)
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no migration directories with atlas state found in compile-time registry")
	}

	dir, err := synthesizeDefaultMigrationDir(sources)
	if err != nil {
		return nil, err
	}
	if err := atlasmigrate.Validate(dir); err != nil {
		return nil, fmt.Errorf("validate synthesized default migration dir: %w", err)
	}
	return dir, nil
}

// parseMigrationDirInput parses a migration directory input string into a categorized migration directory specification.
// It recognizes three input kinds: external paths prefixed with "file:", the default migration directory, and repository-owned selectors starting with "modules/" or "internal/".
// It returns an error if the input is empty, uses server-prefixed paths without explicit prefixes, or lacks required prefixes for external paths.
func parseMigrationDirInput(migrationDir string) (migrationDirInput, error) {
	trimmed := strings.TrimSpace(migrationDir)
	if trimmed == "" {
		return migrationDirInput{}, fmt.Errorf("migration dir is required")
	}

	if strings.HasPrefix(trimmed, externalMigrationDirPrefix) {
		externalPath := strings.TrimSpace(strings.TrimPrefix(trimmed, externalMigrationDirPrefix))
		if externalPath == "" {
			return migrationDirInput{}, fmt.Errorf("external migration dir path is required after %q", externalMigrationDirPrefix)
		}
		return migrationDirInput{
			kind:         migrationDirInputKindExternal,
			displayValue: trimmed,
			externalPath: externalPath,
		}, nil
	}

	normalized := filepath.ToSlash(trimmed)
	if normalized == defaultMigrationDir {
		return migrationDirInput{
			kind:         migrationDirInputKindDefault,
			displayValue: trimmed,
			selector:     defaultMigrationDir,
		}, nil
	}

	if isRepoOwnedMigrationSelector(normalized) {
		return migrationDirInput{
			kind:         migrationDirInputKindRepoOwned,
			displayValue: trimmed,
			selector:     normalized,
		}, nil
	}

	if strings.HasPrefix(normalized, "server/modules/") || strings.HasPrefix(normalized, "server/internal/") {
		return migrationDirInput{}, fmt.Errorf(
			"repo-owned migration selector %q must use owner-aligned path without \"server/\" or explicit %s prefix",
			trimmed,
			externalMigrationDirPrefix,
		)
	}

	return migrationDirInput{}, fmt.Errorf(
		"external migration dir %q must use explicit %s prefix",
		trimmed,
		externalMigrationDirPrefix,
	)
}

// isRepoOwnedMigrationSelector reports whether migrationDir is a repository-owned selector.
//
// A directory is repository-owned if it starts with "modules/" or "internal/".
func isRepoOwnedMigrationSelector(migrationDir string) bool {
	return strings.HasPrefix(migrationDir, "modules/") || strings.HasPrefix(migrationDir, "internal/")
}

// loadRepoOwnedAtlasMigrationDir 从仓库拥有的迁移源加载 Atlas 迁移目录。
func loadRepoOwnedAtlasMigrationDir(migrationDir string) (atlasmigrate.Dir, error) {
	source, err := loadRepoOwnedMigrationDirSource(migrationDir)
	if err != nil {
		return nil, err
	}
	return source.dir, nil
}

// LoadRepoOwnedMigrationDirSource loads a repo-owned migration directory from compile-time embedded sources. It returns an error if the embedded migration directory is not available in the registry.
func loadRepoOwnedMigrationDirSource(migrationDir string) (migrationDirSource, error) {
	embedded, found, err := loadEmbeddedMigrationDirSource(migrationDir)
	if err != nil {
		return migrationDirSource{}, err
	}
	if !found {
		return migrationDirSource{}, fmt.Errorf(
			"compile-time embedded migration dir %q is not available; regenerate registry assets or use %s<path> for an explicit external directory",
			migrationDir,
			externalMigrationDirPrefix,
		)
	}
	return embedded, nil
}

// loadEmbeddedMigrationDirSource 从编译期注册表加载指定迁移目录的嵌入式迁移文件到内存。
// 若不存在返回 false，若成功加载返回包含文件的迁移源及 true。
func loadEmbeddedMigrationDirSource(migrationDir string) (migrationDirSource, bool, error) {
	embedded, ok := migrateEmbeddedMigrationDirByPath(migrationDir)
	if !ok {
		return migrationDirSource{}, false, nil
	}

	dir := &atlasmigrate.MemDir{}
	for _, file := range embedded.Files {
		if err := dir.WriteFile(file.Name, file.Contents); err != nil {
			return migrationDirSource{}, false, fmt.Errorf("write embedded migration file %s/%s: %w", migrationDir, file.Name, err)
		}
	}

	return migrationDirSource{
		path:          migrationDir,
		dir:           dir,
		hasAtlasState: embeddedMigrationDirHasAtlasState(embedded),
	}, true, nil
}

// loadExternalAtlasMigrationDir 加载外部文件系统路径中的迁移目录。
// 若目录打开成功,返回 atlasmigrate.Dir;否则返回错误。
func loadExternalAtlasMigrationDir(baseDir string, externalPath string) (atlasmigrate.Dir, error) {
	absDir, err := resolveExternalMigrationDir(baseDir, externalPath)
	if err != nil {
		return nil, err
	}

	dir, err := atlasmigrate.NewLocalDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("open migration dir %s: %w", absDir, err)
	}
	return dir, nil
}

// resolveExternalMigrationDir 解析 externalPath 为绝对目录路径，
// 若其为相对路径，则相对于 baseDir 进行合并。
// 若目录不存在或非目录，则返回错误。
func resolveExternalMigrationDir(baseDir string, externalPath string) (string, error) {
	candidate := externalPath
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(baseDir, candidate)
	}

	info, err := os.Stat(candidate)
	if err != nil {
		return "", fmt.Errorf("cannot find migration dir %q from %s: %w", externalPath, baseDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("migration dir %s is not a directory", candidate)
	}

	absDir, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve migration dir %s: %w", candidate, err)
	}
	return absDir, nil
}

// embeddedMigrationDirHasAtlasState 报告嵌入式迁移目录中是否存在 Atlas 哈希文件。
func embeddedMigrationDirHasAtlasState(dir moduleregistry.EmbeddedMigrationDir) bool {
	for _, file := range dir.Files {
		if file.Name == atlasmigrate.HashFileName {
			return true
		}
	}
	return false
}

// synthesizeDefaultMigrationDir 将多个迁移源目录合并到单个内存目录中，并计算该目录的校验和。
// 合并过程中会检测并拒绝文件名和版本号的重复。如果合并后的目录不包含任何 SQL 迁移文件，返回错误。
func synthesizeDefaultMigrationDir(sourceDirs []migrationDirSource) (atlasmigrate.Dir, error) {
	memDir := &atlasmigrate.MemDir{}
	copiedNames := make(map[string]string, len(sourceDirs))
	copiedVersions := make(map[string]string, len(sourceDirs))
	totalCopied := 0

	for _, sourceDir := range sourceDirs {
		copied, err := copyMigrationSourceFiles(memDir, sourceDir, copiedNames, copiedVersions)
		if err != nil {
			return nil, err
		}
		totalCopied += copied
	}
	if totalCopied == 0 {
		return nil, fmt.Errorf("default migration chain has no SQL migration files")
	}
	sum, err := memDir.Checksum()
	if err != nil {
		return nil, fmt.Errorf("compute synthesized migration checksum: %w", err)
	}
	if err := atlasmigrate.WriteSumFile(memDir, sum); err != nil {
		return nil, fmt.Errorf("write synthesized migration checksum: %w", err)
	}

	return memDir, nil
}

// CopyMigrationSourceFiles copies migration files from a source directory to a memory directory, validating against duplicate filenames and versions. It returns the number of files copied and any error encountered.
func copyMigrationSourceFiles(
	memDir *atlasmigrate.MemDir,
	sourceDir migrationDirSource,
	copiedNames map[string]string,
	copiedVersions map[string]string,
) (int, error) {
	files, err := sourceDir.dir.Files()
	if err != nil {
		return 0, fmt.Errorf("read migration dir %s: %w", sourceDir.path, err)
	}

	copiedCount := 0
	for _, file := range files {
		if err := validateSynthesizedMigrationFile(sourceDir.path, file.Name(), copiedNames, copiedVersions); err != nil {
			return 0, err
		}
		if err := memDir.WriteFile(file.Name(), file.Bytes()); err != nil {
			return 0, fmt.Errorf("write synthesized migration file %s: %w", file.Name(), err)
		}
		copiedNames[file.Name()] = sourceDir.path
		copiedCount++
	}

	return copiedCount, nil
}

// validateSynthesizedMigrationFile 验证迁移文件不存在重复的文件名或版本号。sourcePath 为该文件所来自的源目录路径。copiedNames 记录已复制的文件名，copiedVersions 记录已复制的版本号。若验证通过，该函数将版本号记录到 copiedVersions 中；若发现重复的文件名或版本号，返回相应的错误。
func validateSynthesizedMigrationFile(
	sourcePath string,
	name string,
	copiedNames map[string]string,
	copiedVersions map[string]string,
) error {
	if previousSource, exists := copiedNames[name]; exists {
		return fmt.Errorf("duplicate migration filename %s from %s and %s", name, previousSource, sourcePath)
	}
	if version := migrationFileVersion(name); version != "" {
		if previousSource, exists := copiedVersions[version]; exists {
			return fmt.Errorf("duplicate migration version %s from %s and %s", version, previousSource, sourcePath)
		}
		copiedVersions[version] = sourcePath
	}

	return nil
}

// migrationFileVersion extracts the leading numeric version from a migration filename. It returns the version number if the filename matches the migration pattern, or an empty string if the filename does not match.
func migrationFileVersion(name string) string {
	matches := migrationVersionPattern.FindStringSubmatch(name)
	if len(matches) != migrationVersionMatchCount {
		return ""
	}

	return matches[1]
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

// directoryContainsAtlasState reports whether a directory contains an Atlas state file.
func directoryContainsAtlasState(absDir string) (bool, error) {
	entries, err := migrateReadDir(absDir)
	if err != nil {
		return false, fmt.Errorf("read migration dir %s: %w", absDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == atlasmigrate.HashFileName {
			return true, nil
		}
	}

	return false, nil
}
