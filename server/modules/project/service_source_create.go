package project

import (
	"context"
	"errors"
	"fmt"
	"strings"

	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
)

const (
	defaultTemplateKey     = "empty-compose"
	defaultTemplateVersion = "v1"
)

// TemplateProjectCreateRequest selects one bundled, module-owned template.
type TemplateProjectCreateRequest struct {
	DisplayName              string
	CanonicalProjectName     string
	RelativeProjectDirectory string
	TemplateKey              string
	TemplateVersion          string
	TemplateInstanceName     string
	LifecycleConfig          *LifecycleStandardConfig
}

// CreateTemplateProject materializes a bundled template and registers it through the shared pipeline.
func (s *Service) CreateTemplateProject(ctx context.Context, request TemplateProjectCreateRequest, actorID *uint64) (ManagedProjectCreateResult, error) {
	workspace, metadata, err := resolveTemplateWorkspace(request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	return s.createMaterializedSourceProject(ctx, workspace, projectcontract.SourceKindTemplate.String(), metadata, actorID)
}

// ValidateTemplateProject checks a bundled template and its eventual managed-root target without writing it.
func (s *Service) ValidateTemplateProject(ctx context.Context, request TemplateProjectCreateRequest) (ManagedProjectCreateValidationResult, error) {
	workspace, metadata, err := resolveTemplateWorkspace(request)
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
	aggregate, now, err := s.createProjectFromWorkspace(ctx, CreationCommand{DisplayName: normalized.DisplayName, CanonicalProjectName: normalized.CanonicalProjectName, CanonicalProjectNameSource: projectcontract.CanonicalProjectNameSourceOverride.String(), SourceKind: sourceType, HostScope: projectcontract.HostScopeLocal.String(), WorkingDirectory: validation.WorkingDirectory, OwnershipMode: projectcontract.OwnershipModeManagedRootDedicated.String(), SourceMetadata: metadata, LifecycleConfig: defaultManagedLifecycleConfig(normalized.LifecycleConfig), ParseResult: parseResult, ActorID: actorID})
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	cleanup = false
	return ManagedProjectCreateResult{Validation: validation, SourceType: sourceType, ProjectID: aggregate.Project.ID, ConfigHash: parseResult.ConfigHash, DeclaredServiceCount: len(parseResult.ServiceNames), RefreshedAt: now}, nil
}

func resolveTemplateWorkspace(request TemplateProjectCreateRequest) (ManagedProjectCreateRequest, map[string]string, error) {
	key := strings.TrimSpace(request.TemplateKey)
	if key == "" {
		key = defaultTemplateKey
	}
	version := strings.TrimSpace(request.TemplateVersion)
	if version == "" {
		version = defaultTemplateVersion
	}
	if key != defaultTemplateKey || version != defaultTemplateVersion {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("%w: unknown template key or version", errProjectInvalidArgument)
	}
	instance := strings.TrimSpace(request.TemplateInstanceName)
	if instance == "" {
		instance = request.CanonicalProjectName
	}
	return ManagedProjectCreateRequest{DisplayName: request.DisplayName, CanonicalProjectName: request.CanonicalProjectName, RelativeProjectDirectory: request.RelativeProjectDirectory, ComposeFileName: "compose.yaml", ComposeFileContent: "services: {}\n", EnvFileName: stringPointer(".env"), EnvFileContent: stringPointer("\n"), WorkspaceFiles: []ManagedWorkspaceFile{{Path: "compose.yaml", Content: "services: {}\n"}, {Path: ".env", Content: "\n"}}, LifecycleConfig: request.LifecycleConfig}, map[string]string{"template_key": key, "template_version": version, "template_instance_name": instance}, nil
}
