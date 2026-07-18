package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func (s *Service) scanDiscoveryCandidates(
	ctx context.Context,
	repository projectstore.Repository,
	rootDirectory string,
	configKey string,
) ([]DiscoveryCandidateResult, error) {
	entries, err := os.ReadDir(rootDirectory)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errProjectImportValidation, err)
	}
	candidates := make([]DiscoveryCandidateResult, 0, projectDiscoveryScanSize)
	for _, entry := range entries {
		if len(candidates) >= projectDiscoveryScanSize {
			break
		}
		name, ok := visibleDirectoryEntryName(entry)
		if !ok {
			continue
		}
		candidate, err := s.buildDiscoveryCandidate(ctx, repository, rootDirectory, name, configKey)
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return strings.Compare(candidates[i].WorkspacePath, candidates[j].WorkspacePath) < 0
	})
	return candidates, nil
}

// buildImportDirectoryItems 构建可浏览的子目录条目列表。
// 仅包含目录条目，并为每个条目填充规范化路径；如果可获取修改时间，则同时记录其 UTC 时间。
func buildImportDirectoryItems(currentPath string, entries []os.DirEntry) []ImportDirectoryItem {
	items := make([]ImportDirectoryItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		item := ImportDirectoryItem{
			Name: name,
			Path: normalizeBrowsePath(filepath.Join(currentPath, name)),
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			modifiedAt := info.ModTime().UTC()
			item.ModifiedAt = &modifiedAt
		}
		items = append(items, item)
	}
	return items
}

// sortImportDirectoryItems 按指定字段和顺序排列导入目录项。
// 当按修改时间排序时，会优先比较修改时间；无法区分时按名称排序。
func sortImportDirectoryItems(items []ImportDirectoryItem, sortBy string, order string) {
	sort.Slice(items, func(i, j int) bool {
		if sortBy == importDirectorySortByModified {
			if decided, ok := compareModifiedTime(items[i].ModifiedAt, items[j].ModifiedAt, order); ok {
				return decided
			}
		}
		return compareDirectoryNames(items[i].Name, items[j].Name, order)
	})
}

// compareModifiedTime 根据修改时间和排序方向比较两个时间。
// 左右时间为空时按零值时间处理；当时间相同时返回 false, false。
// 返回的第一个值表示左侧是否应排在右侧之前，第二个值表示两者是否可比较。
func compareModifiedTime(leftAt *time.Time, rightAt *time.Time, order string) (bool, bool) {
	left := time.Time{}
	right := time.Time{}
	if leftAt != nil {
		left = *leftAt
	}
	if rightAt != nil {
		right = *rightAt
	}
	if left.Equal(right) {
		return false, false
	}
	if order == importDirectoryOrderDesc {
		return left.After(right), true
	}
	return left.Before(right), true
}

// compareDirectoryNames 按指定顺序比较两个目录名的先后。
//
// 当 order 为降序时，左值大于右值返回 true；否则左值小于右值返回 true。
func compareDirectoryNames(left string, right string, order string) bool {
	if order == importDirectoryOrderDesc {
		return strings.Compare(left, right) > 0
	}
	return strings.Compare(left, right) < 0
}

// visibleDirectoryEntryName 返回可见目录项的名称。
//
// 它仅接受目录项，并过滤掉空名称、以 `.` 开头的名称以及非目录项。
// 当目录项可用时返回其修剪后的名称。
func visibleDirectoryEntryName(entry os.DirEntry) (string, bool) {
	if !entry.IsDir() {
		return "", false
	}
	name := strings.TrimSpace(entry.Name())
	if name == "" || strings.HasPrefix(name, ".") {
		return "", false
	}
	return name, true
}

func (s *Service) buildDiscoveryCandidate(
	ctx context.Context,
	repository projectstore.Repository,
	rootDirectory string,
	name string,
	configKey string,
) (DiscoveryCandidateResult, error) {
	workingDirectory := filepath.Join(rootDirectory, name)
	discovered, err := discoverImportFiles(workingDirectory)
	if err != nil {
		return DiscoveryCandidateResult{}, err
	}
	session, err := s.inspectImportRequest(ctx, repository, ImportRequest{
		WorkspacePath: workingDirectory,
		ComposeFiles:  discovered.composeFiles,
		EnvFiles:      discovered.envFiles,
	})
	if err != nil {
		return DiscoveryCandidateResult{}, err
	}
	if len(discovered.warnings) > 0 {
		session.Warnings = append(session.Warnings, discovered.warnings...)
	}
	status, recommendedAction, statusReason := discoveryCandidateStatus(session.Conflicts)
	return DiscoveryCandidateResult{
		CandidateKey:  candidateKeyForWorkspacePath(workingDirectory),
		CandidateKind: "directory-scan",
		SourceType:    projectcontract.SourceTypeManaged.String(),
		SourceMetadata: map[string]string{
			"managed_root_key":           configKey,
			"managed_relative_directory": name,
			"managed_compose_file_name":  firstProjectFileDisplayName(toGeneratedFilesFromCompose(session.ParseResult.ComposeFiles)),
			"managed_env_file_name":      firstProjectFileDisplayName(toGeneratedFilesFromCompose(session.ParseResult.EnvFiles)),
		},
		DisplayName:              session.CanonicalName,
		ComposeProjectName:       session.CanonicalName,
		ComposeProjectNameSource: session.CanonicalSource,
		WorkspacePath:            session.WorkingDir,
		OwnershipMode:            projectcontract.OwnershipModeManagedRootDedicated.String(),
		Status:                   status,
		RecommendedAction:        recommendedAction,
		StatusReason:             statusReason,
		ComposeFiles:             toGeneratedFilesFromCompose(session.ParseResult.ComposeFiles),
		EnvFiles:                 toGeneratedFilesFromCompose(session.ParseResult.EnvFiles),
		DeclaredServiceNames:     append([]string(nil), session.ParseResult.ServiceNames...),
		ServiceCount:             len(session.ParseResult.ServiceNames),
		ConfigHash:               session.ParseResult.ConfigHash,
		Warnings:                 append([]string(nil), session.Warnings...),
		Conflicts:                append([]string(nil), session.Conflicts...),
	}, nil
}

// discoveryCandidateStatus 根据冲突列表返回发现候选的状态、建议操作和状态原因。
// 当不存在冲突时返回“ready”和“import”；当存在冲突时返回“conflict”和“review”，并提供原因说明。
// @returns 状态字符串、建议操作字符串，以及在存在冲突时指向状态原因的指针。
func discoveryCandidateStatus(conflicts []string) (string, string, *string) {
	if len(conflicts) == 0 {
		return "ready", "import", nil
	}
	reason := "Existing registry ownership or canonical-name conflicts require review before import."
	return "conflict", "review", &reason
}

// candidateKeyForWorkspacePath 生成工作目录的扫描候选键。
// @returns 基于修剪后的工作目录生成的键，格式为 `scan:` 加上 SHA-256 摘要前 8 字节的十六进制值。
func candidateKeyForWorkspacePath(workingDirectory string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(workingDirectory)))
	return "scan:" + hex.EncodeToString(sum[:8])
}

// firstProjectFileDisplayName 返回首个项目文件展示路径的基名；当列表为空时返回空字符串。
func firstProjectFileDisplayName(items []generated.ApplicationFileItem) string {
	if len(items) == 0 {
		return ""
	}
	return filepath.Base(items[0].DisplayPath)
}

// displayPathsFromCompose 返回一组 compose 文件投影的显示路径列表。
//
// @returns 按输入顺序提取的显示路径；当输入为空时返回 nil。
func displayPathsFromCompose(files []projectcompose.FileProjection) []string {
	if len(files) == 0 {
		return nil
	}
	result := make([]string, 0, len(files))
	for _, file := range files {
		result = append(result, file.DisplayPath)
	}
	return result
}

func (s *Service) importRootDefinitions(ctx context.Context) ([]importRootDefinition, error) {
	managedRootInfo, err := s.ManagedRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve managed root info: %w", err)
	}
	var managedRoot *string
	if managedRootInfo.Status == projectcontract.ManagedRootStatusReady.String() {
		managedRoot = managedRootInfo.ConfiguredRootDirectory
	}
	if s.configResolver == nil {
		return fallbackImportRoots(normalizeImportRootDefinitions(nil, managedRoot)), nil
	}
	raw, err := s.configResolver.ResolveDefaultConfig(ctx, projectcontract.ApplicationImportAllowedRootsConfig.String())
	if err != nil {
		return fallbackImportRoots(normalizeImportRootDefinitions(nil, managedRoot)), nil
	}
	decoded, decodeErr := decodeAllowedImportRoots(raw)
	if decodeErr != nil {
		return nil, fmt.Errorf("%w: invalid import root config", errProjectInvalidArgument)
	}
	return fallbackImportRoots(normalizeImportRootDefinitions(decoded, managedRoot)), nil
}

// fallbackImportRoots 尝试为导入根列表补充当前工作目录作为回退根。
// 如果无法获取当前工作目录，则直接返回原列表。
func fallbackImportRoots(roots []importRootDefinition) []importRootDefinition {
	return injectFallbackImportRoot(roots, "")
}

func (s *Service) resolveImportRoot(ctx context.Context, provider string, rootID string) (importRootDefinition, error) {
	if strings.TrimSpace(provider) != "" && strings.TrimSpace(provider) != importProviderLocal {
		return importRootDefinition{}, fmt.Errorf("%w: unsupported provider", errProjectDirectoryForbidden)
	}
	roots, err := s.importRootDefinitions(ctx)
	if err != nil {
		return importRootDefinition{}, err
	}
	for _, root := range roots {
		if root.id == strings.TrimSpace(rootID) {
			return root, nil
		}
	}
	return importRootDefinition{}, fmt.Errorf("%w: unknown root", errProjectDirectoryForbidden)
}
