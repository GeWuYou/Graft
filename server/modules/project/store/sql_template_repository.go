package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	projectcontract "graft/server/modules/project/contract"
)

// 模板仓储错误保持独立，避免调用方将模板状态与 Application 注册状态混淆。
var (
	ErrTemplateNotFound = errors.New("application template not found")
	// ErrTemplateConflict 表示模板写入违反模板身份或其它唯一性约束。
	ErrTemplateConflict = errors.New("application template conflict")
	// ErrTemplateNameOccupied 表示存活模板已占用当前展示名称。
	ErrTemplateNameOccupied   = errors.New("application template display name is already in use")
	ErrTemplateDraftNotFound  = errors.New("application template draft not found")
	ErrTemplatePublishedState = errors.New("application template published version is immutable")
)

// ListTemplates 返回管理目录的模板聚合；每个模板优先返回草稿，其次返回已发布版本。
func (r *SQLRepository) ListTemplates(ctx context.Context, query TemplateListQuery) ([]ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return nil, err
	}
	where := []string{"t.deleted_at = 0"}
	args := []any{}
	if !query.IncludeArchived {
		where = append(where, "t.archived_at IS NULL")
	}
	if kind := strings.TrimSpace(query.DeploymentAdapterKind); kind != "" {
		where = append(where, "t.deployment_adapter_kind = ?")
		args = append(args, kind)
	}
	versionJoin := "LEFT JOIN application_template_versions v ON v.template_id = t.template_id AND v.deleted_at = 0 AND (v.status = 'draft' OR (v.status = 'published' AND NOT EXISTS (SELECT 1 FROM application_template_versions draft WHERE draft.template_id = t.template_id AND draft.deleted_at = 0 AND draft.status = 'draft')))"
	if query.PublishedOnly {
		versionJoin = "JOIN application_template_versions v ON v.template_id = t.template_id AND v.deleted_at = 0 AND v.status = 'published'"
	}
	querySQL := `SELECT t.template_id, t.display_name, t.description, t.category, t.deployment_adapter_kind, t.archived_at, t.created_by, t.updated_by, t.deleted_by, t.created_at, t.updated_at, t.deleted_at,
		v.template_version_id, v.version_number, v.status, v.definition_schema_version, v.definition_json, v.published_at, v.published_by, v.withdrawn_at, v.withdrawn_by, v.created_by, v.updated_by, v.created_at, v.updated_at, v.deleted_at
		FROM application_templates t ` + versionJoin + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY t.updated_at DESC, t.template_id DESC`
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(querySQL), args...)
	if err != nil {
		return nil, fmt.Errorf("list application templates: %w", err)
	}
	defer closeRows(rows)
	items := []ApplicationTemplateAggregate{}
	for rows.Next() {
		item, err := scanTemplateAggregate(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListTemplateManagementPage 返回管理列表的过滤分页结果；归档状态始终优先于当前版本状态。
func (r *SQLRepository) ListTemplateManagementPage(ctx context.Context, query TemplateManagementQuery) (TemplateManagementPage, error) {
	if err := r.ensureReady(); err != nil {
		return TemplateManagementPage{}, err
	}
	where, args := templateManagementFilters(query)
	join := "JOIN application_template_versions v ON v.template_id = t.template_id AND v.deleted_at = 0 AND (v.status = 'draft' OR (v.status = 'published' AND NOT EXISTS (SELECT 1 FROM application_template_versions draft WHERE draft.template_id = t.template_id AND draft.deleted_at = 0 AND draft.status = 'draft')))"
	countSQL := `SELECT COUNT(*) FROM application_templates t ` + join + ` WHERE ` + strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(countSQL), args...).Scan(&total); err != nil {
		return TemplateManagementPage{}, fmt.Errorf("count application templates: %w", err)
	}
	querySQL := `SELECT t.template_id, t.display_name, t.description, t.category, t.deployment_adapter_kind, t.archived_at, t.created_by, t.updated_by, t.deleted_by, t.created_at, t.updated_at, t.deleted_at,
		v.template_version_id, v.version_number, v.status, v.definition_schema_version, v.definition_json, v.published_at, v.published_by, v.withdrawn_at, v.withdrawn_by, v.created_by, v.updated_by, v.created_at, v.updated_at, v.deleted_at
		FROM application_templates t ` + join + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + templateManagementOrder(query.Sort) + ` LIMIT ? OFFSET ?`
	args = append(args, query.Limit, query.Offset)
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(querySQL), args...)
	if err != nil {
		return TemplateManagementPage{}, fmt.Errorf("list managed application templates: %w", err)
	}
	defer closeRows(rows)
	items := make([]ApplicationTemplateAggregate, 0, query.Limit)
	for rows.Next() {
		item, scanErr := scanTemplateAggregate(rows)
		if scanErr != nil {
			return TemplateManagementPage{}, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return TemplateManagementPage{}, err
	}
	return TemplateManagementPage{Items: items, Total: total}, nil
}

func templateManagementFilters(query TemplateManagementQuery) ([]string, []any) {
	where, args := []string{"t.deleted_at = 0"}, []any{}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		where = append(where, "(LOWER(t.display_name) LIKE LOWER(?) OR LOWER(t.description) LIKE LOWER(?))")
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	if query.UpdatedAfter != nil {
		where = append(where, "t.updated_at >= ?")
		args = append(args, *query.UpdatedAfter)
	}
	if query.UpdatedBefore != nil {
		where = append(where, "t.updated_at <= ?")
		args = append(args, *query.UpdatedBefore)
	}
	switch query.Status {
	case "draft":
		where = append(where, "t.archived_at IS NULL AND v.status = 'draft'")
	case "published":
		where = append(where, "t.archived_at IS NULL AND v.status = 'published'")
	case "archived":
		where = append(where, "t.archived_at IS NOT NULL")
	}
	return where, args
}

func templateManagementOrder(sort string) string {
	switch sort {
	case "updated_at:asc":
		return "t.updated_at ASC, t.template_id ASC"
	case "display_name:asc":
		return "t.display_name ASC, t.template_id ASC"
	case "display_name:desc":
		return "t.display_name DESC, t.template_id DESC"
	case "status:asc":
		return "CASE WHEN t.archived_at IS NOT NULL THEN 'archived' ELSE v.status END ASC, t.template_id ASC"
	case "status:desc":
		return "CASE WHEN t.archived_at IS NOT NULL THEN 'archived' ELSE v.status END DESC, t.template_id DESC"
	case "version_number:asc":
		return "v.version_number ASC, t.template_id ASC"
	case "version_number:desc":
		return "v.version_number DESC, t.template_id DESC"
	default:
		return "t.updated_at DESC, t.template_id DESC"
	}
}

// ListTemplateCatalog 返回面向创建者的已发布模板摘要，并以多取一条记录判断后续页。
func (r *SQLRepository) ListTemplateCatalog(ctx context.Context, query TemplateCatalogQuery) (TemplateCatalogPage, error) {
	if err := r.ensureReady(); err != nil {
		return TemplateCatalogPage{}, err
	}
	page, pageSize, sort := normalizeTemplateCatalogQuery(query)
	where, args := templateCatalogFilters(query)
	offset := (page - 1) * pageSize
	querySQL := `SELECT t.template_id, t.display_name, t.description, t.category, t.deployment_adapter_kind, t.updated_at,
		v.template_version_id, v.version_number, v.published_at
		FROM application_templates t JOIN application_template_versions v ON v.template_id = t.template_id
		WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + templateCatalogOrder(sort) + ` LIMIT ? OFFSET ?`
	args = append(args, pageSize+1, offset)
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(querySQL), args...)
	if err != nil {
		return TemplateCatalogPage{}, fmt.Errorf("list application template catalog: %w", err)
	}
	defer closeRows(rows)
	items := make([]ApplicationTemplateCatalogItem, 0, templateCatalogPageSizeMax)
	for rows.Next() {
		var item ApplicationTemplateCatalogItem
		if scanErr := rows.Scan(&item.TemplateID, &item.DisplayName, &item.Description, &item.Category, &item.DeploymentAdapterKind, &item.UpdatedAt, &item.TemplateVersionID, &item.VersionNumber, &item.PublishedAt); scanErr != nil {
			return TemplateCatalogPage{}, scanErr
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return TemplateCatalogPage{}, err
	}
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}
	return TemplateCatalogPage{Items: items, HasMore: hasMore}, nil
}

// GetTemplate 返回存活模板身份的当前草稿；没有草稿时返回最新已发布或撤回版本。
func (r *SQLRepository) GetTemplate(ctx context.Context, templateID string) (ApplicationTemplateAggregate, error) {
	return r.getTemplate(ctx, templateID, "")
}

// GetPublishedTemplateVersion 返回可用于创建的已发布、未归档模板版本。
func (r *SQLRepository) GetPublishedTemplateVersion(ctx context.Context, versionID string) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	query := `SELECT t.template_id, t.display_name, t.description, t.category, t.deployment_adapter_kind, t.archived_at, t.created_by, t.updated_by, t.deleted_by, t.created_at, t.updated_at, t.deleted_at,
		v.template_version_id, v.version_number, v.status, v.definition_schema_version, v.definition_json, v.published_at, v.published_by, v.withdrawn_at, v.withdrawn_by, v.created_by, v.updated_by, v.created_at, v.updated_at, v.deleted_at
		FROM application_templates t JOIN application_template_versions v ON v.template_id = t.template_id AND v.deleted_at = 0
		WHERE t.deleted_at = 0 AND t.archived_at IS NULL AND v.template_version_id = ? AND v.status = 'published'`
	item, err := scanTemplateAggregate(r.db.QueryRowContext(ctx, r.placeholder.rebind(query), strings.TrimSpace(versionID)))
	if errors.Is(err, sql.ErrNoRows) {
		return ApplicationTemplateAggregate{}, ErrTemplateNotFound
	}
	if err != nil {
		return ApplicationTemplateAggregate{}, fmt.Errorf("get published application template version: %w", err)
	}
	return item, nil
}

// GetPublishedTemplate 返回当前仍可用于创建的已发布模板详情。
func (r *SQLRepository) GetPublishedTemplate(ctx context.Context, templateID string) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	query := `SELECT t.template_id, t.display_name, t.description, t.category, t.deployment_adapter_kind, t.archived_at, t.created_by, t.updated_by, t.deleted_by, t.created_at, t.updated_at, t.deleted_at,
		v.template_version_id, v.version_number, v.status, v.definition_schema_version, v.definition_json, v.published_at, v.published_by, v.withdrawn_at, v.withdrawn_by, v.created_by, v.updated_by, v.created_at, v.updated_at, v.deleted_at
		FROM application_templates t JOIN application_template_versions v ON v.template_id = t.template_id AND v.deleted_at = 0
		WHERE t.deleted_at = 0 AND t.archived_at IS NULL AND t.template_id = ? AND v.status = 'published'
		ORDER BY v.version_number DESC LIMIT 1`
	item, err := scanTemplateAggregate(r.db.QueryRowContext(ctx, r.placeholder.rebind(query), strings.TrimSpace(templateID)))
	if errors.Is(err, sql.ErrNoRows) {
		return ApplicationTemplateAggregate{}, ErrTemplateNotFound
	}
	if err != nil {
		return ApplicationTemplateAggregate{}, fmt.Errorf("get published application template: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) getTemplate(ctx context.Context, templateID, displayName string) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	column, value := "t.template_id", strings.TrimSpace(templateID)
	if displayName != "" {
		column, value = "t.display_name", displayName
	}
	if value == "" {
		return ApplicationTemplateAggregate{}, ErrInvalidInput
	}
	query := `SELECT t.template_id, t.display_name, t.description, t.category, t.deployment_adapter_kind, t.archived_at, t.created_by, t.updated_by, t.deleted_by, t.created_at, t.updated_at, t.deleted_at,
		v.template_version_id, v.version_number, v.status, v.definition_schema_version, v.definition_json, v.published_at, v.published_by, v.withdrawn_at, v.withdrawn_by, v.created_by, v.updated_by, v.created_at, v.updated_at, v.deleted_at
		FROM application_templates t JOIN application_template_versions v ON v.template_id = t.template_id AND v.deleted_at = 0
		WHERE t.deleted_at = 0 AND ` + column + ` = ? ORDER BY CASE v.status WHEN 'draft' THEN 0 WHEN 'published' THEN 1 ELSE 2 END, v.version_number DESC LIMIT 1`
	item, err := scanTemplateAggregate(r.db.QueryRowContext(ctx, r.placeholder.rebind(query), value))
	if errors.Is(err, sql.ErrNoRows) {
		return ApplicationTemplateAggregate{}, ErrTemplateNotFound
	}
	if err != nil {
		return ApplicationTemplateAggregate{}, fmt.Errorf("get application template: %w", err)
	}
	return item, nil
}

// CreateTemplateDraft 原子创建模板身份及其第一个可编辑草稿。
func (r *SQLRepository) CreateTemplateDraft(ctx context.Context, input CreateTemplateDraftInput) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	if err := validateCreateTemplateDraft(input); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	defer func() { _ = tx.Rollback() }()
	insertTemplate := r.placeholder.rebind(`INSERT INTO application_templates (template_id, display_name, description, category, deployment_adapter_kind, created_by, updated_by, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0)`)
	if _, err = tx.ExecContext(ctx, insertTemplate, input.TemplateID, input.DisplayName, input.Description, input.Category, input.DeploymentAdapterKind, input.ActorID, input.ActorID); err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	insertVersion := r.placeholder.rebind(`INSERT INTO application_template_versions (template_version_id, template_id, version_number, status, definition_schema_version, definition_json, created_by, updated_by, created_at, updated_at, deleted_at) VALUES (?, ?, 1, 'draft', ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0)`)
	if _, err = tx.ExecContext(ctx, insertVersion, input.VersionID, input.TemplateID, input.DefinitionSchemaVersion, input.DefinitionJSON, input.ActorID, input.ActorID); err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	if err = tx.Commit(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	return r.GetTemplate(ctx, input.TemplateID)
}

// UpdateTemplateDraft 仅修改存活且未归档模板的唯一可编辑草稿。
//
//nolint:cyclop // 事务需要显式区分身份、草稿与数据库失败状态。
func (r *SQLRepository) UpdateTemplateDraft(ctx context.Context, input UpdateTemplateDraftInput) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	category := strings.TrimSpace(input.Category)
	if strings.TrimSpace(input.TemplateID) == "" || strings.TrimSpace(input.DisplayName) == "" || !projectcontract.ApplicationTemplateCategory(category).Valid() || input.DefinitionSchemaVersion < 1 || len(input.DefinitionJSON) == 0 {
		return ApplicationTemplateAggregate{}, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	defer func() { _ = tx.Rollback() }()
	updateTemplate := r.placeholder.rebind(`UPDATE application_templates SET display_name = ?, description = ?, category = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE template_id = ? AND deleted_at = 0 AND archived_at IS NULL`)
	result, err := tx.ExecContext(ctx, updateTemplate, strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.Description), category, input.ActorID, strings.TrimSpace(input.TemplateID))
	if err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ApplicationTemplateAggregate{}, ErrTemplateNotFound
	}
	updateDraft := r.placeholder.rebind(`UPDATE application_template_versions SET definition_schema_version = ?, definition_json = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE template_id = ? AND status = 'draft' AND deleted_at = 0`)
	result, err = tx.ExecContext(ctx, updateDraft, input.DefinitionSchemaVersion, input.DefinitionJSON, input.ActorID, strings.TrimSpace(input.TemplateID))
	if err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	count, _ = result.RowsAffected()
	if count != 1 {
		return ApplicationTemplateAggregate{}, ErrTemplateDraftNotFound
	}
	if err = tx.Commit(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	return r.GetTemplate(ctx, input.TemplateID)
}

// CloneTemplate 将来源模板当前选中的定义复制为具有独立身份的草稿。
//
//nolint:cyclop // 来源读取和新模板写入必须在同一事务内保持可审计。
func (r *SQLRepository) CloneTemplate(ctx context.Context, input CloneTemplateInput) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	if strings.TrimSpace(input.SourceTemplateID) == "" || strings.TrimSpace(input.TemplateID) == "" || strings.TrimSpace(input.VersionID) == "" || strings.TrimSpace(input.DisplayName) == "" {
		return ApplicationTemplateAggregate{}, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var description, category, adapterKind string
	var schema int
	var definition []byte
	read := r.placeholder.rebind(`SELECT t.description, t.category, t.deployment_adapter_kind, v.definition_schema_version, v.definition_json
		FROM application_templates t JOIN application_template_versions v ON v.template_id = t.template_id AND v.deleted_at = 0
		WHERE t.template_id = ? AND t.deleted_at = 0
		ORDER BY CASE v.status WHEN 'draft' THEN 0 WHEN 'published' THEN 1 ELSE 2 END, v.version_number DESC LIMIT 1`)
	if err = tx.QueryRowContext(ctx, read, input.SourceTemplateID).Scan(&description, &category, &adapterKind, &schema, &definition); errors.Is(err, sql.ErrNoRows) {
		return ApplicationTemplateAggregate{}, ErrTemplateNotFound
	}
	if err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	insertTemplate := r.placeholder.rebind(`INSERT INTO application_templates (template_id, display_name, description, category, deployment_adapter_kind, created_by, updated_by, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0)`)
	if _, err = tx.ExecContext(ctx, insertTemplate, input.TemplateID, strings.TrimSpace(input.DisplayName), description, category, adapterKind, input.ActorID, input.ActorID); err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	insertVersion := r.placeholder.rebind(`INSERT INTO application_template_versions (template_version_id, template_id, version_number, status, definition_schema_version, definition_json, created_by, updated_by, created_at, updated_at, deleted_at) VALUES (?, ?, 1, 'draft', ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0)`)
	if _, err = tx.ExecContext(ctx, insertVersion, input.VersionID, input.TemplateID, schema, definition, input.ActorID, input.ActorID); err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	if err = tx.Commit(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	return r.GetTemplate(ctx, input.TemplateID)
}

// WithdrawTemplate 将当前已发布版本变更为撤回历史，并原子创建携带同一定义的下一草稿。
//
//nolint:cyclop // 状态转换和版本复制必须作为单一事务完成，避免出现无草稿的中间状态。
func (r *SQLRepository) WithdrawTemplate(ctx context.Context, input WithdrawTemplateInput) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	if strings.TrimSpace(input.TemplateID) == "" || strings.TrimSpace(input.VersionID) == "" {
		return ApplicationTemplateAggregate{}, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var schema int
	var definition []byte
	read := r.placeholder.rebind(`SELECT definition_schema_version, definition_json FROM application_template_versions WHERE template_id = ? AND status = 'published' AND deleted_at = 0 ORDER BY version_number DESC LIMIT 1`)
	if err = tx.QueryRowContext(ctx, read, input.TemplateID).Scan(&schema, &definition); errors.Is(err, sql.ErrNoRows) {
		return ApplicationTemplateAggregate{}, ErrTemplateDraftNotFound
	}
	if err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	update := r.placeholder.rebind(`UPDATE application_template_versions SET status = 'withdrawn', withdrawn_at = CURRENT_TIMESTAMP, withdrawn_by = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE template_id = ? AND status = 'published' AND deleted_at = 0`)
	result, err := tx.ExecContext(ctx, update, input.ActorID, input.ActorID, input.TemplateID)
	if err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ApplicationTemplateAggregate{}, ErrTemplateDraftNotFound
	}
	var next int
	if err = tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT COALESCE(MAX(version_number), 0) + 1 FROM application_template_versions WHERE template_id = ?`), input.TemplateID).Scan(&next); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	insert := r.placeholder.rebind(`INSERT INTO application_template_versions (template_version_id, template_id, version_number, status, definition_schema_version, definition_json, created_by, updated_by, created_at, updated_at, deleted_at) VALUES (?, ?, ?, 'draft', ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0)`)
	if _, err = tx.ExecContext(ctx, insert, input.VersionID, input.TemplateID, next, schema, definition, input.ActorID, input.ActorID); err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	if err = tx.Commit(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	return r.GetTemplate(ctx, input.TemplateID)
}

// PublishTemplateDraft 将唯一草稿转换为不可变的发布快照。
func (r *SQLRepository) PublishTemplateDraft(ctx context.Context, templateID string, actorID *uint64) (ApplicationTemplateAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationTemplateAggregate{}, err
	}
	query := r.placeholder.rebind(`UPDATE application_template_versions SET status = 'published', published_at = CURRENT_TIMESTAMP, published_by = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE template_id = ? AND status = 'draft' AND deleted_at = 0`)
	result, err := r.db.ExecContext(ctx, query, actorID, actorID, strings.TrimSpace(templateID))
	if err != nil {
		return ApplicationTemplateAggregate{}, mapTemplateWriteError(err)
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ApplicationTemplateAggregate{}, ErrTemplateDraftNotFound
	}
	return r.GetTemplate(ctx, templateID)
}

// ArchiveTemplate 从创建者目录隐藏存活模板，但不删除版本溯源。
func (r *SQLRepository) ArchiveTemplate(ctx context.Context, templateID string, actorID *uint64) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE application_templates SET archived_at = CURRENT_TIMESTAMP, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE template_id = ? AND deleted_at = 0 AND archived_at IS NULL`), actorID, strings.TrimSpace(templateID))
	if err != nil {
		return mapTemplateWriteError(err)
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ErrTemplateNotFound
	}
	return nil
}

// DeleteTemplate 写入模板软删除审计字段；版本快照保留，以支持既有应用来源追溯。
func (r *SQLRepository) DeleteTemplate(ctx context.Context, templateID string, actorID *uint64) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE application_templates SET deleted_at = ?, deleted_by = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP WHERE template_id = ? AND deleted_at = 0`), time.Now().Unix(), actorID, actorID, strings.TrimSpace(templateID))
	if err != nil {
		return mapTemplateWriteError(err)
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ErrTemplateNotFound
	}
	return nil
}

type templateScanner interface{ Scan(...any) error }

func scanTemplateAggregate(scanner templateScanner) (ApplicationTemplateAggregate, error) {
	var item ApplicationTemplateAggregate
	var versionID, versionStatus sql.NullString
	var versionNumber, schemaVersion, publishedBy, withdrawnBy, createdBy, updatedBy, deletedAt sql.NullInt64
	var publishedAt, withdrawnAt, createdAt, updatedAt sql.NullTime
	err := scanner.Scan(&item.Template.ID, &item.Template.DisplayName, &item.Template.Description, &item.Template.Category, &item.Template.DeploymentAdapterKind, &item.Template.ArchivedAt, &item.Template.CreatedBy, &item.Template.UpdatedBy, &item.Template.DeletedBy, &item.Template.CreatedAt, &item.Template.UpdatedAt, &item.Template.DeletedAt, &versionID, &versionNumber, &versionStatus, &schemaVersion, &item.Version.DefinitionJSON, &publishedAt, &publishedBy, &withdrawnAt, &withdrawnBy, &createdBy, &updatedBy, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		return item, err
	}
	if !versionID.Valid {
		return item, nil
	}
	item.Version.ID = versionID.String
	item.Version.TemplateID = item.Template.ID
	item.Version.VersionNumber = int(versionNumber.Int64)
	item.Version.Status = versionStatus.String
	item.Version.DefinitionSchemaVersion = int(schemaVersion.Int64)
	item.Version.PublishedAt = nullableTimePointer(publishedAt)
	item.Version.PublishedBy = nullableUint64Pointer(publishedBy)
	item.Version.WithdrawnAt = nullableTimePointer(withdrawnAt)
	item.Version.WithdrawnBy = nullableUint64Pointer(withdrawnBy)
	item.Version.CreatedBy = nullableUint64Pointer(createdBy)
	item.Version.UpdatedBy = nullableUint64Pointer(updatedBy)
	item.Version.CreatedAt = createdAt.Time
	item.Version.UpdatedAt = updatedAt.Time
	item.Version.DeletedAt = deletedAt.Int64
	return item, err
}

func nullableTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullableUint64Pointer(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}
	result := uint64(value.Int64) // #nosec G115 -- 数据库中的审计主体 ID 约束为非负整数。
	return &result
}

func validateCreateTemplateDraft(input CreateTemplateDraftInput) error {
	if strings.TrimSpace(input.TemplateID) == "" || strings.TrimSpace(input.VersionID) == "" || strings.TrimSpace(input.DisplayName) == "" || strings.TrimSpace(input.Category) == "" || strings.TrimSpace(input.DeploymentAdapterKind) == "" || input.DefinitionSchemaVersion < 1 || len(input.DefinitionJSON) == 0 {
		return ErrInvalidInput
	}
	return nil
}

func normalizeTemplateCatalogQuery(query TemplateCatalogQuery) (int, int, string) {
	page, pageSize := query.Page, query.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = templateCatalogPageSizeDefault
	}
	if pageSize > templateCatalogPageSizeMax {
		pageSize = templateCatalogPageSizeMax
	}
	sort := strings.TrimSpace(query.Sort)
	if sort != "name_asc" {
		sort = "updated_desc"
	}
	return page, pageSize, sort
}

func templateCatalogFilters(query TemplateCatalogQuery) ([]string, []any) {
	where := []string{"t.deleted_at = 0", "t.archived_at IS NULL", "v.deleted_at = 0", "v.status = 'published'"}
	args := []any{}
	if kind := strings.TrimSpace(query.DeploymentAdapterKind); kind != "" {
		where, args = append(where, "t.deployment_adapter_kind = ?"), append(args, kind)
	}
	if category := strings.TrimSpace(query.Category); category != "" {
		where, args = append(where, "t.category = ?"), append(args, category)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + escapeTemplateCatalogLikePattern(search) + "%"
		where = append(where, "(LOWER(t.display_name) LIKE LOWER(?) ESCAPE '\\' OR LOWER(t.description) LIKE LOWER(?) ESCAPE '\\')")
		args = append(args, pattern, pattern)
	}
	return where, args
}

// escapeTemplateCatalogLikePattern 转义模板目录搜索词中的 SQL LIKE 模式字符。
func escapeTemplateCatalogLikePattern(value string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value)
}

func templateCatalogOrder(sort string) string {
	if sort == "name_asc" {
		return "LOWER(t.display_name) ASC, t.template_id ASC"
	}
	return "t.updated_at DESC, t.template_id DESC"
}

func mapTemplateWriteError(err error) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "application_templates_display_name_live") ||
		strings.Contains(lower, "application_templates.display_name") {
		return fmt.Errorf("%w: %w", ErrTemplateNameOccupied, err)
	}
	if strings.Contains(lower, "unique") || strings.Contains(lower, "duplicate") {
		return fmt.Errorf("%w: %w", ErrTemplateConflict, err)
	}
	return err
}

// 编译期断言确保 SQL 实现持续满足窄化的模板仓储边界。
var _ TemplateRepository = (*SQLRepository)(nil)
