package project

import (
	"context"
	"errors"
	"fmt"
	"strings"

	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
)

// TemplateProjectCreateRequest selects one operator-managed workspace template.
type TemplateProjectCreateRequest struct {
	DisplayName          string
	RuntimeTargetID      uint64
	ApplicationName      *string
	TemplateKey          string
	TemplateVersion      string
	TemplateInstanceName string
	LifecycleConfig      *LifecycleStandardConfig
}

// CreateTemplateProject materializes an operator-managed runtime template and registers it through the shared pipeline.
func (s *Service) CreateTemplateProject(ctx context.Context, request TemplateProjectCreateRequest, actorID *uint64) (ManagedProjectCreateResult, error) {
	workspace, metadata, err := s.resolveTemplateWorkspace(ctx, request, true)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	return s.createMaterializedSourceProject(ctx, workspace, projectcontract.SourceKindTemplate.String(), metadata, actorID)
}

// ValidateTemplateProject checks a runtime template and its eventual managed-root target without writing it.
func (s *Service) ValidateTemplateProject(ctx context.Context, request TemplateProjectCreateRequest) (ManagedProjectCreateValidationResult, error) {
	workspace, metadata, err := s.resolveTemplateWorkspace(ctx, request, false)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	return s.validateMaterializedSource(ctx, workspace, projectcontract.SourceKindTemplate.String(), metadata)
}

func (s *Service) validateMaterializedSource(ctx context.Context, request ManagedProjectCreateRequest, sourceType string, metadata map[string]string) (ManagedProjectCreateValidationResult, error) {
	result, err := s.ValidateManagedCreate(ctx, request)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	result.SourceType = sourceType
	result.SourceMetadata = metadata
	result.Warnings = append(result.Warnings, "Source resolution only prepares a workspace; creation does not deploy or start containers.")
	return result, nil
}

func (s *Service) createMaterializedSourceProject(ctx context.Context, request ManagedProjectCreateRequest, sourceType string, metadata map[string]string, actorID *uint64) (result ManagedProjectCreateResult, err error) {
	validation, err := s.validateMaterializedSource(ctx, request, sourceType, metadata)
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
	cleanup := true
	defer func() {
		if cleanup {
			err = errors.Join(err, cleanupManagedCreate(createdDir, createdFiles))
		}
	}()
	parseResult, err := projectcompose.Load(projectcompose.Input{WorkingDirectory: validation.WorkingDirectory, ComposeFiles: []string{validation.ComposeFileAbsolutePath}, EnvFiles: managedCreateEnvFileList(validation.EnvFileAbsolutePath)})
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	aggregate, now, err := s.createProjectFromWorkspace(ctx, CreationCommand{DisplayName: normalized.DisplayName, CanonicalProjectName: validation.ComposeProjectName, CanonicalProjectNameSource: "generated", SourceKind: sourceType, HostScope: projectcontract.HostScopeLocal.String(), WorkingDirectory: validation.WorkspacePath, OwnershipMode: projectcontract.OwnershipModeManagedRootDedicated.String(), SourceMetadata: metadata, LifecycleConfig: defaultManagedLifecycleConfig(normalized.LifecycleConfig), ParseResult: parseResult, ActorID: actorID, RuntimeTargetID: normalized.RuntimeTargetID, ApplicationName: validation.ApplicationName})
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	cleanup = false
	return ManagedProjectCreateResult{Validation: validation, SourceType: sourceType, ProjectID: aggregate.Project.ID, ApplicationID: aggregate.Project.ApplicationID, ConfigHash: parseResult.ConfigHash, DeclaredServiceCount: len(parseResult.ServiceNames), RefreshedAt: now}, nil
}

// resolveTemplateWorkspace loads a complete runtime template directory into one managed creation request.
func (s *Service) resolveTemplateWorkspace(ctx context.Context, request TemplateProjectCreateRequest, seedDefault bool) (ManagedProjectCreateRequest, map[string]string, error) {
	key := strings.TrimSpace(request.TemplateKey)
	if key == "" {
		key = defaultTemplateKey
	}
	version := strings.TrimSpace(request.TemplateVersion)
	if version == "" {
		version = defaultTemplateVersion
	}
	if version != defaultTemplateVersion {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("%w: unsupported template version", errProjectInvalidArgument)
	}
	root, err := s.applicationRootDirectory(ctx)
	if err != nil {
		return ManagedProjectCreateRequest{}, nil, err
	}
	if seedDefault {
		if err := seedDefaultWorkspaceTemplate(root); err != nil {
			return ManagedProjectCreateRequest{}, nil, err
		}
	}
	workspaceEntries, composeContent, err := loadTemplateCreateEntries(root, key)
	if err != nil {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("%w: template %s is unavailable", errProjectInvalidArgument, key)
	}
	composePath := "compose.yaml"
	instance := strings.TrimSpace(request.TemplateInstanceName)
	if instance == "" {
		instance = stringValue(request.ApplicationName)
	}
	return ManagedProjectCreateRequest{DisplayName: request.DisplayName, RuntimeTargetID: request.RuntimeTargetID, ApplicationName: request.ApplicationName, ComposeFileName: composePath, ComposeFileContent: composeContent, ComposeFilePath: composePath, WorkspaceEntries: workspaceEntries, LifecycleConfig: request.LifecycleConfig}, map[string]string{"template_key": key, "template_version": version, "template_instance_name": instance}, nil
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
