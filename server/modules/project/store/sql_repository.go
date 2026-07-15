package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SQLRepository persists project registry state in module-owned SQL tables.
type SQLRepository struct {
	db          *sql.DB
	placeholder placeholderStyle
}

const (
	projectListWhereArgCapacity         = 6
	workspaceAnnotationUpdateRetryLimit = 3
)

// NewSQLRepository 创建一个基于 SQL 的项目仓库。
// 当 db 为空时返回错误；否则返回可用于访问项目数据的仓库实例，并根据数据库类型确定占位符样式。
// @param db 数据库连接池。
// @returns SQL 仓库实例及错误信息。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("project repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db, placeholder: detectPlaceholderStyle(db)}, nil
}

// List returns one page of registered projects.
func (r *SQLRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if err := r.ensureReady(); err != nil {
		return ListResult{}, err
	}
	var err error
	query, err = normalizeListQuery(query)
	if err != nil {
		return ListResult{}, err
	}

	where, args := buildListWhere(query)
	countSQL := r.placeholder.rebind(`SELECT COUNT(*)
		FROM compose_projects
		WHERE ` + strings.Join(where, " AND "))
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count projects: %w", err)
	}

	projects, projectIDs, err := r.listProjectsPage(ctx, where, args, query, total)
	if err != nil {
		return ListResult{}, err
	}

	fileMap, snapshotMap, err := r.loadFilesAndSnapshots(ctx, projectIDs)
	if err != nil {
		return ListResult{}, err
	}
	items := buildProjectAggregates(projects, fileMap, snapshotMap)
	return ListResult{Items: items, Total: total}, nil
}

func (r *SQLRepository) listProjectsPage(
	ctx context.Context,
	where []string,
	args []any,
	query ListQuery,
	total int,
) ([]Project, []uint64, error) {
	argsWithPage := append(append([]any(nil), args...), query.Limit, query.Offset)
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			id, application_id, application_name, workspace_path, compose_project_name, compose_project_name_source,
			runtime_target_id, display_name, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
			working_directory, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
			last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM compose_projects
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+buildListOrderBy(query.Sort)+`
		LIMIT ? OFFSET ?`),
		argsWithPage...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list projects: %w", err)
	}
	defer closeRows(rows)

	pageCap := listPageCapacity(total, query.Offset, query.Limit)
	projects := make([]Project, 0, pageCap)
	projectIDs := make([]uint64, 0, pageCap)
	for rows.Next() {
		item, scanErr := scanProject(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan project row: %w", scanErr)
		}
		projects = append(projects, item)
		projectIDs = append(projectIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, projectIDs, nil
}

func buildProjectAggregates(
	projects []Project,
	fileMap map[uint64][]ProjectFile,
	snapshotMap map[uint64]Snapshot,
) []ProjectAggregate {
	items := make([]ProjectAggregate, 0, len(projects))
	for _, item := range projects {
		aggregate := ProjectAggregate{
			Project: item,
			Files:   fileMap[item.ID],
		}
		if snapshot, ok := snapshotMap[item.ID]; ok {
			snapshotCopy := snapshot
			aggregate.Snapshot = &snapshotCopy
		}
		items = append(items, aggregate)
	}
	return items
}

// Get returns one registered project aggregate.
func (r *SQLRepository) Get(ctx context.Context, projectID uint64) (ProjectAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ProjectAggregate{}, err
	}
	projectDBID, err := toDBID(projectID)
	if err != nil {
		return ProjectAggregate{}, err
	}
	project, err := scanProject(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT
			id, application_id, application_name, workspace_path, compose_project_name, compose_project_name_source,
			runtime_target_id, display_name, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
			working_directory, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
			last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM compose_projects
		WHERE id = ? AND deleted_at = 0`),
		projectDBID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProjectAggregate{}, ErrProjectNotFound
		}
		return ProjectAggregate{}, fmt.Errorf("get project: %w", err)
	}

	files, err := r.listFiles(ctx, projectID)
	if err != nil {
		return ProjectAggregate{}, err
	}
	snapshot, err := r.getSnapshot(ctx, projectID)
	if err != nil {
		return ProjectAggregate{}, err
	}
	aggregate := ProjectAggregate{
		Project: project,
		Files:   files,
	}
	if snapshot != nil {
		aggregate.Snapshot = snapshot
	}
	return aggregate, nil
}

// GetByApplicationID resolves the public Application ID without exposing the
// private database key in a caller-visible contract.
func (r *SQLRepository) GetByApplicationID(ctx context.Context, applicationID string) (ProjectAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ProjectAggregate{}, err
	}
	applicationID = strings.TrimSpace(applicationID)
	if applicationID == "" {
		return ProjectAggregate{}, ErrInvalidInput
	}
	var projectID int64
	err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT id FROM compose_projects WHERE application_id = ? AND deleted_at = 0`), applicationID).Scan(&projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProjectAggregate{}, ErrProjectNotFound
		}
		return ProjectAggregate{}, fmt.Errorf("get project by application id: %w", err)
	}
	if projectID < 1 {
		return ProjectAggregate{}, ErrProjectNotFound
	}
	return r.Get(ctx, uint64(projectID)) // #nosec G115 -- positivity is checked above.
}

// GetIDsByApplicationIDs resolves public application identifiers in one query.
func (r *SQLRepository) GetIDsByApplicationIDs(ctx context.Context, applicationIDs []string) (map[string]uint64, error) {
	result := make(map[string]uint64, len(applicationIDs))
	if len(applicationIDs) == 0 {
		return result, nil
	}
	placeholders := make([]string, 0, len(applicationIDs))
	args := make([]any, 0, len(applicationIDs))
	for _, value := range applicationIDs {
		if value = strings.TrimSpace(value); value != "" {
			placeholders = append(placeholders, "?")
			args = append(args, value)
		}
	}
	if len(args) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT application_id, id FROM compose_projects WHERE deleted_at = 0 AND application_id IN (`+strings.Join(placeholders, ",")+`)`), args...)
	if err != nil {
		return nil, fmt.Errorf("resolve project application ids: %w", err)
	}
	defer closeRows(rows)
	for rows.Next() {
		var applicationID string
		var projectID int64
		if err := rows.Scan(&applicationID, &projectID); err != nil {
			return nil, fmt.Errorf("scan project application id: %w", err)
		}
		if projectID > 0 {
			result[applicationID] = uint64(projectID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project application ids: %w", err)
	}
	return result, nil
}

// GetFile returns one file within the requested project scope.
func (r *SQLRepository) GetFile(ctx context.Context, projectID uint64, fileID uint64) (ProjectFile, error) {
	if err := r.ensureReady(); err != nil {
		return ProjectFile{}, err
	}
	projectDBID, err := toDBID(projectID)
	if err != nil {
		return ProjectFile{}, err
	}
	fileDBID, err := toDBID(fileID)
	if err != nil {
		return ProjectFile{}, err
	}
	item, err := scanProjectFile(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT
			f.id, f.project_id, f.kind, f.role, f.absolute_path, f.display_path, f.order_index,
			f.last_observed_hash, f.created_at, f.updated_at
		FROM compose_project_files f
		INNER JOIN compose_projects p ON p.id = f.project_id
		WHERE f.id = ? AND f.project_id = ? AND p.deleted_at = 0`),
		fileDBID,
		projectDBID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProjectFile{}, ErrFileNotFound
		}
		return ProjectFile{}, fmt.Errorf("get project file: %w", err)
	}
	return item, nil
}

// ImportProject creates or replaces one live project registry row.
func (r *SQLRepository) ImportProject(ctx context.Context, input ImportProjectInput) (ProjectAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ProjectAggregate{}, err
	}
	input, err := validateImportInput(input)
	if err != nil {
		return ProjectAggregate{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ProjectAggregate{}, fmt.Errorf("begin project import tx: %w", err)
	}
	defer rollbackTx(tx)

	now := time.Now().UTC()
	var projectID uint64
	if input.StrictCreate {
		projectID, err = r.createProject(ctx, tx, input, now)
	} else {
		projectID, err = r.upsertProject(ctx, tx, input, now)
	}
	if err != nil {
		return ProjectAggregate{}, err
	}
	if err := r.replaceFiles(ctx, tx, projectID, input.Files, now); err != nil {
		return ProjectAggregate{}, err
	}
	if err := r.replaceSnapshot(ctx, tx, projectID, input.Snapshot); err != nil {
		return ProjectAggregate{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProjectAggregate{}, fmt.Errorf("commit project import: %w", err)
	}
	return r.Get(ctx, projectID)
}

// RefreshProject updates one existing project's snapshot and drift metadata.
func (r *SQLRepository) RefreshProject(ctx context.Context, input RefreshProjectInput) (ProjectAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ProjectAggregate{}, err
	}
	input, err := validateRefreshInput(input)
	if err != nil {
		return ProjectAggregate{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ProjectAggregate{}, fmt.Errorf("begin project refresh tx: %w", err)
	}
	defer rollbackTx(tx)

	if err := r.ensureProjectExists(ctx, tx, input.ProjectID); err != nil {
		return ProjectAggregate{}, err
	}
	if err := r.updateRefreshState(ctx, tx, input); err != nil {
		return ProjectAggregate{}, err
	}
	if err := r.replaceRefreshFiles(ctx, tx, input); err != nil {
		return ProjectAggregate{}, err
	}
	if err := r.replaceSnapshot(ctx, tx, input.ProjectID, input.Snapshot); err != nil {
		return ProjectAggregate{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProjectAggregate{}, fmt.Errorf("commit project refresh: %w", err)
	}
	return r.Get(ctx, input.ProjectID)
}

// UpdateLifecycleConfig updates one existing project's saved lifecycle configuration.
func (r *SQLRepository) UpdateLifecycleConfig(ctx context.Context, input UpdateLifecycleConfigInput) (ProjectAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ProjectAggregate{}, err
	}
	input, err := validateUpdateLifecycleConfigInput(input)
	if err != nil {
		return ProjectAggregate{}, err
	}
	lifecycleConfigJSON, err := encodeLifecycleConfigJSON(input.LifecycleConfig)
	if err != nil {
		return ProjectAggregate{}, err
	}
	projectDBID, err := toDBID(input.ProjectID)
	if err != nil {
		return ProjectAggregate{}, err
	}
	result, err := r.db.ExecContext(
		ctx,
		r.placeholder.rebind(`UPDATE compose_projects
		SET lifecycle_strategy_kind = ?,
			lifecycle_review_status = ?,
			lifecycle_config_json = ?::jsonb,
			updated_by = ?,
			updated_at = NOW()
		WHERE id = ? AND deleted_at = 0`),
		input.LifecycleStrategyKind,
		input.LifecycleReviewStatus,
		string(lifecycleConfigJSON),
		input.ActorID,
		projectDBID,
	)
	if err != nil {
		return ProjectAggregate{}, mapWriteErr("update lifecycle config", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ProjectAggregate{}, fmt.Errorf("read lifecycle config update rows: %w", err)
	}
	if rowsAffected == 0 {
		return ProjectAggregate{}, ErrProjectNotFound
	}
	return r.Get(ctx, input.ProjectID)
}

// UpdateWorkspaceAnnotation updates or removes one workspace annotation on the owning project row.
func (r *SQLRepository) UpdateWorkspaceAnnotation(ctx context.Context, input UpdateWorkspaceAnnotationInput) (ProjectAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ProjectAggregate{}, err
	}
	input, err := validateUpdateWorkspaceAnnotationInput(input)
	if err != nil {
		return ProjectAggregate{}, err
	}
	projectDBID, err := toDBID(input.ProjectID)
	if err != nil {
		return ProjectAggregate{}, err
	}
	for attempt := 0; attempt < workspaceAnnotationUpdateRetryLimit; attempt++ {
		updated, err := r.tryUpdateWorkspaceAnnotation(ctx, projectDBID, input)
		if err != nil {
			return ProjectAggregate{}, err
		}
		if updated {
			return r.Get(ctx, input.ProjectID)
		}
	}
	return ProjectAggregate{}, ErrProjectConflict
}

func applyWorkspaceAnnotationUpdate(current map[string]string, relativePath string, annotation *string) map[string]string {
	next := make(map[string]string, len(current))
	for key, value := range current {
		next[key] = value
	}
	if annotation == nil {
		delete(next, relativePath)
		return next
	}
	next[relativePath] = *annotation
	return next
}

func (r *SQLRepository) tryUpdateWorkspaceAnnotation(
	ctx context.Context,
	projectDBID int64,
	input UpdateWorkspaceAnnotationInput,
) (bool, error) {
	currentEncoded, err := r.loadWorkspaceAnnotationsJSON(ctx, projectDBID)
	if err != nil {
		return false, err
	}
	annotations, err := decodeWorkspaceAnnotationsJSON([]byte(currentEncoded))
	if err != nil {
		return false, err
	}
	nextAnnotations := applyWorkspaceAnnotationUpdate(annotations, input.RelativePath, input.Annotation)
	encoded, err := encodeWorkspaceAnnotationsJSON(nextAnnotations)
	if err != nil {
		return false, err
	}
	result, err := r.db.ExecContext(
		ctx,
		r.placeholder.rebind(`UPDATE compose_projects
		SET workspace_annotations_json = `+r.placeholder.jsonParamExpr()+`,
			updated_by = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at = 0 AND workspace_annotations_json = ?`),
		string(encoded),
		input.ActorID,
		projectDBID,
		currentEncoded,
	)
	if err != nil {
		return false, mapWriteErr("update workspace annotation", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read workspace annotation update rows: %w", err)
	}
	return rowsAffected > 0, nil
}

func (r *SQLRepository) loadWorkspaceAnnotationsJSON(ctx context.Context, projectDBID int64) (string, error) {
	var raw string
	err := r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT workspace_annotations_json
		FROM compose_projects
		WHERE id = ? AND deleted_at = 0`),
		projectDBID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrProjectNotFound
		}
		return "", fmt.Errorf("load workspace annotations: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return "{}", nil
	}
	return raw, nil
}

// UnregisterProject soft-deletes one live project registry row without deleting host files.
func (r *SQLRepository) UnregisterProject(ctx context.Context, input UnregisterProjectInput) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	input, err := validateUnregisterInput(input)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin project unregister tx: %w", err)
	}
	defer rollbackTx(tx)

	if err := r.ensureProjectExists(ctx, tx, input.ProjectID); err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	projectDBID, err := toDBID(input.ProjectID)
	if err != nil {
		return err
	}
	var deletedBy any
	if input.ActorID != nil {
		actorID, convErr := toDBID(*input.ActorID)
		if convErr != nil {
			return convErr
		}
		deletedBy = actorID
	}
	if _, err := tx.ExecContext(
		ctx,
		r.placeholder.rebind(`UPDATE compose_projects
		SET deleted_at = ?, deleted_by = ?, updated_at = NOW(), updated_by = ?
		WHERE id = ? AND deleted_at = 0`),
		now,
		deletedBy,
		deletedBy,
		projectDBID,
	); err != nil {
		return fmt.Errorf("unregister project: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project unregister: %w", err)
	}
	return nil
}

// buildListWhere 构建项目列表查询的 WHERE 条件和参数。
// buildListWhere 构建项目列表查询的过滤条件及其参数，始终排除已删除项目，并按需筛选来源类型、漂移状态、关键字和运行时目标。
// 返回条件片段及其对应的参数值。
func buildListWhere(query ListQuery) ([]string, []any) {
	where := []string{"deleted_at = 0"}
	args := make([]any, 0, projectListWhereArgCapacity)
	if query.SourceKind != "" {
		where = append(where, "source_kind = ?")
		args = append(args, query.SourceKind)
	}
	if query.DriftStatus != "" {
		where = append(where, "drift_status = ?")
		args = append(args, query.DriftStatus)
	}
	if query.Keyword != "" {
		where = append(where, "(LOWER(display_name) LIKE LOWER(?) ESCAPE '\\' OR LOWER(canonical_project_name) LIKE LOWER(?) ESCAPE '\\' OR LOWER(working_directory) LIKE LOWER(?) ESCAPE '\\')")
		keyword := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(query.Keyword) + "%"
		args = append(args, keyword, keyword, keyword)
	}
	if query.RuntimeTargetID != nil {
		where = append(where, "runtime_target_id = ?")
		args = append(args, *query.RuntimeTargetID)
	}
	return where, args
}

// BackfillRuntimeTarget assigns the discovered Local Docker target to historical unbound local records.
func (r *SQLRepository) BackfillRuntimeTarget(ctx context.Context, runtimeTargetID uint64) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	id, err := toDBID(runtimeTargetID)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE compose_projects SET runtime_target_id = ?, updated_at = NOW(), updated_by = 0 WHERE deleted_at = 0 AND host_scope = 'local' AND runtime_target_id IS NULL`), id)
	return err
}

func (r *SQLRepository) ensureProjectExists(ctx context.Context, tx *sql.Tx, projectID uint64) error {
	exists, err := r.projectExists(ctx, tx, projectID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProjectNotFound
	}
	return nil
}

func (r *SQLRepository) replaceRefreshFiles(ctx context.Context, tx *sql.Tx, input RefreshProjectInput) error {
	if len(input.Files) == 0 {
		return nil
	}
	return r.replaceFiles(ctx, tx, input.ProjectID, input.Files, time.Now().UTC())
}

func (r *SQLRepository) listFiles(ctx context.Context, projectID uint64) ([]ProjectFile, error) {
	projectDBID, err := toDBID(projectID)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			id, project_id, kind, role, absolute_path, display_path, order_index,
			last_observed_hash, created_at, updated_at
		FROM compose_project_files
		WHERE project_id = ?
		ORDER BY order_index ASC, id ASC`),
		projectDBID,
	)
	if err != nil {
		return nil, fmt.Errorf("list project files: %w", err)
	}
	defer closeRows(rows)
	items := make([]ProjectFile, 0)
	for rows.Next() {
		item, scanErr := scanProjectFile(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan project file: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project files: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) getSnapshot(ctx context.Context, projectID uint64) (*Snapshot, error) {
	projectDBID, err := toDBID(projectID)
	if err != nil {
		return nil, err
	}
	item, err := scanSnapshot(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT
			project_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
		FROM compose_project_snapshots
		WHERE project_id = ?`),
		projectDBID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project snapshot: %w", err)
	}
	return &item, nil
}

func (r *SQLRepository) loadFilesAndSnapshots(
	ctx context.Context,
	projectIDs []uint64,
) (map[uint64][]ProjectFile, map[uint64]Snapshot, error) {
	fileMap := make(map[uint64][]ProjectFile, len(projectIDs))
	snapshotMap := make(map[uint64]Snapshot, len(projectIDs))
	if len(projectIDs) == 0 {
		return fileMap, snapshotMap, nil
	}
	fileMap, err := r.loadFilesByProjectIDs(ctx, projectIDs)
	if err != nil {
		return nil, nil, err
	}
	snapshotMap, err = r.loadSnapshotsByProjectIDs(ctx, projectIDs)
	if err != nil {
		return nil, nil, err
	}
	return fileMap, snapshotMap, nil
}

func (r *SQLRepository) loadFilesByProjectIDs(
	ctx context.Context,
	projectIDs []uint64,
) (map[uint64][]ProjectFile, error) {
	for _, id := range projectIDs {
		if id == 0 {
			return nil, ErrInvalidInput
		}
	}
	args, err := toDBArgs(projectIDs)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			id, project_id, kind, role, absolute_path, display_path, order_index,
			last_observed_hash, created_at, updated_at
		FROM compose_project_files
		WHERE project_id IN (`+placeholderList(len(args))+`)
		ORDER BY project_id ASC, order_index ASC, id ASC`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list project files: %w", err)
	}
	defer closeRows(rows)

	fileMap := make(map[uint64][]ProjectFile, len(projectIDs))
	for rows.Next() {
		item, scanErr := scanProjectFileSummary(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan project file: %w", scanErr)
		}
		fileMap[item.ProjectID] = append(fileMap[item.ProjectID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project files: %w", err)
	}
	return fileMap, nil
}

func (r *SQLRepository) loadSnapshotsByProjectIDs(
	ctx context.Context,
	projectIDs []uint64,
) (map[uint64]Snapshot, error) {
	for _, id := range projectIDs {
		if id == 0 {
			return nil, ErrInvalidInput
		}
	}
	args, err := toDBArgs(projectIDs)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			project_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
		FROM compose_project_snapshots
		WHERE project_id IN (`+placeholderList(len(args))+`)
		ORDER BY project_id ASC`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list project snapshots: %w", err)
	}
	defer closeRows(rows)

	snapshotMap := make(map[uint64]Snapshot, len(projectIDs))
	for rows.Next() {
		item, scanErr := scanSnapshot(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan project snapshot: %w", scanErr)
		}
		snapshotMap[item.ProjectID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project snapshots: %w", err)
	}
	return snapshotMap, nil
}

func (r *SQLRepository) upsertProject(
	ctx context.Context,
	tx *sql.Tx,
	input ImportProjectInput,
	now time.Time,
) (uint64, error) {
	var projectID uint64
	lifecycleConfigJSON, err := encodeLifecycleConfigJSON(input.LifecycleConfig)
	if err != nil {
		return 0, err
	}
	sourceMetadataJSON, err := encodeSourceMetadataJSON(input.SourceMetadata)
	if err != nil {
		return 0, err
	}
	var runtimeTargetID any
	if input.RuntimeTargetID > 0 {
		runtimeTargetID = input.RuntimeTargetID
	}
	err = tx.QueryRowContext(
		ctx,
		r.placeholder.rebind(composeProjectsUpsertSQL()),
		input.ApplicationID,
		input.ApplicationName,
		input.WorkspacePath,
		input.ComposeProjectName,
		input.ComposeProjectNameSource,
		input.DisplayName,
		runtimeTargetID,
		input.CanonicalProjectName,
		input.CanonicalProjectNameSource,
		input.SourceKind,
		input.HostScope,
		input.WorkingDirectory,
		input.OwnershipMode,
		string(sourceMetadataJSON),
		input.LifecycleStrategyKind,
		input.LifecycleReviewStatus,
		string(lifecycleConfigJSON),
		input.LastObservedConfigHash,
		`{}`,
		input.LastDriftCheckedAt,
		input.DriftStatus,
		input.ActorID,
		input.ActorID,
		now,
		now,
	).Scan(&projectID)
	if err != nil {
		return 0, mapWriteErr("upsert project", err)
	}
	return projectID, nil
}

func (r *SQLRepository) createProject(ctx context.Context, tx *sql.Tx, input ImportProjectInput, now time.Time) (uint64, error) {
	lifecycleConfigJSON, err := encodeLifecycleConfigJSON(input.LifecycleConfig)
	if err != nil {
		return 0, err
	}
	sourceMetadataJSON, err := encodeSourceMetadataJSON(input.SourceMetadata)
	if err != nil {
		return 0, err
	}
	var runtimeTargetID any
	if input.RuntimeTargetID > 0 {
		runtimeTargetID = input.RuntimeTargetID
	}
	var projectID uint64
	err = tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO compose_projects (
		application_id, application_name, workspace_path, compose_project_name, compose_project_name_source,
		display_name, runtime_target_id, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
		working_directory, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status,
		lifecycle_config_json, last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
		created_by, updated_by, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?, ?::jsonb, ?, ?::jsonb, ?, ?, ?, ?, ?, ?)
	RETURNING id`),
		input.ApplicationID, input.ApplicationName, input.WorkspacePath, input.ComposeProjectName, input.ComposeProjectNameSource,
		input.DisplayName, runtimeTargetID, input.CanonicalProjectName, input.CanonicalProjectNameSource, input.SourceKind, input.HostScope,
		input.WorkingDirectory, input.OwnershipMode, string(sourceMetadataJSON), input.LifecycleStrategyKind, input.LifecycleReviewStatus,
		string(lifecycleConfigJSON), input.LastObservedConfigHash, `{}`, input.LastDriftCheckedAt, input.DriftStatus,
		input.ActorID, input.ActorID, now, now,
	).Scan(&projectID)
	if err != nil {
		return 0, mapWriteErr("create project", err)
	}
	return projectID, nil
}

// composeProjectsUpsertSQL 返回用于插入或更新项目记录的 SQL 语句，并通过 RETURNING 子句返回项目 ID。
func composeProjectsUpsertSQL() string {
	return `INSERT INTO compose_projects (
			application_id, application_name, workspace_path, compose_project_name, compose_project_name_source,
			display_name, runtime_target_id, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
			working_directory, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
			last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
			created_by, updated_by, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?, ?::jsonb, ?, ?::jsonb, ?, ?, ?, ?, ?, ?
		)
		ON CONFLICT (runtime_target_id, compose_project_name) WHERE deleted_at = 0 DO UPDATE SET
			display_name = excluded.display_name,
			compose_project_name_source = excluded.compose_project_name_source,
			application_name = COALESCE(excluded.application_name, compose_projects.application_name),
			workspace_path = excluded.workspace_path,
			runtime_target_id = excluded.runtime_target_id,
			canonical_project_name_source = excluded.canonical_project_name_source,
			source_kind = excluded.source_kind,
			working_directory = excluded.working_directory,
			ownership_mode = excluded.ownership_mode,
			source_metadata_json = excluded.source_metadata_json,
			lifecycle_strategy_kind = excluded.lifecycle_strategy_kind,
			lifecycle_review_status = excluded.lifecycle_review_status,
			lifecycle_config_json = excluded.lifecycle_config_json,
			last_observed_config_hash = excluded.last_observed_config_hash,
			last_drift_checked_at = excluded.last_drift_checked_at,
			drift_status = excluded.drift_status,
			updated_by = excluded.updated_by,
			updated_at = excluded.updated_at
		RETURNING id`
}

func (r *SQLRepository) replaceFiles(
	ctx context.Context,
	tx *sql.Tx,
	projectID uint64,
	files []ProjectFile,
	now time.Time,
) error {
	projectDBID, err := toDBID(projectID)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		r.placeholder.rebind(`DELETE FROM compose_project_files WHERE project_id = ?`),
		projectDBID,
	); err != nil {
		return fmt.Errorf("delete project files: %w", err)
	}
	for _, item := range files {
		if _, err := tx.ExecContext(
			ctx,
			r.placeholder.rebind(`INSERT INTO compose_project_files (
				project_id, kind, role, absolute_path, display_path, order_index,
				last_observed_hash, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
			projectDBID,
			item.Kind,
			item.Role,
			item.AbsolutePath,
			item.DisplayPath,
			item.OrderIndex,
			item.LastObservedHash,
			now,
			now,
		); err != nil {
			return mapWriteErr("insert project file", err)
		}
	}
	return nil
}

func (r *SQLRepository) replaceSnapshot(
	ctx context.Context,
	tx *sql.Tx,
	projectID uint64,
	snapshot *Snapshot,
) error {
	projectDBID, err := toDBID(projectID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		if _, err := tx.ExecContext(
			ctx,
			r.placeholder.rebind(`DELETE FROM compose_project_snapshots WHERE project_id = ?`),
			projectDBID,
		); err != nil {
			return fmt.Errorf("delete project snapshot: %w", err)
		}
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		r.placeholder.rebind(`INSERT INTO compose_project_snapshots (
			project_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
		) VALUES (?, ?::jsonb, ?, ?, ?, ?)
		ON CONFLICT (project_id) DO UPDATE SET
			normalized_compose_json = excluded.normalized_compose_json,
			config_hash = excluded.config_hash,
			declared_service_count = excluded.declared_service_count,
			declared_services_digest = excluded.declared_services_digest,
			refreshed_at = excluded.refreshed_at`),
		projectDBID,
		string(snapshot.NormalizedComposeJSON),
		snapshot.ConfigHash,
		snapshot.DeclaredServiceCount,
		snapshot.DeclaredServicesDigest,
		snapshot.RefreshedAt,
	); err != nil {
		return mapWriteErr("upsert project snapshot", err)
	}
	return nil
}

func (r *SQLRepository) projectExists(ctx context.Context, tx *sql.Tx, projectID uint64) (bool, error) {
	projectDBID, err := toDBID(projectID)
	if err != nil {
		return false, err
	}
	var count int
	if err := tx.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT COUNT(*) FROM compose_projects WHERE id = ? AND deleted_at = 0`),
		projectDBID,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check project existence: %w", err)
	}
	return count > 0, nil
}

func (r *SQLRepository) updateRefreshState(ctx context.Context, tx *sql.Tx, input RefreshProjectInput) error {
	projectDBID, err := toDBID(input.ProjectID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		r.placeholder.rebind(`UPDATE compose_projects
		SET last_observed_config_hash = ?,
			last_drift_checked_at = ?,
			drift_status = ?,
			updated_by = ?,
			updated_at = NOW()
		WHERE id = ? AND deleted_at = 0`),
		input.LastObservedConfigHash,
		input.LastDriftCheckedAt,
		input.DriftStatus,
		input.ActorID,
		projectDBID,
	)
	if err != nil {
		return mapWriteErr("update project refresh state", err)
	}
	return nil
}
