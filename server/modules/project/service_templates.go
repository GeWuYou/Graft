package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

var (
	errProjectTemplateUnavailable = errors.New("application template repository is unavailable")
	errProjectTemplateArchived    = errors.New("application template is archived")
	errProjectTemplateUnpublished = errors.New("application template version is not published")
	templateVariableNamePattern   = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

const templateCatalogSearchMaxLength = 128
const templateManagementSearchMaxLength = 128

// ComposeTemplateDefinition 是 Compose adapter 对通用模板 definition 的唯一解释。
// 它不把 Docker、Podman 或 Swarm 字段泄漏到通用模板表。
type ComposeTemplateDefinition struct {
	WorkspaceEntries     []ManagedWorkspaceEntry                  `json:"workspace_entries"`
	ComposeFilePath      string                                   `json:"compose_file_path"`
	LifecycleConfig      *LifecycleStandardConfig                 `json:"lifecycle_configuration,omitempty"`
	CatalogDocumentation *ApplicationTemplateCatalogDocumentation `json:"catalog_documentation,omitempty"`
}

// ApplicationTemplateCatalogDocumentation 是随发布定义冻结的目录详情内容。
type ApplicationTemplateCatalogDocumentation struct {
	ReadmeMarkdown string                               `json:"readme_markdown,omitempty"`
	Variables      []ApplicationTemplateCatalogVariable `json:"variables,omitempty"`
}

// ApplicationTemplateCatalogVariable 描述模板使用者在创建时需要理解的变量。
type ApplicationTemplateCatalogVariable struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// ApplicationTemplateDraftRequest 是创建或更新通用模板草稿的服务输入。
type ApplicationTemplateDraftRequest struct {
	DisplayName             string
	Description             string
	Category                projectcontract.ApplicationTemplateCategory
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

// ListPublishedApplicationTemplates 返回创建者可发现的已发布、未归档模板目录页。
func (s *Service) ListPublishedApplicationTemplates(ctx context.Context, kind projectcontract.DeploymentAdapterKind, query projectstore.TemplateCatalogQuery) (projectstore.TemplateCatalogPage, error) {
	if _, err := s.deploymentAdapter(kind); err != nil {
		return projectstore.TemplateCatalogPage{}, err
	}
	query.Search, query.Category, query.Sort = strings.TrimSpace(query.Search), strings.TrimSpace(query.Category), strings.TrimSpace(query.Sort)
	if err := validateTemplateCatalogPagination(query); err != nil {
		return projectstore.TemplateCatalogPage{}, err
	}
	if err := validateTemplateCatalogFilters(query); err != nil {
		return projectstore.TemplateCatalogPage{}, errProjectInvalidArgument
	}
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.TemplateCatalogPage{}, err
	}
	query.DeploymentAdapterKind = kind.String()
	return repository.ListTemplateCatalog(ctx, query)
}

func validateTemplateCatalogPagination(query projectstore.TemplateCatalogQuery) error {
	if query.Page < 1 || query.PageSize < 1 || query.PageSize > 100 {
		return errProjectInvalidArgument
	}
	if query.Page > int(^uint(0)>>1)/query.PageSize+1 {
		return errProjectInvalidArgument
	}
	return nil
}

func validateTemplateCatalogFilters(query projectstore.TemplateCatalogQuery) error {
	if len(query.Search) > templateCatalogSearchMaxLength {
		return errProjectInvalidArgument
	}
	if query.Category != "" && !projectcontract.ApplicationTemplateCategory(query.Category).Valid() {
		return errProjectInvalidArgument
	}
	if query.Sort != "" && query.Sort != "updated_desc" && query.Sort != "name_asc" {
		return errProjectInvalidArgument
	}
	return nil
}

// GetPublishedApplicationTemplate 返回创建者可浏览的当前已发布模板详情。
func (s *Service) GetPublishedApplicationTemplate(ctx context.Context, templateID string) (projectstore.ApplicationTemplateAggregate, error) {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	return repository.GetPublishedTemplate(ctx, templateID)
}

// GetPublishedApplicationTemplateVersion 返回创建时可安全预填的不可变发布版本。
func (s *Service) GetPublishedApplicationTemplateVersion(ctx context.Context, versionID string) (projectstore.ApplicationTemplateAggregate, error) {
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.ApplicationTemplateAggregate{}, err
	}
	return repository.GetPublishedTemplateVersion(ctx, versionID)
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

// ListApplicationTemplateManagementPage 返回管理员模板目录的服务端分页结果。
//
//nolint:gocyclo,cyclop // 查询状态白名单与边界值必须显式校验，避免将排序传入仓储 SQL。
func (s *Service) ListApplicationTemplateManagementPage(ctx context.Context, query projectstore.TemplateManagementQuery) (projectstore.TemplateManagementPage, error) {
	query.Keyword, query.Status, query.Sort = strings.TrimSpace(query.Keyword), strings.TrimSpace(query.Status), strings.TrimSpace(query.Sort)
	if !isTemplateManagementPageSize(query.Limit) || query.Offset < 0 || len(query.Keyword) > templateManagementSearchMaxLength {
		return projectstore.TemplateManagementPage{}, errProjectInvalidArgument
	}
	if query.Status != "" && query.Status != "draft" && query.Status != "published" && query.Status != "archived" {
		return projectstore.TemplateManagementPage{}, errProjectInvalidArgument
	}
	if !isTemplateManagementSort(query.Sort) {
		return projectstore.TemplateManagementPage{}, errProjectInvalidArgument
	}
	if query.UpdatedAfter != nil && query.UpdatedBefore != nil && query.UpdatedAfter.After(*query.UpdatedBefore) {
		return projectstore.TemplateManagementPage{}, errProjectInvalidArgument
	}
	repository, err := s.templateRepositoryOrErr()
	if err != nil {
		return projectstore.TemplateManagementPage{}, err
	}
	return repository.ListTemplateManagementPage(ctx, query)
}

func isTemplateManagementPageSize(size int) bool {
	return size == 10 || size == 20 || size == 50 || size == 100
}

func isTemplateManagementSort(value string) bool {
	switch value {
	case "", "updated_at:asc", "updated_at:desc", "display_name:asc", "display_name:desc", "status:asc", "status:desc", "version_number:asc", "version_number:desc":
		return true
	default:
		return false
	}
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
	return repository.CreateTemplateDraft(ctx, projectstore.CreateTemplateDraftInput{TemplateID: newTemplateID(), VersionID: newTemplateVersionID(), DisplayName: normalized.DisplayName, Description: normalized.Description, Category: normalized.Category.String(), DeploymentAdapterKind: normalized.DeploymentAdapterKind.String(), DefinitionSchemaVersion: normalized.DefinitionSchemaVersion, DefinitionJSON: normalized.DefinitionJSON, ActorID: actorID})
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
	return repository.UpdateTemplateDraft(ctx, projectstore.UpdateTemplateDraftInput{TemplateID: templateID, DisplayName: normalized.DisplayName, Description: normalized.Description, Category: normalized.Category.String(), DefinitionSchemaVersion: normalized.DefinitionSchemaVersion, DefinitionJSON: normalized.DefinitionJSON, ActorID: actorID})
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
	if request.DisplayName == "" || len(request.DisplayName) > 128 || !request.Category.Valid() || request.DefinitionSchemaVersion < 1 {
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
		return ComposeTemplateDefinition{}, fmt.Errorf("%w: %w", errProjectInvalidArgument, err)
	}
	return s.validateComposeTemplateDefinition(kind, definition)
}

func (s *Service) validateComposeTemplateDefinition(kind projectcontract.DeploymentAdapterKind, definition []byte) (ComposeTemplateDefinition, error) {
	if kind != projectcontract.DeploymentAdapterKindCompose {
		return ComposeTemplateDefinition{}, errProjectInvalidArgument
	}
	var parsed ComposeTemplateDefinition
	if err := json.Unmarshal(definition, &parsed); err != nil {
		return ComposeTemplateDefinition{}, errProjectInvalidArgument
	}
	if strings.TrimSpace(parsed.ComposeFilePath) == "" {
		return ComposeTemplateDefinition{}, errProjectInvalidArgument
	}
	if _, err := normalizeManagedWorkspaceEntries(parsed.WorkspaceEntries, parsed.ComposeFilePath); err != nil {
		return ComposeTemplateDefinition{}, err
	}
	if _, err := workspaceEntryContent(parsed.WorkspaceEntries, parsed.ComposeFilePath); err != nil {
		return ComposeTemplateDefinition{}, err
	}
	if err := validateTemplateLifecycleConfig(parsed.LifecycleConfig); err != nil {
		return ComposeTemplateDefinition{}, err
	}
	if err := validateTemplateCatalogDocumentation(parsed.CatalogDocumentation); err != nil {
		return ComposeTemplateDefinition{}, err
	}
	return parsed, nil
}

func validateTemplateLifecycleConfig(config *LifecycleStandardConfig) error {
	if config == nil {
		return nil
	}
	_, err := normalizeLifecycleStandardConfig(*config)
	return err
}

func validateTemplateCatalogDocumentation(documentation *ApplicationTemplateCatalogDocumentation) error {
	if documentation == nil {
		return nil
	}
	if len(documentation.ReadmeMarkdown) > 65535 || len(documentation.Variables) > 100 {
		return errProjectInvalidArgument
	}
	seen := make(map[string]struct{}, len(documentation.Variables))
	for _, variable := range documentation.Variables {
		name, description := strings.TrimSpace(variable.Name), strings.TrimSpace(variable.Description)
		if name != variable.Name || !validTemplateVariableName(name) || description == "" || len(description) > 512 {
			return errProjectInvalidArgument
		}
		if _, exists := seen[name]; exists {
			return errProjectInvalidArgument
		}
		seen[name] = struct{}{}
	}
	return nil
}

func validTemplateVariableName(value string) bool {
	return templateVariableNamePattern.MatchString(value)
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
