package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	projectcompose "graft/server/modules/project/compose"
)

var applicationNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// CreateManagedProject writes managed project files under the configured managed root and persists the registry bootstrap.
func (s *Service) CreateManagedProject(
	ctx context.Context,
	request ManagedProjectCreateRequest,
	actorID *uint64,
) (result ManagedProjectCreateResult, err error) {
	s.applicationNameMu.Lock()
	defer s.applicationNameMu.Unlock()
	s.logManagedCreateDiagnostic("create_started",
		zap.String("application_name", stringValue(request.ApplicationName)),
		zap.Uint64("runtime_target_id", request.RuntimeTargetID),
		zap.Int("workspace_entry_count", len(request.WorkspaceEntries)),
		zap.Bool("reuse_existing_workspace", request.ReuseExistingWorkspace),
	)
	validation, normalized, err := s.prepareManagedCreate(ctx, request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	restoreState, err := snapshotManagedCreateWorkspaceIfNeeded(validation, normalized.WorkspaceEntries)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}

	var createdDir string
	var createdFiles []string
	shouldCleanup := !validation.ReusedExistingWorkspace
	// 复用工作区失败时必须先用快照根执行回滚，再关闭根句柄，避免恢复退回到不受约束的绝对路径操作。
	defer func() {
		if shouldCleanup {
			err = errors.Join(err, cleanupManagedCreate(createdDir, createdFiles))
		} else if err != nil {
			err = errors.Join(err, restoreManagedCreateWorkspace(restoreState))
		}
		if restoreState != nil {
			err = errors.Join(err, closeManagedRootFS(restoreState.root))
		}
	}()

	materialization, err := s.materializeManagedCreate(validation, normalized)
	createdDir = materialization.directory
	createdFiles = materialization.files
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	parseResult := materialization.parse

	aggregate, now, err := s.createProjectFromWorkspace(ctx, managedCreationCommand(validation, normalized, parseResult, actorID))
	if err != nil {
		s.logManagedCreateDiagnostic("registry_persist_failed", zap.Error(err))
		return ManagedProjectCreateResult{}, err
	}
	shouldCleanup = false
	if restoreState != nil {
		restoreState.items = nil
	}
	s.logManagedCreateDiagnostic("create_succeeded", zap.Uint64("project_id", aggregate.Project.ID))
	result = ManagedProjectCreateResult{
		Validation:           validation,
		SourceType:           "managed",
		ProjectID:            aggregate.Project.ID,
		ApplicationID:        aggregate.Project.ApplicationID,
		ConfigHash:           parseResult.ConfigHash,
		DeclaredServiceCount: len(parseResult.ServiceNames),
		RefreshedAt:          now,
	}
	return result, nil
}

func (s *Service) prepareManagedCreate(ctx context.Context, request ManagedProjectCreateRequest) (ManagedProjectCreateValidationResult, normalizedManagedCreateRequest, error) {
	validation, err := s.ValidateManagedCreate(ctx, request)
	if err != nil {
		s.logManagedCreateDiagnostic("validation_failed", zap.Error(err))
		return ManagedProjectCreateValidationResult{}, normalizedManagedCreateRequest{}, err
	}
	s.logManagedCreateDiagnostic("validation_succeeded",
		zap.String("application_name", stringValue(validation.ApplicationName)),
		zap.Bool("reused_existing_workspace", validation.ReusedExistingWorkspace),
	)
	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		s.logManagedCreateDiagnostic("normalization_failed", zap.Error(err))
		return ManagedProjectCreateValidationResult{}, normalizedManagedCreateRequest{}, err
	}
	if err := ensureManagedCreatePathsUnderRoot(validation); err != nil {
		s.logManagedCreateDiagnostic("workspace_boundary_failed", zap.Error(err))
		return ManagedProjectCreateValidationResult{}, normalizedManagedCreateRequest{}, err
	}
	return validation, normalized, nil
}

type managedCreateRestoreState struct {
	// root 在整个创建事务期间保持打开，确保恢复操作绑定到快照时校验过的工作区目录。
	root  *managedRootFS
	items []managedCreateRestoreItem
}

func snapshotManagedCreateWorkspaceIfNeeded(validation ManagedProjectCreateValidationResult, entries []ManagedWorkspaceEntry) (*managedCreateRestoreState, error) {
	if !validation.ReusedExistingWorkspace {
		return nil, nil
	}
	root, err := openManagedRootFS(validation.WorkingDirectory)
	if err != nil {
		return nil, fmt.Errorf("open existing workspace root: %w", err)
	}
	items, err := snapshotManagedCreateWorkspace(root, entries)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("snapshot reused workspace: %w", err), closeManagedRootFS(root))
	}
	return &managedCreateRestoreState{root: root, items: items}, nil
}

func (s *Service) materializeManagedCreate(validation ManagedProjectCreateValidationResult, normalized normalizedManagedCreateRequest) (managedCreateMaterialization, error) {
	createdDir, createdFiles, err := writeManagedProjectFiles(validation, normalized)
	if err != nil {
		s.logManagedCreateDiagnostic("workspace_materialization_failed", zap.Error(err))
		return managedCreateMaterialization{directory: createdDir, files: createdFiles}, fmt.Errorf("%w: %w", errors.Join(errProjectImportValidation, errProjectWorkspaceWriteFailed), err)
	}
	s.logManagedCreateDiagnostic("workspace_materialized", zap.Int("created_file_count", len(createdFiles)))
	parseResult, err := projectcompose.Load(projectcompose.Input{
		WorkingDirectory: validation.WorkspacePath,
		ComposeFiles:     []string{validation.ComposeFileAbsolutePath},
		EnvFiles:         managedCreateEnvFileList(validation.EnvFileAbsolutePath),
	})
	if err != nil {
		s.logManagedCreateDiagnostic("compose_parse_failed", zap.Error(err))
		return managedCreateMaterialization{directory: createdDir, files: createdFiles}, fmt.Errorf("%w: %w", errors.Join(errProjectImportValidation, errProjectInvalidCompose), err)
	}
	return managedCreateMaterialization{directory: createdDir, files: createdFiles, parse: parseResult}, nil
}

type managedCreateRestoreItem struct {
	path    string
	content []byte
	exists  bool
}

func snapshotManagedCreateWorkspace(root *managedRootFS, entries []ManagedWorkspaceEntry) ([]managedCreateRestoreItem, error) {
	items := make([]managedCreateRestoreItem, 0, len(entries))
	for _, entry := range entries {
		if entry.NodeType != "file" {
			continue
		}
		content, err := root.root.ReadFile(entry.Path)
		if err == nil {
			items = append(items, managedCreateRestoreItem{path: entry.Path, content: append([]byte(nil), content...), exists: true})
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read existing workspace file %s: %w", entry.Path, err)
		}
		items = append(items, managedCreateRestoreItem{path: entry.Path})
	}
	return items, nil
}

func restoreManagedCreateWorkspace(state *managedCreateRestoreState) (err error) {
	if state == nil || state.root == nil {
		return nil
	}
	for index := len(state.items) - 1; index >= 0; index-- {
		item := state.items[index]
		if item.exists {
			err = errors.Join(err, state.root.root.WriteFile(item.path, item.content, managedCreateFileMode))
			continue
		}
		removeErr := state.root.root.Remove(item.path)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}
	return err
}

type normalizedManagedCreateRequest struct {
	DisplayName        string
	RuntimeTargetID    uint64
	ApplicationName    *string
	ComposeFileName    string
	ComposeFileContent string
	EnvFileName        *string
	EnvFileContent     *string
	WorkspaceEntries   []ManagedWorkspaceEntry
	ComposeFilePath    string
	EnvFilePaths       []string
	LifecycleConfig    *LifecycleStandardConfig
}

type managedCreateMaterialization struct {
	directory string
	files     []string
	parse     projectcompose.Result
}

type normalizedManagedWorkspaceEntry struct {
	Path     string
	NodeType string
	Content  *string
}

// normalizeManagedCreateRequest 规范化并验证受控项目创建请求，生成用于后续项目创建的输入数据。
// 它会校验项目名称、目录、Compose 文件、可选环境文件及工作区文件，并保留生命周期配置和环境文件路径。
// 返回规范化后的请求数据及验证错误。
//
// normalizeManagedCreateRequest 校验并规范化受控项目创建请求，生成用于后续创建流程的工作区配置。
// normalizeManagedCreateRequest 规范化受控项目创建请求及其工作区条目。
// normalizeManagedCreateRequest 规范化并校验受控项目创建请求，生成可用于创建项目的文件与工作区配置。
// @param request 待规范化的项目创建请求。
// @return 规范化后的项目创建请求及可能发生的校验错误。
func normalizeManagedCreateRequest(request ManagedProjectCreateRequest) (normalizedManagedCreateRequest, error) {
	identity, err := normalizeManagedCreateIdentity(request)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	composeName, composeFileContent, err := ensureComposeProjectName(identity.composeContent, *identity.applicationName)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	if err := rejectComposeProjectNameOverride(identity.envContent, composeName); err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	workspaceEntries, err := normalizeManagedWorkspaceEntries(request.WorkspaceEntries, identity.composePath)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	materializedEntries := make([]ManagedWorkspaceEntry, 0, len(workspaceEntries))
	for _, item := range workspaceEntries {
		content := item.Content
		if item.NodeType == "file" && item.Path == identity.composePath {
			content = stringPointer(composeFileContent)
		}
		materializedEntries = append(materializedEntries, ManagedWorkspaceEntry{Path: item.Path, NodeType: item.NodeType, Content: content})
	}
	return normalizedManagedCreateRequest{
		DisplayName:        identity.displayName,
		RuntimeTargetID:    request.RuntimeTargetID,
		ApplicationName:    identity.applicationName,
		ComposeFileName:    identity.composePath,
		ComposeFileContent: composeFileContent,
		EnvFileName:        identity.envName,
		EnvFileContent:     identity.envContent,
		WorkspaceEntries:   materializedEntries,
		ComposeFilePath:    identity.composePath,
		EnvFilePaths:       append([]string(nil), request.EnvFilePaths...),
		LifecycleConfig:    request.LifecycleConfig,
	}, nil
}

// normalizeManagedWorkspaceEntries 标准化并验证工作区条目，确保其中包含 Compose 文件。
//
// 返回标准化后的工作区条目；条目为空或未包含 Compose 文件时返回错误。
func normalizeManagedWorkspaceEntries(entries []ManagedWorkspaceEntry, composePath string) ([]normalizedManagedWorkspaceEntry, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: workspace entries are required", errProjectInvalidArgument)
	}
	if len(entries) > maxWorkspaceEntryCount {
		return nil, fmt.Errorf("%w: workspace entry limit exceeded", errProjectInvalidArgument)
	}
	result := make([]normalizedManagedWorkspaceEntry, 0, len(entries))
	seen := make(map[string]string, len(entries))
	foundCompose := false
	totalBytes := 0
	for _, entry := range entries {
		item, isCompose, err := normalizeManagedWorkspaceEntry(entry, composePath, seen)
		if err != nil {
			return nil, err
		}
		if isCompose {
			foundCompose = true
		}
		seen[item.Path] = item.NodeType
		if err := accumulateWorkspaceEntrySize(item, &totalBytes); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := validateWorkspaceEntryRelationships(seen); err != nil {
		return nil, err
	}
	if !foundCompose {
		return nil, fmt.Errorf("%w: compose file is absent from workspace", errProjectInvalidArgument)
	}
	return result, nil
}

func accumulateWorkspaceEntrySize(item normalizedManagedWorkspaceEntry, totalBytes *int) error {
	if item.NodeType != "file" {
		return nil
	}
	if len(*item.Content) > maxWorkspaceFileBytes {
		return fmt.Errorf("%w: workspace file size limit exceeded", errProjectInvalidArgument)
	}
	*totalBytes += len(*item.Content)
	if *totalBytes > maxWorkspaceTotalBytes {
		return fmt.Errorf("%w: workspace total size limit exceeded", errProjectInvalidArgument)
	}
	return nil
}

// normalizeManagedWorkspaceEntry 规范化并验证工作区条目，并指示该条目是否为 Compose 文件。
//
// 返回规范化后的路径、节点类型和内容；如果条目表示指定路径的文件，返回值中的布尔值为 true。
func normalizeManagedWorkspaceEntry(entry ManagedWorkspaceEntry, composePath string, seen map[string]string) (normalizedManagedWorkspaceEntry, bool, error) {
	path, err := normalizeManagedWorkspacePath(entry.Path)
	if err != nil {
		return normalizedManagedWorkspaceEntry{}, false, err
	}
	nodeType := strings.TrimSpace(entry.NodeType)
	if err := validateManagedWorkspaceEntry(path, nodeType, entry.Content, seen); err != nil {
		return normalizedManagedWorkspaceEntry{}, false, err
	}
	return normalizedManagedWorkspaceEntry{Path: path, NodeType: nodeType, Content: entry.Content}, nodeType == "file" && path == composePath, nil
}

// validateManagedWorkspaceEntry validates a workspace entry's type, uniqueness, ancestor relationships, and content.
func validateManagedWorkspaceEntry(path, nodeType string, content *string, seen map[string]string) error {
	if nodeType != "file" && nodeType != "directory" {
		return fmt.Errorf("%w: invalid workspace entry type", errProjectInvalidArgument)
	}
	if _, ok := seen[path]; ok {
		return fmt.Errorf("%w: duplicate workspace entry", errProjectInvalidArgument)
	}
	if nodeType == "directory" {
		return validateWorkspaceDirectoryContent(content)
	}
	return validateWorkspaceFileContent(content)
}

// validateWorkspaceEntryRelationships rejects every file/descendant conflict after all paths are known.
func validateWorkspaceEntryRelationships(entries map[string]string) error {
	for path := range entries {
		for ancestor := filepath.Dir(path); ancestor != "."; ancestor = filepath.Dir(ancestor) {
			if entries[ancestor] == "file" {
				return fmt.Errorf("%w: workspace entry has file ancestor", errProjectInvalidArgument)
			}
		}
	}
	return nil
}

// validateWorkspaceDirectoryContent validates that a workspace directory has no content.
func validateWorkspaceDirectoryContent(content *string) error {
	if content != nil {
		return fmt.Errorf("%w: workspace directory cannot have content", errProjectInvalidArgument)
	}
	return nil
}

// validateWorkspaceFileContent validates that workspace file content is present, valid UTF-8 text, and contains no NUL characters.
func validateWorkspaceFileContent(content *string) error {
	if content == nil || !utf8.ValidString(*content) || strings.Contains(*content, "\x00") {
		return fmt.Errorf("%w: workspace file must be UTF-8 text", errProjectInvalidArgument)
	}
	return nil
}

type managedCreateIdentity struct {
	displayName, composeContent, composePath string
	applicationName                          *string
	envName                                  *string
	envContent                               *string
}

type managedCreateFiles struct {
	composePath string
	envName     *string
	envContent  *string
}

// normalizeManagedCreateIdentity trims and validates the required managed-project identity and file fields.
// It returns the normalized display name, application name, compose content and path, and optional
// environment-file information.
func normalizeManagedCreateIdentity(request ManagedProjectCreateRequest) (managedCreateIdentity, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	composeContent := strings.TrimSpace(request.ComposeFileContent)
	if displayName == "" || composeContent == "" {
		return managedCreateIdentity{}, fmt.Errorf("%w: missing required managed-create fields", errProjectInvalidArgument)
	}
	applicationName, err := normalizeApplicationName(request.ApplicationName)
	if err != nil {
		return managedCreateIdentity{}, err
	}
	files, err := normalizeManagedCreateFiles(request)
	if err != nil {
		return managedCreateIdentity{}, err
	}
	return managedCreateIdentity{displayName: displayName, composeContent: composeContent, composePath: files.composePath, applicationName: applicationName, envName: files.envName, envContent: files.envContent}, nil
}

// normalizeManagedCreateFiles normalizes the compose and optional environment file names and content for a managed project creation request.
func normalizeManagedCreateFiles(request ManagedProjectCreateRequest) (managedCreateFiles, error) {
	composePath, err := normalizeManagedFileName(request.ComposeFileName, "compose")
	if err != nil {
		return managedCreateFiles{}, err
	}
	if strings.TrimSpace(request.ComposeFilePath) != "" {
		composePath, err = normalizeManagedWorkspacePath(request.ComposeFilePath)
		if err != nil {
			return managedCreateFiles{}, err
		}
	}
	envName, err := normalizeManagedOptionalFileName(request.EnvFileName, "env")
	if err != nil {
		return managedCreateFiles{}, err
	}
	envContent := normalizeManagedOptionalContent(request.EnvFileContent)
	if envName == nil {
		envContent = nil
	}
	return managedCreateFiles{composePath: composePath, envName: envName, envContent: envContent}, nil
}

// rejectComposeProjectNameOverride 检查内容中的 COMPOSE_PROJECT_NAME 是否与指定的 compose 名称一致。
// rejectComposeProjectNameOverride 校验内容中的 COMPOSE_PROJECT_NAME 配置是否与指定的 Compose 项目名称一致；内容为空或未包含该配置时通过，否则返回参数错误。
func rejectComposeProjectNameOverride(content *string, composeName string) error {
	if content == nil {
		return nil
	}
	for _, line := range strings.Split(*content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "COMPOSE_PROJECT_NAME=") {
			continue
		}
		valueRaw := strings.TrimPrefix(line, "COMPOSE_PROJECT_NAME=")
		if index := strings.Index(valueRaw, "#"); index >= 0 {
			valueRaw = valueRaw[:index]
		}
		value := strings.Trim(strings.TrimSpace(valueRaw), "\"'")
		if value != composeName {
			return fmt.Errorf("%w: COMPOSE_PROJECT_NAME conflicts with compose name", errProjectInvalidArgument)
		}
	}
	return nil
}

// normalizeApplicationName 验证并返回规范化的机器安全应用名称。
//
// 名称必须非空，且只能包含小写字母、数字和连字符，并以小写字母或数字开头。
func normalizeApplicationName(requested *string) (*string, error) {
	if requested == nil {
		return nil, errProjectApplicationNameRequired
	}
	name := strings.TrimSpace(*requested)
	if name == "" {
		return nil, errProjectApplicationNameRequired
	}
	if !applicationNamePattern.MatchString(name) {
		return nil, errProjectInvalidApplicationName
	}
	return &name, nil
}

// normalizeManagedWorkspacePath validates and normalizes a relative workspace file path.
// normalizeManagedWorkspacePath 规范化工作区相对路径，并拒绝空路径、绝对路径、包含反斜杠或逃逸出项目目录的路径。
// normalizeManagedWorkspacePath 规范化工作区相对路径，并返回使用斜杠分隔的路径。
// 如果路径为空、为绝对路径、包含反斜杠或会逃逸项目目录，则返回错误。
func normalizeManagedWorkspacePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, `\`) {
		return "", fmt.Errorf("%w: invalid workspace file path", errProjectInvalidArgument)
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: workspace file path escapes project", errProjectInvalidArgument)
	}
	return filepath.ToSlash(cleaned), nil
}

// normalizeManagedFileName 校验并规范化文件名，去除首尾空白。
// 返回有效的文件名；输入为空、包含路径、为 "." 或路径分隔符时返回错误。
func normalizeManagedFileName(value string, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "." || trimmed == string(filepath.Separator) {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	if filepath.IsAbs(trimmed) || strings.Contains(trimmed, "/") || strings.Contains(trimmed, `\`) {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	fileName := filepath.Base(trimmed)
	if fileName == "" || fileName == "." || fileName != trimmed {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	return fileName, nil
}

// normalizeManagedOptionalFileName 规范化可选文件名，空值或纯空白时返回 nil。
//
// 当输入为空或仅包含空白字符时，返回 nil。
func normalizeManagedOptionalFileName(value *string, label string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	fileName, err := normalizeManagedFileName(*value, label)
	if err != nil {
		return nil, fmt.Errorf("normalize %s file name: %w", label, err)
	}
	return &fileName, nil
}

// managedCreateEnvAbsolutePath 生成受控项目环境文件的绝对路径。
//
// 当未提供环境文件名时返回 nil；否则返回工作目录下对应环境文件的绝对路径。
func managedCreateEnvAbsolutePath(workingDirectory string, envFileName *string) *string {
	if envFileName == nil {
		return nil
	}
	envAbs := filepath.Join(workingDirectory, *envFileName)
	return &envAbs
}

// normalizeManagedOptionalContent 去除可选内容两端的空白字符并返回结果。
//
// @param value 待规范化的内容指针。
// @returns 规范化后的内容指针；当输入为 nil 时返回 nil。
func normalizeManagedOptionalContent(value *string) *string {
	if value == nil {
		return nil
	}
	content := strings.TrimSpace(*value)
	return &content
}

// ensureManagedCreatePathsUnderRoot 验证托管项目工作目录位于已配置的 managed root 下。
// ensureManagedCreatePathsUnderRoot 验证托管项目工作目录位于已配置的 managed root 内。
//
// 当未配置根目录，或工作目录与根目录之间不存在有效的相对关系，或工作目录超出 managed root 时，返回 errProjectInvalidArgument。
func ensureManagedCreatePathsUnderRoot(validation ManagedProjectCreateValidationResult) error {
	if validation.ManagedRoot.ConfiguredRootDirectory == nil {
		return errProjectInvalidArgument
	}
	root := filepath.Clean(*validation.ManagedRoot.ConfiguredRootDirectory)
	workingDirectory := filepath.Clean(validation.WorkingDirectory)
	relative, err := filepath.Rel(root, workingDirectory)
	if err != nil {
		return fmt.Errorf("%w: invalid managed root relationship", errProjectInvalidArgument)
	}
	if relative == "." || relative == "" || strings.HasPrefix(relative, "..") {
		return fmt.Errorf("%w: managed project directory must stay under managed root", errProjectInvalidArgument)
	}
	return nil
}

// guardCode 返回仅包含指定代码的守卫结果。
func guardCode(code string) GuardResult {
	return GuardResult{Code: code}
}

func guardDetail(code string, detail string) GuardResult {
	detail = strings.TrimSpace(detail)
	return GuardResult{Code: code, Detail: &detail}
}
