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

	projectcompose "graft/server/modules/project/compose"
)

var workspaceKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// CreateManagedProject writes managed project files under the configured managed root and persists the registry bootstrap.
func (s *Service) CreateManagedProject(
	ctx context.Context,
	request ManagedProjectCreateRequest,
	actorID *uint64,
) (result ManagedProjectCreateResult, err error) {
	validation, err := s.ValidateManagedCreate(ctx, request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	if err := ensureManagedCreatePathsUnderRoot(validation); err != nil {
		return ManagedProjectCreateResult{}, err
	}

	createdDir, createdFiles, err := writeManagedProjectFiles(validation, normalized)
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	shouldCleanup := true
	defer func() {
		if shouldCleanup {
			err = errors.Join(err, cleanupManagedCreate(createdDir, createdFiles))
		}
	}()

	parseResult, err := projectcompose.Load(projectcompose.Input{
		WorkingDirectory: validation.WorkspacePath,
		ComposeFiles:     []string{validation.ComposeFileAbsolutePath},
		EnvFiles:         managedCreateEnvFileList(validation.EnvFileAbsolutePath),
	})
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}

	aggregate, now, err := s.createProjectFromWorkspace(ctx, managedCreationCommand(validation, normalized, parseResult, actorID))
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	shouldCleanup = false
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

type normalizedManagedCreateRequest struct {
	DisplayName          string
	RuntimeTargetID      uint64
	WorkspaceKey         *string
	CanonicalProjectName string
	ComposeFileName      string
	ComposeFileContent   string
	EnvFileName          *string
	EnvFileContent       *string
	WorkspaceFiles       []ManagedWorkspaceFile
	ComposeFilePath      string
	EnvFilePaths         []string
	LifecycleConfig      *LifecycleStandardConfig
}

type normalizedManagedWorkspaceFile struct {
	Path    string
	Content string
}

// normalizeManagedCreateRequest 规范化并验证受控项目创建请求，生成用于后续项目创建的输入数据。
// 它会校验项目名称、目录、Compose 文件、可选环境文件及工作区文件，并保留生命周期配置和环境文件路径。
// 返回规范化后的请求数据及验证错误。
//
//nolint:cyclop,gocognit,gocyclo // The coupled request fields must be normalized together before any managed-root write.
func normalizeManagedCreateRequest(request ManagedProjectCreateRequest) (normalizedManagedCreateRequest, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	composeFileContent := strings.TrimSpace(request.ComposeFileContent)
	workspaceKey, err := normalizeOrDeriveWorkspaceKey(request.DisplayName, request.WorkspaceKey)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	if strings.TrimSpace(request.CanonicalProjectName) != "" {
		if _, err := validateExplicitCanonicalProjectName(request.CanonicalProjectName); err != nil {
			return normalizedManagedCreateRequest{}, err
		}
	}
	if strings.TrimSpace(request.RelativeProjectDirectory) != "" {
		legacyKey, err := normalizeManagedRelativeDirectory(request.RelativeProjectDirectory)
		if err != nil {
			return normalizedManagedCreateRequest{}, err
		}
		workspaceKey = &legacyKey
	}
	composeFileName, err := normalizeManagedFileName(request.ComposeFileName, "compose")
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	if displayName == "" || composeFileContent == "" {
		return normalizedManagedCreateRequest{}, fmt.Errorf("%w: missing required managed-create fields", errProjectInvalidArgument)
	}
	envFileName, err := normalizeManagedOptionalFileName(request.EnvFileName, "env")
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	envFileContent := normalizeManagedOptionalContent(request.EnvFileContent)
	if envFileName == nil {
		envFileContent = nil
	}
	composeFilePath := composeFileName
	if strings.TrimSpace(request.ComposeFilePath) != "" {
		composeFilePath, err = normalizeManagedWorkspacePath(request.ComposeFilePath)
		if err != nil {
			return normalizedManagedCreateRequest{}, err
		}
	}
	composeName, composeFileContent, err := ensureComposeProjectName(composeFileContent, displayName)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	if err := rejectComposeProjectNameOverride(envFileContent, composeName); err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	workspaceFiles, err := normalizeManagedWorkspaceFiles(request.WorkspaceFiles, composeFilePath, composeFileContent, envFileName, envFileContent)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	materializedFiles := make([]ManagedWorkspaceFile, 0, len(workspaceFiles))
	for _, item := range workspaceFiles {
		content := item.Content
		if item.Path == composeFilePath {
			content = composeFileContent
		}
		materializedFiles = append(materializedFiles, ManagedWorkspaceFile{Path: item.Path, Content: content})
	}
	return normalizedManagedCreateRequest{
		DisplayName:          displayName,
		RuntimeTargetID:      request.RuntimeTargetID,
		WorkspaceKey:         workspaceKey,
		CanonicalProjectName: "",
		ComposeFileName:      composeFilePath,
		ComposeFileContent:   composeFileContent,
		EnvFileName:          envFileName,
		EnvFileContent:       envFileContent,
		WorkspaceFiles:       materializedFiles,
		ComposeFilePath:      request.ComposeFilePath,
		EnvFilePaths:         append([]string(nil), request.EnvFilePaths...),
		LifecycleConfig:      request.LifecycleConfig,
	}, nil
}

func rejectComposeProjectNameOverride(content *string, composeName string) error {
	if content == nil {
		return nil
	}
	for _, line := range strings.Split(*content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "COMPOSE_PROJECT_NAME=") {
			continue
		}
		value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "COMPOSE_PROJECT_NAME=")), "\"'")
		if value != composeName {
			return fmt.Errorf("%w: COMPOSE_PROJECT_NAME conflicts with compose name", errProjectInvalidArgument)
		}
	}
	return nil
}

func normalizeOrDeriveWorkspaceKey(displayName string, requested *string) (*string, error) {
	key := ""
	if requested != nil {
		key = strings.TrimSpace(*requested)
	}
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(displayName))
		key = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			return '-'
		}, key)
		key = strings.Trim(key, "-")
	}
	if key == "" || !workspaceKeyPattern.MatchString(key) {
		return nil, fmt.Errorf("%w: invalid workspace key", errProjectInvalidArgument)
	}
	return &key, nil
}

func chooseWorkspacePath(root string, requested *string, explicit bool) (string, *string, error) {
	if requested == nil {
		return "", nil, errProjectInvalidArgument
	}
	base := *requested
	for suffix := 1; suffix < 10000; suffix++ {
		key := base
		if suffix > 1 {
			key = fmt.Sprintf("%s-%d", base, suffix)
		}
		path := filepath.Join(root, key)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return path, &key, nil
		}
		if err != nil {
			return "", nil, fmt.Errorf("%w: workspace path unavailable", errProjectInvalidArgument)
		}
		if explicit {
			return "", nil, fmt.Errorf("%w: workspace key already exists; suggested=%s-%d", errProjectConflict, base, suffix)
		}
	}
	return "", nil, errProjectConflict
}

// normalizeManagedWorkspaceFiles validates and normalizes workspace files, supplying default compose and environment files when none are provided.
func normalizeManagedWorkspaceFiles(items []ManagedWorkspaceFile, composeFileName, composeContent string, envFileName *string, envContent *string) ([]normalizedManagedWorkspaceFile, error) {
	if len(items) == 0 {
		items = []ManagedWorkspaceFile{{Path: composeFileName, Content: composeContent}}
		if envFileName != nil && envContent != nil {
			items = append(items, ManagedWorkspaceFile{Path: *envFileName, Content: *envContent})
		}
	}
	result := make([]normalizedManagedWorkspaceFile, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		path, err := normalizeManagedWorkspacePath(item.Path)
		if err != nil {
			return nil, err
		}
		if !utf8.ValidString(item.Content) || strings.Contains(item.Content, "\x00") {
			return nil, fmt.Errorf("%w: workspace file must be text", errProjectInvalidArgument)
		}
		if _, ok := seen[path]; ok {
			return nil, fmt.Errorf("%w: duplicate workspace file", errProjectInvalidArgument)
		}
		seen[path] = struct{}{}
		result = append(result, normalizedManagedWorkspaceFile{Path: path, Content: item.Content})
	}
	if _, ok := seen[composeFileName]; !ok {
		return nil, fmt.Errorf("%w: compose file is absent from workspace", errProjectInvalidArgument)
	}
	return result, nil
}

// normalizeManagedWorkspacePath validates and normalizes a relative workspace file path.
// It rejects absolute paths, backslashes, empty paths, and paths that escape the project.
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

// normalizeManagedRelativeDirectory 规范化并校验 managed-create 的相对项目目录。
// 它会去除首尾空白、统一路径分隔符并清理路径，同时确保目录保持在 managed root 下。
// @param value 待规范化的相对目录。
// @returns 规范化后的相对目录；当目录为空、为绝对路径或会逃逸 managed root 时返回错误。
func normalizeManagedRelativeDirectory(value string) (string, error) {
	relativeDir := strings.TrimSpace(value)
	if relativeDir == "" {
		return "", fmt.Errorf("%w: missing required managed-create fields", errProjectInvalidArgument)
	}
	if filepath.IsAbs(relativeDir) || relativeDir == "." || relativeDir == ".." {
		return "", fmt.Errorf("%w: relative project directory must stay under managed root", errProjectInvalidArgument)
	}
	if strings.Contains(relativeDir, `\`) {
		relativeDir = strings.ReplaceAll(relativeDir, `\`, "/")
	}
	relativeDir = filepath.Clean(relativeDir)
	if relativeDir == "." || strings.HasPrefix(relativeDir, "..") {
		return "", fmt.Errorf("%w: relative project directory escapes managed root", errProjectInvalidArgument)
	}
	return relativeDir, nil
}

// 它会去除首尾空白并提取最后一个路径段；当结果为空、`.` 或路径分隔符时返回错误。
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
