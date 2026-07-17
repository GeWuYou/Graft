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

// resolveManagedCreateSource 将可选模板版本转换为受管创建的来源溯源信息。
// 用户提交的工作区始终保持可编辑；版本仅证明其最初预填来源，不要求内容与快照持续一致。
func (s *Service) resolveManagedCreateSource(ctx context.Context, versionID, rootKey, applicationName, composeFileName string, envFileName *string) (string, map[string]string, error) {
	metadata := managedCreateSourceMetadata(rootKey, applicationName, composeFileName, envFileName)
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return projectcontract.SourceTypeManaged.String(), metadata, nil
	}
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return "", nil, err
	}
	item, err := repository.GetPublishedTemplateVersion(ctx, versionID)
	if err != nil {
		return "", nil, err
	}
	kind := projectcontract.DeploymentAdapterKind(item.Template.DeploymentAdapterKind)
	if kind != projectcontract.DeploymentAdapterKindCompose {
		return "", nil, errProjectInvalidArgument
	}
	if _, err = s.validateTemplateDefinition(kind, item.Version.DefinitionJSON); err != nil {
		return "", nil, err
	}
	metadata["template_id"] = item.Template.ID
	metadata["template_version_id"] = item.Version.ID
	return projectcontract.SourceTypeTemplate.String(), metadata, nil
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

// CloneApplicationTemplate 将当前定义复制为具有独立身份的新草稿模板。
func (s *Service) CloneApplicationTemplate(ctx context.Context, templateID, displayName string, actorID *uint64) (projectstore.ApplicationTemplateAggregate, error) {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	current, err := repository.GetTemplate(ctx, templateID)
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(displayName) > 128 {
		return projectstore.ApplicationTemplateAggregate{}, errProjectInvalidArgument
	}
	return repository.CloneTemplate(ctx, projectstore.CloneTemplateInput{SourceTemplateID: current.Template.ID, TemplateID: newTemplateID(), VersionID: newTemplateVersionID(), DisplayName: displayName, ActorID: actorID})
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

// WithdrawApplicationTemplate 撤回已发布快照，并原子创建相同定义的下一可编辑草稿。
func (s *Service) WithdrawApplicationTemplate(ctx context.Context, templateID string, actorID *uint64) (projectstore.ApplicationTemplateAggregate, error) {
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
	if current.Version.Status != projectcontract.ApplicationTemplateStatusPublished.String() {
		return projectstore.ApplicationTemplateAggregate{}, errProjectTemplateUnpublished
	}
	return repository.WithdrawTemplate(ctx, projectstore.WithdrawTemplateInput{TemplateID: current.Template.ID, VersionID: newTemplateVersionID(), ActorID: actorID})
}

// DeleteApplicationTemplate 软删除模板身份，使其不再出现在任何模板目录或创建来源中。
func (s *Service) DeleteApplicationTemplate(ctx context.Context, templateID string, actorID *uint64) error {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return err
	}
	return repository.DeleteTemplate(ctx, templateID, actorID)
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

func workspaceEntryContent(entries []ManagedWorkspaceEntry, path string) (string, error) {
	for _, entry := range entries {
		if entry.NodeType == "file" && entry.Path == path && entry.Content != nil {
			return *entry.Content, nil
		}
	}
	return "", fmt.Errorf("%w: template has no %s", errProjectInvalidArgument, path)
}

func newTemplateID() string        { return "tpl_" + strings.TrimPrefix(newApplicationID(), "app_") }
func newTemplateVersionID() string { return "tplv_" + strings.TrimPrefix(newApplicationID(), "app_") }
