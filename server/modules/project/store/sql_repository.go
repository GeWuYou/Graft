package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SQLRepository 持久化 Application 模块拥有的注册表、文件快照和生命周期配置。
// 对外读取以 deleted_at = 0 的存活应用为边界，写入通过事务保持应用主记录与文件、快照一致。
type SQLRepository struct {
	db          *sql.DB
	placeholder placeholderStyle
}

const (
	projectListWhereArgCapacity         = 6
	workspaceAnnotationUpdateRetryLimit = 3
)

// NewSQLRepository 创建基于 SQL 的应用仓库，并根据数据库类型选择占位符样式。
// db 为空时返回错误；返回的仓库只允许访问模块拥有的应用表。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("application repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db, placeholder: detectPlaceholderStyle(db)}, nil
}

// List 返回一页存活应用及其文件、快照聚合结果。
// 查询先使用同一过滤条件计算总数，再按白名单排序表达式分页；附属文件和快照通过批量查询装配，避免逐应用查询。
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
		FROM applications
		WHERE ` + strings.Join(where, " AND "))
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count applications: %w", err)
	}

	applications, applicationRecordIDs, err := r.listApplicationsPage(ctx, where, args, query, total)
	if err != nil {
		return ListResult{}, err
	}

	fileMap, snapshotMap, err := r.loadFilesAndSnapshots(ctx, applicationRecordIDs)
	if err != nil {
		return ListResult{}, err
	}
	items := buildApplicationAggregates(applications, fileMap, snapshotMap)
	return ListResult{Items: items, Total: total}, nil
}

func (r *SQLRepository) listApplicationsPage(
	ctx context.Context,
	where []string,
	args []any,
	query ListQuery,
	total int,
) ([]Application, []uint64, error) {
	// 排序片段只能来自 normalizeListQuery 的白名单，避免把请求值直接拼入 SQL。
	argsWithPage := append(append([]any(nil), args...), query.Limit, query.Offset)
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			application_record_id, application_id, deployment_adapter_kind, application_name, workspace_path, compose_project_name, compose_project_name_source,
			runtime_target_id, display_name, source_type, ownership_mode, source_metadata_json,
			lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
			last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM applications
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+buildListOrderBy(query.Sort)+`
		LIMIT ? OFFSET ?`),
		argsWithPage...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list applications: %w", err)
	}
	defer closeRows(rows)

	pageCap := listPageCapacity(total, query.Offset, query.Limit)
	applications := make([]Application, 0, pageCap)
	applicationRecordIDs := make([]uint64, 0, pageCap)
	for rows.Next() {
		item, scanErr := scanApplication(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan application row: %w", scanErr)
		}
		applications = append(applications, item)
		applicationRecordIDs = append(applicationRecordIDs, item.ApplicationRecordID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate applications: %w", err)
	}
	return applications, applicationRecordIDs, nil
}

func buildApplicationAggregates(
	applications []Application,
	fileMap map[uint64][]ApplicationFile,
	snapshotMap map[uint64]Snapshot,
) []ApplicationAggregate {
	items := make([]ApplicationAggregate, 0, len(applications))
	for _, item := range applications {
		aggregate := ApplicationAggregate{
			Application: item,
			Files:       fileMap[item.ApplicationRecordID],
		}
		if snapshot, ok := snapshotMap[item.ApplicationRecordID]; ok {
			snapshotCopy := snapshot
			aggregate.Snapshot = &snapshotCopy
		}
		items = append(items, aggregate)
	}
	return items
}

// Get 返回一个已登记的应用聚合；不存在或应用已软删除时返回 ErrApplicationNotFound。
func (r *SQLRepository) Get(ctx context.Context, applicationRecordID uint64) (ApplicationAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationAggregate{}, err
	}
	projectDBID, err := toDBID(applicationRecordID)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	application, err := scanApplication(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT
			application_record_id, application_id, deployment_adapter_kind, application_name, workspace_path, compose_project_name, compose_project_name_source,
			runtime_target_id, display_name, source_type, ownership_mode, source_metadata_json,
			lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
			last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
			created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		FROM applications
		WHERE application_record_id = ? AND deleted_at = 0`),
		projectDBID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ApplicationAggregate{}, ErrApplicationNotFound
		}
		return ApplicationAggregate{}, fmt.Errorf("get application: %w", err)
	}

	files, err := r.listFiles(ctx, applicationRecordID)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	snapshot, err := r.getSnapshot(ctx, applicationRecordID)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	aggregate := ApplicationAggregate{
		Application: application,
		Files:       files,
	}
	if snapshot != nil {
		aggregate.Snapshot = snapshot
	}
	return aggregate, nil
}

// GetByApplicationID 根据公开 Application ID 解析应用，不向调用方暴露模块私有数据库主键。
func (r *SQLRepository) GetByApplicationID(ctx context.Context, applicationID string) (ApplicationAggregate, error) {
	return r.getByLiveIdentifier(ctx, applicationID, "application_id", "application id")
}

// GetByApplicationName 根据存活的受管应用名称解析应用，不向调用方暴露模块私有数据库主键。
func (r *SQLRepository) GetByApplicationName(ctx context.Context, applicationName string) (ApplicationAggregate, error) {
	return r.getByLiveIdentifier(ctx, applicationName, "application_name", "application name")
}

func (r *SQLRepository) getByLiveIdentifier(ctx context.Context, value, column, label string) (ApplicationAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationAggregate{}, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ApplicationAggregate{}, ErrInvalidInput
	}
	query, ok := liveApplicationIdentifierQueries[column]
	if !ok {
		return ApplicationAggregate{}, ErrInvalidInput
	}
	var applicationRecordID int64
	err := r.db.QueryRowContext(ctx, r.placeholder.rebind(query), value).Scan(&applicationRecordID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ApplicationAggregate{}, ErrApplicationNotFound
		}
		return ApplicationAggregate{}, fmt.Errorf("get application by %s: %w", label, err)
	}
	if applicationRecordID < 1 {
		return ApplicationAggregate{}, ErrApplicationNotFound
	}
	return r.Get(ctx, uint64(applicationRecordID)) // #nosec G115 -- positivity is checked immediately above.
}

var liveApplicationIdentifierQueries = map[string]string{
	"application_id":   `SELECT application_record_id FROM applications WHERE application_id = ? AND deleted_at = 0`,
	"application_name": `SELECT application_record_id FROM applications WHERE application_name = ? AND deleted_at = 0`,
}

// GetRecordIDsByApplicationIDs 在一次查询中解析公开应用标识，避免为批量请求逐项读取聚合。
func (r *SQLRepository) GetRecordIDsByApplicationIDs(ctx context.Context, applicationIDs []string) (map[string]uint64, error) {
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
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT application_id, application_record_id FROM applications WHERE deleted_at = 0 AND application_id IN (`+strings.Join(placeholders, ",")+`)`), args...)
	if err != nil {
		return nil, fmt.Errorf("resolve application application ids: %w", err)
	}
	defer closeRows(rows)
	for rows.Next() {
		var applicationID string
		var applicationRecordID int64
		if err := rows.Scan(&applicationID, &applicationRecordID); err != nil {
			return nil, fmt.Errorf("scan application application id: %w", err)
		}
		if applicationRecordID > 0 {
			result[applicationID] = uint64(applicationRecordID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application application ids: %w", err)
	}
	return result, nil
}

// GetFile 返回指定应用范围内的一个文件；应用或文件不存在时返回对应错误。
func (r *SQLRepository) GetFile(ctx context.Context, applicationRecordID uint64, fileID uint64) (ApplicationFile, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationFile{}, err
	}
	projectDBID, err := toDBID(applicationRecordID)
	if err != nil {
		return ApplicationFile{}, err
	}
	fileDBID, err := toDBID(fileID)
	if err != nil {
		return ApplicationFile{}, err
	}
	item, err := scanApplicationFile(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT
			f.id, f.application_record_id, f.kind, f.role, f.absolute_path, f.display_path, f.order_index,
			f.last_observed_hash, f.created_at, f.updated_at
		FROM application_files f
		INNER JOIN applications p ON p.application_record_id = f.application_record_id
		WHERE f.id = ? AND f.application_record_id = ? AND p.deleted_at = 0`),
		fileDBID,
		projectDBID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ApplicationFile{}, ErrFileNotFound
		}
		return ApplicationFile{}, fmt.Errorf("get application file: %w", err)
	}
	return item, nil
}

// ImportApplication 在一个事务中创建或替换存活应用，并同步替换文件与快照。
// 提交前任一步骤失败都会回滚，提交后再读取完整聚合，确保调用方看到的是数据库已确认的状态。
func (r *SQLRepository) ImportApplication(ctx context.Context, input ImportApplicationInput) (ApplicationAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationAggregate{}, err
	}
	input, err := validateImportInput(input)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplicationAggregate{}, fmt.Errorf("begin application import tx: %w", err)
	}
	defer rollbackTx(tx)

	now := time.Now().UTC()
	var applicationRecordID uint64
	if input.StrictCreate {
		applicationRecordID, err = r.createApplication(ctx, tx, input, now)
	} else {
		applicationRecordID, err = r.upsertApplication(ctx, tx, input, now)
	}
	if err != nil {
		return ApplicationAggregate{}, err
	}
	if err := r.replaceFiles(ctx, tx, applicationRecordID, input.Files, now); err != nil {
		return ApplicationAggregate{}, err
	}
	if err := r.replaceSnapshot(ctx, tx, applicationRecordID, input.Snapshot); err != nil {
		return ApplicationAggregate{}, err
	}
	if err := tx.Commit(); err != nil {
		return ApplicationAggregate{}, fmt.Errorf("commit application import: %w", err)
	}
	return r.Get(ctx, applicationRecordID)
}

// RefreshApplication 在一个事务中更新应用快照、漂移元数据及文件投影。
// 应用必须在事务内仍为存活记录；事务提交后才重新读取聚合结果。
func (r *SQLRepository) RefreshApplication(ctx context.Context, input RefreshApplicationInput) (ApplicationAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationAggregate{}, err
	}
	input, err := validateRefreshInput(input)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplicationAggregate{}, fmt.Errorf("begin application refresh tx: %w", err)
	}
	defer rollbackTx(tx)

	if err := r.ensureApplicationExists(ctx, tx, input.ApplicationRecordID); err != nil {
		return ApplicationAggregate{}, err
	}
	if err := r.updateRefreshState(ctx, tx, input); err != nil {
		return ApplicationAggregate{}, err
	}
	if err := r.replaceRefreshFiles(ctx, tx, input); err != nil {
		return ApplicationAggregate{}, err
	}
	if err := r.replaceSnapshot(ctx, tx, input.ApplicationRecordID, input.Snapshot); err != nil {
		return ApplicationAggregate{}, err
	}
	if err := tx.Commit(); err != nil {
		return ApplicationAggregate{}, fmt.Errorf("commit application refresh: %w", err)
	}
	return r.Get(ctx, input.ApplicationRecordID)
}

// UpdateLifecycleConfig 更新存活应用保存的生命周期配置，并保留配置确认状态这一持久化契约。
func (r *SQLRepository) UpdateLifecycleConfig(ctx context.Context, input UpdateLifecycleConfigInput) (ApplicationAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationAggregate{}, err
	}
	input, err := validateUpdateLifecycleConfigInput(input)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	lifecycleConfigJSON, err := encodeLifecycleConfigJSON(input.LifecycleConfig)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	projectDBID, err := toDBID(input.ApplicationRecordID)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	result, err := r.db.ExecContext(
		ctx,
		r.placeholder.rebind(`UPDATE applications
		SET lifecycle_strategy_kind = ?,
			lifecycle_review_status = ?,
			lifecycle_config_json = ?::jsonb,
			updated_by = ?,
			updated_at = NOW()
		WHERE application_record_id = ? AND deleted_at = 0`),
		input.LifecycleStrategyKind,
		input.LifecycleReviewStatus,
		string(lifecycleConfigJSON),
		input.ActorID,
		projectDBID,
	)
	if err != nil {
		return ApplicationAggregate{}, mapWriteErr("update lifecycle config", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ApplicationAggregate{}, fmt.Errorf("read lifecycle config update rows: %w", err)
	}
	if rowsAffected == 0 {
		return ApplicationAggregate{}, ErrApplicationNotFound
	}
	return r.Get(ctx, input.ApplicationRecordID)
}

// UpdateWorkspaceAnnotation 更新或删除应用行上的一个工作区注释，注释归属始终由应用记录限定。
func (r *SQLRepository) UpdateWorkspaceAnnotation(ctx context.Context, input UpdateWorkspaceAnnotationInput) (ApplicationAggregate, error) {
	if err := r.ensureReady(); err != nil {
		return ApplicationAggregate{}, err
	}
	input, err := validateUpdateWorkspaceAnnotationInput(input)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	projectDBID, err := toDBID(input.ApplicationRecordID)
	if err != nil {
		return ApplicationAggregate{}, err
	}
	for attempt := 0; attempt < workspaceAnnotationUpdateRetryLimit; attempt++ {
		updated, err := r.tryUpdateWorkspaceAnnotation(ctx, projectDBID, input)
		if err != nil {
			return ApplicationAggregate{}, err
		}
		if updated {
			return r.Get(ctx, input.ApplicationRecordID)
		}
	}
	return ApplicationAggregate{}, ErrApplicationConflict
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
		r.placeholder.rebind(`UPDATE applications
		SET workspace_annotations_json = `+r.placeholder.jsonParamExpr()+`,
			updated_by = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE application_record_id = ? AND deleted_at = 0 AND workspace_annotations_json = ?`),
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
		FROM applications
		WHERE application_record_id = ? AND deleted_at = 0`),
		projectDBID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrApplicationNotFound
		}
		return "", fmt.Errorf("load workspace annotations: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return "{}", nil
	}
	return raw, nil
}

// UnregisterApplication 软删除一条存活应用记录，但不删除工作区文件。
// 该边界允许后续审计或人工恢复引用历史注册信息，同时由 deleted_at 过滤后续业务读取。
func (r *SQLRepository) UnregisterApplication(ctx context.Context, input UnregisterApplicationInput) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	input, err := validateUnregisterInput(input)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin application unregister tx: %w", err)
	}
	defer rollbackTx(tx)

	if err := r.ensureApplicationExists(ctx, tx, input.ApplicationRecordID); err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	projectDBID, err := toDBID(input.ApplicationRecordID)
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
		r.placeholder.rebind(`UPDATE applications
		SET deleted_at = ?, deleted_by = ?, updated_at = NOW(), updated_by = ?
		WHERE application_record_id = ? AND deleted_at = 0`),
		now,
		deletedBy,
		deletedBy,
		projectDBID,
	); err != nil {
		return fmt.Errorf("unregister application: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit application unregister: %w", err)
	}
	return nil
}

// buildListWhere 构建项目列表查询的过滤条件及参数。
// 条件始终排除已删除项目；关键字会转义 LIKE 通配符，避免用户输入改变匹配语义。
func buildListWhere(query ListQuery) ([]string, []any) {
	where := []string{"deleted_at = 0"}
	args := make([]any, 0, projectListWhereArgCapacity)
	if query.SourceType != "" {
		where = append(where, "source_type = ?")
		args = append(args, query.SourceType)
	}
	if query.DriftStatus != "" {
		where = append(where, "drift_status = ?")
		args = append(args, query.DriftStatus)
	}
	if query.Keyword != "" {
		where = append(where, "(LOWER(display_name) LIKE LOWER(?) ESCAPE '\\' OR LOWER(compose_project_name) LIKE LOWER(?) ESCAPE '\\' OR LOWER(workspace_path) LIKE LOWER(?) ESCAPE '\\')")
		keyword := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(query.Keyword) + "%"
		args = append(args, keyword, keyword, keyword)
	}
	if query.RuntimeTargetID != nil {
		where = append(where, "runtime_target_id = ?")
		args = append(args, *query.RuntimeTargetID)
	}
	return where, args
}

// BackfillRuntimeTarget 为历史上未绑定运行时目标的本地应用补齐已发现的 Local Docker 目标。
func (r *SQLRepository) BackfillRuntimeTarget(ctx context.Context, runtimeTargetID uint64) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	id, err := toDBID(runtimeTargetID)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE applications SET runtime_target_id = ?, updated_at = NOW(), updated_by = 0 WHERE deleted_at = 0 AND runtime_target_id IS NULL`), id)
	return err
}

func (r *SQLRepository) ensureApplicationExists(ctx context.Context, tx *sql.Tx, applicationRecordID uint64) error {
	exists, err := r.applicationExists(ctx, tx, applicationRecordID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrApplicationNotFound
	}
	return nil
}

func (r *SQLRepository) replaceRefreshFiles(ctx context.Context, tx *sql.Tx, input RefreshApplicationInput) error {
	if len(input.Files) == 0 {
		return nil
	}
	return r.replaceFiles(ctx, tx, input.ApplicationRecordID, input.Files, time.Now().UTC())
}

func (r *SQLRepository) listFiles(ctx context.Context, applicationRecordID uint64) ([]ApplicationFile, error) {
	// order_index 表示导入时的文件顺序，id 只用于处理并列顺序，不能改为按路径排序。
	projectDBID, err := toDBID(applicationRecordID)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			id, application_record_id, kind, role, absolute_path, display_path, order_index,
			last_observed_hash, created_at, updated_at
		FROM application_files
		WHERE application_record_id = ?
		ORDER BY order_index ASC, id ASC`),
		projectDBID,
	)
	if err != nil {
		return nil, fmt.Errorf("list application files: %w", err)
	}
	defer closeRows(rows)
	items := make([]ApplicationFile, 0)
	for rows.Next() {
		item, scanErr := scanApplicationFile(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan application file: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application files: %w", err)
	}
	return items, nil
}

func (r *SQLRepository) getSnapshot(ctx context.Context, applicationRecordID uint64) (*Snapshot, error) {
	projectDBID, err := toDBID(applicationRecordID)
	if err != nil {
		return nil, err
	}
	item, err := scanSnapshot(r.db.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT
			application_record_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
		FROM application_snapshots
		WHERE application_record_id = ?`),
		projectDBID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get application snapshot: %w", err)
	}
	return &item, nil
}

func (r *SQLRepository) loadFilesAndSnapshots(
	ctx context.Context,
	applicationRecordIDs []uint64,
) (map[uint64][]ApplicationFile, map[uint64]Snapshot, error) {
	fileMap := make(map[uint64][]ApplicationFile, len(applicationRecordIDs))
	snapshotMap := make(map[uint64]Snapshot, len(applicationRecordIDs))
	if len(applicationRecordIDs) == 0 {
		return fileMap, snapshotMap, nil
	}
	fileMap, err := r.loadFilesByApplicationRecordIDs(ctx, applicationRecordIDs)
	if err != nil {
		return nil, nil, err
	}
	snapshotMap, err = r.loadSnapshotsByApplicationRecordIDs(ctx, applicationRecordIDs)
	if err != nil {
		return nil, nil, err
	}
	return fileMap, snapshotMap, nil
}

func (r *SQLRepository) loadFilesByApplicationRecordIDs(
	ctx context.Context,
	applicationRecordIDs []uint64,
) (map[uint64][]ApplicationFile, error) {
	for _, id := range applicationRecordIDs {
		if id == 0 {
			return nil, ErrInvalidInput
		}
	}
	args, err := toDBArgs(applicationRecordIDs)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			id, application_record_id, kind, role, absolute_path, display_path, order_index,
			last_observed_hash, created_at, updated_at
		FROM application_files
		WHERE application_record_id IN (`+placeholderList(len(args))+`)
		ORDER BY application_record_id ASC, order_index ASC, id ASC`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list application files: %w", err)
	}
	defer closeRows(rows)

	fileMap := make(map[uint64][]ApplicationFile, len(applicationRecordIDs))
	for rows.Next() {
		item, scanErr := scanApplicationFileSummary(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan application file: %w", scanErr)
		}
		fileMap[item.ApplicationRecordID] = append(fileMap[item.ApplicationRecordID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application files: %w", err)
	}
	return fileMap, nil
}

func (r *SQLRepository) loadSnapshotsByApplicationRecordIDs(
	ctx context.Context,
	applicationRecordIDs []uint64,
) (map[uint64]Snapshot, error) {
	for _, id := range applicationRecordIDs {
		if id == 0 {
			return nil, ErrInvalidInput
		}
	}
	args, err := toDBArgs(applicationRecordIDs)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(
		ctx,
		r.placeholder.rebind(`SELECT
			application_record_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
		FROM application_snapshots
		WHERE application_record_id IN (`+placeholderList(len(args))+`)
		ORDER BY application_record_id ASC`),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list application snapshots: %w", err)
	}
	defer closeRows(rows)

	snapshotMap := make(map[uint64]Snapshot, len(applicationRecordIDs))
	for rows.Next() {
		item, scanErr := scanSnapshot(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan application snapshot: %w", scanErr)
		}
		snapshotMap[item.ApplicationRecordID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application snapshots: %w", err)
	}
	return snapshotMap, nil
}

func (r *SQLRepository) upsertApplication(
	ctx context.Context,
	tx *sql.Tx,
	input ImportApplicationInput,
	now time.Time,
) (uint64, error) {
	var applicationRecordID uint64
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
		r.placeholder.rebind(applicationsUpsertSQL()),
		input.ApplicationID,
		input.DeploymentAdapterKind,
		input.ApplicationName,
		input.WorkspacePath,
		input.DisplayName,
		runtimeTargetID,
		input.ComposeProjectName,
		input.ComposeProjectNameSource,
		input.SourceType,
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
	).Scan(&applicationRecordID)
	if err != nil {
		return 0, mapWriteErr("upsert application", err)
	}
	return applicationRecordID, nil
}

func (r *SQLRepository) createApplication(ctx context.Context, tx *sql.Tx, input ImportApplicationInput, now time.Time) (uint64, error) {
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
	var applicationRecordID uint64
	err = tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO applications (
		application_id, deployment_adapter_kind, application_name, workspace_path, compose_project_name, compose_project_name_source,
		display_name, runtime_target_id, source_type, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status,
		lifecycle_config_json, last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
		created_by, updated_by, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?, ?::jsonb, ?, ?::jsonb, ?, ?, ?, ?, ?, ?)
	RETURNING application_record_id`),
		input.ApplicationID, input.DeploymentAdapterKind, input.ApplicationName, input.WorkspacePath, input.ComposeProjectName, input.ComposeProjectNameSource,
		input.DisplayName, runtimeTargetID, input.SourceType, input.OwnershipMode, string(sourceMetadataJSON), input.LifecycleStrategyKind, input.LifecycleReviewStatus,
		string(lifecycleConfigJSON), input.LastObservedConfigHash, `{}`, input.LastDriftCheckedAt, input.DriftStatus,
		input.ActorID, input.ActorID, now, now,
	).Scan(&applicationRecordID)
	if err != nil {
		return 0, mapWriteErr("create application", err)
	}
	return applicationRecordID, nil
}

// applicationsUpsertSQL 返回用于插入或更新应用记录的 SQL，并通过 RETURNING 返回内部数值键。
func applicationsUpsertSQL() string {
	return `INSERT INTO applications (
			application_id, deployment_adapter_kind, application_name, workspace_path, compose_project_name, compose_project_name_source,
			display_name, runtime_target_id, source_type, ownership_mode, source_metadata_json,
			lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
			last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status,
			created_by, updated_by, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?, ?::jsonb, ?, ?::jsonb, ?, ?, ?, ?, ?, ?
		)
		ON CONFLICT (runtime_target_id, compose_project_name) WHERE deleted_at = 0 DO UPDATE SET
			deployment_adapter_kind = excluded.deployment_adapter_kind,
			display_name = excluded.display_name,
			application_name = COALESCE(excluded.application_name, applications.application_name),
			runtime_target_id = excluded.runtime_target_id,
			compose_project_name_source = excluded.compose_project_name_source,
			source_type = excluded.source_type,
			workspace_path = excluded.workspace_path,
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
		RETURNING application_record_id`
}

func (r *SQLRepository) replaceFiles(
	ctx context.Context,
	tx *sql.Tx,
	applicationRecordID uint64,
	files []ApplicationFile,
	now time.Time,
) error {
	projectDBID, err := toDBID(applicationRecordID)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		r.placeholder.rebind(`DELETE FROM application_files WHERE application_record_id = ?`),
		projectDBID,
	); err != nil {
		return fmt.Errorf("delete application files: %w", err)
	}
	for _, item := range files {
		if _, err := tx.ExecContext(
			ctx,
			r.placeholder.rebind(`INSERT INTO application_files (
				application_record_id, kind, role, absolute_path, display_path, order_index,
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
			return mapWriteErr("insert application file", err)
		}
	}
	return nil
}

func (r *SQLRepository) replaceSnapshot(
	ctx context.Context,
	tx *sql.Tx,
	applicationRecordID uint64,
	snapshot *Snapshot,
) error {
	projectDBID, err := toDBID(applicationRecordID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		if _, err := tx.ExecContext(
			ctx,
			r.placeholder.rebind(`DELETE FROM application_snapshots WHERE application_record_id = ?`),
			projectDBID,
		); err != nil {
			return fmt.Errorf("delete application snapshot: %w", err)
		}
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		r.placeholder.rebind(`INSERT INTO application_snapshots (
			application_record_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
		) VALUES (?, ?::jsonb, ?, ?, ?, ?)
		ON CONFLICT (application_record_id) DO UPDATE SET
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
		return mapWriteErr("upsert application snapshot", err)
	}
	return nil
}

func (r *SQLRepository) applicationExists(ctx context.Context, tx *sql.Tx, applicationRecordID uint64) (bool, error) {
	projectDBID, err := toDBID(applicationRecordID)
	if err != nil {
		return false, err
	}
	var count int
	if err := tx.QueryRowContext(
		ctx,
		r.placeholder.rebind(`SELECT COUNT(*) FROM applications WHERE application_record_id = ? AND deleted_at = 0`),
		projectDBID,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check application existence: %w", err)
	}
	return count > 0, nil
}

func (r *SQLRepository) updateRefreshState(ctx context.Context, tx *sql.Tx, input RefreshApplicationInput) error {
	projectDBID, err := toDBID(input.ApplicationRecordID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		r.placeholder.rebind(`UPDATE applications
		SET last_observed_config_hash = ?,
			last_drift_checked_at = ?,
			drift_status = ?,
			updated_by = ?,
			updated_at = NOW()
		WHERE application_record_id = ? AND deleted_at = 0`),
		input.LastObservedConfigHash,
		input.LastDriftCheckedAt,
		input.DriftStatus,
		input.ActorID,
		projectDBID,
	)
	if err != nil {
		return mapWriteErr("update application refresh state", err)
	}
	return nil
}
