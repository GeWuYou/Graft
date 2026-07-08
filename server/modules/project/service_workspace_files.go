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

type workspaceTreeBuildContext struct {
	RootPath              string
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
	fileTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ProjectWorkspaceFileTooltipRulesConfig.String(), defaultWorkspaceFileTooltipRules)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	directoryTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ProjectWorkspaceDirectoryTooltipRulesConfig.String(), defaultWorkspaceDirectoryTooltipRules)
	if err != nil {
		return workspaceFilesResult{}, err
	}
	buildContext := workspaceTreeBuildContext{
		RootPath:              rootDir,
		TrackedKinds:          trackedProjectFileKinds(rootDir, aggregate.Files),
		HiddenDirectories:     hiddenDirectories,
		FileTooltipRules:      fileTooltipRules,
		DirectoryTooltipRules: directoryTooltipRules,
		Annotations:           aggregate.Project.WorkspaceAnnotations,
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
	// #nosec G304 -- absolutePath is validated to stay under the project's working_directory before reading.
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return workspaceFileContentResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	state := resolveWorkspaceFileState(relativePath, trackedProjectFileKinds(rootDir, aggregate.Files), content)
	if !state.Readable {
		return workspaceFileContentResult{}, errProjectInvalidArgument
	}
	return workspaceFileContentResult{
		ProjectID:    projectID,
		RelativePath: relativePath,
		FileKind:     state.FileKind,
		LanguageHint: state.LanguageHint,
		Readable:     state.Readable,
		Editable:     state.Editable,
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
	absolutePath := filepath.Join(rootDir, relativePath)
	if err := ensureWorkspaceSaveTarget(absolutePath); err != nil {
		return workspaceFileSaveResult{}, err
	}
	// #nosec G304 -- absolutePath is validated to stay under the project's working_directory before reading.
	existingContent, err := os.ReadFile(absolutePath)
	if err != nil {
		return workspaceFileSaveResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	state := resolveWorkspaceFileState(relativePath, trackedProjectFileKinds(rootDir, aggregate.Files), existingContent)
	if !state.Editable {
		return workspaceFileSaveResult{}, errProjectInvalidArgument
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
	rootDir, relativePath, err := resolveProjectWorkspaceDirectory(aggregate.Project.WorkingDirectory, path)
	if err != nil {
		return workspaceFileItem{}, err
	}
	if relativePath == "" {
		return workspaceFileItem{}, errProjectInvalidArgument
	}
	absolutePath := filepath.Join(rootDir, relativePath)
	info, err := os.Stat(absolutePath)
	if err != nil {
		return workspaceFileItem{}, mapWorkspacePathError(err)
	}
	updatedAggregate, err := s.repository.UpdateWorkspaceAnnotation(ctx, projectstore.UpdateWorkspaceAnnotationInput{
		ProjectID:    projectID,
		RelativePath: relativePath,
		Annotation:   annotation,
		ActorID:      actorID,
	})
	if err != nil {
		return workspaceFileItem{}, mapStoreError(err)
	}
	entry := fs.FileInfoToDirEntry(info)
	hiddenDirectories, err := s.workspaceHiddenDirectories(ctx)
	if err != nil {
		return workspaceFileItem{}, err
	}
	fileTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ProjectWorkspaceFileTooltipRulesConfig.String(), defaultWorkspaceFileTooltipRules)
	if err != nil {
		return workspaceFileItem{}, err
	}
	directoryTooltipRules, err := s.workspaceTooltipRules(ctx, projectcontract.ProjectWorkspaceDirectoryTooltipRulesConfig.String(), defaultWorkspaceDirectoryTooltipRules)
	if err != nil {
		return workspaceFileItem{}, err
	}
	return buildProjectWorkspaceFileItem(relativePath, entry, workspaceTreeBuildContext{
		TrackedKinds:          trackedProjectFileKinds(rootDir, updatedAggregate.Files),
		HiddenDirectories:     hiddenDirectories,
		FileTooltipRules:      fileTooltipRules,
		DirectoryTooltipRules: directoryTooltipRules,
		Annotations:           updatedAggregate.Project.WorkspaceAnnotations,
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
		return workspaceFileItem{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	state, err := resolveWorkspaceTreeItemState(buildContext.RootPath, relativePath, entry, buildContext.TrackedKinds)
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
		ProjectNote:     projectNote,
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
	rootDir string,
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
	return resolveWorkspaceFileStateFromPath(filepath.Join(rootDir, relativePath), relativePath, trackedKinds)
}

func resolveWorkspaceFileStateFromPath(
	absolutePath string,
	relativePath string,
	trackedKinds map[string]string,
) (workspaceFileState, error) {
	// #nosec G304 -- absolutePath is already constrained to the validated project root before probing file content.
	sample, err := readWorkspaceFileSample(absolutePath)
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

func readWorkspaceFileSample(path string) ([]byte, error) {
	// #nosec G304 -- path is already constrained to a validated file path under the project working directory.
	file, err := os.Open(path)
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
