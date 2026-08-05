// Package store 持久化 Build 所有的冻结作业快照与不可变产物。
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

// ErrNotFound 表示不存在与给定 Task 标识对应的 Build 记录。
var ErrNotFound = errors.New("build record not found")

// JobSnapshot 是请求期授权后冻结的 Build 所有执行输入。
type JobSnapshot struct {
	BuildID             string
	SubmissionID        string
	TaskID              uint64
	ApplicationID       string
	ApplicationRecordID uint64
	ApplicationName     string
	WorkspaceRoot       string
	ContextPath         string
	DockerfilePath      string
	RuntimeTargetID     uint64
	RuntimeProvider     string
	ImageRepository     string
	ImageTag            string
	BuildArgs           []moduleapi.DockerImageBuildArg
	RequestedBy         uint64
}

// Artifact 是 Build 作业完成后由 Docker 结果结算的只读产物证据。
type Artifact struct {
	ArtifactID, ImageID, Digest, Repository, Tag, Platform string
	SizeBytes                                              int64
}

// JobProjection 是 API 消费的 Build-owned 快照投影，不包含 Project 或 Container 内部数据。
type JobProjection struct {
	JobSnapshot
	CreatedAt time.Time
	Artifact  *Artifact
}

// ListResult 保存 Build 作业分页投影。
type ListResult struct {
	Items []JobProjection
	Total int64
}

// ListQuery 描述 Build 历史列表支持的冻结快照过滤条件和分页窗口。
type ListQuery struct {
	Limit, Offset               int
	ApplicationID               *string
	ImageRepository, ImageTag   *string
	CreatedAfter, CreatedBefore *time.Time
}

const (
	// DefaultListLimit 是未指定分页窗口时的 Build 历史页大小。
	DefaultListLimit = 20
	// MaxListLimit 是 Build 历史列表允许的最大页大小。
	MaxListLimit      = 100
	pageArgumentCount = 2
	jobListFilterCap  = 5
)

// Repository 是提交与执行器路径使用的窄 Build 持久化边界。
type Repository interface {
	CreateJob(context.Context, JobSnapshot) error
	MaterializeSubmissionSnapshot(context.Context, *sql.Tx, moduleapi.TaskSubmission, JobSnapshot) (string, error)
	GetJobByTaskID(context.Context, uint64) (JobSnapshot, error)
	SettleDockerArtifact(context.Context, uint64, moduleapi.DockerImageBuildResult) error
	ListJobs(context.Context, ListQuery) (ListResult, error)
	GetJobByBuildID(context.Context, string) (JobProjection, error)
}

// MaterializeSubmissionSnapshot 在 Task Runtime 拥有的事务中写入 Build 快照。
// 调用方只能传入已分配 TaskID 的 reserved Submission，事务提交前 worker 无法观察到该 Task。
//
//nolint:cyclop // Idempotent snapshot materialization keeps all validation at the transaction boundary.
func (r *SQLRepository) MaterializeSubmissionSnapshot(ctx context.Context, tx *sql.Tx, submission moduleapi.TaskSubmission, value JobSnapshot) (string, error) {
	if r == nil || tx == nil || submission.ID == "" || submission.TaskID == nil || *submission.TaskID == 0 {
		return "", errors.New("invalid build submission snapshot")
	}
	value.SubmissionID = submission.ID
	value.TaskID = *submission.TaskID
	if !validJobSnapshot(value) {
		return "", errors.New("invalid build submission snapshot")
	}
	jobID, created, err := insertSubmissionJob(ctx, tx, value)
	if err != nil {
		return "", err
	}
	if !created {
		if err := r.verifyExistingJob(ctx, tx, value); err != nil {
			return "", err
		}
		return value.BuildID, nil
	}
	if err := insertBuildArgs(ctx, tx, jobID, value.BuildArgs); err != nil {
		return "", err
	}
	return value.BuildID, nil
}

// ListJobs 读取 Build 域自己的作业与已结算 artifact，不联结 Task、Project 或 Container 内部表。
func (r *SQLRepository) ListJobs(ctx context.Context, query ListQuery) (result ListResult, err error) {
	if r == nil || r.db == nil {
		return result, errors.New("build repository is unavailable")
	}
	query.Limit, query.Offset = normalizedPagination(query.Limit, query.Offset)
	where, args := jobListFilters(query)
	if result.Total, err = r.countJobs(ctx, where, args); err != nil {
		return result, err
	}
	result.Items, err = r.listJobProjections(ctx, where, args, query.Limit, query.Offset)
	return result, err
}

func normalizedPagination(limit, offset int) (int, int) {
	if limit < 1 {
		limit = DefaultListLimit
	} else if limit > MaxListLimit {
		limit = MaxListLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (r *SQLRepository) countJobs(ctx context.Context, where []string, args []any) (int64, error) {
	var total int64
	query := `SELECT COUNT(*) FROM build_jobs j WHERE ` + strings.Join(where, " AND ")
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count build jobs: %w", err)
	}
	return total, nil
}

func (r *SQLRepository) listJobProjections(ctx context.Context, where []string, args []any, limit, offset int) (items []JobProjection, err error) {
	pageArgs := append(append([]any{}, args...), limit, offset)
	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + pageArgumentCount
	// #nosec G202 -- where clauses are assembled only from static column fragments in jobListFilters.
	query := jobProjectionQuery + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY j.created_at DESC, j.id DESC LIMIT $` + strconv.Itoa(limitPlaceholder) + ` OFFSET $` + strconv.Itoa(offsetPlaceholder)
	rows, err := r.db.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("list build jobs: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close build job rows: %w", closeErr)
		}
	}()
	for rows.Next() {
		item, scanErr := scanJobProjection(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate build jobs: %w", err)
	}
	return items, nil
}

func jobListFilters(query ListQuery) ([]string, []any) {
	where := []string{"1 = 1"}
	args := make([]any, 0, jobListFilterCap)
	if query.ApplicationID != nil {
		where = append(where, `j.application_id = $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.ApplicationID)
	}
	if query.ImageRepository != nil {
		where = append(where, `j.image_repository = $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.ImageRepository)
	}
	if query.ImageTag != nil {
		where = append(where, `j.image_tag = $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.ImageTag)
	}
	if query.CreatedAfter != nil {
		where = append(where, `j.created_at >= $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.CreatedAfter)
	}
	if query.CreatedBefore != nil {
		where = append(where, `j.created_at <= $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.CreatedBefore)
	}
	return where, args
}

// GetJobByBuildID 读取一个 Build-owned job projection。
func (r *SQLRepository) GetJobByBuildID(ctx context.Context, buildID string) (JobProjection, error) {
	if r == nil || r.db == nil {
		return JobProjection{}, errors.New("build repository is unavailable")
	}
	row := r.db.QueryRowContext(ctx, jobProjectionQuery+` WHERE j.build_id = $1`, buildID)
	item, err := scanJobProjection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return JobProjection{}, ErrNotFound
	}
	if err != nil {
		return JobProjection{}, err
	}
	item.BuildArgs, err = r.listBuildArgs(ctx, r.db, item.TaskID)
	if err != nil {
		return JobProjection{}, err
	}
	return item, nil
}

const jobProjectionQuery = `SELECT j.build_id, j.task_id, j.application_id, j.application_record_id, j.application_name_snapshot, j.workspace_context_path, j.workspace_root, j.dockerfile_path, j.runtime_target_id, j.runtime_provider, j.image_repository, j.image_tag, COALESCE(j.created_by, 0), j.created_at, a.artifact_id, a.image_id, COALESCE(a.digest, ''), a.repository, a.tag, a.size_bytes, CONCAT_WS('/', NULLIF(a.os, ''), NULLIF(a.architecture, '')) FROM build_jobs j LEFT JOIN build_artifacts a ON a.build_job_id = j.id AND a.role = 'primary'`

type rowScanner interface{ Scan(...any) error }

func scanJobProjection(row rowScanner) (JobProjection, error) {
	var item JobProjection
	var artifactID, imageID, digest, repository, tag, platform sql.NullString
	var size sql.NullInt64
	err := row.Scan(&item.BuildID, &item.TaskID, &item.ApplicationID, &item.ApplicationRecordID, &item.ApplicationName, &item.ContextPath, &item.WorkspaceRoot, &item.DockerfilePath, &item.RuntimeTargetID, &item.RuntimeProvider, &item.ImageRepository, &item.ImageTag, &item.RequestedBy, &item.CreatedAt, &artifactID, &imageID, &digest, &repository, &tag, &size, &platform)
	if err != nil {
		return JobProjection{}, err
	}
	if artifactID.Valid {
		item.Artifact = &Artifact{ArtifactID: artifactID.String, ImageID: imageID.String, Digest: digest.String, Repository: repository.String, Tag: tag.String, SizeBytes: size.Int64, Platform: platform.String}
	}
	return item, nil
}

// SQLRepository 持久化 Build 事实而不拥有 Task 执行状态。
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository 从平台 SQL 连接创建 Build 仓储。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("build repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db}, nil
}

// CreateJob 在 Task 分配稳定任务标识后存储 Build 快照。
func (r *SQLRepository) CreateJob(ctx context.Context, value JobSnapshot) error {
	if r == nil || r.db == nil || !validJobSnapshot(value) {
		return errors.New("invalid build job snapshot")
	}
	for attempt := 0; attempt < 2; attempt++ {
		err, retry := r.createJobAttempt(ctx, value)
		if !retry || err == nil {
			return err
		}
	}
	return ErrNotFound
}

func (r *SQLRepository) createJobAttempt(ctx context.Context, value JobSnapshot) (error, bool) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin build job: %w", err), false
	}
	jobID, created, err := insertJob(ctx, tx, value)
	if err != nil {
		_ = tx.Rollback()
		return err, false
	}
	if !created {
		err = r.verifyExistingJob(ctx, tx, value)
		_ = tx.Rollback()
		return err, errors.Is(err, ErrNotFound)
	}
	if err := insertBuildArgs(ctx, tx, jobID, value.BuildArgs); err != nil {
		_ = tx.Rollback()
		return err, false
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit build job: %w", err), false
	}
	return nil, false
}

func validJobSnapshot(value JobSnapshot) bool {
	return value.TaskID != 0 && value.BuildID != "" && value.WorkspaceRoot != ""
}

func insertSubmissionJob(ctx context.Context, tx *sql.Tx, value JobSnapshot) (uint64, bool, error) {
	var jobID uint64
	var created bool
	err := tx.QueryRowContext(ctx, `INSERT INTO build_jobs (build_id, submission_id, task_id, application_id, application_record_id, application_name_snapshot, workspace_context_path, workspace_root, dockerfile_path, runtime_target_id, runtime_provider, executor_kind, image_repository, image_tag, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15, 0))
		ON CONFLICT (submission_id) WHERE submission_id IS NOT NULL DO UPDATE SET task_id = EXCLUDED.task_id
		RETURNING id, (xmax = 0)`, value.BuildID, value.SubmissionID, value.TaskID, value.ApplicationID, value.ApplicationRecordID, value.ApplicationName, value.ContextPath, value.WorkspaceRoot, value.DockerfilePath, value.RuntimeTargetID, value.RuntimeProvider, "dockerfile", value.ImageRepository, value.ImageTag, value.RequestedBy).Scan(&jobID, &created)
	if err != nil {
		return 0, false, fmt.Errorf("insert build submission snapshot: %w", err)
	}
	return jobID, created, nil
}

func insertJob(ctx context.Context, tx *sql.Tx, value JobSnapshot) (uint64, bool, error) {
	var jobID uint64
	var created bool
	err := tx.QueryRowContext(ctx, `INSERT INTO build_jobs (build_id, task_id, application_id, application_record_id, application_name_snapshot, workspace_context_path, workspace_root, dockerfile_path, runtime_target_id, runtime_provider, executor_kind, image_repository, image_tag, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14, 0))
		ON CONFLICT (task_id) DO UPDATE SET task_id = EXCLUDED.task_id
		RETURNING id, (xmax = 0)`, value.BuildID, value.TaskID, value.ApplicationID, value.ApplicationRecordID, value.ApplicationName, value.ContextPath, value.WorkspaceRoot, value.DockerfilePath, value.RuntimeTargetID, value.RuntimeProvider, "dockerfile", value.ImageRepository, value.ImageTag, value.RequestedBy).Scan(&jobID, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, errors.New("insert build job returned no row")
	}
	if err != nil {
		return 0, false, fmt.Errorf("insert build job: %w", err)
	}
	return jobID, created, nil
}

func (r *SQLRepository) verifyExistingJob(ctx context.Context, query jobQueryer, value JobSnapshot) error {
	existing, err := getJobByTaskID(ctx, query, value.TaskID)
	if err != nil {
		return err
	}
	if existing.ApplicationID != value.ApplicationID || existing.ApplicationRecordID != value.ApplicationRecordID || existing.WorkspaceRoot != value.WorkspaceRoot || existing.ContextPath != value.ContextPath || existing.DockerfilePath != value.DockerfilePath || existing.ImageRepository != value.ImageRepository || existing.ImageTag != value.ImageTag {
		return errors.New("build task snapshot conflict")
	}
	return nil
}

func insertBuildArgs(ctx context.Context, tx *sql.Tx, jobID uint64, args []moduleapi.DockerImageBuildArg) error {
	for _, arg := range args {
		if _, err := tx.ExecContext(ctx, `INSERT INTO build_job_args (build_job_id, name, value) VALUES ($1,$2,$3)`, jobID, arg.Name, arg.Value); err != nil {
			return fmt.Errorf("insert build argument: %w", err)
		}
	}
	return nil
}

// GetJobByTaskID 为后台执行只读取 Build 所有的冻结数据。
func (r *SQLRepository) GetJobByTaskID(ctx context.Context, taskID uint64) (JobSnapshot, error) {
	if r == nil || r.db == nil {
		return JobSnapshot{}, errors.New("build repository is unavailable")
	}
	return getJobByTaskID(ctx, r.db, taskID)
}

type jobQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func getJobByTaskID(ctx context.Context, query jobQueryer, taskID uint64) (JobSnapshot, error) {
	var value JobSnapshot
	err := query.QueryRowContext(ctx, `SELECT build_id, task_id, application_id, application_record_id, application_name_snapshot, workspace_context_path, workspace_root, dockerfile_path, runtime_target_id, runtime_provider, image_repository, image_tag, COALESCE(created_by, 0)
		FROM build_jobs WHERE task_id = $1`, taskID).Scan(&value.BuildID, &value.TaskID, &value.ApplicationID, &value.ApplicationRecordID, &value.ApplicationName, &value.ContextPath, &value.WorkspaceRoot, &value.DockerfilePath, &value.RuntimeTargetID, &value.RuntimeProvider, &value.ImageRepository, &value.ImageTag, &value.RequestedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return JobSnapshot{}, ErrNotFound
	}
	if err != nil {
		return JobSnapshot{}, fmt.Errorf("get build job: %w", err)
	}
	args, err := listBuildArgs(ctx, query, taskID)
	if err != nil {
		return JobSnapshot{}, err
	}
	value.BuildArgs = args
	return value, nil
}

func (r *SQLRepository) listBuildArgs(ctx context.Context, query jobQueryer, taskID uint64) ([]moduleapi.DockerImageBuildArg, error) {
	return listBuildArgs(ctx, query, taskID)
}

func listBuildArgs(ctx context.Context, query jobQueryer, taskID uint64) (args []moduleapi.DockerImageBuildArg, err error) {
	rows, err := query.QueryContext(ctx, `SELECT name, value FROM build_job_args WHERE build_job_id = (SELECT id FROM build_jobs WHERE task_id = $1) ORDER BY id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list build arguments: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close build argument rows: %w", closeErr)
		}
	}()
	for rows.Next() {
		var arg moduleapi.DockerImageBuildArg
		if err := rows.Scan(&arg.Name, &arg.Value); err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return args, nil
}

// SettleDockerArtifact 为 Build 作业写入或更新唯一主 Docker 产物。
func (r *SQLRepository) SettleDockerArtifact(ctx context.Context, taskID uint64, result moduleapi.DockerImageBuildResult) error {
	if r == nil || r.db == nil || result.ImageID == "" {
		return errors.New("docker build result has no image id")
	}
	resultInfo, err := r.db.ExecContext(ctx, `INSERT INTO build_artifacts (artifact_id, build_job_id, role, artifact_type, media_type, runtime_provider, runtime_target_id, image_id, digest, repository, tag, size_bytes, os, architecture, variant, producer_version)
		SELECT 'artifact-' || task_id::text, id, 'primary', 'container_image', 'application/vnd.oci.image.manifest.v1+json', runtime_provider, runtime_target_id, $2, $3, $4, $5, $6, $7, $8, $9, 'docker'
		FROM build_jobs WHERE task_id = $1
		ON CONFLICT (build_job_id, role) DO UPDATE SET image_id=EXCLUDED.image_id, digest=EXCLUDED.digest, repository=EXCLUDED.repository, tag=EXCLUDED.tag, size_bytes=EXCLUDED.size_bytes, os=EXCLUDED.os, architecture=EXCLUDED.architecture, variant=EXCLUDED.variant`, taskID, result.ImageID, result.Digest, result.Repository, result.Tag, result.SizeBytes, result.OS, result.Architecture, result.Variant)
	if err != nil {
		return fmt.Errorf("settle build artifact: %w", err)
	}
	rowsAffected, err := resultInfo.RowsAffected()
	if err != nil {
		return fmt.Errorf("count settled build artifacts: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
