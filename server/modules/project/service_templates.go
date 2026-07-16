package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

var (
	errProjectTemplateUnavailable = errors.New("application template repository is unavailable")
	errProjectTemplateArchived    = errors.New("application template is archived")
	errProjectTemplateUnpublished = errors.New("application template version is not published")
)

// ComposeTemplateDefinition 是 Compose adapter 对通用模板 definition 的唯一解释。
// 它不把 Docker、Podman 或 Swarm 字段泄漏到通用模板表。
type ComposeTemplateDefinition struct {
	WorkspaceEntries []ManagedWorkspaceEntry  `json:"workspace_entries"`
	ComposeFilePath  string                   `json:"compose_file_path"`
	LifecycleConfig  *LifecycleStandardConfig `json:"lifecycle_configuration,omitempty"`
}

// ApplicationTemplateDraftRequest 是创建或更新通用模板草稿的服务输入。
type ApplicationTemplateDraftRequest struct {
	DisplayName             string
	Description             string
	DeploymentAdapterKind   projectcontract.DeploymentAdapterKind
	DefinitionSchemaVersion int
	DefinitionJSON          []byte
}

func (s *Service) templateRepositoryOrErr() (projectstore.TemplateRepository, error) {
	repository, ok := s.repository.(projectstore.TemplateRepository)
	if !ok || repository == nil {
		return nil, errProjectTemplateUnavailable
	}
	return repository, nil
}

// ListPublishedApplicationTemplates 返回创建者可使用的已发布、未归档模板。
func (s *Service) ListPublishedApplicationTemplates(ctx context.Context, kind projectcontract.DeploymentAdapterKind) ([]projectstore.ApplicationTemplateAggregate, error) {
	if _, err := s.deploymentAdapter(kind); err != nil {
		return nil, err
	}
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return nil, err
	}
	return repository.ListTemplates(ctx, projectstore.TemplateListQuery{DeploymentAdapterKind: kind.String(), PublishedOnly: true})
}

// ListApplicationTemplates 返回管理员模板目录；仅管理页面使用 IncludeArchived。
func (s *Service) ListApplicationTemplates(ctx context.Context, includeArchived bool) ([]projectstore.ApplicationTemplateAggregate, error) {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return nil, err
	}
	return repository.ListTemplates(ctx, projectstore.TemplateListQuery{IncludeArchived: includeArchived})
}

// GetApplicationTemplate 返回模板当前草稿；没有草稿时返回最新已发布版本。
func (s *Service) GetApplicationTemplate(ctx context.Context, templateID string) (projectstore.ApplicationTemplateAggregate, error) {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	return repository.GetTemplate(ctx, templateID)
}

// CreateApplicationTemplateDraft 仅从空白定义创建模板，明确不允许从现有 Application 复制。
func (s *Service) CreateApplicationTemplateDraft(ctx context.Context, request ApplicationTemplateDraftRequest, actorID *uint64) (projectstore.ApplicationTemplateAggregate, error) {
	normalized, err := s.normalizeTemplateDraft(request)
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	return repository.CreateTemplateDraft(ctx, projectstore.CreateTemplateDraftInput{TemplateID: newTemplateID(), VersionID: newTemplateVersionID(), DisplayName: normalized.DisplayName, Description: normalized.Description, DeploymentAdapterKind: normalized.DeploymentAdapterKind.String(), DefinitionSchemaVersion: normalized.DefinitionSchemaVersion, DefinitionJSON: normalized.DefinitionJSON, ActorID: actorID})
}

// UpdateApplicationTemplateDraft 校验并替换单个模板唯一可编辑草稿的定义快照。
func (s *Service) UpdateApplicationTemplateDraft(ctx context.Context, templateID string, request ApplicationTemplateDraftRequest, actorID *uint64) (projectstore.ApplicationTemplateAggregate, error) {
	normalized, err := s.normalizeTemplateDraft(request)
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	current, err := repository.GetTemplate(ctx, templateID)
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	if current.Template.ArchivedAt != nil {
		return projectstore.ApplicationTemplateAggregate{}, errProjectTemplateArchived
	}
	if current.Template.DeploymentAdapterKind != normalized.DeploymentAdapterKind.String() {
		return projectstore.ApplicationTemplateAggregate{}, errProjectInvalidArgument
	}
	return repository.UpdateTemplateDraft(ctx, projectstore.UpdateTemplateDraftInput{TemplateID: templateID, DisplayName: normalized.DisplayName, Description: normalized.Description, DefinitionSchemaVersion: normalized.DefinitionSchemaVersion, DefinitionJSON: normalized.DefinitionJSON, ActorID: actorID})
}

// DeriveApplicationTemplateDraft 只接受已发布版本并建立同一模板的下一个可编辑草稿。
func (s *Service) DeriveApplicationTemplateDraft(ctx context.Context, templateID, versionID string, actorID *uint64) (projectstore.ApplicationTemplateAggregate, error) {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	current, err := repository.GetTemplate(ctx, templateID)
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	if current.Template.ArchivedAt != nil {
		return projectstore.ApplicationTemplateAggregate{}, errProjectTemplateArchived
	}
	return repository.DeriveTemplateDraft(ctx, projectstore.DeriveTemplateDraftInput{TemplateID: templateID, SourceVersionID: versionID, VersionID: newTemplateVersionID(), ActorID: actorID})
}

// PublishApplicationTemplateDraft 校验草稿后将其发布为不可变且创建者可见的版本。
func (s *Service) PublishApplicationTemplateDraft(ctx context.Context, templateID string, actorID *uint64) (projectstore.ApplicationTemplateAggregate, error) {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	current, err := repository.GetTemplate(ctx, templateID)
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	if current.Template.ArchivedAt != nil {
		return projectstore.ApplicationTemplateAggregate{}, errProjectTemplateArchived
	}
	if _, err = s.validateTemplateDefinition(projectcontract.DeploymentAdapterKind(current.Template.DeploymentAdapterKind), current.Version.DefinitionJSON); err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	return repository.PublishTemplateDraft(ctx, templateID, actorID)
}

// ArchiveApplicationTemplate 阻止模板继续用于创建，同时保留既有版本溯源。
func (s *Service) ArchiveApplicationTemplate(ctx context.Context, templateID string, actorID *uint64) error {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return err
	}
	return repository.ArchiveTemplate(ctx, templateID, actorID)
}

// EnsureBundledApplicationTemplate 保存内嵌基线为持久化平台模板。它必须由显式 Boot/管理路径调用，不能回退到目录读取。
func (s *Service) EnsureBundledApplicationTemplate(ctx context.Context) error {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return err
	}
	if _, err = repository.FindTemplateByDisplayName(ctx, "Compose Baseline"); err == nil {
		return nil
	}
	if !errors.Is(err, projectstore.ErrTemplateNotFound) {
		return err
	}
	composeContent, err := bundledWorkspaceTemplates.ReadFile("templates/default/compose.yaml")
	if err != nil {
		return err
	}
	envContent, err := bundledWorkspaceTemplates.ReadFile("templates/default/.env")
	if err != nil {
		return err
	}
	definition, err := encodeComposeTemplateDefinition(ComposeTemplateDefinition{ComposeFilePath: "compose.yaml", WorkspaceEntries: []ManagedWorkspaceEntry{{Path: ".env", NodeType: "file", Content: stringPointer(string(envContent))}, {Path: "compose.yaml", NodeType: "file", Content: stringPointer(string(composeContent))}}, LifecycleConfig: templateLifecycleConfigPointer(defaultLifecycleStandardConfig())})
	if err != nil {
		return err
	}
	created, err := s.CreateApplicationTemplateDraft(ctx, ApplicationTemplateDraftRequest{DisplayName: "Compose Baseline", DeploymentAdapterKind: projectcontract.DeploymentAdapterKindCompose, DefinitionSchemaVersion: projectcontract.ApplicationTemplateDefinitionSchemaVersionCurrent, DefinitionJSON: definition}, nil)
	if err != nil && errors.Is(err, projectstore.ErrTemplateConflict) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.PublishApplicationTemplateDraft(ctx, created.Template.ID, nil)
	return err
}

// ImportLegacyApplicationTemplate 显式将 Application Root templates/<key> 转换为 Compose 草稿。
// 它只在管理员动作中读取旧目录，不产生启动时种子或运行时回退。
func (s *Service) ImportLegacyApplicationTemplate(ctx context.Context, key, displayName string, actorID *uint64) (projectstore.ApplicationTemplateAggregate, error) {
	root, err := s.applicationRootDirectory(ctx)
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	entries, err := loadWorkspaceTemplate(root, strings.TrimSpace(key))
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, fmt.Errorf("%w: legacy template is unavailable", errProjectInvalidArgument)
	}
	workspace := make([]ManagedWorkspaceEntry, 0, len(entries))
	for _, entry := range entries {
		workspace = append(workspace, ManagedWorkspaceEntry(entry))
	}
	definition, err := encodeComposeTemplateDefinition(ComposeTemplateDefinition{WorkspaceEntries: workspace, ComposeFilePath: "compose.yaml", LifecycleConfig: templateLifecycleConfigPointer(defaultLifecycleStandardConfig())})
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = strings.TrimSpace(key)
	}
	return s.CreateApplicationTemplateDraft(ctx, ApplicationTemplateDraftRequest{DisplayName: displayName, DeploymentAdapterKind: projectcontract.DeploymentAdapterKindCompose, DefinitionSchemaVersion: projectcontract.ApplicationTemplateDefinitionSchemaVersionCurrent, DefinitionJSON: definition}, actorID)
}

func (s *Service) normalizeTemplateDraft(request ApplicationTemplateDraftRequest) (ApplicationTemplateDraftRequest, error) {
	request.DisplayName, request.Description = strings.TrimSpace(request.DisplayName), strings.TrimSpace(request.Description)
	if request.DisplayName == "" || len(request.DisplayName) > 128 || request.DefinitionSchemaVersion < 1 {
		return ApplicationTemplateDraftRequest{}, errProjectInvalidArgument
	}
	if _, err := s.validateTemplateDefinition(request.DeploymentAdapterKind, request.DefinitionJSON); err != nil {
		return ApplicationTemplateDraftRequest{}, err
	}
	return request, nil
}

func (s *Service) deploymentAdapter(kind projectcontract.DeploymentAdapterKind) (projectcontract.DeploymentAdapter, error) {
	registry, err := projectcontract.NewDeploymentAdapterRegistry(projectcontract.ComposeDeploymentAdapter{})
	if err != nil {
		return nil, err
	}
	adapter, ok := registry.Get(kind)
	if !ok {
		return nil, errProjectInvalidArgument
	}
	return adapter, nil
}

func (s *Service) validateTemplateDefinition(kind projectcontract.DeploymentAdapterKind, definition []byte) (ComposeTemplateDefinition, error) {
	adapter, err := s.deploymentAdapter(kind)
	if err != nil {
		return ComposeTemplateDefinition{}, err
	}
	if err = adapter.ValidateDefinition(definition); err != nil {
		return ComposeTemplateDefinition{}, fmt.Errorf("%w: %v", errProjectInvalidArgument, err)
	}
	if kind != projectcontract.DeploymentAdapterKindCompose {
		return ComposeTemplateDefinition{}, errProjectInvalidArgument
	}
	var parsed ComposeTemplateDefinition
	if err = json.Unmarshal(definition, &parsed); err != nil {
		return ComposeTemplateDefinition{}, errProjectInvalidArgument
	}
	if strings.TrimSpace(parsed.ComposeFilePath) == "" {
		return ComposeTemplateDefinition{}, errProjectInvalidArgument
	}
	if _, err = normalizeManagedWorkspaceEntries(parsed.WorkspaceEntries, parsed.ComposeFilePath); err != nil {
		return ComposeTemplateDefinition{}, err
	}
	if _, err = workspaceEntryContent(parsed.WorkspaceEntries, parsed.ComposeFilePath); err != nil {
		return ComposeTemplateDefinition{}, err
	}
	if parsed.LifecycleConfig != nil {
		if _, err = normalizeLifecycleStandardConfig(*parsed.LifecycleConfig); err != nil {
			return ComposeTemplateDefinition{}, err
		}
	}
	return parsed, nil
}

func encodeComposeTemplateDefinition(definition ComposeTemplateDefinition) ([]byte, error) {
	return json.Marshal(definition)
}
func templateLifecycleConfigPointer(value LifecycleStandardConfig) *LifecycleStandardConfig {
	return &value
}
func newTemplateID() string        { return "tpl_" + strings.TrimPrefix(newApplicationID(), "app_") }
func newTemplateVersionID() string { return "tplv_" + strings.TrimPrefix(newApplicationID(), "app_") }
