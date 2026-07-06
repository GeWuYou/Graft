package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const projectWorkspaceEncodingUTF8 = "utf-8"

type trackedWorkspaceFile struct {
	Kind             string
	Path             string
	LastObservedHash string
	Content          string
	BaselineContent  string
}

func (s *Service) browseProjectFiles(
	ctx context.Context,
	projectID uint64,
	query workspaceFileBrowseQuery,
) (workspaceFilesResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	rootDir, currentPath, err := resolveProjectWorkspaceDirectory(aggregate.Project.WorkingDirectory, query.Path)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	browsePath := filepath.Join(rootDir, currentPath)
	if err := ensureWorkspaceBrowsePath(browsePath); err != nil {
		return workspaceFilesResult{}, err
	}
	entries, err := os.ReadDir(browsePath)
	if err != nil {
		return workspaceFilesResult{}, mapWorkspacePathError(err)
	}
	hiddenDirectories, err := s.workspaceHiddenDirectories(ctx)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	trackedKinds := trackedProjectFileKinds(rootDir, aggregate.Files)
	items, hasMoreHidden, err := buildVisibleWorkspaceItems(entries, currentPath, trackedKinds, hiddenDirectories, query.ShowHidden)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	parentPath := workspaceParentPath(currentPath)
	return workspaceFilesResult{
		ProjectID:     projectID,
		RootPath:      rootDir,
		CurrentPath:   currentPath,
		ParentPath:    parentPath,
		HasMoreHidden: hasMoreHidden,
		Items:         items,
	}, nil
}

func (s *Service) projectFileContent(
	ctx context.Context,
	projectID uint64,
	path string,
) (workspaceFileContentResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return workspaceFileContentResult{}, err
	}
	rootDir, relativePath, err := resolveProjectWorkspaceFilePath(aggregate.Project.WorkingDirectory, path)
	if err != nil {
		return workspaceFileContentResult{}, err
	}
	absolutePath := filepath.Join(rootDir, relativePath)
	info, err := os.Stat(absolutePath)
	if err != nil {
		if os.IsNotExist(err) {
			return workspaceFileContentResult{}, errProjectFileNotFound
		}
		return workspaceFileContentResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	if info.IsDir() {
		return workspaceFileContentResult{}, errProjectInvalidArgument
	}
	fileKind, languageHint, editable := classifyWorkspaceFile(relativePath, trackedProjectFileKinds(rootDir, aggregate.Files))
	// #nosec G304 -- absolutePath is validated to stay under the project's working_directory before reading.
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return workspaceFileContentResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	if !utf8.Valid(content) {
		return workspaceFileContentResult{}, errProjectInvalidArgument
	}
	return workspaceFileContentResult{
		ProjectID:    projectID,
		RelativePath: relativePath,
		FileKind:     fileKind,
		LanguageHint: languageHint,
		Editable:     editable,
		Encoding:     projectWorkspaceEncodingUTF8,
		Content:      string(content),
		SizeBytes:    info.Size(),
	}, nil
}

func (s *Service) saveProjectFileContent(
	ctx context.Context,
	projectID uint64,
	path string,
	request workspaceFileSaveRequest,
) (workspaceFileSaveResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return workspaceFileSaveResult{}, err
	}
	rootDir, relativePath, err := resolveProjectWorkspaceFilePath(aggregate.Project.WorkingDirectory, path)
	if err != nil {
		return workspaceFileSaveResult{}, err
	}
	fileKind, _, editable := classifyWorkspaceFile(relativePath, trackedProjectFileKinds(rootDir, aggregate.Files))
	if !editable || fileKind == "unsupported" {
		return workspaceFileSaveResult{}, errProjectInvalidArgument
	}
	absolutePath := filepath.Join(rootDir, relativePath)
	if err := ensureWorkspaceSaveTarget(absolutePath); err != nil {
		return workspaceFileSaveResult{}, err
	}
	fsRoot, err := openManagedRootFS(rootDir)
	if err != nil {
		return workspaceFileSaveResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	defer func() {
		_ = closeManagedRootFS(fsRoot)
	}()
	normalized := normalizeTextBlock(request.Content)
	if err := fsRoot.root.WriteFile(relativePath, []byte(normalized), managedCreateFileMode); err != nil {
		return workspaceFileSaveResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	return workspaceFileSaveResult{
		ProjectID:    projectID,
		RelativePath: relativePath,
		SavedAt:      time.Now().UTC(),
		ContentHash:  hashString(normalized),
		SizeBytes:    int64(len(normalized)),
	}, nil
}

func loadTrackedFileContents(aggregate projectstore.ProjectAggregate) ([]trackedWorkspaceFile, error) {
	rootDir, _, err := resolveProjectWorkspaceDirectory(aggregate.Project.WorkingDirectory, "")
	if err != nil {
		return nil, err
	}
	result := make([]trackedWorkspaceFile, 0, len(aggregate.Files))
	for _, item := range aggregate.Files {
		relativePath, relErr := relativePathWithinRoot(rootDir, item.AbsolutePath)
		if relErr != nil {
			return nil, fmt.Errorf("%w: %v", errProjectInvalidArgument, relErr)
		}
		// #nosec G304 -- rootDir and relativePath are normalized within the validated project root.
		content, readErr := os.ReadFile(filepath.Join(rootDir, relativePath))
		if readErr != nil {
			return nil, fmt.Errorf("%w: %v", errProjectImportValidation, readErr)
		}
		baselineContent := ""
		if item.Kind == projectcontract.FileKindCompose.String() && aggregate.Snapshot != nil {
			baselineContent = string(normalizeSnapshotJSON(aggregate.Snapshot.NormalizedComposeJSON))
		}
		result = append(result, trackedWorkspaceFile{
			Kind:             item.Kind,
			Path:             item.AbsolutePath,
			LastObservedHash: item.LastObservedHash,
			Content:          string(content),
			BaselineContent:  baselineContent,
		})
	}
	return result, nil
}

func buildProjectWorkspaceFileItem(
	relativePath string,
	entry fs.DirEntry,
	trackedKinds map[string]string,
	hiddenDirectories []string,
) (workspaceFileItem, error) {
	nodeType := "file"
	if entry.IsDir() {
		nodeType = "directory"
	}
	info, err := entry.Info()
	if err != nil {
		return workspaceFileItem{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	fileKind, languageHint, editable := classifyWorkspaceFile(relativePath, trackedKinds)
	return workspaceFileItem{
		Name:            entry.Name(),
		RelativePath:    relativePath,
		NodeType:        nodeType,
		FileKind:        fileKind,
		Editable:        editable,
		LanguageHint:    languageHint,
		SizeBytes:       info.Size(),
		HiddenByDefault: shouldHideWorkspaceEntry(entry.Name(), hiddenDirectories),
		HasChildren:     nodeType == "directory",
	}, nil
}

func trackedProjectFileKinds(rootDir string, files []projectstore.ProjectFile) map[string]string {
	result := make(map[string]string, len(files))
	for _, item := range files {
		if relativePath, err := relativePathWithinRoot(rootDir, item.AbsolutePath); err == nil {
			result[normalizeWorkspaceRelative(relativePath)] = item.Kind
		}
		if relativePath, err := relativePathWithinRoot(rootDir, item.DisplayPath); err == nil {
			result[normalizeWorkspaceRelative(relativePath)] = item.Kind
		}
	}
	return result
}

func classifyWorkspaceFile(relativePath string, trackedKinds map[string]string) (string, string, bool) {
	normalized := normalizeWorkspaceRelative(relativePath)
	base := strings.ToLower(filepath.Base(normalized))
	if kind, ok := trackedKinds[normalized]; ok {
		return classifyTrackedWorkspaceKind(kind)
	}
	if base == ".env" || strings.HasSuffix(base, ".env") {
		return "env", "dotenv", true
	}
	return classifyWorkspaceExtension(strings.ToLower(filepath.Ext(base)))
}

func classifyTrackedWorkspaceKind(kind string) (string, string, bool) {
	switch kind {
	case projectcontract.FileKindCompose.String():
		return "compose", "yaml", true
	case projectcontract.FileKindEnv.String():
		return "env", "dotenv", true
	default:
		return "config", "plaintext", false
	}
}

func classifyWorkspaceExtension(ext string) (string, string, bool) {
	switch ext {
	case ".yaml", ".yml":
		return "config", "yaml", true
	case ".json":
		return "config", "json", true
	case ".toml":
		return "config", "toml", true
	case ".ini", ".conf", ".cfg":
		return "config", "ini", true
	case ".properties":
		return "config", "properties", true
	case ".txt", ".md", ".log":
		return "text", "plaintext", false
	case "":
		return "unsupported", "plaintext", false
	default:
		return "unsupported", "plaintext", false
	}
}

func resolveProjectWorkspaceDirectory(workingDirectory string, requestedPath string) (string, string, error) {
	rootDir := filepath.Clean(strings.TrimSpace(workingDirectory))
	if rootDir == "" || !filepath.IsAbs(rootDir) {
		return "", "", errProjectInvalidArgument
	}
	candidate := filepath.Join(rootDir, strings.TrimSpace(requestedPath))
	relativePath, err := relativePathWithinRoot(rootDir, candidate)
	if err != nil {
		return "", "", errProjectInvalidArgument
	}
	if relativePath == "." {
		return rootDir, "", nil
	}
	return rootDir, normalizeWorkspaceRelative(relativePath), nil
}

func resolveProjectWorkspaceFilePath(workingDirectory string, requestedPath string) (string, string, error) {
	rootDir, relativePath, err := resolveProjectWorkspaceDirectory(workingDirectory, requestedPath)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(relativePath) == "" {
		return "", "", errProjectInvalidArgument
	}
	return rootDir, relativePath, nil
}

func normalizeWorkspaceRelative(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "." {
		return ""
	}
	normalized := filepath.ToSlash(filepath.Clean(trimmed))
	return strings.TrimPrefix(normalized, "./")
}

func buildVisibleWorkspaceItems(
	entries []os.DirEntry,
	currentPath string,
	trackedKinds map[string]string,
	hiddenDirectories []string,
	showHidden bool,
) ([]workspaceFileItem, bool, error) {
	items := make([]workspaceFileItem, 0, len(entries))
	hasMoreHidden := false
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if !showHidden && shouldHideWorkspaceEntry(name, hiddenDirectories) {
			hasMoreHidden = true
			continue
		}
		relativePath := normalizeWorkspaceRelative(filepath.Join(currentPath, name))
		item, err := buildProjectWorkspaceFileItem(relativePath, entry, trackedKinds, hiddenDirectories)
		if err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	return items, hasMoreHidden, nil
}

func ensureWorkspaceBrowsePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return mapWorkspacePathError(err)
	}
	if !info.IsDir() {
		return errProjectInvalidArgument
	}
	return nil
}

func ensureWorkspaceSaveTarget(path string) error {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return errProjectInvalidArgument
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return mapWorkspacePathError(err)
	}
	return nil
}

func mapWorkspacePathError(err error) error {
	if os.IsNotExist(err) {
		return errProjectFileNotFound
	}
	return fmt.Errorf("%w: %v", errProjectImportValidation, err)
}

func workspaceParentPath(currentPath string) *string {
	normalized := normalizeWorkspaceRelative(currentPath)
	if normalized == "" {
		return nil
	}
	parent := normalizeWorkspaceRelative(filepath.Dir(normalized))
	return &parent
}

func shouldHideWorkspaceEntry(name string, extraHidden []string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	return slices.Contains(extraHidden, name)
}

func (s *Service) workspaceHiddenDirectories(ctx context.Context) ([]string, error) {
	if s == nil || s.configResolver == nil {
		return decodeWorkspaceHiddenDirectories(defaultWorkspaceHiddenDirectories)
	}
	raw, err := s.configResolver.ResolveDefaultConfig(ctx, projectcontract.ProjectWorkspaceHiddenDirectoriesConfig.String())
	if err != nil {
		return decodeWorkspaceHiddenDirectories(defaultWorkspaceHiddenDirectories)
	}
	var encoded string
	if unmarshalErr := json.Unmarshal([]byte(raw), &encoded); unmarshalErr != nil {
		encoded = raw
	}
	hiddenDirectories, decodeErr := decodeWorkspaceHiddenDirectories(encoded)
	if decodeErr == nil {
		return hiddenDirectories, nil
	}
	return decodeWorkspaceHiddenDirectories(defaultWorkspaceHiddenDirectories)
}

func decodeWorkspaceHiddenDirectories(raw string) ([]string, error) {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("%w: invalid workspace hidden directory config", errProjectInvalidArgument)
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		result = append(result, filepath.Base(trimmed))
	}
	return result, nil
}
