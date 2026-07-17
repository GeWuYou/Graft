package store

import (
	"context"
	"time"
)

// ApplicationTemplate 是 Application 创建蓝图的稳定身份记录。
// 它只保存通用元数据；定义格式由版本的 deployment_adapter_kind 决定。
type ApplicationTemplate struct {
	ID                    string
	DisplayName           string
	Description           string
	Category              string
	DeploymentAdapterKind string
	ArchivedAt            *time.Time
	CreatedBy             *uint64
	UpdatedBy             *uint64
	DeletedBy             *uint64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             int64
}

// ApplicationTemplateVersion 保存模板的草稿或不可变发布快照。
type ApplicationTemplateVersion struct {
	ID                      string
	TemplateID              string
	VersionNumber           int
	Status                  string
	DefinitionSchemaVersion int
	DefinitionJSON          []byte
	PublishedAt             *time.Time
	PublishedBy             *uint64
	WithdrawnAt             *time.Time
	WithdrawnBy             *uint64
	CreatedBy               *uint64
	UpdatedBy               *uint64
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               int64
}

// ApplicationTemplateAggregate 组合模板身份与当前选中的版本。
type ApplicationTemplateAggregate struct {
	Template ApplicationTemplate
	Version  ApplicationTemplateVersion
}

// TemplateListQuery 限定模板管理列表的类型和发布状态筛选。
type TemplateListQuery struct {
	DeploymentAdapterKind string
	PublishedOnly         bool
	IncludeArchived       bool
}

// TemplateCatalogQuery 限定创建者可发现的已发布模板目录。
type TemplateCatalogQuery struct {
	DeploymentAdapterKind string
	Search                string
	Category              string
	Sort                  string
	Page                  int
	PageSize              int
}

const (
	templateCatalogPageSizeDefault = 24
	templateCatalogPageSizeMax     = 100
)

// TemplateCatalogPage 是无总数目录查询的有界结果。
type TemplateCatalogPage struct {
	Items   []ApplicationTemplateCatalogItem
	HasMore bool
}

// ApplicationTemplateCatalogItem 是目录列表的轻量投影，不包含工作区定义快照。
type ApplicationTemplateCatalogItem struct {
	TemplateID            string
	DisplayName           string
	Description           string
	Category              string
	DeploymentAdapterKind string
	UpdatedAt             time.Time
	TemplateVersionID     string
	VersionNumber         int
	PublishedAt           time.Time
}

// CreateTemplateDraftInput 创建模板身份及其第一个草稿版本。
type CreateTemplateDraftInput struct {
	TemplateID              string
	VersionID               string
	DisplayName             string
	Description             string
	Category                string
	DeploymentAdapterKind   string
	DefinitionSchemaVersion int
	DefinitionJSON          []byte
	ActorID                 *uint64
}

// UpdateTemplateDraftInput 只允许修改当前模板的草稿版本。
type UpdateTemplateDraftInput struct {
	TemplateID              string
	DisplayName             string
	Description             string
	Category                string
	DefinitionSchemaVersion int
	DefinitionJSON          []byte
	ActorID                 *uint64
}

// CloneTemplateInput 从当前选中的定义创建独立的草稿模板。
type CloneTemplateInput struct {
	SourceTemplateID string
	TemplateID       string
	VersionID        string
	DisplayName      string
	ActorID          *uint64
}

// WithdrawTemplateInput 将当前已发布版本变为撤回历史，并创建后继草稿。
type WithdrawTemplateInput struct {
	TemplateID string
	VersionID  string
	ActorID    *uint64
}

// TemplateRepository 是模板管理的模块自有持久化边界。
// 它独立于 Application 注册表，避免给既有 Application repository mock 增加无关责任。
type TemplateRepository interface {
	ListTemplates(ctx context.Context, query TemplateListQuery) ([]ApplicationTemplateAggregate, error)
	ListTemplateCatalog(ctx context.Context, query TemplateCatalogQuery) (TemplateCatalogPage, error)
	GetTemplate(ctx context.Context, templateID string) (ApplicationTemplateAggregate, error)
	GetPublishedTemplate(ctx context.Context, templateID string) (ApplicationTemplateAggregate, error)
	GetPublishedTemplateVersion(ctx context.Context, versionID string) (ApplicationTemplateAggregate, error)
	CreateTemplateDraft(ctx context.Context, input CreateTemplateDraftInput) (ApplicationTemplateAggregate, error)
	UpdateTemplateDraft(ctx context.Context, input UpdateTemplateDraftInput) (ApplicationTemplateAggregate, error)
	CloneTemplate(ctx context.Context, input CloneTemplateInput) (ApplicationTemplateAggregate, error)
	WithdrawTemplate(ctx context.Context, input WithdrawTemplateInput) (ApplicationTemplateAggregate, error)
	PublishTemplateDraft(ctx context.Context, templateID string, actorID *uint64) (ApplicationTemplateAggregate, error)
	ArchiveTemplate(ctx context.Context, templateID string, actorID *uint64) error
	DeleteTemplate(ctx context.Context, templateID string, actorID *uint64) error
}
