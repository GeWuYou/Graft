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

const projectListWhereArgCapacity = 3

// NewSQLRepository creates a SQL-backed project repository.
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
	query = normalizeListQuery(query)

	where, args := buildListWhere(query)
	countSQL := r.placeholder.rebind(`SELECT COUNT(*)
		FROM compose_projects
		WHERE ` + strings.Join(where, " AND "))
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count projects: %w", err)
	}

	argsWithPage := append(append([]any(nil), args...), query.Limit, query.Offset)
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			id, display_name, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
			working_directory, ownership_mode, last_refresh_status, last_refresh_at, last_refresh_error_code,
			last_refresh_error_message, last_refresh_config_hash, last_observed_config_hash, last_drift_checked_at,
			drift_status, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM compose_projects
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY updated_at DESC, id DESC
		LIMIT ? OFFSET ?`),
		argsWithPage...,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("list projects: %w", err)
	}
	defer closeRows(rows)

	projects := make([]Project, 0, query.Limit)
	projectIDs := make([]uint64, 0, query.Limit)
	for rows.Next() {
		item, scanErr := scanProject(rows)
		if scanErr != nil {
			return ListResult{}, fmt.Errorf("scan project row: %w", scanErr)
		}
		projects = append(projects, item)
		projectIDs = append(projectIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate projects: %w", err)
	}

	fileMap, snapshotMap, err := r.loadFilesAndSnapshots(ctx, projectIDs)
	if err != nil {
		return ListResult{}, err
	}

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
	return ListResult{Items: items, Total: total}, nil
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
			id, display_name, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
			working_directory, ownership_mode, last_refresh_status, last_refresh_at, last_refresh_error_code,
			last_refresh_error_message, last_refresh_config_hash, last_observed_config_hash, last_drift_checked_at,
			drift_status, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
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
			id, project_id, kind, role, absolute_path, display_path, order_index,
			exists_on_last_refresh, last_observed_hash, created_at, updated_at
		FROM compose_project_files
		WHERE id = ? AND project_id = ?`),
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
	projectID, err := r.upsertProject(ctx, tx, input, now)
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
	if query.LastRefreshStatus != "" {
		where = append(where, "last_refresh_status = ?")
		args = append(args, query.LastRefreshStatus)
	}
	return where, args
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
			exists_on_last_refresh, last_observed_hash, created_at, updated_at
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
	for _, id := range projectIDs {
		files, err := r.listFiles(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		fileMap[id] = files
		snapshot, err := r.getSnapshot(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if snapshot != nil {
			snapshotMap[id] = *snapshot
		}
	}
	return fileMap, snapshotMap, nil
}

func (r *SQLRepository) upsertProject(
	ctx context.Context,
	tx *sql.Tx,
	input ImportProjectInput,
	now time.Time,
) (uint64, error) {
	var projectID uint64
	err := tx.QueryRowContext(
		ctx,
		r.placeholder.rebind(`INSERT INTO compose_projects (
			display_name, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
			working_directory, ownership_mode, last_refresh_status, last_refresh_at, last_refresh_error_code,
			last_refresh_error_message, last_refresh_config_hash, last_observed_config_hash, last_drift_checked_at,
			drift_status, created_by, updated_by, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
		ON CONFLICT (host_scope, canonical_project_name) WHERE deleted_at = 0 DO UPDATE SET
			display_name = excluded.display_name,
			canonical_project_name_source = excluded.canonical_project_name_source,
			source_kind = excluded.source_kind,
			working_directory = excluded.working_directory,
			ownership_mode = excluded.ownership_mode,
			last_refresh_status = excluded.last_refresh_status,
			last_refresh_at = excluded.last_refresh_at,
			last_refresh_error_code = excluded.last_refresh_error_code,
			last_refresh_error_message = excluded.last_refresh_error_message,
			last_refresh_config_hash = excluded.last_refresh_config_hash,
			last_observed_config_hash = excluded.last_observed_config_hash,
			last_drift_checked_at = excluded.last_drift_checked_at,
			drift_status = excluded.drift_status,
			updated_by = excluded.updated_by,
			updated_at = excluded.updated_at
		RETURNING id`),
		input.DisplayName,
		input.CanonicalProjectName,
		input.CanonicalProjectNameSource,
		input.SourceKind,
		input.HostScope,
		input.WorkingDirectory,
		input.OwnershipMode,
		input.LastRefreshStatus,
		input.LastRefreshAt,
		input.LastRefreshErrorCode,
		input.LastRefreshErrorMessage,
		input.LastRefreshConfigHash,
		input.LastObservedConfigHash,
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
				exists_on_last_refresh, last_observed_hash, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
			projectDBID,
			item.Kind,
			item.Role,
			item.AbsolutePath,
			item.DisplayPath,
			item.OrderIndex,
			item.ExistsOnLastRefresh,
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
		SET last_refresh_status = ?,
			last_refresh_at = ?,
			last_refresh_error_code = ?,
			last_refresh_error_message = ?,
			last_refresh_config_hash = ?,
			last_observed_config_hash = ?,
			last_drift_checked_at = ?,
			drift_status = ?,
			updated_by = ?,
			updated_at = NOW()
		WHERE id = ? AND deleted_at = 0`),
		input.LastRefreshStatus,
		input.LastRefreshAt,
		input.LastRefreshErrorCode,
		input.LastRefreshErrorMessage,
		input.LastRefreshConfigHash,
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
