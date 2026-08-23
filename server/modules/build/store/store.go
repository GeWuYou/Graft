// Package store 持久化 Build 所有的冻结作业快照与不可变产物。
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"graft/server/internal/moduleapi"
)

// ErrNotFound 表示不存在与给定 Task 标识对应的 Build 记录。
var ErrNotFound = errors.New("build record not found")

// ErrConflict 表示并发写入在有限重试内未能收敛。
var ErrConflict = errors.New("build record write conflict")

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
	RuntimeTargetName   string
	RuntimeProvider     string
	ImageRepository     string
	ImageTag            string
	BuildArgs           []moduleapi.BuildArgument
	RequestedBy         uint64
	InputSnapshotID     string
	InputSnapshotDigest string
	SourceKind          string
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
	Execution TaskExecution
}

// TaskExecution 是 Build 列表消费的 Task Runtime 执行投影；状态事实仍归 Task Runtime 所有。
type TaskExecution struct {
	Status              moduleapi.TaskStatus
	CurrentStageKey     *string
	CompletedStageCount int
	StageCount          int
	DurationMS          *int64
	FailureCode         *string
	FailureMessage      *string
	RecoveryReason      *string
	Capabilities        moduleapi.TaskCapabilities
}

// ListResult 保存 Build 作业分页投影。
type ListResult struct {
	Items []JobProjection
	Total int64
}

// WorkspaceListResult 保存调用者可见 Workspace 的分页投影。
type WorkspaceListResult struct {
	Items []moduleapi.BuildWorkspace
	Total int64
}

// WorkspaceListQuery 描述 Workspace 列表的分页窗口和可选搜索词。
type WorkspaceListQuery struct {
	Limit, Offset int
	Search        *string
}

// V2ArtifactProjection 是独立于 Build Job 的不可变 Artifact 读取投影。
// Publication 是可变引用，不改变 Artifact 摘要身份。
type V2ArtifactProjection struct {
	ArtifactID string
	Digest     string
	MediaType  string
	Platforms  []string
	SizeBytes  int64
	CreatedAt  time.Time
}

// V2ArtifactListResult 是 Artifact 读取的分页投影。
type V2ArtifactListResult struct {
	Items []V2ArtifactProjection
	Total int64
}

// ListQuery 描述 Build 历史列表支持的冻结快照过滤条件和分页窗口。
type ListQuery struct {
	Limit, Offset               int
	ApplicationID               *string
	ImageRepository, ImageTag   *string
	Search                      *string
	BuildStatus                 *StatusFilter
	BuilderID                   *uint64
	CreatedAfter, CreatedBefore *time.Time
}

// StatusFilter 是 Build Task Center 对 Task Runtime 状态的稳定产品级归类。
type StatusFilter string

const (
	// StatusFilterQueued 覆盖尚未由 worker 执行的全部排队态。
	StatusFilterQueued StatusFilter = "queued"
	// StatusFilterRunning 表示正在执行的构建任务。
	StatusFilterRunning StatusFilter = "running"
	// StatusFilterSuccess 表示已成功完成的构建任务。
	StatusFilterSuccess StatusFilter = "success"
	// StatusFilterFailed 覆盖失败和需要人工处置的构建任务。
	StatusFilterFailed StatusFilter = "failed"
	// StatusFilterCancelled 表示已取消的构建任务。
	StatusFilterCancelled StatusFilter = "cancelled"
)

// MatchesTaskStatus 判断 Task Runtime 状态是否属于当前 Build 产品状态。
func (s StatusFilter) MatchesTaskStatus(status moduleapi.TaskStatus) bool {
	for _, candidate := range s.taskStatuses() {
		if status == candidate {
			return true
		}
	}
	return false
}

func (s StatusFilter) taskStatuses() []moduleapi.TaskStatus {
	switch s {
	case StatusFilterQueued:
		return []moduleapi.TaskStatus{moduleapi.TaskStatusPending, moduleapi.TaskStatusReady, moduleapi.TaskStatusScheduled}
	case StatusFilterRunning:
		return []moduleapi.TaskStatus{moduleapi.TaskStatusRunning}
	case StatusFilterSuccess:
		return []moduleapi.TaskStatus{moduleapi.TaskStatusSuccess}
	case StatusFilterFailed:
		return []moduleapi.TaskStatus{moduleapi.TaskStatusFailed, moduleapi.TaskStatusNeedsAttention}
	case StatusFilterCancelled:
		return []moduleapi.TaskStatus{moduleapi.TaskStatusCancelled}
	default:
		return nil
	}
}

const (
	// DefaultListLimit 是未指定分页窗口时的 Build 历史页大小。
	DefaultListLimit = 20
	// MaxListLimit 是 Build 历史列表允许的最大页大小。
	MaxListLimit                 = 100
	pageArgumentCount            = 2
	jobListFilterCap             = 13
	artifactIdentityPrefixLength = 24
)

// Repository 是提交与执行器路径使用的窄 Build 持久化边界。
type Repository interface {
	CreateJob(context.Context, JobSnapshot) error
	MaterializeSubmissionSnapshot(context.Context, *sql.Tx, moduleapi.TaskSubmission, JobSnapshot) (string, error)
	GetJobByTaskID(context.Context, uint64) (JobSnapshot, error)
	SettleBuildArtifact(context.Context, uint64, moduleapi.BuildArtifactResult) error
	ListJobs(context.Context, ListQuery) (ListResult, error)
	GetJobByBuildID(context.Context, string) (JobProjection, error)
}

// ExecutionPlanRepository 是 v2 write boundary；它刻意与 legacy job projection
// 分离，避免 v2 submission 回退到 Docker-first persistence。
type ExecutionPlanRepository interface {
	MaterializeExecutionPlan(context.Context, *sql.Tx, moduleapi.TaskSubmission, moduleapi.BuildExecutionPlan, uint64) (string, error)
}

// InputSnapshotRepository 持久化 Build-owned 上传输入；它复用既有
// build_workspace_snapshots 表，不创建第二套 Snapshot/Blob 存储。
type InputSnapshotRepository interface {
	CreateBuildInputSnapshot(context.Context, moduleapi.WorkspaceSnapshot, uint64) (moduleapi.WorkspaceSnapshot, error)
	GetBuildInputSnapshot(context.Context, string) (moduleapi.WorkspaceSnapshot, error)
}

// InputSnapshotListResult 保存调用者可复用的 Build 输入快照分页投影。
type InputSnapshotListResult struct {
	Items []moduleapi.WorkspaceSnapshot
	Total int64
}

// InputSnapshotReader 暴露按所有权过滤的输入快照查询边界。
type InputSnapshotReader interface {
	ListBuildInputSnapshots(context.Context, uint64, int, int) (InputSnapshotListResult, error)
}

// BuilderReservationRepository 暴露 Build 容量租约的事务化写入与终态更新。
type BuilderReservationRepository interface {
	moduleapi.BuilderReservationRepository
}

// BuilderReservationLeaseTTL 限制尚未进入运行态的容量租约。运行态 lease 由 fencing
// 和显式释放控制，不通过该超时回收。
const BuilderReservationLeaseTTL = 5 * time.Minute

// WorkspaceRepository 是 Build-owned Workspace 定义的持久化边界；来源适配器只
// 负责提供授权输入，不能直接写入 Build 表或改变已冻结 Snapshot。
type WorkspaceRepository interface {
	CreateWorkspace(context.Context, moduleapi.BuildWorkspace) error
	GetWorkspace(context.Context, string) (moduleapi.BuildWorkspace, error)
}

// ExecutionPlanReader 是 frozen plan 的 execution-time read authority。
type ExecutionPlanReader interface {
	GetExecutionPlanByTaskID(context.Context, uint64) (moduleapi.BuildExecutionPlan, error)
}

// V2JobReader projects new Build jobs solely from execution plans, Tasks and
// immutable v2 artifacts; legacy build_jobs remains historical read-only evidence.
type V2JobReader interface {
	ListV2Jobs(context.Context, ListQuery) (ListResult, error)
	GetV2JobByBuildID(context.Context, string) (JobProjection, error)
}

// V2ArtifactSettlementRepository 在 target executor 完成两项动作后记录 immutable
// artifact evidence 及其 publication reference。
type V2ArtifactSettlementRepository interface {
	SettleV2Artifact(context.Context, uint64, moduleapi.BuildExecutionPlan, moduleapi.BuildArtifactResult, moduleapi.RegistryAuthExecution) error
}

// V2ArtifactReader 为 Artifact、Promotion 和 Deployment 提供 Build-owned 的摘要读取边界。
type V2ArtifactReader interface {
	ListV2Artifacts(context.Context, int, int) (V2ArtifactListResult, error)
}

// ArtifactPublicationReader 为摘要寻址 Artifact 解析 Build-owned 的可变 Publication 记录。
// Registry 会在 copy 前重新授权两个仓库；此 reader 从不解析 endpoint 或 credential。
type ArtifactPublicationReader interface {
	ListArtifactPublicationSources(context.Context, string) ([]moduleapi.ArtifactPublicationSource, error)
}

// ArtifactPromotionSettlementRepository 记录 provider 已完成的 digest-preserving promotion。
// 它只接受 Build 已解析的不可变 source 和 Registry 已授权的 destination，不改变 Artifact 身份。
type ArtifactPromotionSettlementRepository interface {
	SettleArtifactPromotion(context.Context, moduleapi.OCIArtifactCopyInput, moduleapi.OCIArtifactCopyResult, moduleapi.RegistryAuthExecution) error
}

// ProviderExecutionEvidenceRepository 记录执行前已验证的 Runtime provider 资格；相同阶段只能重放相同证据。
type ProviderExecutionEvidenceRepository interface {
	RecordProviderExecutionEvidence(context.Context, moduleapi.BuildExecutionPlan, moduleapi.ProviderExecutionEvidence) error
}

// PlatformArtifactRepository 记录协调任务的每平台 Artifact；它不创建 mutable Publication，也不执行 Manifest 合并。
type PlatformArtifactRepository interface {
	RecordPlatformArtifact(context.Context, uint64, moduleapi.BuildExecutionPlan, moduleapi.PlatformArtifact) error
	ListPlatformArtifacts(context.Context, uint64, moduleapi.BuildExecutionPlan) ([]moduleapi.PlatformArtifact, error)
	PrepareOCIManifestPublication(context.Context, uint64, moduleapi.BuildExecutionPlan) (moduleapi.OCIManifestPublicationInput, error)
}

// OCIManifestSettlementRepository 持久化 Driver 已发布的最终 Manifest Artifact 与 mutable Publication。
type OCIManifestSettlementRepository interface {
	SettleOCIManifestPublication(context.Context, uint64, moduleapi.BuildExecutionPlan, moduleapi.OCIManifestPublicationResult, moduleapi.RegistryAuthExecution) error
}

// ExpiredSnapshotMaterialization 是已由 Build 领取清理的私有物化引用。Snapshot identity
// 不在这里删除；调用方只可在验证私有路径后清理其字节内容。
type ExpiredSnapshotMaterialization struct {
	SnapshotID         string
	MaterializationRef string
}

// SnapshotMaterializationRetentionRepository 维护 Build-owned 快照物化的清理租约和终态。
// 它不允许调用方删除 Snapshot、Execution Plan 或 Artifact 审计记录。
type SnapshotMaterializationRetentionRepository interface {
	ClaimExpiredSnapshotMaterializations(context.Context, time.Time, time.Time, int) ([]ExpiredSnapshotMaterialization, error)
	MarkSnapshotMaterializationPurged(context.Context, string) error
	ReleaseSnapshotMaterializationClaim(context.Context, string) error
}

// ClaimExpiredSnapshotMaterializations 以短租约领取已经到期的 Build-owned 物化内容。
// `purging` 的过期租约会回收，以避免进程中断永久阻塞清理；Snapshot 和计划身份保持不变。
func (r *SQLRepository) ClaimExpiredSnapshotMaterializations(ctx context.Context, now, claimBefore time.Time, limit int) ([]ExpiredSnapshotMaterialization, error) {
	if !validSnapshotMaterializationClaim(r, now, claimBefore, limit) {
		return nil, errors.New("invalid snapshot materialization cleanup claim")
	}
	rows, err := r.db.QueryContext(ctx, `WITH candidates AS (
		SELECT id FROM build_workspace_snapshots
		WHERE materialization_owner = 'build'
		  AND retention_expires_at IS NOT NULL
		  AND retention_expires_at <= $1
		  AND (materialization_state IN ('available', 'expired') OR (materialization_state = 'purging' AND materialization_claimed_at <= $2))
		ORDER BY retention_expires_at ASC, id ASC
		LIMIT $3
		FOR UPDATE SKIP LOCKED
	)
	UPDATE build_workspace_snapshots snapshot
	SET materialization_state = 'purging', materialization_claimed_at = $1
	FROM candidates
	WHERE snapshot.id = candidates.id
	RETURNING snapshot.snapshot_id, snapshot.materialization_ref`, now.UTC(), claimBefore.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("claim expired snapshot materializations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanExpiredSnapshotMaterializations(rows, limit)
}

func validSnapshotMaterializationClaim(repository *SQLRepository, now, claimBefore time.Time, limit int) bool {
	return repository != nil && repository.db != nil && !now.IsZero() && !claimBefore.IsZero() && !claimBefore.After(now) && limit > 0
}

func scanExpiredSnapshotMaterializations(rows *sql.Rows, limit int) ([]ExpiredSnapshotMaterialization, error) {
	items := make([]ExpiredSnapshotMaterialization, 0, limit)
	for rows.Next() {
		var item ExpiredSnapshotMaterialization
		if err := rows.Scan(&item.SnapshotID, &item.MaterializationRef); err != nil {
			return nil, fmt.Errorf("scan expired snapshot materialization: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired snapshot materializations: %w", err)
	}
	return items, nil
}

// MarkSnapshotMaterializationPurged 将已经删除私有物化字节的 Snapshot 标记为 purged。
// 物化引用被清空，避免后续执行器或投影误把已经删除的路径当作可用内容。
func (r *SQLRepository) MarkSnapshotMaterializationPurged(ctx context.Context, snapshotID string) error {
	if r == nil || r.db == nil || strings.TrimSpace(snapshotID) == "" {
		return ErrNotFound
	}
	result, err := r.db.ExecContext(ctx, `UPDATE build_workspace_snapshots
	SET materialization_state = 'purged', materialization_ref = '', materialization_claimed_at = NULL
	WHERE snapshot_id = $1 AND materialization_owner = 'build' AND materialization_state = 'purging'`, strings.TrimSpace(snapshotID))
	if err != nil {
		return fmt.Errorf("mark snapshot materialization purged: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count purged snapshot materialization: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// ReleaseSnapshotMaterializationClaim 将未完成清理的 lease 退回 expired，使其可在下一次清理中重试。
func (r *SQLRepository) ReleaseSnapshotMaterializationClaim(ctx context.Context, snapshotID string) error {
	if r == nil || r.db == nil || strings.TrimSpace(snapshotID) == "" {
		return ErrNotFound
	}
	result, err := r.db.ExecContext(ctx, `UPDATE build_workspace_snapshots
	SET materialization_state = 'expired', materialization_claimed_at = NULL
	WHERE snapshot_id = $1 AND materialization_owner = 'build' AND materialization_state = 'purging'`, strings.TrimSpace(snapshotID))
	if err != nil {
		return fmt.Errorf("release snapshot materialization claim: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count released snapshot materialization claim: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// MaterializeExecutionPlan 在创建 Task 的 Task Runtime transaction 中持久化
// immutable workspace 与 plan。
//
//nolint:cyclop // Plan、Snapshot 与容量 lease 必须在同一 Task materialization transaction 内保持原子。
func (r *SQLRepository) MaterializeExecutionPlan(ctx context.Context, tx *sql.Tx, submission moduleapi.TaskSubmission, plan moduleapi.BuildExecutionPlan, requestedBy uint64) (string, error) {
	if r == nil || tx == nil || submission.TaskID == nil || *submission.TaskID == 0 || !validExecutionPlan(plan) {
		return "", errors.New("invalid build execution plan")
	}
	snapshotPK, err := materializeWorkspaceSnapshot(ctx, tx, plan.Workspace, requestedBy)
	if err != nil {
		return "", err
	}
	encodedPlan, err := marshalExecutionPlanFields(plan)
	if err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO build_execution_plans (plan_id, plan_digest, workspace_snapshot_id, task_id, builder_pool_id, builder_instance_id, runtime_target_id, driver, template_ref, cache_policy, security_policy, platforms_json, builder_placements_json, destination_json, created_by)
VALUES ($1,$2,$3,$4,NULLIF($5, ''),NULLIF($6, ''),$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (plan_id) DO UPDATE SET plan_id = EXCLUDED.plan_id`, plan.ID, plan.Digest, snapshotPK, *submission.TaskID, plan.BuilderPoolID, plan.BuilderInstanceID, plan.RuntimeTargetID, plan.Driver, plan.TemplateRef, plan.CachePolicy, plan.SecurityPolicy, encodedPlan.platforms, encodedPlan.placements, encodedPlan.destination, nullableUint64(requestedBy)); err != nil {
		return "", fmt.Errorf("materialize execution plan: %w", err)
	}
	placements := plan.BuilderPlacements
	if len(placements) == 0 {
		return "", errors.New("execution plan has no frozen builder placements")
	}
	now := time.Now().UTC()
	for _, placement := range placements {
		legID := placement.Platform
		slotBudget, observedAt, slotBudgetErr := placementReservationCapacity(placement)
		if slotBudgetErr != nil {
			return "", slotBudgetErr
		}
		if _, err := r.reserveBuilderCapacity(ctx, tx, moduleapi.BuilderReservation{
			ID:             fmt.Sprintf("reservation_%s_%s", plan.ID, legID),
			InstanceID:     placement.BuilderInstanceID,
			PlanID:         plan.ID,
			TaskID:         *submission.TaskID,
			Attempt:        1,
			LegID:          legID,
			FenceToken:     BuilderReservationFence(plan.ID, *submission.TaskID, legID, 1),
			State:          moduleapi.BuilderReservationAccepted,
			LeaseExpiresAt: now.Add(BuilderReservationLeaseTTL),
			CreatedAt:      now,
			UpdatedAt:      now,
		}, slotBudget, observedAt); err != nil {
			return "", err
		}
	}
	return plan.ID, nil
}

// placementReservationSlotBudget 读取 Build Placement Policy 已冻结的 slot 预算；不得以 Runtime Target 默认值替代。
func placementReservationCapacity(placement moduleapi.BuilderPlacement) (int, time.Time, error) {
	var evidence struct {
		ReservationSlotBudget int       `json:"reservation_slot_budget"`
		ReservationObservedAt time.Time `json:"reservation_observed_at"`
	}
	if len(placement.SchedulingEvidence) == 0 || json.Unmarshal(placement.SchedulingEvidence, &evidence) != nil || evidence.ReservationSlotBudget < 1 {
		return 0, time.Time{}, errors.New("builder placement has no frozen reservation slot budget")
	}
	if isDynamicSchedulingPolicy(placement.SchedulingPolicy) && evidence.ReservationObservedAt.IsZero() {
		return 0, time.Time{}, errors.New("dynamic builder placement has no frozen telemetry observation time")
	}
	return evidence.ReservationSlotBudget, evidence.ReservationObservedAt.UTC(), nil
}

// BuilderReservationFence 生成与冻结计划绑定的稳定 fencing token。
func BuilderReservationFence(planID string, taskID uint64, legID string, attempt int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s:%d", planID, taskID, legID, attempt)))
	return hex.EncodeToString(sum[:])
}

// ReserveBuilder 在 Task materialization transaction 内取得唯一的 Builder 容量 lease。
//
//nolint:cyclop,gocyclo // 唯一性、per-leg fence replay 与数据库错误必须在容量 owner 边界集中校验。
func (r *SQLRepository) ReserveBuilder(ctx context.Context, tx *sql.Tx, reservation moduleapi.BuilderReservation) (moduleapi.BuilderReservation, error) {
	return r.reserveBuilderCapacity(ctx, tx, reservation, 1, time.Time{})
}

// reserveBuilderCapacity 对冻结的 Builder slot 预算原子分配一个 Build capacity unit。
// 动态 Placement 仅统计遥测观察之后创建的 live lease，避免重复扣减 provider 已计入的运行容量。
//
//nolint:cyclop,gocyclo // 容量裁决必须在同一事务中保持锁、过期回收、预算核对和 fencing 写入的顺序。
func (r *SQLRepository) reserveBuilderCapacity(ctx context.Context, tx *sql.Tx, reservation moduleapi.BuilderReservation, slotBudget int, observedAt time.Time) (moduleapi.BuilderReservation, error) {
	if r == nil || tx == nil || strings.TrimSpace(reservation.ID) == "" || strings.TrimSpace(reservation.InstanceID) == "" || strings.TrimSpace(reservation.PlanID) == "" || reservation.TaskID == 0 || strings.TrimSpace(reservation.LegID) == "" || strings.TrimSpace(reservation.FenceToken) == "" || reservation.LeaseExpiresAt.IsZero() {
		return moduleapi.BuilderReservation{}, errors.New("invalid builder reservation")
	}
	if slotBudget < 1 {
		return moduleapi.BuilderReservation{}, errors.New("builder reservation slot budget is invalid")
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, reservation.InstanceID); err != nil {
		return moduleapi.BuilderReservation{}, fmt.Errorf("lock builder capacity: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE build_builder_reservations SET state = 'expired', updated_at = NOW() WHERE builder_instance_id = $1 AND state IN ('reserved','accepted') AND lease_expires_at <= NOW()`, reservation.InstanceID); err != nil {
		return moduleapi.BuilderReservation{}, fmt.Errorf("expire builder reservation: %w", err)
	}
	var existing moduleapi.BuilderReservation
	err := tx.QueryRowContext(ctx, `SELECT reservation_id, builder_instance_id, plan_id, task_id, attempt, leg_id, fence_token, state, lease_expires_at, created_at, updated_at
FROM build_builder_reservations
WHERE plan_id = $1 AND task_id = $2 AND attempt = $3 AND leg_id = $4`, reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID).Scan(&existing.ID, &existing.InstanceID, &existing.PlanID, &existing.TaskID, &existing.Attempt, &existing.LegID, &existing.FenceToken, &existing.State, &existing.LeaseExpiresAt, &existing.CreatedAt, &existing.UpdatedAt)
	if err == nil {
		if existing.InstanceID != reservation.InstanceID || existing.FenceToken != reservation.FenceToken || existing.State != reservation.State {
			return moduleapi.BuilderReservation{}, ErrConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return moduleapi.BuilderReservation{}, fmt.Errorf("read existing builder reservation: %w", err)
	}
	var usedUnits int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(capacity_units), 0) FROM build_builder_reservations WHERE builder_instance_id = $1 AND state IN ('reserved','accepted','running') AND ($2::timestamptz IS NULL OR created_at > $2)`, reservation.InstanceID, nullableObservationTime(observedAt)).Scan(&usedUnits); err != nil {
		return moduleapi.BuilderReservation{}, fmt.Errorf("read builder reserved capacity: %w", err)
	}
	if usedUnits+1 > slotBudget {
		return moduleapi.BuilderReservation{}, ErrConflict
	}
	var stored moduleapi.BuilderReservation
	err = tx.QueryRowContext(ctx, `INSERT INTO build_builder_reservations (reservation_id, builder_instance_id, plan_id, task_id, attempt, leg_id, fence_token, state, lease_expires_at, capacity_units, slot_budget)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,1,$10)
ON CONFLICT (plan_id, task_id, attempt, leg_id) DO UPDATE SET reservation_id = build_builder_reservations.reservation_id
RETURNING reservation_id, builder_instance_id, plan_id, task_id, attempt, leg_id, fence_token, state, lease_expires_at, created_at, updated_at`, reservation.ID, reservation.InstanceID, reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID, reservation.FenceToken, reservation.State, reservation.LeaseExpiresAt, slotBudget).Scan(&stored.ID, &stored.InstanceID, &stored.PlanID, &stored.TaskID, &stored.Attempt, &stored.LegID, &stored.FenceToken, &stored.State, &stored.LeaseExpiresAt, &stored.CreatedAt, &stored.UpdatedAt)
	if err != nil {
		return moduleapi.BuilderReservation{}, fmt.Errorf("reserve builder capacity: %w", err)
	}
	if stored.InstanceID != reservation.InstanceID || stored.LegID != reservation.LegID || stored.FenceToken != reservation.FenceToken || stored.State != reservation.State {
		return moduleapi.BuilderReservation{}, ErrConflict
	}
	return stored, nil
}

// ReserveBuilderAttempt 在旧尝试已经进入终态后取得新的容量 lease 与 fence。
func (r *SQLRepository) ReserveBuilderAttempt(ctx context.Context, reservation moduleapi.BuilderReservation) (moduleapi.BuilderReservation, error) {
	return r.ReserveBuilderAttemptWithCapacity(ctx, reservation, 1)
}

// ReserveBuilderAttemptWithCapacity 使用原始 Placement 冻结的 slot 预算取得新的 attempt-scoped lease。
func (r *SQLRepository) ReserveBuilderAttemptWithCapacity(ctx context.Context, reservation moduleapi.BuilderReservation, slotBudget int) (moduleapi.BuilderReservation, error) {
	return r.ReserveBuilderAttemptWithCapacityAfterObservation(ctx, reservation, slotBudget, time.Time{})
}

// ReserveBuilderAttemptWithCapacityAfterObservation 使用冻结遥测观察之后的 live lease 取得 retry capacity unit。
func (r *SQLRepository) ReserveBuilderAttemptWithCapacityAfterObservation(ctx context.Context, reservation moduleapi.BuilderReservation, slotBudget int, observedAt time.Time) (moduleapi.BuilderReservation, error) {
	if r == nil || r.db == nil || reservation.Attempt < 2 {
		return moduleapi.BuilderReservation{}, errors.New("invalid builder retry reservation")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return moduleapi.BuilderReservation{}, fmt.Errorf("begin builder retry reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE build_builder_reservations SET state = 'abandoned', updated_at = NOW() WHERE task_id = $1 AND leg_id = $2 AND attempt < $3 AND state IN ('accepted','running')`, reservation.TaskID, reservation.LegID, reservation.Attempt); err != nil {
		return moduleapi.BuilderReservation{}, fmt.Errorf("abandon prior builder reservation: %w", err)
	}
	stored, err := r.reserveBuilderCapacity(ctx, tx, reservation, slotBudget, observedAt)
	if err != nil {
		return moduleapi.BuilderReservation{}, err
	}
	if err := tx.Commit(); err != nil {
		return moduleapi.BuilderReservation{}, fmt.Errorf("commit builder retry reservation: %w", err)
	}
	return stored, nil
}

func nullableObservationTime(observedAt time.Time) any {
	if observedAt.IsZero() {
		return nil
	}
	return observedAt.UTC()
}

func isDynamicSchedulingPolicy(policy string) bool {
	return policy == "least_load" || policy == "capacity" || policy == "affinity"
}

// MarkBuilderReservationRunning 只允许 accepted lease 进入执行中，旧 fence 会被拒绝。
func (r *SQLRepository) MarkBuilderReservationRunning(ctx context.Context, taskID uint64, legID, fenceToken string) error {
	if r == nil || r.db == nil || taskID == 0 || strings.TrimSpace(legID) == "" || strings.TrimSpace(fenceToken) == "" {
		return errors.New("invalid builder reservation transition")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE build_builder_reservations SET state = 'running', updated_at = NOW() WHERE task_id = $1 AND leg_id = $2 AND fence_token = $3 AND state = 'accepted'`, taskID, legID, fenceToken)
	if err != nil {
		return fmt.Errorf("start builder reservation: %w", err)
	}
	return requireReservationUpdate(result)
}

// RenewBuilderReservation extends a running lease only when the current leg still owns its fence.
func (r *SQLRepository) RenewBuilderReservation(ctx context.Context, taskID uint64, legID, fenceToken string, leaseExpiresAt time.Time) error {
	if r == nil || r.db == nil || taskID == 0 || strings.TrimSpace(legID) == "" || strings.TrimSpace(fenceToken) == "" || !leaseExpiresAt.After(time.Now().UTC()) {
		return errors.New("invalid builder reservation renewal")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE build_builder_reservations SET lease_expires_at = $4, updated_at = NOW() WHERE task_id = $1 AND leg_id = $2 AND fence_token = $3 AND state = 'running'`, taskID, legID, fenceToken, leaseExpiresAt)
	if err != nil {
		return fmt.Errorf("renew builder reservation: %w", err)
	}
	return requireReservationUpdate(result)
}

// ReleaseBuilderReservation 仅以匹配 fence 更新 lease，避免 retry 释放新尝试的容量。
func (r *SQLRepository) ReleaseBuilderReservation(ctx context.Context, taskID uint64, legID, fenceToken, state string) error {
	if r == nil || r.db == nil || taskID == 0 || strings.TrimSpace(legID) == "" || strings.TrimSpace(fenceToken) == "" || (state != moduleapi.BuilderReservationReleased && state != moduleapi.BuilderReservationAbandoned) {
		return errors.New("invalid builder reservation release")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE build_builder_reservations SET state = $4, updated_at = NOW() WHERE task_id = $1 AND leg_id = $2 AND fence_token = $3 AND state IN ('accepted','running')`, taskID, legID, fenceToken, state)
	if err != nil {
		return fmt.Errorf("release builder reservation: %w", err)
	}
	return requireReservationUpdate(result)
}

func requireReservationUpdate(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count builder reservation transition: %w", err)
	}
	if count != 1 {
		return ErrConflict
	}
	return nil
}

func materializeWorkspaceSnapshot(ctx context.Context, tx *sql.Tx, snapshot moduleapi.WorkspaceSnapshot, requestedBy uint64) (uint64, error) {
	var snapshotPK uint64
	if err := tx.QueryRowContext(ctx, `INSERT INTO build_workspace_snapshots (snapshot_id, source_kind, source_reference, content_digest, materialization_ref, materialization_owner, materialization_state, retention_policy, retention_expires_at, created_by)
VALUES ($1, $2, $3, $4, $5, 'build', 'available', 'task_lifetime', NULL, $6)
ON CONFLICT (snapshot_id) DO UPDATE SET snapshot_id = EXCLUDED.snapshot_id, retention_policy = 'task_lifetime', retention_expires_at = NULL, materialization_state = 'available', materialization_ref = EXCLUDED.materialization_ref
RETURNING id`, snapshot.ID, snapshot.SourceKind, snapshot.SourceReference, snapshot.ContentDigest, snapshot.MaterializationRef, nullableUint64(requestedBy)).Scan(&snapshotPK); err != nil {
		return 0, fmt.Errorf("materialize workspace snapshot: %w", err)
	}
	return snapshotPK, nil
}

// CreateBuildInputSnapshot 将已由 Build 校验并物化到 Build-owned 临时目录的归档
// 注册为可复用 Snapshot。内容摘要冲突时返回已有可用 Snapshot，保证去重幂等。
func (r *SQLRepository) CreateBuildInputSnapshot(ctx context.Context, snapshot moduleapi.WorkspaceSnapshot, requestedBy uint64) (moduleapi.WorkspaceSnapshot, error) {
	if r == nil || r.db == nil || strings.TrimSpace(snapshot.ID) == "" || strings.TrimSpace(snapshot.ContentDigest) == "" || strings.TrimSpace(snapshot.MaterializationRef) == "" {
		return moduleapi.WorkspaceSnapshot{}, errors.New("invalid build input snapshot")
	}
	var existing moduleapi.WorkspaceSnapshot
	err := r.db.QueryRowContext(ctx, `INSERT INTO build_workspace_snapshots (snapshot_id, source_kind, source_reference, content_digest, materialization_ref, materialization_owner, materialization_state, retention_policy, retention_expires_at, created_by)
VALUES ($1,$2,$3,$4,$5,'build','available','snapshot_lifetime',NOW() + INTERVAL '24 hours',$6)
ON CONFLICT (content_digest) DO UPDATE SET
  materialization_state = CASE WHEN build_workspace_snapshots.materialization_state = 'purged' OR build_workspace_snapshots.retention_expires_at <= NOW() THEN EXCLUDED.materialization_state ELSE 'available' END,
  materialization_ref = CASE WHEN build_workspace_snapshots.materialization_state = 'purged' OR build_workspace_snapshots.retention_expires_at <= NOW() THEN EXCLUDED.materialization_ref ELSE build_workspace_snapshots.materialization_ref END,
  retention_expires_at = GREATEST(COALESCE(build_workspace_snapshots.retention_expires_at, NOW()), NOW() + INTERVAL '24 hours')
RETURNING snapshot_id, source_kind, source_reference, content_digest, materialization_ref, created_at`, snapshot.ID, snapshot.SourceKind, snapshot.SourceReference, snapshot.ContentDigest, snapshot.MaterializationRef, nullableUint64(requestedBy)).Scan(&existing.ID, &existing.SourceKind, &existing.SourceReference, &existing.ContentDigest, &existing.MaterializationRef, &existing.CreatedAt)
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, fmt.Errorf("create build input snapshot: %w", err)
	}
	return existing, nil
}

// GetBuildInputSnapshot 返回仍可用于新 Build 的 Snapshot；已清理物化内容的身份
// 保留为历史证据，但不能重新提交执行计划。
func (r *SQLRepository) GetBuildInputSnapshot(ctx context.Context, snapshotID string, requestedBy uint64) (moduleapi.WorkspaceSnapshot, error) {
	if r == nil || r.db == nil || strings.TrimSpace(snapshotID) == "" {
		return moduleapi.WorkspaceSnapshot{}, ErrNotFound
	}
	var snapshot moduleapi.WorkspaceSnapshot
	err := r.db.QueryRowContext(ctx, `SELECT snapshot_id, source_kind, source_reference, content_digest, materialization_ref, created_at
FROM build_workspace_snapshots
WHERE snapshot_id = $1 AND materialization_owner = 'build' AND materialization_state = 'available' AND (retention_expires_at IS NULL OR retention_expires_at > NOW()) AND ($2 = 0 OR created_by = $2)`, strings.TrimSpace(snapshotID), nullableUint64(requestedBy)).Scan(&snapshot.ID, &snapshot.SourceKind, &snapshot.SourceReference, &snapshot.ContentDigest, &snapshot.MaterializationRef, &snapshot.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.WorkspaceSnapshot{}, ErrNotFound
	}
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, fmt.Errorf("get build input snapshot: %w", err)
	}
	return snapshot, nil
}

// ListBuildInputSnapshots 返回当前用户仍可复用的上传快照，结果按创建时间倒序稳定分页。
//
//nolint:cyclop // 查询同时校验分页边界、统计结果和行扫描错误。
func (r *SQLRepository) ListBuildInputSnapshots(ctx context.Context, requestedBy uint64, limit, offset int) (InputSnapshotListResult, error) {
	if r == nil || r.db == nil || requestedBy == 0 || limit < 1 || limit > 100 || offset < 0 {
		return InputSnapshotListResult{}, errors.New("invalid build input snapshot list query")
	}
	const predicate = `FROM build_workspace_snapshots
WHERE materialization_owner = 'build' AND materialization_state = 'available'
  AND (retention_expires_at IS NULL OR retention_expires_at > NOW()) AND created_by = $1`
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) `+predicate, requestedBy).Scan(&total); err != nil {
		return InputSnapshotListResult{}, fmt.Errorf("count build input snapshots: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT snapshot_id, source_kind, source_reference, content_digest, materialization_ref, created_at `+predicate+` ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`, requestedBy, limit, offset)
	if err != nil {
		return InputSnapshotListResult{}, fmt.Errorf("list build input snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]moduleapi.WorkspaceSnapshot, 0, limit)
	for rows.Next() {
		var item moduleapi.WorkspaceSnapshot
		if err := rows.Scan(&item.ID, &item.SourceKind, &item.SourceReference, &item.ContentDigest, &item.MaterializationRef, &item.CreatedAt); err != nil {
			return InputSnapshotListResult{}, fmt.Errorf("scan build input snapshot: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return InputSnapshotListResult{}, fmt.Errorf("iterate build input snapshots: %w", err)
	}
	return InputSnapshotListResult{Items: items, Total: total}, nil
}

type executionPlanFields struct {
	platforms, placements, destination []byte
}

func marshalExecutionPlanFields(plan moduleapi.BuildExecutionPlan) (executionPlanFields, error) {
	platforms, err := json.Marshal(plan.Platforms)
	if err != nil {
		return executionPlanFields{}, fmt.Errorf("marshal plan platforms: %w", err)
	}
	destination, err := json.Marshal(plan.Destination)
	if err != nil {
		return executionPlanFields{}, fmt.Errorf("marshal plan destination: %w", err)
	}
	placements, err := json.Marshal(plan.BuilderPlacements)
	if err != nil {
		return executionPlanFields{}, fmt.Errorf("marshal plan builder placements: %w", err)
	}
	return executionPlanFields{platforms: platforms, placements: placements, destination: destination}, nil
}

//nolint:cyclop,gocyclo // 显式校验必须枚举每一个不可变计划边界。
func validExecutionPlan(plan moduleapi.BuildExecutionPlan) bool {
	if plan.ID == "" || plan.Digest == "" || plan.Workspace.ID == "" || plan.Workspace.ContentDigest == "" || plan.Workspace.MaterializationRef == "" || plan.RuntimeTargetID < 1 || plan.Driver == "" || plan.TemplateRef == "" || plan.CachePolicy == "" || plan.SecurityPolicy == "" || len(plan.Platforms) == 0 || plan.Destination.Kind == "" || plan.Destination.ConnectionRef == "" || plan.Destination.RepositoryRef == "" || plan.Destination.Reference == "" {
		return false
	}
	if len(plan.BuilderPlacements) > 0 {
		if len(plan.BuilderPlacements) != len(plan.Platforms) {
			return false
		}
		seen := make(map[string]struct{}, len(plan.BuilderPlacements))
		for _, placement := range plan.BuilderPlacements {
			if placement.Platform == "" || placement.RuntimeTargetID < 1 || !containsPlatform(plan.Platforms, placement.Platform) {
				return false
			}
			if _, exists := seen[placement.Platform]; exists {
				return false
			}
			seen[placement.Platform] = struct{}{}
		}
	}
	return (plan.BuilderPoolID == "" && plan.BuilderInstanceID == "") || (plan.BuilderPoolID != "" && plan.BuilderInstanceID != "")
}

// GetExecutionPlanByTaskID 仅返回 Task executor 所需的 immutable input；endpoint
// 与 credential detail 由其它 authority 单独解析。
func (r *SQLRepository) GetExecutionPlanByTaskID(ctx context.Context, taskID uint64) (moduleapi.BuildExecutionPlan, error) {
	if r == nil || r.db == nil || taskID == 0 {
		return moduleapi.BuildExecutionPlan{}, ErrNotFound
	}
	const query = `SELECT p.plan_id, p.plan_digest, s.snapshot_id, s.source_kind, s.source_reference, s.content_digest, s.materialization_ref, s.created_at, COALESCE(p.builder_pool_id, ''), COALESCE(p.builder_instance_id, ''), p.runtime_target_id, p.driver, p.template_ref, p.cache_policy, p.security_policy, p.platforms_json, p.builder_placements_json, p.destination_json, p.created_at
FROM build_execution_plans p JOIN build_workspace_snapshots s ON s.id = p.workspace_snapshot_id WHERE p.task_id = $1`
	var plan moduleapi.BuildExecutionPlan
	var platformsJSON, placementsJSON, destinationJSON []byte
	if err := r.db.QueryRowContext(ctx, query, taskID).Scan(&plan.ID, &plan.Digest, &plan.Workspace.ID, &plan.Workspace.SourceKind, &plan.Workspace.SourceReference, &plan.Workspace.ContentDigest, &plan.Workspace.MaterializationRef, &plan.Workspace.CreatedAt, &plan.BuilderPoolID, &plan.BuilderInstanceID, &plan.RuntimeTargetID, &plan.Driver, &plan.TemplateRef, &plan.CachePolicy, &plan.SecurityPolicy, &platformsJSON, &placementsJSON, &destinationJSON, &plan.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return moduleapi.BuildExecutionPlan{}, ErrNotFound
		}
		return moduleapi.BuildExecutionPlan{}, fmt.Errorf("query build execution plan: %w", err)
	}
	if err := json.Unmarshal(platformsJSON, &plan.Platforms); err != nil {
		return moduleapi.BuildExecutionPlan{}, fmt.Errorf("decode build plan platforms: %w", err)
	}
	if err := json.Unmarshal(placementsJSON, &plan.BuilderPlacements); err != nil {
		return moduleapi.BuildExecutionPlan{}, fmt.Errorf("decode build plan builder placements: %w", err)
	}
	if err := json.Unmarshal(destinationJSON, &plan.Destination); err != nil {
		return moduleapi.BuildExecutionPlan{}, fmt.Errorf("decode build plan destination: %w", err)
	}
	return plan, nil
}

// RecordProviderExecutionEvidence 以执行计划和阶段为不可变边界持久化 provider 能力证明。
//
//nolint:cyclop,gocyclo // 事务同时校验计划归属、证据完整性与幂等重放边界。
func (r *SQLRepository) RecordProviderExecutionEvidence(ctx context.Context, plan moduleapi.BuildExecutionPlan, evidence moduleapi.ProviderExecutionEvidence) error {
	if r == nil || r.db == nil || plan.ID == "" || evidence.TaskID == 0 || evidence.StageID == 0 || evidence.TargetID < 1 || strings.TrimSpace(evidence.Platform) == "" || !validProviderEvidence(evidence.Conformance) {
		return errors.New("invalid provider execution evidence")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin provider evidence: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var planPK int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM build_execution_plans WHERE plan_id = $1 AND task_id = $2`, plan.ID, evidence.TaskID).Scan(&planPK); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load provider evidence plan: %w", err)
	}
	result := evidence.Conformance
	if _, err := tx.ExecContext(ctx, `INSERT INTO build_provider_execution_evidence (execution_plan_id, task_id, stage_id, runtime_target_id, platform, provider_id, conformance_version, snapshot_delivery_proof, driver_execution_proof, publication_proof, cancellation_proof, cleanup_proof) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (execution_plan_id, stage_id) DO NOTHING`, planPK, evidence.TaskID, evidence.StageID, evidence.TargetID, evidence.Platform, result.ProviderID, result.ConformanceVersion, result.SnapshotDeliveryProof, result.DriverExecutionProof, result.PublicationProof, result.CancellationProof, result.CleanupProof); err != nil {
		return fmt.Errorf("insert provider execution evidence: %w", err)
	}
	var storedTargetID int64
	var storedPlatform, storedProviderID, storedVersion string
	var storedDelivery, storedDriver, storedPublication, storedCancellation, storedCleanup bool
	if err := tx.QueryRowContext(ctx, `SELECT runtime_target_id, platform, provider_id, conformance_version, snapshot_delivery_proof, driver_execution_proof, publication_proof, cancellation_proof, cleanup_proof FROM build_provider_execution_evidence WHERE execution_plan_id = $1 AND stage_id = $2`, planPK, evidence.StageID).Scan(&storedTargetID, &storedPlatform, &storedProviderID, &storedVersion, &storedDelivery, &storedDriver, &storedPublication, &storedCancellation, &storedCleanup); err != nil {
		return fmt.Errorf("verify provider execution evidence replay: %w", err)
	}
	if storedTargetID != evidence.TargetID || storedPlatform != evidence.Platform || storedProviderID != result.ProviderID || storedVersion != result.ConformanceVersion || storedDelivery != result.SnapshotDeliveryProof || storedDriver != result.DriverExecutionProof || storedPublication != result.PublicationProof || storedCancellation != result.CancellationProof || storedCleanup != result.CleanupProof {
		return ErrConflict
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit provider execution evidence: %w", err)
	}
	committed = true
	return nil
}

func validProviderEvidence(result moduleapi.ProviderExecutionConformanceResult) bool {
	return result.Executable && strings.TrimSpace(result.ProviderID) != "" && strings.TrimSpace(result.ConformanceVersion) != "" && result.SnapshotDeliveryProof && result.DriverExecutionProof && result.PublicationProof && result.CancellationProof && result.CleanupProof
}

// SettleV2Artifact 原子记录 digest-addressed artifact 及其 mutable publication
// reference，绝不改变 artifact identity。
//
//nolint:cyclop // Settlement 在同一 transaction boundary 内维护 plan ownership、immutable identity 与 publication idempotency。
func (r *SQLRepository) SettleV2Artifact(ctx context.Context, taskID uint64, plan moduleapi.BuildExecutionPlan, result moduleapi.BuildArtifactResult, authExecution moduleapi.RegistryAuthExecution) error {
	if r == nil || r.db == nil || taskID == 0 || plan.ID == "" || strings.TrimSpace(result.Digest) == "" || authExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return errors.New("invalid v2 artifact settlement")
	}
	platforms, err := json.Marshal([]string{firstPlatform(result.OS, result.Architecture, result.Variant)})
	if err != nil {
		return fmt.Errorf("marshal artifact platforms: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin v2 artifact settlement: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var planPK int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM build_execution_plans WHERE plan_id = $1 AND task_id = $2`, plan.ID, taskID).Scan(&planPK); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load v2 execution plan: %w", err)
	}
	artifactID := "artifact_" + strings.TrimPrefix(result.Digest, "sha256:")[:minStringLength(artifactIdentityPrefixLength, len(strings.TrimPrefix(result.Digest, "sha256:")))]
	var artifactPK int64
	if err := tx.QueryRowContext(ctx, `INSERT INTO build_v2_artifacts (artifact_id, artifact_digest, media_type, platforms_json, size_bytes, produced_plan_id) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (artifact_digest) DO UPDATE SET artifact_digest = EXCLUDED.artifact_digest RETURNING id`, artifactID, result.Digest, "application/vnd.oci.image.manifest.v1+json", platforms, result.SizeBytes, planPK).Scan(&artifactPK); err != nil {
		return fmt.Errorf("persist v2 artifact: %w", err)
	}
	publicationID := publicationIDFor(result.Digest, moduleapi.AuthorizedArtifactDestination(plan.Destination))
	if _, err := tx.ExecContext(ctx, `INSERT INTO build_publications (publication_id, artifact_id, destination_kind, connection_ref, repository_ref, mutable_reference, credential_execution_mode) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (publication_id) DO UPDATE SET artifact_id = EXCLUDED.artifact_id, credential_execution_mode = EXCLUDED.credential_execution_mode`, publicationID, artifactPK, plan.Destination.Kind, plan.Destination.ConnectionRef, plan.Destination.RepositoryRef, plan.Destination.Reference, authExecution.Mode); err != nil {
		return fmt.Errorf("persist v2 publication: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit v2 artifact settlement: %w", err)
	}
	committed = true
	return nil
}

// RecordPlatformArtifact 将一个已完成协调 leg 的不可变产物关联到冻结执行计划。它拒绝覆盖同一 leg 的既有结果，
// 因此重试只能重放同一 digest，不能改写审计事实。
//
//nolint:gocognit,gocyclo,cyclop // 单个事务必须同时维护计划归属、摘要去重与同 leg 不可变重放校验。
func (r *SQLRepository) RecordPlatformArtifact(ctx context.Context, taskID uint64, plan moduleapi.BuildExecutionPlan, artifact moduleapi.PlatformArtifact) error {
	if r == nil || r.db == nil || taskID == 0 || plan.ID == "" || strings.TrimSpace(artifact.LegID) == "" || strings.TrimSpace(artifact.Platform) == "" || strings.TrimSpace(artifact.Digest) == "" || strings.TrimSpace(artifact.MediaType) == "" || artifact.SizeBytes < 0 || !containsPlatform(plan.Platforms, artifact.Platform) {
		return errors.New("invalid platform artifact")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin platform artifact settlement: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var planPK int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM build_execution_plans WHERE plan_id = $1 AND task_id = $2`, plan.ID, taskID).Scan(&planPK); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load execution plan for platform artifact: %w", err)
	}
	artifactID := "artifact_" + strings.TrimPrefix(artifact.Digest, "sha256:")[:minStringLength(artifactIdentityPrefixLength, len(strings.TrimPrefix(artifact.Digest, "sha256:")))]
	platforms, err := json.Marshal([]string{artifact.Platform})
	if err != nil {
		return fmt.Errorf("marshal platform artifact platform: %w", err)
	}
	var artifactPK int64
	if err := tx.QueryRowContext(ctx, `INSERT INTO build_v2_artifacts (artifact_id, artifact_digest, media_type, platforms_json, size_bytes, produced_plan_id) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (artifact_digest) DO UPDATE SET artifact_digest = EXCLUDED.artifact_digest RETURNING id`, artifactID, artifact.Digest, artifact.MediaType, platforms, artifact.SizeBytes, planPK).Scan(&artifactPK); err != nil {
		return fmt.Errorf("persist platform artifact: %w", err)
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO build_platform_artifacts (execution_plan_id, leg_id, artifact_id, platform) VALUES ($1,$2,$3,$4) ON CONFLICT (execution_plan_id, leg_id) DO NOTHING`, planPK, artifact.LegID, artifactPK, artifact.Platform)
	if err != nil {
		return fmt.Errorf("link platform artifact: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count platform artifact link: %w", err)
	}
	if affected == 0 {
		var existingDigest, existingPlatform string
		if err := tx.QueryRowContext(ctx, `SELECT artifact.artifact_digest, platform.platform FROM build_platform_artifacts platform JOIN build_v2_artifacts artifact ON artifact.id = platform.artifact_id WHERE platform.execution_plan_id = $1 AND platform.leg_id = $2`, planPK, artifact.LegID).Scan(&existingDigest, &existingPlatform); err != nil {
			return fmt.Errorf("load existing platform artifact: %w", err)
		}
		if existingDigest != artifact.Digest || existingPlatform != artifact.Platform {
			return ErrConflict
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit platform artifact settlement: %w", err)
	}
	committed = true
	return nil
}

func containsPlatform(platforms []string, platform string) bool {
	for _, candidate := range platforms {
		if candidate == platform {
			return true
		}
	}
	return false
}

// ListPlatformArtifacts 返回一个执行计划已经持久化的平台 Artifact，并拒绝不属于该 Task 的计划引用。
func (r *SQLRepository) ListPlatformArtifacts(ctx context.Context, taskID uint64, plan moduleapi.BuildExecutionPlan) ([]moduleapi.PlatformArtifact, error) {
	if r == nil || r.db == nil || taskID == 0 || plan.ID == "" {
		return nil, ErrNotFound
	}
	rows, err := r.db.QueryContext(ctx, `SELECT link.leg_id, link.platform, artifact.artifact_digest, artifact.media_type, artifact.size_bytes, link.created_at
		FROM build_platform_artifacts link
		JOIN build_execution_plans execution_plan ON execution_plan.id = link.execution_plan_id
		JOIN build_v2_artifacts artifact ON artifact.id = link.artifact_id
		WHERE execution_plan.plan_id = $1 AND execution_plan.task_id = $2
		ORDER BY link.platform ASC`, plan.ID, taskID)
	if err != nil {
		return nil, fmt.Errorf("list platform artifacts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]moduleapi.PlatformArtifact, 0, len(plan.Platforms))
	for rows.Next() {
		var item moduleapi.PlatformArtifact
		if err := rows.Scan(&item.LegID, &item.Platform, &item.Digest, &item.MediaType, &item.SizeBytes, &item.ProducedAt); err != nil {
			return nil, fmt.Errorf("scan platform artifact: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate platform artifacts: %w", err)
	}
	return items, nil
}

// PrepareOCIManifestPublication 只在每个冻结目标平台都有一个不可变 Artifact 时构造 Driver 发布输入。
// 它不执行发布，也不允许通过 mutable tag 补齐缺失平台。
func (r *SQLRepository) PrepareOCIManifestPublication(ctx context.Context, taskID uint64, plan moduleapi.BuildExecutionPlan) (moduleapi.OCIManifestPublicationInput, error) {
	artifacts, err := r.ListPlatformArtifacts(ctx, taskID, plan)
	if err != nil {
		return moduleapi.OCIManifestPublicationInput{}, err
	}
	if !platformArtifactSetComplete(plan.Platforms, artifacts) {
		return moduleapi.OCIManifestPublicationInput{}, ErrConflict
	}
	return moduleapi.OCIManifestPublicationInput{Destination: moduleapi.AuthorizedArtifactDestination(plan.Destination), PlatformArtifacts: artifacts}, nil
}

// ListV2Artifacts 返回摘要寻址 Artifact 的稳定读取投影，不解析可变 Publication 引用。
func (r *SQLRepository) ListV2Artifacts(ctx context.Context, limit, offset int) (V2ArtifactListResult, error) {
	if r == nil || r.db == nil || limit < 1 || limit > MaxListLimit || offset < 0 {
		return V2ArtifactListResult{}, errors.New("invalid v2 artifact query")
	}
	total, err := r.countV2Artifacts(ctx)
	if err != nil {
		return V2ArtifactListResult{}, err
	}
	return r.listV2ArtifactPage(ctx, total, limit, offset)
}

// ListArtifactPublicationSources 返回 Artifact 的所有当前 Publication 记录。Promotion
// 必须从这些记录选择不可变 digest source，不能从可变 repository tag 推断 source。
func (r *SQLRepository) ListArtifactPublicationSources(ctx context.Context, artifactID string) ([]moduleapi.ArtifactPublicationSource, error) {
	if r == nil || r.db == nil || strings.TrimSpace(artifactID) == "" {
		return nil, ErrNotFound
	}
	rows, err := r.db.QueryContext(ctx, `SELECT artifact.artifact_id, publication.publication_id, artifact.artifact_digest, artifact.media_type, publication.destination_kind, publication.connection_ref, publication.repository_ref
		FROM build_publications publication
		JOIN build_v2_artifacts artifact ON artifact.id = publication.artifact_id
		WHERE artifact.artifact_id = $1
		ORDER BY publication.created_at DESC, publication.id DESC`, strings.TrimSpace(artifactID))
	if err != nil {
		return nil, fmt.Errorf("list artifact publications: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]moduleapi.ArtifactPublicationSource, 0)
	for rows.Next() {
		var item moduleapi.ArtifactPublicationSource
		if err := rows.Scan(&item.ArtifactID, &item.PublicationID, &item.Digest, &item.MediaType, &item.DestinationKind, &item.ConnectionRef, &item.RepositoryRef); err != nil {
			return nil, fmt.Errorf("scan artifact publication: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artifact publications: %w", err)
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}

func (r *SQLRepository) countV2Artifacts(ctx context.Context) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM build_v2_artifacts`).Scan(&total); err != nil {
		return 0, fmt.Errorf("count v2 artifacts: %w", err)
	}
	return total, nil
}

func (r *SQLRepository) listV2ArtifactPage(ctx context.Context, total int64, limit, offset int) (V2ArtifactListResult, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT artifact_id, artifact_digest, media_type, platforms_json, size_bytes, created_at FROM build_v2_artifacts ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return V2ArtifactListResult{}, fmt.Errorf("list v2 artifacts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := V2ArtifactListResult{Items: make([]V2ArtifactProjection, 0), Total: total}
	for rows.Next() {
		var item V2ArtifactProjection
		var platforms []byte
		if err := rows.Scan(&item.ArtifactID, &item.Digest, &item.MediaType, &platforms, &item.SizeBytes, &item.CreatedAt); err != nil {
			return V2ArtifactListResult{}, fmt.Errorf("scan v2 artifact: %w", err)
		}
		if err := json.Unmarshal(platforms, &item.Platforms); err != nil {
			return V2ArtifactListResult{}, fmt.Errorf("decode v2 artifact platforms: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return V2ArtifactListResult{}, fmt.Errorf("iterate v2 artifacts: %w", err)
	}
	return result, nil
}

func platformArtifactSetComplete(platforms []string, artifacts []moduleapi.PlatformArtifact) bool {
	if len(platforms) < 2 || len(artifacts) != len(platforms) {
		return false
	}
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.LegID == "" || artifact.Digest == "" || artifact.MediaType == "" || artifact.SizeBytes < 0 || !containsPlatform(platforms, artifact.Platform) {
			return false
		}
		if _, exists := seen[artifact.Platform]; exists {
			return false
		}
		seen[artifact.Platform] = struct{}{}
	}
	return true
}

// SettleOCIManifestPublication 在 Build 再次确认全部平台 Artifact 完整后，记录 Driver 返回的最终 OCI Manifest。
// Driver 负责发布，Build 负责不可变 Artifact 与可变 Publication 的事实边界。
//
//nolint:gocyclo,cyclop,gocognit // 事务必须把完整性复验、Artifact 身份与 Publication 指向作为一个不可分割的提交。
func (r *SQLRepository) SettleOCIManifestPublication(ctx context.Context, taskID uint64, plan moduleapi.BuildExecutionPlan, result moduleapi.OCIManifestPublicationResult, authExecution moduleapi.RegistryAuthExecution) error {
	if r == nil || r.db == nil || taskID == 0 || plan.ID == "" || strings.TrimSpace(result.Digest) == "" || strings.TrimSpace(result.MediaType) == "" || result.SizeBytes < 0 || authExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return errors.New("invalid OCI manifest settlement")
	}
	if _, err := r.PrepareOCIManifestPublication(ctx, taskID, plan); err != nil {
		return fmt.Errorf("verify manifest platform artifacts: %w", err)
	}
	platforms, err := json.Marshal(plan.Platforms)
	if err != nil {
		return fmt.Errorf("marshal manifest platforms: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin OCI manifest settlement: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var planPK int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM build_execution_plans WHERE plan_id = $1 AND task_id = $2`, plan.ID, taskID).Scan(&planPK); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load manifest execution plan: %w", err)
	}
	manifestID := "artifact_" + strings.TrimPrefix(result.Digest, "sha256:")[:minStringLength(artifactIdentityPrefixLength, len(strings.TrimPrefix(result.Digest, "sha256:")))]
	var artifactPK int64
	if err := tx.QueryRowContext(ctx, `INSERT INTO build_v2_artifacts (artifact_id, artifact_digest, media_type, platforms_json, size_bytes, produced_plan_id) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (artifact_digest) DO UPDATE SET artifact_digest = EXCLUDED.artifact_digest RETURNING id`, manifestID, result.Digest, result.MediaType, platforms, result.SizeBytes, planPK).Scan(&artifactPK); err != nil {
		return fmt.Errorf("persist OCI manifest artifact: %w", err)
	}
	publicationID := publicationIDFor(result.Digest, moduleapi.AuthorizedArtifactDestination(plan.Destination))
	if _, err := tx.ExecContext(ctx, `INSERT INTO build_publications (publication_id, artifact_id, destination_kind, connection_ref, repository_ref, mutable_reference, credential_execution_mode) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (publication_id) DO UPDATE SET artifact_id = EXCLUDED.artifact_id, credential_execution_mode = EXCLUDED.credential_execution_mode`, publicationID, artifactPK, plan.Destination.Kind, plan.Destination.ConnectionRef, plan.Destination.RepositoryRef, plan.Destination.Reference, authExecution.Mode); err != nil {
		return fmt.Errorf("persist OCI manifest publication: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit OCI manifest settlement: %w", err)
	}
	committed = true
	return nil
}

// SettleArtifactPromotion 重新核对 source Artifact 的 digest 和 media type，再记录新的 Publication。
// provider 返回任何不同 digest 都会 fail-closed；同一 source/destination 重放保持幂等，新的目的地引用保留历史。
//
//nolint:cyclop // 结算必须在同一事务边界显式覆盖完整性校验、来源读取与幂等写入。
func (r *SQLRepository) SettleArtifactPromotion(ctx context.Context, input moduleapi.OCIArtifactCopyInput, result moduleapi.OCIArtifactCopyResult, authExecution moduleapi.RegistryAuthExecution) error {
	if r == nil || r.db == nil || !validPromotionSettlement(input, result, authExecution) {
		return errors.New("invalid artifact promotion settlement")
	}
	if result.Digest != input.Source.Digest || result.MediaType != input.Source.MediaType {
		return ErrConflict
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin artifact promotion settlement: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var artifactPK int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM build_v2_artifacts WHERE artifact_id = $1 AND artifact_digest = $2`, input.Source.ArtifactID, input.Source.Digest).Scan(&artifactPK); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load promotion source artifact: %w", err)
	}
	publicationID := publicationIDFor(input.Source.Digest, input.Destination)
	if _, err := tx.ExecContext(ctx, `INSERT INTO build_publications (publication_id, artifact_id, destination_kind, connection_ref, repository_ref, mutable_reference, credential_execution_mode) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (publication_id) DO UPDATE SET artifact_id = EXCLUDED.artifact_id, credential_execution_mode = EXCLUDED.credential_execution_mode`, publicationID, artifactPK, input.Destination.Kind, input.Destination.ConnectionRef, input.Destination.RepositoryRef, input.Destination.Reference, authExecution.Mode); err != nil {
		return fmt.Errorf("persist promoted publication: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit artifact promotion settlement: %w", err)
	}
	committed = true
	return nil
}

//nolint:cyclop // 复制结果进入持久化前必须逐项验证不可变来源、目标与 provider 证明。
func validPromotionSettlement(input moduleapi.OCIArtifactCopyInput, result moduleapi.OCIArtifactCopyResult, authExecution moduleapi.RegistryAuthExecution) bool {
	digest := strings.TrimSpace(input.Source.Digest)
	return input.Source.DestinationKind == "oci_registry" && input.Destination.Kind == "oci_registry" &&
		strings.TrimSpace(input.Source.ArtifactID) != "" && strings.TrimSpace(input.Source.MediaType) != "" &&
		strings.HasPrefix(digest, "sha256:") && len(strings.TrimPrefix(digest, "sha256:")) == 64 &&
		strings.TrimSpace(input.Destination.ConnectionRef) != "" && strings.TrimSpace(input.Destination.RepositoryRef) != "" && strings.TrimSpace(input.Destination.Reference) != "" &&
		strings.TrimSpace(result.Digest) != "" && strings.TrimSpace(result.MediaType) != "" && result.SizeBytes >= 0 && authExecution.Mode == moduleapi.RegistryAuthExecutionEphemeral
}

func publicationIDFor(digest string, destination moduleapi.AuthorizedArtifactDestination) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{digest, destination.Kind, destination.ConnectionRef, destination.RepositoryRef, destination.Reference}, "\x00")))
	return "publication_" + hex.EncodeToString(sum[:])[:artifactIdentityPrefixLength]
}

func firstPlatform(osName, architecture, variant string) string {
	platform := strings.Trim(strings.TrimSpace(osName)+"/"+strings.TrimSpace(architecture)+"/"+strings.TrimSpace(variant), "/")
	return platform
}

func minStringLength(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func nullableUint64(value uint64) any {
	if value == 0 {
		return nil
	}
	return value
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

// ListV2Jobs 读取新 Build 作业的 canonical execution-plan projection。
func (r *SQLRepository) ListV2Jobs(ctx context.Context, query ListQuery) (result ListResult, err error) {
	if r == nil || r.db == nil {
		return result, errors.New("build repository is unavailable")
	}
	query.Limit, query.Offset = normalizedPagination(query.Limit, query.Offset)
	where, args := v2JobListFilters(query)
	predicate := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM build_execution_plans p JOIN build_workspace_snapshots s ON s.id = p.workspace_snapshot_id WHERE `+predicate, args...).Scan(&total); err != nil {
		return result, fmt.Errorf("count v2 build jobs: %w", err)
	}
	result.Total = total
	args = append(args, query.Limit, query.Offset)
	//nolint:gosec // predicate and placeholders are generated from fixed query fields; values remain bound args.
	rows, err := r.db.QueryContext(ctx, `SELECT p.plan_id, p.task_id, s.snapshot_id, s.content_digest, s.source_kind, p.runtime_target_id, p.created_at, COALESCE(p.destination_json->>'repository_ref',''), COALESCE(p.destination_json->>'reference',''), COALESCE(a.artifact_id,''), COALESCE(a.artifact_digest,''), COALESCE(a.size_bytes,0)
FROM build_execution_plans p JOIN build_workspace_snapshots s ON s.id = p.workspace_snapshot_id LEFT JOIN LATERAL (SELECT artifact_id, artifact_digest, size_bytes FROM build_v2_artifacts WHERE produced_plan_id = p.id ORDER BY created_at DESC, id DESC LIMIT 1) a ON TRUE
WHERE `+predicate+` ORDER BY p.created_at DESC, p.id DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return result, fmt.Errorf("list v2 build jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var item JobProjection
		var artifactID, digest, tag string
		var size int64
		if err := rows.Scan(&item.BuildID, &item.TaskID, &item.InputSnapshotID, &item.InputSnapshotDigest, &item.SourceKind, &item.RuntimeTargetID, &item.CreatedAt, &item.ImageRepository, &tag, &artifactID, &digest, &size); err != nil {
			return result, fmt.Errorf("scan v2 build job: %w", err)
		}
		item.ImageTag = tag
		if artifactID != "" {
			item.Artifact = &Artifact{ArtifactID: artifactID, Digest: digest, Repository: item.ImageRepository, Tag: tag, SizeBytes: size}
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate v2 build jobs: %w", err)
	}
	return result, nil
}

// GetV2JobByBuildID returns one canonical execution-plan projection by plan identity.
func (r *SQLRepository) GetV2JobByBuildID(ctx context.Context, buildID string) (JobProjection, error) {
	if r == nil || r.db == nil || strings.TrimSpace(buildID) == "" {
		return JobProjection{}, ErrNotFound
	}
	var item JobProjection
	var artifactID, digest, tag string
	var size int64
	err := r.db.QueryRowContext(ctx, `SELECT p.plan_id, p.task_id, s.snapshot_id, s.content_digest, s.source_kind, p.runtime_target_id, p.created_at, COALESCE(p.destination_json->>'repository_ref',''), COALESCE(p.destination_json->>'reference',''), COALESCE(a.artifact_id,''), COALESCE(a.artifact_digest,''), COALESCE(a.size_bytes,0)
FROM build_execution_plans p JOIN build_workspace_snapshots s ON s.id = p.workspace_snapshot_id LEFT JOIN LATERAL (SELECT artifact_id, artifact_digest, size_bytes FROM build_v2_artifacts WHERE produced_plan_id = p.id ORDER BY created_at DESC, id DESC LIMIT 1) a ON TRUE
WHERE p.plan_id = $1`, strings.TrimSpace(buildID)).Scan(&item.BuildID, &item.TaskID, &item.InputSnapshotID, &item.InputSnapshotDigest, &item.SourceKind, &item.RuntimeTargetID, &item.CreatedAt, &item.ImageRepository, &tag, &artifactID, &digest, &size)
	if errors.Is(err, sql.ErrNoRows) {
		return JobProjection{}, ErrNotFound
	}
	if err != nil {
		return JobProjection{}, fmt.Errorf("get v2 build job: %w", err)
	}
	item.ImageTag = tag
	if artifactID != "" {
		item.Artifact = &Artifact{ArtifactID: artifactID, Digest: digest, Repository: item.ImageRepository, Tag: tag, SizeBytes: size}
	}
	return item, nil
}

// ListJobs is the legacy projection retained for historical evidence.
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
	if query.Search != nil {
		placeholder := strconv.Itoa(len(args) + 1)
		where = append(where, `(j.build_id ILIKE '%' || $`+placeholder+` || '%' OR j.application_name_snapshot ILIKE '%' || $`+placeholder+` || '%' OR j.image_repository ILIKE '%' || $`+placeholder+` || '%' OR j.image_tag ILIKE '%' || $`+placeholder+` || '%')`)
		args = append(args, *query.Search)
	}
	if query.BuilderID != nil {
		where = append(where, `j.runtime_target_id = $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.BuilderID)
	}
	if query.CreatedAfter != nil {
		where = append(where, `j.created_at >= $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.CreatedAfter)
	}
	if query.CreatedBefore != nil {
		where = append(where, `j.created_at <= $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.CreatedBefore)
	}
	return appendTaskStatusFilter(where, args, query.BuildStatus)
}

func v2JobListFilters(query ListQuery) ([]string, []any) {
	where := []string{"1 = 1"}
	args := make([]any, 0, jobListFilterCap)
	if query.ApplicationID != nil {
		where = append(where, "FALSE")
	}
	if query.ImageRepository != nil {
		where = append(where, `p.destination_json->>'repository_ref' = $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.ImageRepository)
	}
	if query.ImageTag != nil {
		where = append(where, `p.destination_json->>'reference' = $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.ImageTag)
	}
	if query.Search != nil {
		placeholder := strconv.Itoa(len(args) + 1)
		where = append(where, `(p.plan_id ILIKE '%' || $`+placeholder+` || '%' OR s.snapshot_id ILIKE '%' || $`+placeholder+` || '%' OR s.content_digest ILIKE '%' || $`+placeholder+` || '%')`)
		args = append(args, *query.Search)
	}
	if query.BuilderID != nil {
		where = append(where, `p.runtime_target_id = $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.BuilderID)
	}
	if query.CreatedAfter != nil {
		where = append(where, `p.created_at >= $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.CreatedAfter)
	}
	if query.CreatedBefore != nil {
		where = append(where, `p.created_at <= $`+strconv.Itoa(len(args)+1))
		args = append(args, *query.CreatedBefore)
	}
	if query.BuildStatus != nil {
		where, args = appendTaskStatusFilter(where, args, query.BuildStatus)
		where[len(where)-1] = strings.Replace(where[len(where)-1], "j.task_id", "p.task_id", 1)
	}
	return where, args
}

func appendTaskStatusFilter(where []string, args []any, filter *StatusFilter) ([]string, []any) {
	if filter == nil {
		return where, args
	}
	statuses := filter.taskStatuses()
	if len(statuses) == 0 {
		return append(where, "FALSE"), args
	}
	placeholders := make([]string, 0, len(statuses))
	for _, status := range statuses {
		placeholders = append(placeholders, "$"+strconv.Itoa(len(args)+1))
		args = append(args, status)
	}
	where = append(where, `EXISTS (SELECT 1 FROM tasks t WHERE t.id = j.task_id AND t.status IN (`+strings.Join(placeholders, ", ")+`))`)
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

const jobProjectionQuery = `SELECT j.build_id, j.task_id, j.application_id, j.application_record_id, j.application_name_snapshot, j.workspace_context_path, j.workspace_root, j.dockerfile_path, j.runtime_target_id, COALESCE(j.runtime_target_name, ''), j.runtime_provider, j.image_repository, j.image_tag, COALESCE(j.created_by, 0), j.created_at, a.artifact_id, a.image_id, COALESCE(a.digest, ''), a.repository, a.tag, a.size_bytes, CONCAT_WS('/', NULLIF(a.os, ''), NULLIF(a.architecture, '')) FROM build_jobs j LEFT JOIN build_artifacts a ON a.build_job_id = j.id AND a.role = 'primary'`

type rowScanner interface{ Scan(...any) error }

func scanJobProjection(row rowScanner) (JobProjection, error) {
	var item JobProjection
	var artifactID, imageID, digest, repository, tag, platform sql.NullString
	var size sql.NullInt64
	err := row.Scan(&item.BuildID, &item.TaskID, &item.ApplicationID, &item.ApplicationRecordID, &item.ApplicationName, &item.ContextPath, &item.WorkspaceRoot, &item.DockerfilePath, &item.RuntimeTargetID, &item.RuntimeTargetName, &item.RuntimeProvider, &item.ImageRepository, &item.ImageTag, &item.RequestedBy, &item.CreatedAt, &artifactID, &imageID, &digest, &repository, &tag, &size, &platform)
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

// CreateWorkspace 持久化可复用的来源定义；相同身份只能重复提交完全相同的定义，
// 不允许通过幂等写入偷偷改变来源或保留策略。
func (r *SQLRepository) CreateWorkspace(ctx context.Context, workspace moduleapi.BuildWorkspace) error {
	workspace, err := normalizeWorkspace(workspace)
	if err != nil {
		return err
	}
	if r == nil || r.db == nil {
		return errors.New("build repository is unavailable")
	}
	var returnedID string
	err = r.db.QueryRowContext(ctx, `INSERT INTO build_workspaces (workspace_id, display_name, source_kind, source_reference, retention_policy, created_by)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (workspace_id) DO UPDATE SET workspace_id = EXCLUDED.workspace_id
  WHERE build_workspaces.display_name = EXCLUDED.display_name
    AND build_workspaces.source_kind = EXCLUDED.source_kind
    AND build_workspaces.source_reference = EXCLUDED.source_reference
    AND build_workspaces.retention_policy = EXCLUDED.retention_policy
RETURNING workspace_id`, workspace.ID, workspace.Name, workspace.SourceKind, workspace.SourceReference, workspace.RetentionPolicy, nullableUint64(workspace.CreatedBy)).Scan(&returnedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConflict
		}
		return fmt.Errorf("create build workspace: %w", err)
	}
	return nil
}

// GetWorkspace 返回 Build-owned Workspace 定义，不包含任何物化路径或凭据。
func (r *SQLRepository) GetWorkspace(ctx context.Context, workspaceID string) (moduleapi.BuildWorkspace, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" || r == nil || r.db == nil {
		return moduleapi.BuildWorkspace{}, ErrNotFound
	}
	var workspace moduleapi.BuildWorkspace
	var createdBy sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT workspace_id, display_name, source_kind, source_reference, retention_policy, COALESCE(created_by, 0), created_at, updated_at
FROM build_workspaces WHERE workspace_id = $1 AND deleted_at = 0`, workspaceID).Scan(&workspace.ID, &workspace.Name, &workspace.SourceKind, &workspace.SourceReference, &workspace.RetentionPolicy, &createdBy, &workspace.CreatedAt, &workspace.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.BuildWorkspace{}, ErrNotFound
	}
	if err != nil {
		return moduleapi.BuildWorkspace{}, fmt.Errorf("get build workspace: %w", err)
	}
	if createdBy.Valid && createdBy.Int64 > 0 {
		workspace.CreatedBy = uint64(createdBy.Int64)
	}
	return workspace, nil
}

// ListWorkspaces 只返回调用者创建或平台共享的来源定义，不包含物化路径。
func (r *SQLRepository) ListWorkspaces(ctx context.Context, requestedBy uint64, query WorkspaceListQuery) (result WorkspaceListResult, err error) {
	if r == nil || r.db == nil {
		return result, errors.New("build repository is unavailable")
	}
	query.Limit, query.Offset = normalizedPagination(query.Limit, query.Offset)
	where, args, err := buildWorkspaceListFilter(requestedBy, query.Search)
	if err != nil {
		return result, err
	}
	if err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM build_workspaces WHERE `+where, args...).Scan(&result.Total); err != nil {
		return result, fmt.Errorf("count build workspaces: %w", err)
	}
	result.Items, err = r.listWorkspacePage(ctx, where, args, query)
	return result, err
}

func buildWorkspaceListFilter(requestedBy uint64, searchQuery *string) (string, []any, error) {
	where := `deleted_at = 0 AND (created_by = $1 OR created_by IS NULL)`
	args := []any{nullableUint64(requestedBy)}
	if searchQuery != nil {
		search := strings.TrimSpace(*searchQuery)
		if search == "" || utf8.RuneCountInString(search) > 255 {
			return "", nil, errors.New("invalid build workspace query")
		}
		args = append(args, search)
		placeholder := strconv.Itoa(len(args))
		where += ` AND (display_name ILIKE '%' || $` + placeholder + ` || '%' OR workspace_id ILIKE '%' || $` + placeholder + ` || '%' OR source_reference ILIKE '%' || $` + placeholder + ` || '%')`
	}
	return where, args, nil
}

func (r *SQLRepository) listWorkspacePage(ctx context.Context, where string, args []any, query WorkspaceListQuery) (items []moduleapi.BuildWorkspace, err error) {
	pageArgs := append(append([]any{}, args...), query.Limit, query.Offset)
	limitPlaceholder := strconv.Itoa(len(args) + 1)
	offsetPlaceholder := strconv.Itoa(len(args) + pageArgumentCount)
	// #nosec G202 -- where 与占位符只由本函数的静态片段生成，搜索词始终通过参数绑定。
	pageQuery := `SELECT workspace_id, display_name, source_kind, source_reference, retention_policy, COALESCE(created_by, 0), created_at, updated_at
FROM build_workspaces WHERE ` + where + ` ORDER BY display_name, workspace_id LIMIT $` + limitPlaceholder + ` OFFSET $` + offsetPlaceholder
	rows, err := r.db.QueryContext(ctx, pageQuery, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("list build workspaces: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close build workspace rows: %w", closeErr)
		}
	}()
	items = make([]moduleapi.BuildWorkspace, 0)
	for rows.Next() {
		var item moduleapi.BuildWorkspace
		var createdBy sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Name, &item.SourceKind, &item.SourceReference, &item.RetentionPolicy, &createdBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan build workspace: %w", err)
		}
		if createdBy.Valid && createdBy.Int64 > 0 {
			item.CreatedBy = uint64(createdBy.Int64)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate build workspaces: %w", err)
	}
	return items, nil
}

func normalizeWorkspace(workspace moduleapi.BuildWorkspace) (moduleapi.BuildWorkspace, error) {
	workspace.ID = strings.TrimSpace(workspace.ID)
	workspace.Name = strings.TrimSpace(workspace.Name)
	workspace.SourceKind = strings.TrimSpace(workspace.SourceKind)
	workspace.SourceReference = strings.TrimSpace(workspace.SourceReference)
	workspace.RetentionPolicy = strings.TrimSpace(workspace.RetentionPolicy)
	if workspace.RetentionPolicy == "" {
		workspace.RetentionPolicy = "workspace"
	}
	if workspace.ID == "" || workspace.Name == "" || workspace.SourceReference == "" || workspace.SourceKind == "" || !validWorkspaceSourceKind(workspace.SourceKind) || strings.ContainsAny(workspace.ID+workspace.SourceReference, "\x00\r\n") {
		return moduleapi.BuildWorkspace{}, errors.New("invalid build workspace")
	}
	return workspace, nil
}

func validWorkspaceSourceKind(sourceKind string) bool {
	switch sourceKind {
	case moduleapi.WorkspaceSourceApplication, moduleapi.WorkspaceSourceGit, moduleapi.WorkspaceSourceArchive, moduleapi.WorkspaceSourceGenerated, moduleapi.WorkspaceSourceTargetLocal:
		return true
	default:
		return false
	}
}

// CreateBuilderProfile 写入不可携带连接信息的 Builder Profile。
func (r *SQLRepository) CreateBuilderProfile(ctx context.Context, profile moduleapi.BuilderProfile, requestedBy uint64) error {
	if r == nil || r.db == nil {
		return errors.New("build repository is unavailable")
	}
	profile, err := normalizeBuilderProfile(profile)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO build_builder_profiles (profile_id, display_name, driver_ref, driver_version, policy_json, created_by, updated_by)
VALUES ($1,$2,$3,$4,$5,$6,$6)`, profile.ID, profile.DisplayName, profile.DriverRef, profile.DriverVersion, profile.Policy, nullableUint64(requestedBy))
	if err != nil {
		return fmt.Errorf("create builder profile: %w", err)
	}
	return nil
}

// CreateBuilderInstance 将 Build Profile 绑定到 Runtime Target 的 build capability。
func (r *SQLRepository) CreateBuilderInstance(ctx context.Context, instance moduleapi.BuilderInstance, requestedBy uint64) error {
	if r == nil || r.db == nil {
		return errors.New("build repository is unavailable")
	}
	instance, labels, err := normalizeBuilderInstance(instance)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO build_builder_instances (instance_id, profile_id, runtime_target_id, status, labels_json, created_by, updated_by)
SELECT $1, p.id, $3, $4, $5, $6, $6 FROM build_builder_profiles p WHERE p.profile_id = $2 AND p.deleted_at = 0`, instance.ID, instance.ProfileID, instance.RuntimeTargetID, instance.Status, labels, nullableUint64(requestedBy))
	if err != nil {
		return fmt.Errorf("create builder instance: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check builder profile for instance: %w", err)
	}
	if affected != 1 {
		return ErrNotFound
	}
	return nil
}

// CreateBuilderPool 创建不拥有执行状态的 Builder Pool。
func (r *SQLRepository) CreateBuilderPool(ctx context.Context, pool moduleapi.BuilderPool, requestedBy uint64) error {
	if r == nil || r.db == nil {
		return errors.New("build repository is unavailable")
	}
	pool, err := normalizeBuilderPool(pool)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO build_builder_pools (pool_id, display_name, scheduling_policy, selector_json, created_by, updated_by)
VALUES ($1,$2,$3,$4,$5,$5)`, pool.ID, pool.DisplayName, pool.SchedulingPolicy, pool.Selector, nullableUint64(requestedBy))
	if err != nil {
		return fmt.Errorf("create builder pool: %w", err)
	}
	return nil
}

// ReplaceBuilderPoolMembers 在一个事务中替换 Pool 的 live 成员集合；历史关系保留为软删除证据。
//
//nolint:gocognit,gocyclo,cyclop // 成员替换必须在单一事务内完成软删除、引用校验和重建，保持 Pool 读者看到一致集合。
func (r *SQLRepository) ReplaceBuilderPoolMembers(ctx context.Context, poolID string, members []moduleapi.BuilderPoolMember, requestedBy uint64) error {
	if r == nil || r.db == nil || strings.TrimSpace(poolID) == "" {
		return errors.New("invalid builder pool members")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace builder pool members: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var poolPK int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM build_builder_pools WHERE pool_id = $1 AND deleted_at = 0 FOR UPDATE`, strings.TrimSpace(poolID)).Scan(&poolPK); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock builder pool: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE build_builder_pool_members SET deleted_at = EXTRACT(EPOCH FROM NOW())::BIGINT, deleted_by = $2 WHERE pool_id = $1 AND deleted_at = 0`, poolPK, nullableUint64(requestedBy)); err != nil {
		return fmt.Errorf("retire builder pool members: %w", err)
	}
	seen := make(map[string]struct{}, len(members))
	for _, member := range members {
		member.InstanceID = strings.TrimSpace(member.InstanceID)
		if member.InstanceID == "" || member.Priority < 0 {
			return errors.New("invalid builder pool member")
		}
		if _, ok := seen[member.InstanceID]; ok {
			return errors.New("duplicate builder pool member")
		}
		seen[member.InstanceID] = struct{}{}
		result, err := tx.ExecContext(ctx, `INSERT INTO build_builder_pool_members (pool_id, instance_id, priority, created_by)
SELECT $1, i.id, $3, $4 FROM build_builder_instances i WHERE i.instance_id = $2 AND i.deleted_at = 0`, poolPK, member.InstanceID, member.Priority, nullableUint64(requestedBy))
		if err != nil {
			return fmt.Errorf("add builder pool member: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check builder pool member: %w", err)
		}
		if affected != 1 {
			return ErrNotFound
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit builder pool members: %w", err)
	}
	committed = true
	return nil
}

// GetBuilderPool 返回 Pool 的非秘密、非执行状态定义。
func (r *SQLRepository) GetBuilderPool(ctx context.Context, poolID string) (moduleapi.BuilderPool, error) {
	if r == nil || r.db == nil || strings.TrimSpace(poolID) == "" {
		return moduleapi.BuilderPool{}, ErrNotFound
	}
	var pool moduleapi.BuilderPool
	if err := r.db.QueryRowContext(ctx, `SELECT pool_id, display_name, scheduling_policy, selector_json FROM build_builder_pools WHERE pool_id = $1 AND deleted_at = 0`, strings.TrimSpace(poolID)).Scan(&pool.ID, &pool.DisplayName, &pool.SchedulingPolicy, &pool.Selector); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return moduleapi.BuilderPool{}, ErrNotFound
		}
		return moduleapi.BuilderPool{}, fmt.Errorf("get builder pool: %w", err)
	}
	return pool, nil
}

// ListBuilderPools 返回不含成员执行状态或连接信息的 Pool 选择投影。
func (r *SQLRepository) ListBuilderPools(ctx context.Context) ([]moduleapi.BuilderPool, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("build repository is unavailable")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT pool_id, display_name, scheduling_policy, selector_json
FROM build_builder_pools WHERE deleted_at = 0 ORDER BY display_name, pool_id`)
	if err != nil {
		return nil, fmt.Errorf("list builder pools: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]moduleapi.BuilderPool, 0)
	for rows.Next() {
		var item moduleapi.BuilderPool
		if err := rows.Scan(&item.ID, &item.DisplayName, &item.SchedulingPolicy, &item.Selector); err != nil {
			return nil, fmt.Errorf("scan builder pool: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate builder pools: %w", err)
	}
	return items, nil
}

// ListBuilderPoolMembers 按稳定优先级和公开标识排序，供纯选择逻辑消费。
func (r *SQLRepository) ListBuilderPoolMembers(ctx context.Context, poolID string) ([]moduleapi.BuilderInstance, error) {
	if r == nil || r.db == nil || strings.TrimSpace(poolID) == "" {
		return nil, ErrNotFound
	}
	rows, err := r.db.QueryContext(ctx, `SELECT i.instance_id, p.profile_id, i.runtime_target_id, i.status, i.labels_json, p.driver_ref, p.driver_version
FROM build_builder_pool_members m
JOIN build_builder_pools b ON b.id = m.pool_id AND b.deleted_at = 0
JOIN build_builder_instances i ON i.id = m.instance_id AND i.deleted_at = 0
JOIN build_builder_profiles p ON p.id = i.profile_id AND p.deleted_at = 0
WHERE b.pool_id = $1 AND m.deleted_at = 0 ORDER BY m.priority ASC, i.instance_id ASC`, strings.TrimSpace(poolID))
	if err != nil {
		return nil, fmt.Errorf("list builder pool members: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]moduleapi.BuilderInstance, 0)
	for rows.Next() {
		var item moduleapi.BuilderInstance
		var labels []byte
		if err := rows.Scan(&item.ID, &item.ProfileID, &item.RuntimeTargetID, &item.Status, &labels, &item.DriverRef, &item.DriverVersion); err != nil {
			return nil, fmt.Errorf("scan builder pool member: %w", err)
		}
		if err := json.Unmarshal(labels, &item.Labels); err != nil {
			return nil, fmt.Errorf("decode builder instance labels: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate builder pool members: %w", err)
	}
	return items, nil
}

// SelectRoundRobinBuilderInstance 在 Pool 行锁内推进游标并返回下一个 ready 实例。
// 它只决定 Builder resource，不创建任务或维护第二个执行队列。
//
//nolint:cyclop // 选择路径显式处理事务、空集合和策略边界，不能隐藏失败语义。
func (r *SQLRepository) SelectRoundRobinBuilderInstance(ctx context.Context, poolID string) (moduleapi.BuilderPoolSelection, error) {
	if r == nil || r.db == nil || strings.TrimSpace(poolID) == "" {
		return moduleapi.BuilderPoolSelection{}, ErrNotFound
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return moduleapi.BuilderPoolSelection{}, fmt.Errorf("begin select builder instance: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var poolPK, cursor int64
	var policy string
	if err := tx.QueryRowContext(ctx, `SELECT id, selection_cursor, scheduling_policy FROM build_builder_pools WHERE pool_id = $1 AND deleted_at = 0 FOR UPDATE`, strings.TrimSpace(poolID)).Scan(&poolPK, &cursor, &policy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return moduleapi.BuilderPoolSelection{}, ErrNotFound
		}
		return moduleapi.BuilderPoolSelection{}, fmt.Errorf("lock builder pool selection: %w", err)
	}
	if policy != "round_robin" {
		return moduleapi.BuilderPoolSelection{}, errors.New("builder pool does not use round robin scheduling")
	}
	var item moduleapi.BuilderInstance
	var labels []byte
	err = tx.QueryRowContext(ctx, `SELECT i.instance_id, p.profile_id, i.runtime_target_id, i.status, i.labels_json, p.driver_ref, p.driver_version
FROM build_builder_pool_members m
JOIN build_builder_instances i ON i.id = m.instance_id AND i.deleted_at = 0
JOIN build_builder_profiles p ON p.id = i.profile_id AND p.deleted_at = 0
WHERE m.pool_id = $1 AND m.deleted_at = 0 AND i.status = 'ready'
ORDER BY m.priority ASC, i.instance_id ASC
OFFSET ($2 % NULLIF((SELECT COUNT(*) FROM build_builder_pool_members m2 JOIN build_builder_instances i2 ON i2.id = m2.instance_id AND i2.deleted_at = 0 WHERE m2.pool_id = $1 AND m2.deleted_at = 0 AND i2.status = 'ready'), 0)) LIMIT 1`, poolPK, cursor).Scan(&item.ID, &item.ProfileID, &item.RuntimeTargetID, &item.Status, &labels, &item.DriverRef, &item.DriverVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.BuilderPoolSelection{}, ErrNotFound
	}
	if err != nil {
		return moduleapi.BuilderPoolSelection{}, fmt.Errorf("select builder pool member: %w", err)
	}
	if err := json.Unmarshal(labels, &item.Labels); err != nil {
		return moduleapi.BuilderPoolSelection{}, fmt.Errorf("decode selected builder labels: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE build_builder_pools SET selection_cursor = selection_cursor + 1, updated_at = NOW() WHERE id = $1`, poolPK); err != nil {
		return moduleapi.BuilderPoolSelection{}, fmt.Errorf("advance builder pool cursor: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return moduleapi.BuilderPoolSelection{}, fmt.Errorf("commit builder selection: %w", err)
	}
	committed = true
	return moduleapi.BuilderPoolSelection{Instance: item, Cursor: &cursor}, nil
}

func normalizeBuilderProfile(profile moduleapi.BuilderProfile) (moduleapi.BuilderProfile, error) {
	profile.ID, profile.DisplayName = strings.TrimSpace(profile.ID), strings.TrimSpace(profile.DisplayName)
	profile.DriverRef, profile.DriverVersion = strings.TrimSpace(profile.DriverRef), strings.TrimSpace(profile.DriverVersion)
	if profile.ID == "" || profile.DisplayName == "" || profile.DriverRef == "" || profile.DriverVersion == "" {
		return moduleapi.BuilderProfile{}, errors.New("invalid builder profile")
	}
	if len(profile.Policy) == 0 {
		profile.Policy = json.RawMessage(`{}`)
	}
	if !json.Valid(profile.Policy) {
		return moduleapi.BuilderProfile{}, errors.New("invalid builder profile policy")
	}
	return profile, nil
}

func normalizeBuilderInstance(instance moduleapi.BuilderInstance) (moduleapi.BuilderInstance, []byte, error) {
	instance.ID, instance.ProfileID, instance.Status = strings.TrimSpace(instance.ID), strings.TrimSpace(instance.ProfileID), strings.TrimSpace(instance.Status)
	if instance.ID == "" || instance.ProfileID == "" || instance.RuntimeTargetID < 1 || !validBuilderInstanceStatus(instance.Status) {
		return moduleapi.BuilderInstance{}, nil, errors.New("invalid builder instance")
	}
	if instance.Labels == nil {
		instance.Labels = map[string]string{}
	}
	labels, err := json.Marshal(instance.Labels)
	if err != nil {
		return moduleapi.BuilderInstance{}, nil, fmt.Errorf("encode builder instance labels: %w", err)
	}
	return instance, labels, nil
}

func normalizeBuilderPool(pool moduleapi.BuilderPool) (moduleapi.BuilderPool, error) {
	pool.ID, pool.DisplayName, pool.SchedulingPolicy = strings.TrimSpace(pool.ID), strings.TrimSpace(pool.DisplayName), strings.TrimSpace(pool.SchedulingPolicy)
	if pool.ID == "" || pool.DisplayName == "" || !validBuilderPoolPolicy(pool.SchedulingPolicy) {
		return moduleapi.BuilderPool{}, errors.New("invalid builder pool")
	}
	if len(pool.Selector) == 0 {
		pool.Selector = json.RawMessage(`{}`)
	}
	if !json.Valid(pool.Selector) {
		return moduleapi.BuilderPool{}, errors.New("invalid builder pool selector")
	}
	return pool, nil
}

func validBuilderInstanceStatus(status string) bool {
	return status == "pending" || status == "ready" || status == "draining" || status == "unavailable"
}
func validBuilderPoolPolicy(policy string) bool {
	return policy == "manual" || policy == "round_robin" || policy == "random" || policy == "least_load" || policy == "capacity" || policy == "affinity"
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
	return ErrConflict
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
	return value.TaskID != 0 && value.BuildID != "" && len(value.BuildArgs) == 0
}

func insertSubmissionJob(ctx context.Context, tx *sql.Tx, value JobSnapshot) (uint64, bool, error) {
	var jobID uint64
	var created bool
	err := tx.QueryRowContext(ctx, `INSERT INTO build_jobs (build_id, submission_id, task_id, application_id, application_record_id, application_name_snapshot, workspace_context_path, workspace_root, dockerfile_path, runtime_target_id, runtime_target_name, runtime_provider, executor_kind, image_repository, image_tag, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NULLIF($16, 0))
		ON CONFLICT (submission_id) WHERE submission_id IS NOT NULL DO UPDATE SET task_id = EXCLUDED.task_id
		RETURNING id, (xmax = 0)`, value.BuildID, value.SubmissionID, value.TaskID, value.ApplicationID, value.ApplicationRecordID, value.ApplicationName, value.ContextPath, value.WorkspaceRoot, value.DockerfilePath, value.RuntimeTargetID, value.RuntimeTargetName, value.RuntimeProvider, "dockerfile", value.ImageRepository, value.ImageTag, value.RequestedBy).Scan(&jobID, &created)
	if err != nil {
		return 0, false, fmt.Errorf("insert build submission snapshot: %w", err)
	}
	return jobID, created, nil
}

func insertJob(ctx context.Context, tx *sql.Tx, value JobSnapshot) (uint64, bool, error) {
	var jobID uint64
	var created bool
	err := tx.QueryRowContext(ctx, `INSERT INTO build_jobs (build_id, task_id, application_id, application_record_id, application_name_snapshot, workspace_context_path, workspace_root, dockerfile_path, runtime_target_id, runtime_target_name, runtime_provider, executor_kind, image_repository, image_tag, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15, 0))
		ON CONFLICT (task_id) DO UPDATE SET task_id = EXCLUDED.task_id
		RETURNING id, (xmax = 0)`, value.BuildID, value.TaskID, value.ApplicationID, value.ApplicationRecordID, value.ApplicationName, value.ContextPath, value.WorkspaceRoot, value.DockerfilePath, value.RuntimeTargetID, value.RuntimeTargetName, value.RuntimeProvider, "dockerfile", value.ImageRepository, value.ImageTag, value.RequestedBy).Scan(&jobID, &created)
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

func insertBuildArgs(_ context.Context, _ *sql.Tx, _ uint64, args []moduleapi.BuildArgument) error {
	if len(args) != 0 {
		return errors.New("persisted build arguments are not supported")
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
	err := query.QueryRowContext(ctx, `SELECT build_id, task_id, application_id, application_record_id, application_name_snapshot, workspace_context_path, workspace_root, dockerfile_path, runtime_target_id, COALESCE(runtime_target_name, ''), runtime_provider, image_repository, image_tag, COALESCE(created_by, 0)
		FROM build_jobs WHERE task_id = $1`, taskID).Scan(&value.BuildID, &value.TaskID, &value.ApplicationID, &value.ApplicationRecordID, &value.ApplicationName, &value.ContextPath, &value.WorkspaceRoot, &value.DockerfilePath, &value.RuntimeTargetID, &value.RuntimeTargetName, &value.RuntimeProvider, &value.ImageRepository, &value.ImageTag, &value.RequestedBy)
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

func (r *SQLRepository) listBuildArgs(ctx context.Context, query jobQueryer, taskID uint64) ([]moduleapi.BuildArgument, error) {
	return listBuildArgs(ctx, query, taskID)
}

func listBuildArgs(ctx context.Context, query jobQueryer, taskID uint64) (args []moduleapi.BuildArgument, err error) {
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
		var arg moduleapi.BuildArgument
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

// SettleBuildArtifact 为 Build 作业写入或更新唯一主 Docker 产物。
func (r *SQLRepository) SettleBuildArtifact(ctx context.Context, taskID uint64, result moduleapi.BuildArtifactResult) error {
	if r == nil || r.db == nil || result.ImageID == "" {
		return errors.New("build artifact result has no image id")
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
