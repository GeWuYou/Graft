package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const projectWorkspaceEncodingUTF8 = "utf-8"
const workspaceReadableSampleLimit = 8192

type workspaceFileClassification struct {
	FileKind     string
	LanguageHint string
}

var workspaceBaseNameClassifications = map[string]workspaceFileClassification{
	".editorconfig":  {FileKind: "config", LanguageHint: "ini"},
	".gitattributes": {FileKind: "text", LanguageHint: "plaintext"},
	".gitconfig":     {FileKind: "config", LanguageHint: "ini"},
	".gitignore":     {FileKind: "text", LanguageHint: "plaintext"},
	"caddyfile":      {FileKind: "text", LanguageHint: "plaintext"},
	"dockerfile":     {FileKind: "config", LanguageHint: "dockerfile"},
	"makefile":       {FileKind: "text", LanguageHint: "plaintext"},
}

var workspaceExtensionClassifications = map[string]workspaceFileClassification{
	".bash":       {FileKind: "config", LanguageHint: "shell"},
	".cfg":        {FileKind: "config", LanguageHint: "ini"},
	".conf":       {FileKind: "config", LanguageHint: "ini"},
	".dockerfile": {FileKind: "config", LanguageHint: "dockerfile"},
	".hcl":        {FileKind: "config", LanguageHint: "hcl"},
	".ini":        {FileKind: "config", LanguageHint: "ini"},
	".json":       {FileKind: "config", LanguageHint: "json"},
	".jsonc":      {FileKind: "config", LanguageHint: "json"},
	".log":        {FileKind: "text", LanguageHint: "plaintext"},
	".markdown":   {FileKind: "text", LanguageHint: "markdown"},
	".md":         {FileKind: "text", LanguageHint: "markdown"},
	".properties": {FileKind: "config", LanguageHint: "properties"},
	".ps1":        {FileKind: "config", LanguageHint: "powershell"},
	".psd1":       {FileKind: "config", LanguageHint: "powershell"},
	".psm1":       {FileKind: "config", LanguageHint: "powershell"},
	".sh":         {FileKind: "config", LanguageHint: "shell"},
	".sql":        {FileKind: "config", LanguageHint: "sql"},
	".tf":         {FileKind: "config", LanguageHint: "hcl"},
	".tfvars":     {FileKind: "config", LanguageHint: "hcl"},
	".toml":       {FileKind: "config", LanguageHint: "toml"},
	".txt":        {FileKind: "text", LanguageHint: "plaintext"},
	".xml":        {FileKind: "config", LanguageHint: "xml"},
	".yaml":       {FileKind: "config", LanguageHint: "yaml"},
	".yml":        {FileKind: "config", LanguageHint: "yaml"},
	".zsh":        {FileKind: "config", LanguageHint: "shell"},
}

type workspaceTooltipRule struct {
	Enabled bool   `json:"enabled"`
	Pattern string `json:"pattern"`
	Tooltip string `json:"tooltip"`
	regex   *regexp.Regexp
}

type workspaceFileState struct {
	FileKind     string
	LanguageHint string
	Readable     bool
	Editable     bool
}

type workspaceEntryCreateRequest struct {
	Path     string
	NodeType string
	Content  *string
}

type workspaceEntryRenameRequest struct {
	Path    string
	NewPath string
}

type workspaceTreeBuildContext struct {
	Root                  *managedRootFS
	TrackedKinds          map[string]string
	HiddenDirectories     []string
	FileTooltipRules      []workspaceTooltipRule
	DirectoryTooltipRules []workspaceTooltipRule
	Annotations           map[string]string
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
	rootDir, currentPath, err := resolveProjectWorkspaceDirectory(aggregate.Application.WorkspacePath, query.Path)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	root, entries, err := readWorkspaceDirectory(rootDir, currentPath)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	defer func() { _ = closeManagedRootFS(root) }()
	hiddenDirectories, err := s.workspaceHiddenDirectories(ctx)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	fileTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ApplicationWorkspaceFileTooltipRulesConfig.String(), defaultWorkspaceFileTooltipRules)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	directoryTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ApplicationWorkspaceDirectoryTooltipRulesConfig.String(), defaultWorkspaceDirectoryTooltipRules)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	buildContext := workspaceTreeBuildContext{
		Root:                  root,
		TrackedKinds:          trackedProjectFileKinds(rootDir, aggregate.Files),
		HiddenDirectories:     hiddenDirectories,
		FileTooltipRules:      fileTooltipRules,
		DirectoryTooltipRules: directoryTooltipRules,
		Annotations:           aggregate.Application.WorkspaceAnnotations,
	}
	items, hasMoreHidden, err := buildVisibleWorkspaceItems(
		entries,
		currentPath,
		buildContext,
		query.ShowHidden,
	)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	parentPath := workspaceParentPath(currentPath)
	return workspaceFilesResult{
		ApplicationRecordID: projectID,
		ApplicationID:       aggregate.Application.ApplicationID,
		RootPath:            rootDir,
		CurrentPath:         currentPath,
		ParentPath:          parentPath,
		HasMoreHidden:       hasMoreHidden,
		Items:               items,
	}, nil
}

func readWorkspaceDirectory(rootDir string, currentPath string) (*managedRootFS, []fs.DirEntry, error) {
	root, err := openManagedRootFS(rootDir)
	if err != nil {
		return nil, nil, mapWorkspacePathError(err)
	}
	managedPath := currentPath
	if managedPath == "" {
		managedPath = "."
	}
	if err := ensureWorkspaceBrowsePath(root, managedPath); err != nil {
		_ = closeManagedRootFS(root)
		return nil, nil, err
	}
	entries, err := fs.ReadDir(root.root.FS(), managedPath)
	if err != nil {
		_ = closeManagedRootFS(root)
		return nil, nil, mapWorkspacePathError(err)
	}
	return root, entries, nil
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
	rootDir, relativePath, err := resolveProjectWorkspaceFilePath(aggregate.Application.WorkspacePath, path)
	if err != nil {
		return workspaceFileContentResult{}, err
	}
	root, err := openManagedRootFS(rootDir)
	if err != nil {
		return workspaceFileContentResult{}, mapWorkspacePathError(err)
	}
	defer func() { _ = closeManagedRootFS(root) }()
	info, err := root.root.Stat(relativePath)
	if err != nil {
		if os.IsNotExist(err) {
			return workspaceFileContentResult{}, errProjectFileNotFound
		}
		return workspaceFileContentResult{}, fmt.Errorf("%w: %w", errProjectImportValidation, err)
	}
	if info.IsDir() {
		return workspaceFileContentResult{}, errProjectInvalidArgument
	}
	content, err := root.root.ReadFile(relativePath)
	if err != nil {
		return workspaceFileContentResult{}, fmt.Errorf("%w: %w", errProjectImportValidation, err)
	}
	state := resolveWorkspaceFileState(relativePath, trackedProjectFileKinds(rootDir, aggregate.Files), content)
	if !state.Readable {
		return workspaceFileContentResult{}, errProjectInvalidArgument
	}
	return workspaceFileContentResult{
		ApplicationRecordID: projectID,
		ApplicationID:       aggregate.Application.ApplicationID,
		RelativePath:        relativePath,
		FileKind:            state.FileKind,
		LanguageHint:        state.LanguageHint,
		Readable:            state.Readable,
		Editable:            state.Editable,
		Encoding:            projectWorkspaceEncodingUTF8,
		Content:             string(content),
		SizeBytes:           info.Size(),
	}, nil
}

func (s *Service) saveProjectFileContent(
	ctx context.Context,
	projectID uint64,
	path string,
	request workspaceFileSaveRequest,
) (workspaceFileSaveResult, error) {
	s.workspaceMutationMu.Lock()
	defer s.workspaceMutationMu.Unlock()
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return workspaceFileSaveResult{}, err
	}
	rootDir, relativePath, err := resolveProjectWorkspaceFilePath(aggregate.Application.WorkspacePath, path)
	if err != nil {
		return workspaceFileSaveResult{}, err
	}
	fsRoot, err := openManagedRootFS(rootDir)
	if err != nil {
		return workspaceFileSaveResult{}, fmt.Errorf("%w: %w", errProjectImportValidation, err)
	}
	defer func() {
		_ = closeManagedRootFS(fsRoot)
	}()
	if err := ensureWorkspaceSaveTarget(fsRoot, relativePath); err != nil {
		return workspaceFileSaveResult{}, err
	}
	existingContent, err := fsRoot.root.ReadFile(relativePath)
	if err != nil {
		return workspaceFileSaveResult{}, fmt.Errorf("%w: %w", errProjectImportValidation, err)
	}
	state := resolveWorkspaceFileState(relativePath, trackedProjectFileKinds(rootDir, aggregate.Files), existingContent)
	if !state.Editable {
		return workspaceFileSaveResult{}, errProjectInvalidArgument
	}
	normalized := normalizeTextBlock(request.Content)
	if err := fsRoot.root.WriteFile(relativePath, []byte(normalized), managedCreateFileMode); err != nil {
		return workspaceFileSaveResult{}, fmt.Errorf("%w: %w", errProjectImportValidation, err)
	}
	return workspaceFileSaveResult{
		ApplicationRecordID: projectID,
		ApplicationID:       aggregate.Application.ApplicationID,
		RelativePath:        relativePath,
		SavedAt:             time.Now().UTC(),
		ContentHash:         hashString(normalized),
		SizeBytes:           int64(len(normalized)),
	}, nil
}

func (s *Service) createProjectWorkspaceEntry(ctx context.Context, projectID uint64, request workspaceEntryCreateRequest) error {
	s.workspaceMutationMu.Lock()
	defer s.workspaceMutationMu.Unlock()
	relativePath, root, err := s.openProjectWorkspaceRoot(ctx, projectID, request.Path)
	if err != nil {
		return err
	}
	defer func() { _ = closeManagedRootFS(root) }()
	if err := validateWorkspaceEntryRequest(request); err != nil {
		return err
	}
	if err := ensureWorkspaceEntryAbsent(root, relativePath); err != nil {
		return err
	}
	if err := ensureWorkspaceParent(root, relativePath); err != nil {
		return err
	}
	if request.NodeType == "directory" {
		return mapWorkspacePathError(root.root.MkdirAll(relativePath, managedCreateDirMode))
	}
	return mapWorkspacePathError(root.root.WriteFile(relativePath, []byte(*request.Content), managedCreateFileMode))
}

func (s *Service) renameProjectWorkspaceEntry(ctx context.Context, projectID uint64, request workspaceEntryRenameRequest) error {
	s.workspaceMutationMu.Lock()
	defer s.workspaceMutationMu.Unlock()
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return err
	}
	rootDir, source, err := resolveProjectWorkspaceFilePath(aggregate.Application.WorkspacePath, request.Path)
	if err != nil {
		return err
	}
	if workspaceMutationContainsTrackedLifecycleInput(aggregate, source) {
		return errProjectConflict
	}
	root, err := openManagedRootFS(rootDir)
	if err != nil {
		return mapWorkspacePathError(err)
	}
	defer func() { _ = closeManagedRootFS(root) }()
	destination, err := normalizeManagedWorkspacePath(request.NewPath)
	if err != nil {
		return err
	}
	if err := ensureWorkspaceEntrySafe(root, source); err != nil {
		return err
	}
	if err := ensureWorkspaceEntryAbsent(root, destination); err != nil {
		return err
	}
	if err := ensureWorkspaceParent(root, destination); err != nil {
		return err
	}
	if err := root.root.Rename(source, destination); err != nil {
		return mapWorkspacePathError(err)
	}
	return nil
}

func (s *Service) openProjectWorkspaceRoot(ctx context.Context, projectID uint64, path string) (string, *managedRootFS, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return "", nil, err
	}
	rootDir, relativePath, err := resolveProjectWorkspaceFilePath(aggregate.Application.WorkspacePath, path)
	if err != nil {
		return "", nil, err
	}
	root, err := openManagedRootFS(rootDir)
	if err != nil {
		return "", nil, mapWorkspacePathError(err)
	}
	return relativePath, root, nil
}

// validateWorkspaceEntryRequest 校验工作区创建请求的节点类型和内容边界：目录不得携带内容，文件必须是有效 UTF-8 且不得包含 NUL 字节。
func validateWorkspaceEntryRequest(request workspaceEntryCreateRequest) error {
	if request.NodeType != "file" && request.NodeType != "directory" {
		return errProjectInvalidArgument
	}
	if request.NodeType == "directory" {
		if request.Content != nil {
			return errProjectInvalidArgument
		}
		return nil
	}
	if request.Content == nil || !utf8.ValidString(*request.Content) || strings.Contains(*request.Content, "\x00") {
		return errProjectInvalidArgument
	}
	return nil
}

func ensureWorkspaceEntryAbsent(root *managedRootFS, path string) error {
	if _, err := root.root.Lstat(path); err == nil {
		return errProjectConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return mapWorkspacePathError(err)
	}
	return nil
}

func ensureWorkspaceEntrySafe(root *managedRootFS, path string) error {
	info, err := root.root.Lstat(path)
	if err != nil {
		return mapWorkspacePathError(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errProjectInvalidArgument
	}
	return nil
}

func ensureWorkspaceParent(root *managedRootFS, path string) error {
	parent := filepath.Dir(path)
	if parent == "." {
		return nil
	}
	return mapWorkspacePathError(root.root.MkdirAll(parent, managedCreateDirMode))
}

func (s *Service) deleteProjectWorkspaceEntry(ctx context.Context, projectID uint64, path string, recursive bool) error {
	s.workspaceMutationMu.Lock()
	defer s.workspaceMutationMu.Unlock()
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return err
	}
	rootDir, relativePath, err := resolveProjectWorkspaceFilePath(aggregate.Application.WorkspacePath, path)
	if err != nil {
		return err
	}
	if workspaceMutationContainsTrackedLifecycleInput(aggregate, relativePath) {
		return errProjectConflict
	}
	root, err := openManagedRootFS(rootDir)
	if err != nil {
		return mapWorkspacePathError(err)
	}
	defer func() { _ = closeManagedRootFS(root) }()
	info, err := root.root.Lstat(relativePath)
	if err != nil {
		return mapWorkspacePathError(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errProjectInvalidArgument
	}
	if info.IsDir() {
		if recursive {
			return mapWorkspacePathError(root.root.RemoveAll(relativePath))
		}
		return mapWorkspacePathError(root.root.Remove(relativePath))
	}
	return mapWorkspacePathError(root.root.Remove(relativePath))
}

// workspaceMutationContainsTrackedLifecycleInput 防止移动或删除仍被活动生命周期命令引用的文件及其上级目录。
func workspaceMutationContainsTrackedLifecycleInput(aggregate projectstore.ApplicationAggregate, path string) bool {
	for _, file := range aggregate.Files {
		relative, err := filepath.Rel(aggregate.Application.WorkspacePath, file.AbsolutePath)
		if err != nil {
			continue
		}
		relative, err = normalizeManagedWorkspacePath(relative)
		if err != nil {
			continue
		}
		if relative == path || strings.HasPrefix(relative, path+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (s *Service) updateProjectWorkspaceAnnotation(
	ctx context.Context,
	projectID uint64,
	path string,
	annotation *string,
	actorID *uint64,
) (workspaceFileItem, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return workspaceFileItem{}, err
	}
	rootDir, relativePath, err := resolveProjectWorkspaceDirectory(aggregate.Application.WorkspacePath, path)
	if err != nil {
		return workspaceFileItem{}, err
	}
	if relativePath == "" {
		return workspaceFileItem{}, errProjectInvalidArgument
	}
	root, err := openManagedRootFS(rootDir)
	if err != nil {
		return workspaceFileItem{}, mapWorkspacePathError(err)
	}
	defer func() { _ = closeManagedRootFS(root) }()
	info, err := root.root.Stat(relativePath)
	if err != nil {
		return workspaceFileItem{}, mapWorkspacePathError(err)
	}
	updatedAggregate, err := s.repository.UpdateWorkspaceAnnotation(ctx, projectstore.UpdateWorkspaceAnnotationInput{
		ApplicationRecordID: projectID,
		RelativePath:        relativePath,
		Annotation:          annotation,
		ActorID:             actorID,
	})
	if err != nil {
		return workspaceFileItem{}, mapStoreError(err)
	}
	entry := fs.FileInfoToDirEntry(info)
	hiddenDirectories, err := s.workspaceHiddenDirectories(ctx)
	if err != nil {
		return workspaceFileItem{}, err
	}
	fileTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ApplicationWorkspaceFileTooltipRulesConfig.String(), defaultWorkspaceFileTooltipRules)
	if err != nil {
		return workspaceFileItem{}, err
	}
	directoryTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ApplicationWorkspaceDirectoryTooltipRulesConfig.String(), defaultWorkspaceDirectoryTooltipRules)
	if err != nil {
		return workspaceFileItem{}, err
	}
	return buildProjectWorkspaceFileItem(relativePath, entry, workspaceTreeBuildContext{
		Root:                  root,
		TrackedKinds:          trackedProjectFileKinds(rootDir, updatedAggregate.Files),
		HiddenDirectories:     hiddenDirectories,
		FileTooltipRules:      fileTooltipRules,
		DirectoryTooltipRules: directoryTooltipRules,
		Annotations:           updatedAggregate.Application.WorkspaceAnnotations,
	})
}

func buildProjectWorkspaceFileItem(
	relativePath string,
	entry fs.DirEntry,
	buildContext workspaceTreeBuildContext,
) (workspaceFileItem, error) {
	nodeType := "file"
	if entry.IsDir() {
		nodeType = "directory"
	}
	info, err := entry.Info()
	if err != nil {
		return workspaceFileItem{}, fmt.Errorf("%w: %w", errProjectImportValidation, err)
	}
	state, err := resolveWorkspaceTreeItemState(buildContext.Root, relativePath, entry, buildContext.TrackedKinds)
	if err != nil {
		return workspaceFileItem{}, err
	}
	projectNote := strings.TrimSpace(buildContext.Annotations[normalizeWorkspaceRelative(relativePath)])
	tooltip, tooltipSource := resolveWorkspaceTooltip(
		entry.Name(),
		entry.IsDir(),
		projectNote,
		buildContext.FileTooltipRules,
		buildContext.DirectoryTooltipRules,
	)
	return workspaceFileItem{
		Name:            entry.Name(),
		RelativePath:    relativePath,
		NodeType:        nodeType,
		FileKind:        state.FileKind,
		Readable:        state.Readable,
		Editable:        state.Editable,
		LanguageHint:    state.LanguageHint,
		SizeBytes:       info.Size(),
		HiddenByDefault: shouldHideWorkspaceEntry(entry.Name(), entry.IsDir(), buildContext.HiddenDirectories),
		HasChildren:     nodeType == "directory",
		Tooltip:         tooltip,
		TooltipSource:   tooltipSource,
		ApplicationNote: projectNote,
	}, nil
}

func trackedProjectFileKinds(rootDir string, files []projectstore.ApplicationFile) map[string]string {
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

func classifyWorkspaceFile(relativePath string, trackedKinds map[string]string) (string, string) {
	normalized := normalizeWorkspaceRelative(relativePath)
	base := strings.ToLower(filepath.Base(normalized))
	if kind, ok := trackedKinds[normalized]; ok {
		return classifyTrackedWorkspaceKind(kind)
	}
	if classification, ok := classifyWorkspaceBaseName(base); ok {
		return classification.FileKind, classification.LanguageHint
	}
	if isWorkspaceEnvFileBaseName(base) {
		return "env", "dotenv"
	}
	return classifyWorkspaceExtension(strings.ToLower(filepath.Ext(base)))
}

func isWorkspaceEnvFileBaseName(base string) bool {
	return base == ".env" || strings.HasPrefix(base, ".env.") || strings.HasSuffix(base, ".env")
}

func classifyTrackedWorkspaceKind(kind string) (string, string) {
	switch kind {
	case projectcontract.FileKindCompose.String():
		return "compose", "yaml"
	case projectcontract.FileKindEnv.String():
		return "env", "dotenv"
	default:
		return "text", "plaintext"
	}
}

func classifyWorkspaceExtension(ext string) (string, string) {
	if ext == "" {
		return "text", "plaintext"
	}
	if classification, ok := workspaceExtensionClassifications[ext]; ok {
		return classification.FileKind, classification.LanguageHint
	}
	return "text", "plaintext"
}

func classifyWorkspaceBaseName(base string) (workspaceFileClassification, bool) {
	classification, ok := workspaceBaseNameClassifications[base]
	return classification, ok
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
	buildContext workspaceTreeBuildContext,
	showHidden bool,
) ([]workspaceFileItem, bool, error) {
	items := make([]workspaceFileItem, 0, len(entries))
	hasMoreHidden := false
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if !showHidden && shouldHideWorkspaceEntry(name, entry.IsDir(), buildContext.HiddenDirectories) {
			hasMoreHidden = true
			continue
		}
		relativePath := normalizeWorkspaceRelative(filepath.Join(currentPath, name))
		item, err := buildProjectWorkspaceFileItem(relativePath, entry, buildContext)
		if err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	return items, hasMoreHidden, nil
}

func ensureWorkspaceBrowsePath(root *managedRootFS, path string) error {
	info, err := root.root.Stat(path)
	if err != nil {
		return mapWorkspacePathError(err)
	}
	if !info.IsDir() {
		return errProjectInvalidArgument
	}
	return nil
}

func ensureWorkspaceSaveTarget(root *managedRootFS, path string) error {
	info, err := root.root.Stat(path)
	if err != nil {
		return mapWorkspacePathError(err)
	}
	if info.IsDir() {
		return errProjectInvalidArgument
	}
	return nil
}

func mapWorkspacePathError(err error) error {
	if os.IsNotExist(err) {
		return errProjectFileNotFound
	}
	return fmt.Errorf("%w: %w", errProjectImportValidation, err)
}

func workspaceParentPath(currentPath string) *string {
	normalized := normalizeWorkspaceRelative(currentPath)
	if normalized == "" {
		return nil
	}
	parent := normalizeWorkspaceRelative(filepath.Dir(normalized))
	return &parent
}

func shouldHideWorkspaceEntry(name string, isDirectory bool, extraHidden []string) bool {
	if isDirectory && strings.HasPrefix(name, ".") {
		return true
	}
	return isDirectory && slices.Contains(extraHidden, name)
}

func (s *Service) workspaceHiddenDirectories(ctx context.Context) ([]string, error) {
	if s == nil || s.configResolver == nil {
		return decodeWorkspaceHiddenDirectories(defaultWorkspaceHiddenDirectories)
	}
	raw, err := s.configResolver.ResolveDefaultConfig(ctx, projectcontract.ApplicationWorkspaceHiddenDirectoriesConfig.String())
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

func (s *Service) workspaceTooltipRules(
	ctx context.Context,
	configKey string,
	fallback string,
) ([]workspaceTooltipRule, error) {
	if s == nil || s.configResolver == nil {
		return decodeWorkspaceTooltipRules(fallback)
	}
	raw, err := s.configResolver.ResolveDefaultConfig(ctx, configKey)
	if err != nil {
		return decodeWorkspaceTooltipRules(fallback)
	}
	var encoded string
	if unmarshalErr := json.Unmarshal([]byte(raw), &encoded); unmarshalErr != nil {
		encoded = raw
	}
	rules, decodeErr := decodeWorkspaceTooltipRules(encoded)
	if decodeErr == nil {
		return rules, nil
	}
	return decodeWorkspaceTooltipRules(fallback)
}

func decodeWorkspaceTooltipRules(raw string) ([]workspaceTooltipRule, error) {
	var rules []workspaceTooltipRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, fmt.Errorf("%w: invalid workspace tooltip rule config", errProjectInvalidArgument)
	}
	normalized := make([]workspaceTooltipRule, 0, len(rules))
	for _, rule := range rules {
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" {
			continue
		}
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid workspace tooltip rule pattern", errProjectInvalidArgument)
		}
		normalized = append(normalized, workspaceTooltipRule{
			Enabled: rule.Enabled,
			Pattern: pattern,
			Tooltip: strings.TrimSpace(rule.Tooltip),
			regex:   regex,
		})
	}
	return normalized, nil
}

func resolveWorkspaceTooltip(
	name string,
	isDirectory bool,
	projectNote string,
	fileTooltipRules []workspaceTooltipRule,
	directoryTooltipRules []workspaceTooltipRule,
) (string, string) {
	if projectNote != "" {
		return projectNote, "project-note"
	}
	rules := fileTooltipRules
	if isDirectory {
		rules = directoryTooltipRules
	}
	baseName := filepath.Base(strings.TrimSpace(name))
	matchedTooltip := ""
	for _, rule := range rules {
		if rule.regex == nil || !rule.regex.MatchString(baseName) {
			continue
		}
		if !rule.Enabled {
			continue
		}
		matchedTooltip = rule.Tooltip
	}
	if matchedTooltip == "" {
		return "", ""
	}
	return matchedTooltip, "default-rule"
}

func resolveWorkspaceTreeItemState(
	root *managedRootFS,
	relativePath string,
	entry fs.DirEntry,
	trackedKinds map[string]string,
) (workspaceFileState, error) {
	if entry.IsDir() {
		return workspaceFileState{
			FileKind:     "directory",
			LanguageHint: "plaintext",
			Readable:     true,
			Editable:     false,
		}, nil
	}
	return resolveWorkspaceFileStateFromPath(root, relativePath, trackedKinds)
}

func resolveWorkspaceFileStateFromPath(
	root *managedRootFS,
	relativePath string,
	trackedKinds map[string]string,
) (workspaceFileState, error) {
	sample, err := readWorkspaceFileSample(root, relativePath)
	if err != nil {
		return unreadableWorkspaceFileState(relativePath, trackedKinds), nil
	}
	return resolveWorkspaceFileState(relativePath, trackedKinds, sample), nil
}

func resolveWorkspaceFileState(
	relativePath string,
	trackedKinds map[string]string,
	content []byte,
) workspaceFileState {
	fileKind, languageHint := classifyWorkspaceFile(relativePath, trackedKinds)
	if !isWorkspaceReadableText(content) {
		return workspaceFileState{
			FileKind:     "binary",
			LanguageHint: "plaintext",
			Readable:     false,
			Editable:     false,
		}
	}
	return workspaceFileState{
		FileKind:     fileKind,
		LanguageHint: languageHint,
		Readable:     true,
		Editable:     true,
	}
}

func unreadableWorkspaceFileState(relativePath string, trackedKinds map[string]string) workspaceFileState {
	fileKind, languageHint := classifyWorkspaceFile(relativePath, trackedKinds)
	return workspaceFileState{
		FileKind:     fileKind,
		LanguageHint: languageHint,
		Readable:     false,
		Editable:     false,
	}
}

func readWorkspaceFileSample(root *managedRootFS, path string) ([]byte, error) {
	file, err := root.root.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	buffer := make([]byte, workspaceReadableSampleLimit)
	count, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return append([]byte(nil), buffer[:count]...), nil
}

func isWorkspaceReadableText(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return false
	}
	trimmed := append([]byte(nil), content...)
	for len(trimmed) > 0 && !utf8.Valid(trimmed) {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return len(trimmed) > 0 && utf8.Valid(trimmed)
}
