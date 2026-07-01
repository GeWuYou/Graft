package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
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

type importInspectionCache struct {
	mu       sync.Mutex
	sessions map[string]importInspectionSession
}

func newImportInspectionCache() *importInspectionCache {
	return &importInspectionCache{sessions: make(map[string]importInspectionSession)}
}

func (c *importInspectionCache) put(session importInspectionSession) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(time.Now().UTC())
	c.sessions[session.ID] = session
}

func (c *importInspectionCache) get(id string) (importInspectionSession, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(time.Now().UTC())
	session, ok := c.sessions[id]
	return session, ok
}

func (c *importInspectionCache) pruneLocked(now time.Time) {
	for key, session := range c.sessions {
		if now.After(session.ExpiresAt) {
			delete(c.sessions, key)
		}
	}
}

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

func normalizedBrowseLimit(limit int) int {
	if limit <= 0 || limit > importDirectoryBrowseMaxLimit {
		return importDirectoryBrowseDefault
	}
	return limit
}

func normalizedBrowseOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func normalizedBrowseSort(sortBy string) string {
	switch strings.TrimSpace(sortBy) {
	case importDirectorySortByModified:
		return importDirectorySortByModified
	default:
		return importDirectorySortByName
	}
}

func normalizedBrowseOrder(order string) string {
	if strings.EqualFold(strings.TrimSpace(order), importDirectoryOrderDesc) {
		return importDirectoryOrderDesc
	}
	return importDirectoryOrderAsc
}

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

func snapshotFileHashes(parseResult projectcompose.Result) map[string]string {
	hashes := make(map[string]string, len(parseResult.ComposeFiles)+len(parseResult.EnvFiles))
	for _, file := range append(append([]projectcompose.FileProjection(nil), parseResult.ComposeFiles...), parseResult.EnvFiles...) {
		hashes[file.AbsolutePath] = file.Hash
	}
	return hashes
}

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

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func decodeAllowedImportRoots(raw string) ([]importAllowedRootConfig, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	var items []importAllowedRootConfig
	if err := json.Unmarshal([]byte(trimmed), &items); err == nil {
		return items, nil
	}

	var nested string
	if err := json.Unmarshal([]byte(trimmed), &nested); err != nil {
		return nil, err
	}
	nested = strings.TrimSpace(nested)
	if nested == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(nested), &items); err != nil {
		return nil, err
	}
	return items, nil
}

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

func rootIDFromPath(path string) string {
	cleaned := strings.Trim(filepath.Clean(path), string(filepath.Separator))
	if cleaned == "" {
		return "root"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-")
	return replacer.Replace(cleaned)
}

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

func discoverPrimaryComposeFiles(workingDirectory string) []string {
	foundPrimary := make([]string, 0, len(defaultPrimaryComposeCandidates))
	for _, candidate := range defaultPrimaryComposeCandidates {
		if _, statErr := os.Stat(filepath.Join(workingDirectory, candidate)); statErr == nil {
			foundPrimary = append(foundPrimary, candidate)
		}
	}
	return foundPrimary
}

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
