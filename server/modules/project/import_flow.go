package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	projectcompose "graft/server/modules/project/compose"
)

const (
	importProviderLocal            = "local"
	importManagedRootSourceID      = "managed-root"
	importServiceRootSourceID      = "service-root"
	importFallbackRootPath         = "/"
	importInspectionSessionTTL     = 5 * time.Minute
	importDirectoryBrowseMaxLimit  = 200
	importDirectoryBrowseDefault   = 100
	importComposeFileCapacity      = 2
	importDirectorySortByName      = "name"
	importDirectoryOrderAsc        = "asc"
	importDirectoryOrderDesc       = "desc"
	importDirectorySortByModified  = "modified_at"
	importDirectoryWarningMultiple = "Multiple primary compose files were discovered. The highest-priority file was selected for inspection."
)

var defaultPrimaryComposeCandidates = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

const defaultComposeOverrideCandidate = "docker-compose.override.yml"

type importAllowedRootConfig struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

// ImportDirectorySource describes one allowed import root exposed to the folder picker.
type ImportDirectorySource struct {
	Provider    string `json:"provider"`
	RootID      string `json:"root_id"`
	Label       string `json:"label"`
	Path        string `json:"path"`
	InitialPath string `json:"initial_path"`
	Managed     bool   `json:"managed"`
}

// ImportDirectorySourceResult returns the available import roots.
type ImportDirectorySourceResult struct {
	Items []ImportDirectorySource `json:"items"`
}

// ImportDirectoryBrowseQuery defines one bounded root-relative browse request.
type ImportDirectoryBrowseQuery struct {
	Provider string
	RootID   string
	Path     string
	Limit    int
	Offset   int
	SortBy   string
	Order    string
}

// ImportDirectoryItem describes one browsable directory item.
type ImportDirectoryItem struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	HasParent  bool       `json:"has_parent"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

// ImportDirectoryBrowseResult returns a bounded directory page.
type ImportDirectoryBrowseResult struct {
	Provider    string                `json:"provider"`
	RootID      string                `json:"root_id"`
	CurrentPath string                `json:"current_path"`
	ParentPath  *string               `json:"parent_path,omitempty"`
	Limit       int                   `json:"limit"`
	Offset      int                   `json:"offset"`
	HasMore     bool                  `json:"has_more"`
	SortBy      string                `json:"sort_by"`
	Order       string                `json:"order"`
	Items       []ImportDirectoryItem `json:"directories"`
}

// ImportDirectoryReference identifies a root-relative directory for inspect/import.
type ImportDirectoryReference struct {
	Provider string `json:"provider"`
	RootID   string `json:"root_id"`
	Path     string `json:"path"`
}

// ImportInspectRequest captures one inspect request for an import directory.
type ImportInspectRequest struct {
	DirectoryRef                 ImportDirectoryReference `json:"directory_ref"`
	DisplayName                  *string                  `json:"display_name,omitempty"`
	CanonicalProjectNameOverride *string                  `json:"canonical_project_name_override,omitempty"`
}

// ImportInspectResult returns the discovered files and preview derived from one inspect.
type ImportInspectResult struct {
	InspectionID               string                   `json:"inspection_id"`
	DirectoryRef               ImportDirectoryReference `json:"directory_ref"`
	ResolvedWorkingDirectory   string                   `json:"resolved_working_directory"`
	CanonicalProjectName       string                   `json:"canonical_project_name"`
	CanonicalProjectNameSource string                   `json:"canonical_project_name_source"`
	DisplayNameSuggested       string                   `json:"display_name_suggested"`
	ComposeFiles               []FileView               `json:"compose_files"`
	EnvFiles                   []FileView               `json:"env_files"`
	ServiceNames               []string                 `json:"services"`
	NetworkNames               []string                 `json:"networks"`
	VolumeNames                []string                 `json:"volumes"`
	ConfigHash                 string                   `json:"config_hash"`
	Warnings                   []string                 `json:"warnings"`
	Conflicts                  []string                 `json:"conflicts"`
	ValidationStatus           string                   `json:"validation_status"`
}

// ImportExecuteRequest finalizes an import from a prior inspection snapshot.
type ImportExecuteRequest struct {
	InspectionID                 string  `json:"inspection_id"`
	DisplayName                  *string `json:"display_name,omitempty"`
	CanonicalProjectNameOverride *string `json:"canonical_project_name_override,omitempty"`
	ActorID                      *uint64 `json:"-"`
}

// FileView exposes one discovered compose/env file in inspect responses.
type FileView struct {
	Kind                string  `json:"kind"`
	Role                string  `json:"role"`
	AbsolutePath        string  `json:"absolute_path"`
	DisplayPath         string  `json:"display_path"`
	OrderIndex          int     `json:"order_index"`
	ExistsOnLastRefresh bool    `json:"exists_on_last_refresh"`
	LastObservedHash    *string `json:"last_observed_hash,omitempty"`
}

type discoveredImportFiles struct {
	composeFiles []string
	envFiles     []string
	warnings     []string
}

type importInspectionSession struct {
	ID              string
	DirectoryRef    ImportDirectoryReference
	WorkingDir      string
	CanonicalName   string
	CanonicalSource string
	DisplayName     string
	ParseResult     projectcompose.Result
	Conflicts       []string
	Warnings        []string
	CreatedAt       time.Time
	ExpiresAt       time.Time
	FileHashes      map[string]string
}

type importRootDefinition struct {
	id          string
	label       string
	path        string
	initialPath string
	managed     bool
}

// normalizeDirectoryBrowseQuery 规范化目录浏览查询参数。
// 它会修剪 provider 和 root_id，默认使用本地 provider，
// 并对路径、分页、排序字段和排序顺序应用标准化规则。
func normalizeDirectoryBrowseQuery(query ImportDirectoryBrowseQuery) ImportDirectoryBrowseQuery {
	query.Provider = strings.TrimSpace(query.Provider)
	if query.Provider == "" {
		query.Provider = importProviderLocal
	}
	query.RootID = strings.TrimSpace(query.RootID)
	query.Path = normalizeBrowsePath(query.Path)
	query.Limit = normalizedBrowseLimit(query.Limit)
	query.Offset = normalizedBrowseOffset(query.Offset)
	query.SortBy = normalizedBrowseSort(query.SortBy)
	query.Order = normalizedBrowseOrder(query.Order)
	return query
}

// normalizedBrowseLimit 将浏览条目数量限制归一到允许范围内。
// 当 limit 小于等于 0 或大于最大值时，返回默认浏览数量。
func normalizedBrowseLimit(limit int) int {
	if limit <= 0 || limit > importDirectoryBrowseMaxLimit {
		return importDirectoryBrowseDefault
	}
	return limit
}

// normalizedBrowseOffset 将浏览偏移量限制为不小于 0。
func normalizedBrowseOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

// normalizedBrowseSort 规范化目录浏览的排序字段。
// 仅保留按修改时间排序，其余值统一使用名称排序。
func normalizedBrowseSort(sortBy string) string {
	switch strings.TrimSpace(sortBy) {
	case importDirectorySortByModified:
		return importDirectorySortByModified
	default:
		return importDirectorySortByName
	}
}

// normalizedBrowseOrder 规范化目录浏览的排序方向。
// 当输入为 `desc`（忽略大小写和首尾空白）时返回 `desc`，否则返回 `asc`。
func normalizedBrowseOrder(order string) string {
	if strings.EqualFold(strings.TrimSpace(order), importDirectoryOrderDesc) {
		return importDirectoryOrderDesc
	}
	return importDirectoryOrderAsc
}

// normalizeBrowsePath 规范化用于浏览的路径，去除空白并将根路径表示为一个空字符串。
func normalizeBrowsePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "." || trimmed == "/" {
		return ""
	}
	cleaned := filepath.Clean(trimmed)
	cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
	if cleaned == "." {
		return ""
	}
	return cleaned
}

// inspectionSessionID 生成导入检查会话的标识符。
// 该标识符由目录引用、配置哈希和创建时间共同决定。
func inspectionSessionID(ref ImportDirectoryReference, configHash string, createdAt time.Time) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		ref.Provider,
		ref.RootID,
		ref.Path,
		configHash,
		strconv.FormatInt(createdAt.UnixNano(), 10),
	}, "|")))
	return "inspect_" + hex.EncodeToString(sum[:12])
}

// toFileViews 将文件投影转换为 FileView 列表，并保留其元数据与最后观测到的哈希。
func toFileViews(files []projectcompose.FileProjection) []FileView {
	result := make([]FileView, 0, len(files))
	for _, item := range files {
		hash := item.Hash
		result = append(result, FileView{
			Kind:                item.Kind,
			Role:                item.Role,
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			OrderIndex:          item.OrderIndex,
			ExistsOnLastRefresh: item.Exists,
			LastObservedHash:    optionalString(hash),
		})
	}
	return result
}

// 它返回一个以绝对路径为键、文件哈希为值的映射。
func snapshotFileHashes(parseResult projectcompose.Result) map[string]string {
	hashes := make(map[string]string, len(parseResult.ComposeFiles)+len(parseResult.EnvFiles))
	for _, file := range append(append([]projectcompose.FileProjection(nil), parseResult.ComposeFiles...), parseResult.EnvFiles...) {
		hashes[file.AbsolutePath] = file.Hash
	}
	return hashes
}

// sameFileHashes 判断当前解析结果中的文件哈希是否与预期一致。
// 当路径数量一致且每个路径对应的哈希值都匹配时返回 true，否则返回 false。
func sameFileHashes(expected map[string]string, parseResult projectcompose.Result) bool {
	actual := snapshotFileHashes(parseResult)
	if len(actual) != len(expected) {
		return false
	}
	for path, hash := range expected {
		if actual[path] != hash {
			return false
		}
	}
	return true
}

// minInt 返回两个整数中较小的值。
func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// decodeAllowedImportRoots 解析允许导入根目录的配置。
// 输入可以是直接编码的 JSON 数组，或包含该 JSON 数组的 JSON 字符串；空白输入返回空结果。
// @returns 解析后的根目录配置列表；当输入为空时返回 nil 和 nil，解析失败时返回错误。
func decodeAllowedImportRoots(raw string) ([]importAllowedRootConfig, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	items, err := decodeAllowedImportRootItems(trimmed)
	if err == nil {
		return items, nil
	}

	nested, nestedErr := decodeAllowedImportRootString(trimmed)
	if nestedErr != nil {
		return nil, fmt.Errorf("decode allowed import roots: %w", errors.Join(err, nestedErr))
	}
	nested = strings.TrimSpace(nested)
	if nested == "" {
		return nil, nil
	}
	items, nestedItemsErr := decodeAllowedImportRootItems(nested)
	if nestedItemsErr != nil {
		return nil, fmt.Errorf("decode allowed import roots: %w", errors.Join(err, nestedItemsErr))
	}
	return items, nil
}

// decodeAllowedImportRootItems 将 JSON 数组解析为允许的导入根配置列表。
//
// @param raw 要解析的 JSON 数组字符串。
// @returns 解析得到的导入根配置列表，或解析失败时的错误。
func decodeAllowedImportRootItems(raw string) ([]importAllowedRootConfig, error) {
	var items []importAllowedRootConfig
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("decode allowed import roots array: %w", err)
	}
	return items, nil
}

// decodeAllowedImportRootString 解析包含嵌套 JSON 的字符串值。
// 返回解码后的字符串；解析失败时返回带上下文的错误。
func decodeAllowedImportRootString(raw string) (string, error) {
	var nested string
	if err := json.Unmarshal([]byte(raw), &nested); err != nil {
		return "", fmt.Errorf("decode allowed import roots string: %w", err)
	}
	return nested, nil
}

// configured 提供可导入的根配置，managedRoot 提供可选的受管根路径；函数会清理路径、过滤空值和非绝对路径、按绝对路径去重，并在缺少 ID 或标签时补全默认值。
func normalizeImportRootDefinitions(configured []importAllowedRootConfig, managedRoot *string) []importRootDefinition {
	result := make([]importRootDefinition, 0, len(configured)+1)
	seenByPath := make(map[string]struct{})
	appendRoot := func(id string, label string, path string, managed bool) {
		trimmedPath := strings.TrimSpace(path)
		if trimmedPath == "" {
			return
		}
		cleanedPath := filepath.Clean(trimmedPath)
		if !filepath.IsAbs(cleanedPath) {
			return
		}
		if _, ok := seenByPath[cleanedPath]; ok {
			return
		}
		seenByPath[cleanedPath] = struct{}{}
		trimmedID := strings.TrimSpace(id)
		if trimmedID == "" {
			trimmedID = rootIDFromPath(cleanedPath)
		}
		trimmedLabel := strings.TrimSpace(label)
		if trimmedLabel == "" {
			trimmedLabel = cleanedPath
		}
		result = append(result, importRootDefinition{
			id:      trimmedID,
			label:   trimmedLabel,
			path:    cleanedPath,
			managed: managed,
		})
	}
	for _, item := range configured {
		appendRoot(item.ID, item.Label, item.Path, false)
	}
	if managedRoot != nil && strings.TrimSpace(*managedRoot) != "" {
		appendRoot(importManagedRootSourceID, "Managed Root", *managedRoot, true)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].managed != result[j].managed {
			return result[i].managed
		}
		return strings.Compare(result[i].label, result[j].label) < 0
	})
	return result
}

// injectFallbackImportRoot 在未配置导入根目录时注入一个本地文件系统根目录。
// 当已有根目录时直接返回原值；否则根据工作目录或环境变量计算初始路径，并在可用时返回一个默认根目录。
func injectFallbackImportRoot(roots []importRootDefinition, workingDirectory string) []importRootDefinition {
	if len(roots) > 0 {
		return roots
	}
	initialPath := fallbackImportInitialPath(workingDirectory)
	if initialPath == "" {
		return roots
	}
	return []importRootDefinition{
		{
			id:          importServiceRootSourceID,
			label:       "Local Filesystem",
			path:        importFallbackRootPath,
			initialPath: normalizeBrowsePath(initialPath),
			managed:     false,
		},
	}
}

// fallbackImportInitialPath 返回导入流程的默认初始目录路径。
// 它优先使用环境变量 `GRAFT_PROJECT_IMPORT_DEFAULT_PATH`，仅在其为绝对路径时生效；
// 否则使用 workingDirectory，在其为绝对路径时返回规范化后的路径。
func fallbackImportInitialPath(workingDirectory string) string {
	if configured := strings.TrimSpace(os.Getenv("GRAFT_PROJECT_IMPORT_DEFAULT_PATH")); configured != "" {
		cleaned := filepath.Clean(configured)
		if filepath.IsAbs(cleaned) {
			return cleaned
		}
	}
	trimmed := strings.TrimSpace(workingDirectory)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return ""
	}
	return cleaned
}

// rootIDFromPath 将路径转换为稳定的根标识符。
// 空路径返回 "root"；其余路径会清理后将路径分隔符和空格替换为连字符。
func rootIDFromPath(path string) string {
	cleaned := strings.Trim(filepath.Clean(path), string(filepath.Separator))
	if cleaned == "" {
		return "root"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-")
	return replacer.Replace(cleaned)
}

// 当路径试图逃离根目录时返回 errProjectInvalidArgument。
func resolveRootPath(root importRootDefinition, relative string) (string, error) {
	normalized := normalizeBrowsePath(relative)
	joined := filepath.Join(root.path, normalized)
	rel, err := filepath.Rel(root.path, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errProjectInvalidArgument
	}
	return joined, nil
}

// parentBrowsePath 返回路径的父级浏览路径。
//
// 当输入为空或仅包含空白字符时，返回 nil。若父级位于根目录，则返回指向空字符串的指针，以表示根的父级；否则返回规范化后的父级路径。
func parentBrowsePath(path string) *string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	parent := filepath.Dir(path)
	if parent == "." || parent == "/" {
		empty := ""
		return &empty
	}
	parent = normalizeBrowsePath(parent)
	return &parent
}

// 如果未发现任何主 Compose 文件，则返回错误。
func discoverImportFiles(workingDirectory string) (discoveredImportFiles, error) {
	entries, err := os.ReadDir(workingDirectory)
	if err != nil {
		return discoveredImportFiles{}, err
	}
	foundPrimary := discoverPrimaryComposeFiles(workingDirectory)
	if len(foundPrimary) == 0 {
		return discoveredImportFiles{}, fmt.Errorf("no compose file discovered")
	}
	composeFiles := make([]string, 0, importComposeFileCapacity)
	warnings := make([]string, 0, 1)
	composeFiles = append(composeFiles, foundPrimary[0])
	if len(foundPrimary) > 1 {
		warnings = append(warnings, importDirectoryWarningMultiple)
	}
	if _, statErr := os.Stat(filepath.Join(workingDirectory, defaultComposeOverrideCandidate)); statErr == nil {
		composeFiles = append(composeFiles, defaultComposeOverrideCandidate)
	}
	return discoveredImportFiles{
		composeFiles: composeFiles,
		envFiles:     discoverEnvFiles(workingDirectory, entries),
		warnings:     warnings,
	}, nil
}

// discoverPrimaryComposeFiles 返回工作目录中存在的首选 Compose 文件名列表。
// 返回值按候选项顺序排列，只包含实际存在的文件名。
func discoverPrimaryComposeFiles(workingDirectory string) []string {
	foundPrimary := make([]string, 0, len(defaultPrimaryComposeCandidates))
	for _, candidate := range defaultPrimaryComposeCandidates {
		if _, statErr := os.Stat(filepath.Join(workingDirectory, candidate)); statErr == nil {
			foundPrimary = append(foundPrimary, candidate)
		}
	}
	return foundPrimary
}

// discoverEnvFiles 收集工作目录中的环境文件名，按优先级去重后返回。
// 如果存在 `.env`，会优先包含它，再追加 `.env.*` 和 `*.env` 变体。
//
// 返回按发现顺序排列的环境文件名列表。
func discoverEnvFiles(workingDirectory string, entries []os.DirEntry) []string {
	envFiles := make([]string, 0)
	envSet := make(map[string]struct{})
	appendEnv := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := envSet[name]; ok {
			return
		}
		envSet[name] = struct{}{}
		envFiles = append(envFiles, name)
	}
	if _, statErr := os.Stat(filepath.Join(workingDirectory, ".env")); statErr == nil {
		appendEnv(".env")
	}
	dotEnvVariants, suffixEnvVariants := collectEnvVariants(entries)
	for _, name := range dotEnvVariants {
		appendEnv(name)
	}
	for _, name := range suffixEnvVariants {
		appendEnv(name)
	}
	return envFiles
}

// collectEnvVariants 收集目录中的环境变量变体文件名。
// 第一个返回值包含以 `.env.` 开头的文件名，第二个返回值包含以 `.env` 结尾的文件名；两组结果都会按字典序排序。
func collectEnvVariants(entries []os.DirEntry) ([]string, []string) {
	dotEnvVariants := make([]string, 0)
	suffixEnvVariants := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		switch {
		case strings.HasPrefix(name, ".env.") && name != ".env":
			dotEnvVariants = append(dotEnvVariants, name)
		case strings.HasSuffix(name, ".env") && name != ".env":
			suffixEnvVariants = append(suffixEnvVariants, name)
		}
	}
	sort.Strings(dotEnvVariants)
	sort.Strings(suffixEnvVariants)
	return dotEnvVariants, suffixEnvVariants
}
