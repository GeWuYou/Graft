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

type migrationDirResolutionPlan struct {
	dirs                   []string
	alreadyResolved        bool
	includeAllResolvedDirs bool
}

// buildAtlasMigrationDir 根据迁移目录规范构造 Atlas 迁移目录。
// buildAtlasMigrationDir 根据输入构建迁移目录，支持默认链、仓库拥有目录和外部路径。
// @param baseDir 用于解析外部路径的基准目录。
// @param migrationDir 迁移目录输入，支持默认值、仓库选择器或以 `file:` 前缀指定的外部路径。
// @returns 解析后的迁移目录或错误。
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

// buildDefaultAtlasMigrationDir 从编译期注册表加载包含 Atlas 状态的迁移目录，并将其合成为单一迁移目录。
// 它会跳过不包含 Atlas 状态的源目录，并在合成前后校验迁移目录有效性。
// 返回合成后的迁移目录，或在注册表加载、源目录校验、合成失败时返回错误。
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
// parseMigrationDirInput 解析迁移目录输入并识别其类型。
// 支持默认迁移目录、仓库内拥有的迁移选择器以及显式外部路径；输入会先去除首尾空白并规范化路径分隔符。
// 
// @param migrationDir 迁移目录输入。
// @returns 解析后的迁移目录输入及错误；当输入为空、外部路径缺少显式前缀，或以 server/ 开头但未使用允许的选择器格式时返回错误。
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
// isRepoOwnedMigrationSelector 判断迁移目录选择器是否属于仓库内置路径。
// 当路径以前缀 "modules/" 或 "internal/" 开头时，返回 `true`。
func isRepoOwnedMigrationSelector(migrationDir string) bool {
	return strings.HasPrefix(migrationDir, "modules/") || strings.HasPrefix(migrationDir, "internal/")
}

// loadRepoOwnedAtlasMigrationDir 从仓库拥有的迁移源加载迁移目录。
// 
// @param migrationDir 仓库内迁移目录的选择器。
// @returns 加载后的迁移目录。
func loadRepoOwnedAtlasMigrationDir(migrationDir string) (atlasmigrate.Dir, error) {
	source, err := loadRepoOwnedMigrationDirSource(migrationDir)
	if err != nil {
		return nil, err
	}
	return source.dir, nil
}

// loadRepoOwnedMigrationDirSource 从编译期嵌入资源中加载仓库拥有的迁移目录源。
// 如果对应的嵌入迁移目录不可用，则返回错误。
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
// 成功时返回包含该目录文件的内存迁移源，以及是否找到了对应的嵌入式资源。
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
// loadExternalAtlasMigrationDir 解析外部迁移目录并打开对应的本地迁移目录。
// 它会先将 externalPath 解析为绝对路径，然后以该路径创建 atlasmigrate.Dir。
য
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
// resolveExternalMigrationDir 将外部迁移目录解析为绝对路径。
// 目录相对于 baseDir 解析；如果目录不存在、不是目录或无法解析为绝对路径，则返回错误。
// @param baseDir 用于解析相对路径的基准目录。
// @param externalPath 外部迁移目录路径。
// @returns 解析后的绝对目录路径。
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
// synthesizeDefaultMigrationDir 合并多个迁移目录源并生成默认迁移链。
// 它会拒绝重复的文件名或版本号；如果合并结果不包含任何 SQL 迁移文件，返回错误。
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

// copyMigrationSourceFiles 将源目录中的迁移文件复制到内存目录，并检查文件名和版本号是否重复。
// 它会跳过 Atlas 状态哈希文件，只复制迁移文件。
//
// 返回复制的文件数量。
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
		if file.Name() == atlasmigrate.HashFileName {
			continue
		}
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

// 验证通过时，会将该版本号记录到 copiedVersions 中。
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

// migrationFileVersion 提取迁移文件名中的前导数字版本。
// 如果文件名符合迁移命名模式，则返回版本号；否则返回空字符串。
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
// 对仓库拥有和外部输入，会按解析结果返回对应目录路径。
func resolveMigrationDirs(baseDir string, migrationDir string) ([]string, error) {
	input, err := parseMigrationDirInput(migrationDir)
	if err != nil {
		return nil, err
	}

	plan, err := planMigrationDirResolution(baseDir, input)
	if err != nil {
		return nil, err
	}
	if plan.alreadyResolved {
		return plan.dirs, nil
	}

	resolved := make([]string, 0, len(plan.dirs))
	for _, current := range plan.dirs {
		absDir, err := resolveMigrationDir(baseDir, current)
		if err != nil {
			if shouldSkipMissingMigrationDir(plan.includeAllResolvedDirs, err) {
				continue
			}
			return nil, err
		}

		resolved, err = appendResolvedMigrationDir(resolved, absDir, plan.includeAllResolvedDirs)
		if err != nil {
			return nil, err
		}
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("no migration directories with atlas state found in compile-time registry")
	}

	return resolved, nil
}

// planMigrationDirResolution 根据迁移目录输入构建解析计划。
// 默认输入会读取编译期迁移注册表；仓库拥有输入保留选择器以供后续解析；外部输入会先解析为绝对目录。
// 返回值中的标志位用于控制后续是否需要继续解析，以及是否保留所有已解析目录。
func planMigrationDirResolution(baseDir string, input migrationDirInput) (migrationDirResolutionPlan, error) {
	switch input.kind {
	case migrationDirInputKindDefault:
		searchDirs, err := migrateRegistryMigrationDirs()
		if err != nil {
			return migrationDirResolutionPlan{}, fmt.Errorf("load compile-time migration registry: %w", err)
		}
		return migrationDirResolutionPlan{
			dirs:                   searchDirs,
			includeAllResolvedDirs: false,
		}, nil
	case migrationDirInputKindRepoOwned:
		return migrationDirResolutionPlan{
			dirs:                   []string{input.selector},
			includeAllResolvedDirs: true,
		}, nil
	case migrationDirInputKindExternal:
		absDir, err := resolveExternalMigrationDir(baseDir, input.externalPath)
		if err != nil {
			return migrationDirResolutionPlan{}, err
		}
		return migrationDirResolutionPlan{
			dirs:            []string{absDir},
			alreadyResolved: true,
		}, nil
	default:
		return migrationDirResolutionPlan{}, fmt.Errorf("unsupported migration dir input %q", input.displayValue)
	}
}

// shouldSkipMissingMigrationDir 判断是否忽略缺失迁移目录的错误。
// 当未要求包含全部已解析目录且错误可匹配 `os.ErrNotExist` 时，返回 `true`。
func shouldSkipMissingMigrationDir(includeAllResolvedDirs bool, err error) bool {
	return !includeAllResolvedDirs && errors.Is(err, os.ErrNotExist)
}

// appendResolvedMigrationDir 将解析后的迁移目录追加到结果中。
// 当 includeAllResolvedDirs 为 false 时，仅在目录包含 Atlas 状态文件时才追加。
// 返回更新后的目录列表。
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
// resolveMigrationDir 从给定基目录向上查找并解析迁移目录的绝对路径。
// 如果迁移目录名不以 server/ 开头，还会同时尝试 server/ 前缀路径。
// 返回解析到的绝对目录路径，或在未找到、路径不是目录时返回错误。
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

// directoryContainsAtlasState reports whether absDir contains the Atlas state file.
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
