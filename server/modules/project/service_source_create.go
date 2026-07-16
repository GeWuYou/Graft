package project

import (
	"context"
	"errors"
	"fmt"
	"strings"

	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
)

// TemplateApplicationCreateRequest 选择一个由操作员管理的工作区模板。
type TemplateApplicationCreateRequest struct {
	DisplayName          string
	RuntimeTargetID      uint64
	ApplicationName      *string
	TemplateKey          string
	TemplateVersion      string
	TemplateInstanceName string
	LifecycleConfig      *LifecycleStandardConfig
}

// CreateTemplateApplication 将操作员管理的运行时模板物化为工作区，并通过共享流水线登记项目。
func (s *Service) CreateTemplateApplication(ctx context.Context, request TemplateApplicationCreateRequest, actorID *uint64) (ManagedApplicationCreateResult, error) {
	workspace, metadata, err := s.resolveTemplateWorkspace(ctx, request, true)
	if err != nil {
		return ManagedApplicationCreateResult{}, err
	}
	return s.createMaterializedSourceApplication(ctx, workspace, projectcontract.SourceTypeTemplate.String(), metadata, actorID)
}

// ValidateTemplateApplication 校验运行时模板及其最终受管根目录目标，不写入工作区。
func (s *Service) ValidateTemplateApplication(ctx context.Context, request TemplateApplicationCreateRequest) (ManagedApplicationCreateValidationResult, error) {
	workspace, metadata, err := s.resolveTemplateWorkspace(ctx, request, false)
	if err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}
	return s.validateMaterializedSource(ctx, workspace, projectcontract.SourceTypeTemplate.String(), metadata)
}

func (s *Service) validateMaterializedSource(ctx context.Context, request ManagedApplicationCreateRequest, sourceType string, metadata map[string]string) (ManagedApplicationCreateValidationResult, error) {
	result, err := s.ValidateManagedCreate(ctx, request)
	if err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}
	result.SourceType = sourceType
	result.SourceMetadata = metadata
	result.Warnings = append(result.Warnings, "Source resolution only prepares a workspace; creation does not deploy or start containers.")
	return result, nil
}

func (s *Service) createMaterializedSourceApplication(ctx context.Context, request ManagedApplicationCreateRequest, sourceType string, metadata map[string]string, actorID *uint64) (result ManagedApplicationCreateResult, err error) {
	validation, err := s.validateMaterializedSource(ctx, request, sourceType, metadata)
	if err != nil {
		return ManagedApplicationCreateResult{}, err
	}
	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedApplicationCreateResult{}, err
	}
	if err := ensureManagedCreatePathsUnderRoot(validation); err != nil {
		return ManagedApplicationCreateResult{}, err
	}
	createdDir, createdFiles, err := writeManagedApplicationFiles(validation, normalized)
	if err != nil {
		return ManagedApplicationCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			err = errors.Join(err, cleanupManagedCreate(createdDir, createdFiles))
		}
	}()
	parseResult, err := projectcompose.Load(projectcompose.Input{WorkspacePath: validation.WorkspacePath, ComposeFiles: []string{validation.ComposeFileAbsolutePath}, EnvFiles: managedCreateEnvFileList(validation.EnvFileAbsolutePath)})
	if err != nil {
		return ManagedApplicationCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	aggregate, now, err := s.createProjectFromWorkspace(ctx, CreationCommand{DisplayName: normalized.DisplayName, ComposeProjectName: validation.ComposeProjectName, ComposeProjectNameSource: projectcontract.ComposeProjectNameSourceComputed.String(), SourceType: sourceType, WorkspacePath: validation.WorkspacePath, OwnershipMode: projectcontract.OwnershipModeManagedRootDedicated.String(), SourceMetadata: metadata, LifecycleConfig: defaultManagedLifecycleConfig(normalized.LifecycleConfig), ParseResult: parseResult, ActorID: actorID, RuntimeTargetID: normalized.RuntimeTargetID, ApplicationName: validation.ApplicationName})
	if err != nil {
		return ManagedApplicationCreateResult{}, err
	}
	cleanup = false
	return ManagedApplicationCreateResult{Validation: validation, SourceType: sourceType, ApplicationRecordID: aggregate.Application.ApplicationRecordID, ApplicationID: aggregate.Application.ApplicationID, ConfigHash: parseResult.ConfigHash, DeclaredServiceCount: len(parseResult.ServiceNames), RefreshedAt: now}, nil
}

// resolveTemplateWorkspace 将完整运行时模板目录加载为一个受管创建请求。
func (s *Service) resolveTemplateWorkspace(ctx context.Context, request TemplateApplicationCreateRequest, seedDefault bool) (ManagedApplicationCreateRequest, map[string]string, error) {
	key := strings.TrimSpace(request.TemplateKey)
	if key == "" {
		key = defaultTemplateKey
	}
	version := strings.TrimSpace(request.TemplateVersion)
	if version == "" {
		version = defaultTemplateVersion
	}
	if version != defaultTemplateVersion {
		return ManagedApplicationCreateRequest{}, nil, fmt.Errorf("%w: unsupported template version", errProjectInvalidArgument)
	}
	root, err := s.applicationRootDirectory(ctx)
	if err != nil {
		return ManagedApplicationCreateRequest{}, nil, err
	}
	if seedDefault {
		if err := seedDefaultWorkspaceTemplate(root); err != nil {
			return ManagedApplicationCreateRequest{}, nil, err
		}
	}
	workspaceEntries, composeContent, err := loadTemplateCreateEntries(root, key)
	if err != nil {
		return ManagedApplicationCreateRequest{}, nil, fmt.Errorf("%w: template %s is unavailable", errProjectInvalidArgument, key)
	}
	composePath := "compose.yaml"
	instance := strings.TrimSpace(request.TemplateInstanceName)
	if instance == "" {
		instance = stringValue(request.ApplicationName)
	}
	return ManagedApplicationCreateRequest{DisplayName: request.DisplayName, RuntimeTargetID: request.RuntimeTargetID, ApplicationName: request.ApplicationName, ComposeFileName: composePath, ComposeFileContent: composeContent, ComposeFilePath: composePath, WorkspaceEntries: workspaceEntries, LifecycleConfig: request.LifecycleConfig}, map[string]string{"template_key": key, "template_version": version, "template_instance_name": instance}, nil
}

// loadTemplateCreateEntries 加载指定模板的工作区条目，并返回其 compose.yaml 内容。
func loadTemplateCreateEntries(root, key string) ([]ManagedWorkspaceEntry, string, error) {
	entries, err := loadWorkspaceTemplate(root, key)
	if err != nil {
		return nil, "", err
	}
	workspaceEntries := make([]ManagedWorkspaceEntry, 0, len(entries))
	for _, entry := range entries {
		workspaceEntries = append(workspaceEntries, ManagedWorkspaceEntry(entry))
	}
	composeContent, err := workspaceEntryContent(workspaceEntries, "compose.yaml")
	if err != nil {
		return nil, "", err
	}
	return workspaceEntries, composeContent, nil
}

// workspaceEntryContent 返回模板条目中指定文件路径的内容；如果未找到对应文件，则返回错误。
func workspaceEntryContent(entries []ManagedWorkspaceEntry, path string) (string, error) {
	for _, entry := range entries {
		if entry.NodeType == "file" && entry.Path == path && entry.Content != nil {
			return *entry.Content, nil
		}
	}
	return "", fmt.Errorf("%w: template has no %s", errProjectInvalidArgument, path)
}
