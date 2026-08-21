// Package store 负责 runtime-target 的持久化查询。
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

// ErrNotFound 表示查询不到未软删除的运行时目标。
var ErrNotFound = errors.New("runtime target not found")

// ErrAssignmentRevisionConflict 表示授权替换使用了过期的读取版本。
var ErrAssignmentRevisionConflict = errors.New("runtime target assignment revision conflict")

// ErrUnavailable 表示目标记录存在但当前不可用于 provider 执行。
var ErrUnavailable = errors.New("runtime target unavailable")

// ErrTelemetryRejected 表示报告无法通过已绑定 Agent 的原子重放与身份校验。
var ErrTelemetryRejected = errors.New("builder telemetry report rejected")

// ErrLegacyAgentTrustDisabled 表示 Ed25519 遥测绑定已退役，不能再驱动 Agent 信任或动态准入。
var ErrLegacyAgentTrustDisabled = errors.New("legacy builder telemetry trust is disabled")

// ErrBuilderLedgerRejected 表示构建代理执行账本拒绝了不合法或超出容量的状态转换。
var ErrBuilderLedgerRejected = errors.New("builder agent execution ledger rejected")

// LocalDockerProbe 记录一次受限的本机 Docker 探测结果；调用方应保留探测时间和失败原因，供运行时目标状态查询使用。
type LocalDockerProbe struct {
	Endpoint  string
	Available bool
	Error     string
	CheckedAt time.Time
}

// SQLRepository 持久化运行时目标记录，并将已软删除记录排除在公开查询之外。
type SQLRepository struct{ db *sql.DB }

// AssignmentBatchAction identifies an atomic assignment mutation.
type AssignmentBatchAction string

const (
	// AssignmentBatchGrant adds active assignments.
	AssignmentBatchGrant AssignmentBatchAction = "grant"
	// AssignmentBatchRevoke removes active assignments.
	AssignmentBatchRevoke AssignmentBatchAction = "revoke"
)

type transactionContextKey struct{}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// TransactionRunner 为需要将运行时目标事实和 durable event 一起提交的调用方提供受限事务边界。
//
// callback 收到的 context 已绑定当前事务；仓储写入会复用该事务，调用方只能将 tx
// 交给 event.TransactionalPublisher，不能自行提交或回滚。
type TransactionRunner interface {
	RunInTransaction(context.Context, func(context.Context, *sql.Tx) error) error
}

// NewSQLRepository 创建由模块拥有的 SQL 仓储。
func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

// RunInTransaction 将使用同一仓储的写入与 durable event 固定在同一个 SQL transaction 中。
// callback 失败会回滚；只有业务写入与事件写入均成功时才提交。嵌套调用会复用
// context 中的外层事务，避免内层独立提交。
func (r *SQLRepository) RunInTransaction(ctx context.Context, callback func(context.Context, *sql.Tx) error) error {
	if r == nil || r.db == nil {
		return errors.New("runtime target repository is unavailable")
	}
	if callback == nil {
		return errors.New("runtime target transaction callback is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if tx := transactionFromContext(ctx); tx != nil {
		return callback(ctx, tx)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin runtime target transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := callback(context.WithValue(ctx, transactionContextKey{}, tx), tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit runtime target transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *SQLRepository) executor(ctx context.Context) sqlExecutor {
	if tx := transactionFromContext(ctx); tx != nil {
		return tx
	}
	return r.db
}

func transactionFromContext(ctx context.Context) *sql.Tx {
	if ctx == nil {
		return nil
	}
	tx, _ := ctx.Value(transactionContextKey{}).(*sql.Tx)
	return tx
}

// Target 是运行时目标的持久化读取投影；Capabilities 是 provider-neutral 的能力集合，供上层筛选可执行能力。
type Target struct {
	ID             uint64     `json:"id"`
	Provider       string     `json:"provider"`
	DisplayName    string     `json:"displayName"`
	EndpointLabel  string     `json:"endpointLabel"`
	ConnectionKind string     `json:"connectionKind"`
	Capabilities   []string   `json:"capabilities"`
	Availability   bool       `json:"availability"`
	LastError      string     `json:"lastError"`
	CheckedAt      *time.Time `json:"checkedAt"`
}

// DockerTargetConnection 是 Runtime Target 交给 provider adapter 的私有连接事实。
// 它只允许在 runtime-target 与 provider 实现之间流转，不应进入 moduleapi、Build plan 或 HTTP 投影。
type DockerTargetConnection struct {
	TargetID       uint64
	Endpoint       string
	ConnectionKind string
}

// BuilderTelemetryObservation 是 Builder Agent 或控制平面写入的持久化观测事实。
// 它不保存连接端点、凭据或 Docker/主机指标，只保留 Builder 范围的调度输入和可验证出处。
type BuilderTelemetryObservation struct {
	TargetID              int64
	AgentID               string
	TelemetrySequence     int64
	BuilderScope          string
	ProviderID            string
	CapabilityProfile     string
	CapabilityVersion     string
	AffinityKey           string
	Available             bool
	Running               int
	Queued                int
	AllocatableSlots      int
	ObservedAt            time.Time
	ExpiresAt             time.Time
	SourceRef             string
	Provenance            string
	Integrity             string
	UnsupportedDimensions []string
}

// BuilderTelemetryAgent 是获准向指定运行目标提交遥测的 Builder Agent 身份。
// PublicKey 只用于验证报告签名，私钥永远不进入 Runtime Target 持久化或 Build 边界。
type BuilderTelemetryAgent struct {
	TargetID          int64
	AgentID           string
	ProviderID        string
	BuilderScope      string
	CapabilityProfile string
	CapabilityVersion string
	PublicKey         []byte
	LastSequence      int64
	Enabled           bool
}

// BuilderAgentLedgerState 是 Runtime Target 持有的构建代理执行账本快照。
type BuilderAgentLedgerState struct {
	TargetID          int64
	AgentID           string
	SlotBudget        int
	Queued            int
	Running           int
	TelemetrySequence int64
}

// EnsureBuilderAgentLedger 创建或更新一个代理的容量预算，但不会重置已有执行计数或遥测序列。
func (r *SQLRepository) EnsureBuilderAgentLedger(ctx context.Context, targetID int64, agentID string, slotBudget int) error {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" || slotBudget < 1 {
		return ErrBuilderLedgerRejected
	}
	result, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_target_builder_execution_ledgers (runtime_target_id, agent_id, slot_budget, queued_builds, running_builds, telemetry_sequence) VALUES ($1,$2,$3,0,0,0) ON CONFLICT (runtime_target_id, agent_id) DO UPDATE SET slot_budget = EXCLUDED.slot_budget, updated_at = CURRENT_TIMESTAMP WHERE runtime_target_builder_execution_ledgers.running_builds <= EXCLUDED.slot_budget`, targetID, agentID, slotBudget)
	if err != nil {
		return fmt.Errorf("ensure builder execution ledger: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read builder execution ledger result: %w", err)
	}
	if affected != 1 {
		return ErrBuilderLedgerRejected
	}
	return nil
}

// QueueBuilderAgentBuild 将一个构建加入代理自有排队账本。
func (r *SQLRepository) QueueBuilderAgentBuild(ctx context.Context, targetID int64, agentID string) error {
	return r.transitionBuilderAgentLedger(ctx, targetID, agentID, `queued_builds = queued_builds + 1`, "queue builder execution", "")
}

// StartBuilderAgentBuild 原子地将排队构建转为运行构建，并核对 slot budget。
func (r *SQLRepository) StartBuilderAgentBuild(ctx context.Context, targetID int64, agentID string) error {
	return r.transitionBuilderAgentLedger(ctx, targetID, agentID, `queued_builds = queued_builds - 1, running_builds = running_builds + 1`, "start builder execution", "queued_builds > 0 AND running_builds < slot_budget")
}

// CancelQueuedBuilderAgentBuild 移除尚未启动的构建，避免容量裁决失败留下幽灵排队计数。
func (r *SQLRepository) CancelQueuedBuilderAgentBuild(ctx context.Context, targetID int64, agentID string) error {
	return r.transitionBuilderAgentLedger(ctx, targetID, agentID, `queued_builds = queued_builds - 1`, "cancel queued builder execution", "queued_builds > 0")
}

// FinishBuilderAgentBuild 原子地结算一个运行构建。
func (r *SQLRepository) FinishBuilderAgentBuild(ctx context.Context, targetID int64, agentID string) error {
	return r.transitionBuilderAgentLedger(ctx, targetID, agentID, `running_builds = running_builds - 1`, "finish builder execution", "running_builds > 0")
}

func (r *SQLRepository) transitionBuilderAgentLedger(ctx context.Context, targetID int64, agentID, assignments, operation, predicate string) error {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" {
		return ErrBuilderLedgerRejected
	}
	where := "runtime_target_id = $1 AND agent_id = $2"
	if predicate != "" {
		where += " AND " + predicate
	}
	return r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		result, err := r.executor(txCtx).ExecContext(txCtx, "UPDATE runtime_target_builder_execution_ledgers SET "+assignments+", updated_at = CURRENT_TIMESTAMP WHERE "+where, targetID, agentID)
		if err != nil {
			return fmt.Errorf("%s: %w", operation, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("%s rows affected: %w", operation, err)
		}
		if affected != 1 {
			return ErrBuilderLedgerRejected
		}
		return nil
	})
}

// SnapshotBuilderAgentLedger 返回当前受控执行账本状态。
func (r *SQLRepository) SnapshotBuilderAgentLedger(ctx context.Context, targetID int64, agentID string) (BuilderAgentLedgerState, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" {
		return BuilderAgentLedgerState{}, ErrBuilderLedgerRejected
	}
	var state BuilderAgentLedgerState
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT runtime_target_id, agent_id, slot_budget, queued_builds, running_builds, telemetry_sequence FROM runtime_target_builder_execution_ledgers WHERE runtime_target_id = $1 AND agent_id = $2`, targetID, agentID).Scan(&state.TargetID, &state.AgentID, &state.SlotBudget, &state.Queued, &state.Running, &state.TelemetrySequence)
	if errors.Is(err, sql.ErrNoRows) {
		return BuilderAgentLedgerState{}, ErrBuilderLedgerRejected
	}
	if err != nil {
		return BuilderAgentLedgerState{}, fmt.Errorf("snapshot builder execution ledger: %w", err)
	}
	return state, nil
}

// AdvanceBuilderAgentTelemetry 原子推进遥测序列并返回同一时刻的执行账本快照。
func (r *SQLRepository) AdvanceBuilderAgentTelemetry(ctx context.Context, targetID int64, agentID string) (BuilderAgentLedgerState, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" {
		return BuilderAgentLedgerState{}, ErrBuilderLedgerRejected
	}
	var state BuilderAgentLedgerState
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		result, err := r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_builder_execution_ledgers SET telemetry_sequence = telemetry_sequence + 1, updated_at = CURRENT_TIMESTAMP WHERE runtime_target_id = $1 AND agent_id = $2`, targetID, agentID)
		if err != nil {
			return fmt.Errorf("advance builder telemetry sequence: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return ErrBuilderLedgerRejected
		}
		state, err = r.SnapshotBuilderAgentLedger(txCtx, targetID, agentID)
		return err
	})
	return state, err
}

// UserAssignment 表示一条有效部署使用授权，不携带运行目标凭据。
type UserAssignment struct {
	TargetID  uint64
	UserID    uint64
	CreatedAt time.Time
	CreatedBy uint64
}

// Page 表示一个稳定的运行时目标分页窗口；Total 与 Items 使用同一份未软删除数据集计算。
type Page struct {
	Items   []Target
	Total   int64
	Summary Summary
}

// ListQuery 描述运行目标目录允许的服务端筛选与单排序条件。
type ListQuery struct {
	Limit, Offset                                   int
	Keyword, Provider, ConnectionKind, Health, Sort string
}

const targetListPaginationArgumentCount = 2

// Summary 是未软删除运行时目标的全量健康聚合，不受当前分页窗口影响。
type Summary struct {
	Total       int64
	Healthy     int64
	Unavailable int64
}

const readRuntimeTargetSummarySQL = `SELECT
COUNT(*),
COUNT(*) FILTER (WHERE availability = true),
COUNT(*) FILTER (WHERE availability = false)
FROM runtime_targets
WHERE deleted_at = 0`

// ReadSummary 返回未软删除运行目标的目录健康聚合；查询不访问远端 provider。
func (r *SQLRepository) ReadSummary(ctx context.Context) (Summary, error) {
	if r == nil || r.db == nil {
		return Summary{}, errors.New("runtime target repository is unavailable")
	}
	var summary Summary
	err := r.executor(ctx).QueryRowContext(ctx, readRuntimeTargetSummarySQL).Scan(
		&summary.Total,
		&summary.Healthy,
		&summary.Unavailable,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("summarize runtime targets: %w", err)
	}
	return summary, nil
}

// List 返回全部未软删除的运行时目标，并按 provider、显示名称和 ID 排序以保证跨请求顺序稳定。
func (r *SQLRepository) List(ctx context.Context) ([]Target, error) {
	if r == nil || r.db == nil {
		return []Target{}, nil
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE deleted_at = 0 ORDER BY provider, display_name, id`)
	if err != nil {
		return nil, err
	}
	return scanTargets(rows)
}

// ListPage 返回一个未软删除的运行时目标分页窗口及总数；limit 和 offset 由调用方负责先行校验。
func (r *SQLRepository) ListPage(ctx context.Context, limit, offset int) (Page, error) {
	return r.ListQueryPage(ctx, ListQuery{Limit: limit, Offset: offset, Sort: "display_name:asc"})
}

// ListQueryPage 返回过滤结果和未过滤 fleet 摘要；实时资源指标不参与此查询。
func (r *SQLRepository) ListQueryPage(ctx context.Context, query ListQuery) (Page, error) {
	if r == nil || r.db == nil {
		return Page{Items: []Target{}}, nil
	}
	summary, err := r.ReadSummary(ctx)
	if err != nil {
		return Page{}, err
	}
	where, args := targetListFilters(query)
	countSQL := `SELECT COUNT(*) FROM runtime_targets WHERE ` + strings.Join(where, " AND ")
	var total int64
	if err := r.executor(ctx).QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page{}, err
	}
	itemsSQL := `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + targetListOrder(query.Sort) + ` LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+targetListPaginationArgumentCount)
	args = append(args, query.Limit, query.Offset)
	rows, err := r.executor(ctx).QueryContext(ctx, itemsSQL, args...)
	if err != nil {
		return Page{}, err
	}
	items, err := scanTargets(rows)
	if err != nil {
		return Page{}, err
	}
	return Page{Items: items, Total: total, Summary: summary}, nil
}

func targetListFilters(query ListQuery) ([]string, []any) {
	where, args := []string{"deleted_at = 0"}, []any{}
	if value := strings.TrimSpace(query.Keyword); value != "" {
		where = append(where, "(LOWER(display_name) LIKE LOWER($1) OR LOWER(endpoint_label) LIKE LOWER($1))")
		args = append(args, "%"+value+"%")
	}
	next := len(args) + 1
	if value := strings.TrimSpace(query.Provider); value != "" {
		where = append(where, "provider = $"+fmt.Sprint(next))
		args = append(args, value)
		next++
	}
	if value := strings.TrimSpace(query.ConnectionKind); value != "" {
		where = append(where, "connection_kind = $"+fmt.Sprint(next))
		args = append(args, value)
		next++
	}
	if value := strings.TrimSpace(query.Health); value != "" {
		where = append(where, "availability = $"+fmt.Sprint(next))
		args = append(args, value == "healthy")
	}
	return where, args
}
func targetListOrder(sort string) string {
	switch sort {
	case "display_name:desc":
		return "display_name DESC, id DESC"
	case "provider:asc":
		return "provider ASC, display_name ASC, id ASC"
	case "provider:desc":
		return "provider DESC, display_name DESC, id DESC"
	case "health:asc":
		return "availability ASC, display_name ASC, id ASC"
	case "health:desc":
		return "availability DESC, display_name ASC, id ASC"
	default:
		return "display_name ASC, id ASC"
	}
}

// scanTargets 读取 runtime-target 列表投影，并始终关闭结果集以释放数据库资源。
func scanTargets(rows *sql.Rows) ([]Target, error) {
	defer func() { _ = rows.Close() }()
	items := []Target{}
	for rows.Next() {
		var item Target
		var capabilities []byte
		if err := rows.Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// FindSystemLocalDocker 返回此前发现的系统托管本机 Docker 记录；未发现时统一返回 ErrNotFound。
func (r *SQLRepository) FindSystemLocalDocker(ctx context.Context) (Target, error) {
	if r == nil || r.db == nil {
		return Target{}, ErrNotFound
	}
	var item Target
	var capabilities []byte
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE provider = 'docker' AND endpoint = 'unix:///var/run/docker.sock' AND system_managed = true AND deleted_at = 0`).Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrNotFound
	}
	if err != nil {
		return Target{}, err
	}
	if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
		return Target{}, err
	}
	return item, nil
}

// Get 按 ID 返回一个未软删除的运行时目标；记录不存在时统一返回 ErrNotFound。
func (r *SQLRepository) Get(ctx context.Context, id uint64) (Target, error) {
	if r == nil || r.db == nil {
		return Target{}, ErrNotFound
	}
	var item Target
	var capabilities []byte
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE id = $1 AND deleted_at = 0`, id).Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrNotFound
	}
	if err != nil {
		return Target{}, err
	}
	if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
		return Target{}, err
	}
	return item, nil
}

// RecordBuilderTelemetryObservation 追加控制平面已验证的 Builder 遥测观察。
// 该仓储写入仅供 Runtime Target 内部的 Builder Agent/控制平面接入使用，不是 HTTP 或 Build 写入边界。
func (r *SQLRepository) RecordBuilderTelemetryObservation(ctx context.Context, observation BuilderTelemetryObservation) error {
	if r == nil || r.db == nil || !validBuilderTelemetryObservation(observation) {
		return errors.New("invalid builder telemetry observation")
	}
	unsupported, err := json.Marshal(observation.UnsupportedDimensions)
	if err != nil {
		return fmt.Errorf("encode unsupported builder telemetry dimensions: %w", err)
	}
	_, err = r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_target_builder_telemetry_observations (runtime_target_id, agent_id, telemetry_sequence, builder_scope, provider_id, capability_profile, capability_version, affinity_key, available, running_builds, queued_builds, allocatable_slots, observed_at, expires_at, source_ref, provenance, integrity, unsupported_dimensions_json) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`, observation.TargetID, observation.AgentID, observation.TelemetrySequence, observation.BuilderScope, observation.ProviderID, observation.CapabilityProfile, observation.CapabilityVersion, observation.AffinityKey, observation.Available, observation.Running, observation.Queued, observation.AllocatableSlots, observation.ObservedAt, observation.ExpiresAt, observation.SourceRef, observation.Provenance, observation.Integrity, unsupported)
	if err != nil {
		return fmt.Errorf("record builder telemetry observation: %w", err)
	}
	return nil
}

// UpsertBuilderTelemetryAgent 由 Runtime Target 控制平面配置已绑定 Builder Agent 的验证公钥。
// 该方法不是 HTTP 或 Build API；Agent 只能使用已登记公钥签名报告，不能自行注册身份。
func (r *SQLRepository) UpsertBuilderTelemetryAgent(ctx context.Context, agent BuilderTelemetryAgent) error {
	if r == nil || r.db == nil || !validBuilderTelemetryAgent(agent) {
		return errors.New("invalid builder telemetry agent")
	}
	result, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_target_builder_telemetry_agents (runtime_target_id, agent_id, provider_id, builder_scope, capability_profile, capability_version, public_key, last_sequence, enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,0,$8) ON CONFLICT (runtime_target_id, agent_id) DO UPDATE SET provider_id = EXCLUDED.provider_id, builder_scope = EXCLUDED.builder_scope, capability_profile = EXCLUDED.capability_profile, capability_version = EXCLUDED.capability_version, public_key = EXCLUDED.public_key, last_sequence = CASE WHEN runtime_target_builder_telemetry_agents.public_key = EXCLUDED.public_key THEN runtime_target_builder_telemetry_agents.last_sequence ELSE 0 END, enabled = EXCLUDED.enabled WHERE runtime_target_builder_telemetry_agents.public_key = EXCLUDED.public_key OR runtime_target_builder_telemetry_agents.enabled = false`, agent.TargetID, agent.AgentID, agent.ProviderID, agent.BuilderScope, agent.CapabilityProfile, agent.CapabilityVersion, agent.PublicKey, agent.Enabled)
	if err != nil {
		return fmt.Errorf("upsert builder telemetry agent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read builder telemetry agent upsert result: %w", err)
	}
	if affected == 0 {
		return errors.New("builder telemetry key rotation requires disabled agent binding")
	}
	return nil
}

// ReadLegacyBuilderTelemetryAgent 只读返回已退役的 Ed25519 遥测绑定，供审计诊断使用。
// 返回值永远不得用于 Agent 信任、遥测准入或动态放置。
func (r *SQLRepository) ReadLegacyBuilderTelemetryAgent(ctx context.Context, targetID int64, agentID string) (BuilderTelemetryAgent, error) {
	if r == nil || r.db == nil || targetID < 1 || strings.TrimSpace(agentID) == "" {
		return BuilderTelemetryAgent{}, ErrNotFound
	}
	var agent BuilderTelemetryAgent
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT runtime_target_id, agent_id, provider_id, builder_scope, capability_profile, capability_version, public_key, last_sequence, enabled FROM runtime_target_builder_telemetry_agents WHERE runtime_target_id = $1 AND agent_id = $2`, targetID, agentID).Scan(&agent.TargetID, &agent.AgentID, &agent.ProviderID, &agent.BuilderScope, &agent.CapabilityProfile, &agent.CapabilityVersion, &agent.PublicKey, &agent.LastSequence, &agent.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return BuilderTelemetryAgent{}, ErrNotFound
	}
	if err != nil {
		return BuilderTelemetryAgent{}, fmt.Errorf("read legacy builder telemetry agent: %w", err)
	}
	return agent, nil
}

// GetBuilderTelemetryAgent 拒绝将历史 Ed25519 绑定作为受信任 Agent 身份读取。
func (r *SQLRepository) GetBuilderTelemetryAgent(_ context.Context, _ int64, _ string) (BuilderTelemetryAgent, error) {
	return BuilderTelemetryAgent{}, ErrLegacyAgentTrustDisabled
}

// GetActiveDockerBuilderTelemetryAgent 返回目标唯一启用的 Docker Agent 绑定。
// Build 的动态 Placement 目前只携带 Runtime Target 标识，因此一个目标不能同时暴露多个可执行 Builder scope。
func (r *SQLRepository) GetActiveDockerBuilderTelemetryAgent(_ context.Context, _ int64) (BuilderTelemetryAgent, error) {
	return BuilderTelemetryAgent{}, ErrLegacyAgentTrustDisabled
}

// RecordBoundBuilderTelemetryObservation 将已经验签的报告与单调序列原子写入。
// 先更新绑定行可防止同一 Agent 的并发或重放报告超过最后已接受的序列。
func (r *SQLRepository) RecordBoundBuilderTelemetryObservation(_ context.Context, _ BuilderTelemetryAgent, _ int64, _ BuilderTelemetryObservation) error {
	return ErrLegacyAgentTrustDisabled
}

// ListLatestBuilderTelemetry 返回每个目标最新的一条控制平面 Builder 观测。
// 缺失记录由调用方按 fail-closed 处理，不能回退到 Docker 或宿主机诊断指标。
func (r *SQLRepository) ListLatestBuilderTelemetry(ctx context.Context, targetIDs []int64) ([]BuilderTelemetryObservation, error) {
	if r == nil || r.db == nil || len(targetIDs) == 0 {
		return []BuilderTelemetryObservation{}, nil
	}
	placeholders, args, err := builderTelemetryTargetArguments(targetIDs)
	if err != nil {
		return nil, err
	}
	query := latestBuilderTelemetryQuery(placeholders)
	rows, err := r.executor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list latest builder telemetry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanBuilderTelemetryObservations(rows, len(targetIDs))
}

func validBuilderTelemetryObservation(observation BuilderTelemetryObservation) bool {
	return builderTelemetryIdentityValid(observation) && builderTelemetryCapacityValid(observation) && builderTelemetryWindowValid(observation) && builderTelemetryEvidenceValid(observation)
}

func builderTelemetryTargetArguments(targetIDs []int64) ([]string, []any, error) {
	placeholders := make([]string, 0, len(targetIDs))
	args := make([]any, 0, len(targetIDs))
	for index, targetID := range targetIDs {
		if targetID < 1 {
			return nil, nil, errors.New("invalid builder telemetry target id")
		}
		placeholders = append(placeholders, "$"+strconv.Itoa(index+1))
		args = append(args, targetID)
	}
	return placeholders, args, nil
}

func latestBuilderTelemetryQuery(placeholders []string) string {
	return `SELECT runtime_target_id, agent_id, telemetry_sequence, builder_scope, provider_id, capability_profile, capability_version, affinity_key, available, running_builds, queued_builds, allocatable_slots, observed_at, expires_at, source_ref, provenance, integrity, unsupported_dimensions_json FROM (SELECT runtime_target_id, agent_id, telemetry_sequence, builder_scope, provider_id, capability_profile, capability_version, affinity_key, available, running_builds, queued_builds, allocatable_slots, observed_at, expires_at, source_ref, provenance, integrity, unsupported_dimensions_json, ROW_NUMBER() OVER (PARTITION BY runtime_target_id ORDER BY observed_at DESC, id DESC) AS observation_rank FROM runtime_target_builder_telemetry_observations WHERE runtime_target_id IN (` + strings.Join(placeholders, ",") + `)) latest WHERE observation_rank = 1 ORDER BY runtime_target_id`
}

func scanBuilderTelemetryObservations(rows *sql.Rows, capacity int) ([]BuilderTelemetryObservation, error) {
	results := make([]BuilderTelemetryObservation, 0, capacity)
	for rows.Next() {
		observation, err := scanBuilderTelemetryObservation(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, observation)
	}
	return results, rows.Err()
}

func scanBuilderTelemetryObservation(rows *sql.Rows) (BuilderTelemetryObservation, error) {
	var observation BuilderTelemetryObservation
	var unsupported []byte
	if err := rows.Scan(&observation.TargetID, &observation.AgentID, &observation.TelemetrySequence, &observation.BuilderScope, &observation.ProviderID, &observation.CapabilityProfile, &observation.CapabilityVersion, &observation.AffinityKey, &observation.Available, &observation.Running, &observation.Queued, &observation.AllocatableSlots, &observation.ObservedAt, &observation.ExpiresAt, &observation.SourceRef, &observation.Provenance, &observation.Integrity, &unsupported); err != nil {
		return BuilderTelemetryObservation{}, fmt.Errorf("scan latest builder telemetry: %w", err)
	}
	if err := json.Unmarshal(unsupported, &observation.UnsupportedDimensions); err != nil {
		return BuilderTelemetryObservation{}, fmt.Errorf("decode unsupported builder telemetry dimensions: %w", err)
	}
	return observation, nil
}

func builderTelemetryIdentityValid(observation BuilderTelemetryObservation) bool {
	return observation.TargetID > 0 && strings.TrimSpace(observation.BuilderScope) != "" && strings.TrimSpace(observation.ProviderID) != "" && strings.TrimSpace(observation.CapabilityProfile) != "" && strings.TrimSpace(observation.CapabilityVersion) != ""
}

func builderTelemetryCapacityValid(observation BuilderTelemetryObservation) bool {
	return observation.Running >= 0 && observation.Queued >= 0 && observation.AllocatableSlots >= 0
}

func builderTelemetryWindowValid(observation BuilderTelemetryObservation) bool {
	return !observation.ObservedAt.IsZero() && !observation.ExpiresAt.IsZero() && observation.ExpiresAt.After(observation.ObservedAt)
}

func builderTelemetryEvidenceValid(observation BuilderTelemetryObservation) bool {
	return strings.TrimSpace(observation.SourceRef) != "" && strings.TrimSpace(observation.Provenance) != "" && strings.TrimSpace(observation.Integrity) != ""
}

func validBuilderTelemetryAgent(agent BuilderTelemetryAgent) bool {
	return agent.TargetID > 0 && strings.TrimSpace(agent.AgentID) != "" && agent.ProviderID == "docker" && strings.TrimSpace(agent.BuilderScope) != "" && strings.TrimSpace(agent.CapabilityProfile) != "" && strings.TrimSpace(agent.CapabilityVersion) != "" && len(agent.PublicKey) == 32 && agent.LastSequence >= 0
}

// ListBuildTargets 返回仍存活且声明镜像构建能力的 Docker Target；连接事实仍由 provider 私有查询读取。
func (r *SQLRepository) ListBuildTargets(ctx context.Context) ([]Target, error) {
	if r == nil || r.db == nil {
		return []Target{}, nil
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE provider = 'docker' AND deleted_at = 0 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]Target, 0)
	for rows.Next() {
		var item Target
		var capabilities []byte
		if err := rows.Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
			return nil, err
		}
		if containsCapability(item.Capabilities, "image_build") {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetDockerTargetConnection 读取仍存活的 Docker Target 私有连接事实；调用方必须在同一 provider boundary
// 内完成 endpoint 校验和连接使用，不能将结果复制到公共资源投影。
func (r *SQLRepository) GetDockerTargetConnection(ctx context.Context, id uint64) (DockerTargetConnection, error) {
	if r == nil || r.db == nil || id == 0 {
		return DockerTargetConnection{}, ErrNotFound
	}
	var connection DockerTargetConnection
	var available bool
	var capabilities []byte
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT id, endpoint, connection_kind, availability, capabilities_json FROM runtime_targets WHERE id = $1 AND provider = 'docker' AND deleted_at = 0`, id).Scan(&connection.TargetID, &connection.Endpoint, &connection.ConnectionKind, &available, &capabilities)
	if errors.Is(err, sql.ErrNoRows) {
		return DockerTargetConnection{}, ErrNotFound
	}
	if err != nil {
		return DockerTargetConnection{}, fmt.Errorf("get Docker target connection: %w", err)
	}
	if !available {
		return DockerTargetConnection{}, ErrUnavailable
	}
	var capabilityNames []string
	if err := json.Unmarshal(capabilities, &capabilityNames); err != nil {
		return DockerTargetConnection{}, fmt.Errorf("decode Docker target capabilities: %w", err)
	}
	if !containsCapability(capabilityNames, "image_build") {
		return DockerTargetConnection{}, ErrUnavailable
	}
	if err := validateDockerEndpoint(connection.Endpoint, connection.ConnectionKind); err != nil {
		return DockerTargetConnection{}, err
	}
	return connection, nil
}

// GetProviderConnection 读取任意 Build-capable Runtime Target 的私有连接事实；provider 适配器必须在本边界内消费 endpoint。
//
//nolint:cyclop // 连接读取在同一私有边界内完成存活、能力、格式和 provider 身份校验。
func (r *SQLRepository) GetProviderConnection(ctx context.Context, targetID int64) (moduleapi.RuntimeTargetProviderConnection, error) {
	if r == nil || r.db == nil || targetID < 1 {
		return moduleapi.RuntimeTargetProviderConnection{}, ErrNotFound
	}
	var connection moduleapi.RuntimeTargetProviderConnection
	var available bool
	var capabilities []byte
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT id, provider, endpoint, connection_kind, availability, capabilities_json FROM runtime_targets WHERE id = $1 AND deleted_at = 0`, targetID).Scan(&connection.TargetID, &connection.Provider, &connection.Endpoint, &connection.ConnectionKind, &available, &capabilities)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.RuntimeTargetProviderConnection{}, ErrNotFound
	}
	if err != nil {
		return moduleapi.RuntimeTargetProviderConnection{}, fmt.Errorf("get provider connection: %w", err)
	}
	if !available {
		return moduleapi.RuntimeTargetProviderConnection{}, ErrUnavailable
	}
	var capabilityNames []string
	if err := json.Unmarshal(capabilities, &capabilityNames); err != nil {
		return moduleapi.RuntimeTargetProviderConnection{}, fmt.Errorf("decode provider capabilities: %w", err)
	}
	if strings.TrimSpace(connection.Provider) == "" || !containsCapability(capabilityNames, "image_build") || !validProviderEndpoint(connection.Endpoint) {
		return moduleapi.RuntimeTargetProviderConnection{}, ErrUnavailable
	}
	return connection, nil
}

func validProviderEndpoint(endpoint string) bool {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	return err == nil && parsed.Scheme != "" && parsed.User == nil && !strings.ContainsAny(endpoint, "\x00\r\n")
}

func containsCapability(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validateDockerEndpoint(endpoint, connectionKind string) error {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Scheme == "" || parsed.User != nil || strings.ContainsAny(endpoint, "\x00\r\n") {
		return errors.New("runtime target Docker endpoint is invalid")
	}
	return validateDockerEndpointScheme(parsed, connectionKind)
}

func validateDockerEndpointScheme(parsed *url.URL, connectionKind string) error {
	switch connectionKind {
	case "unix_socket":
		if parsed.Scheme != "unix" || parsed.Path == "" {
			return errors.New("runtime target Docker endpoint does not match connection kind")
		}
	case "tcp", "ssh":
		if parsed.Scheme != connectionKind || parsed.Host == "" {
			return errors.New("runtime target Docker endpoint does not match connection kind")
		}
	default:
		return errors.New("runtime target Docker connection kind is unsupported")
	}
	return nil
}

// ListAssignedComposeTargets 仅返回指定用户获授权且具备 Compose 能力的存活目标。
func (r *SQLRepository) ListAssignedComposeTargets(ctx context.Context, userID uint64) ([]Target, error) {
	if r == nil || r.db == nil || userID == 0 {
		return []Target{}, nil
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT t.id, t.provider, t.display_name, t.endpoint_label, t.connection_kind, t.capabilities_json, t.availability, t.last_error, t.checked_at
		FROM runtime_targets t INNER JOIN runtime_target_user_assignments a ON a.runtime_target_id = t.id
		WHERE t.deleted_at = 0 AND a.deleted_at = 0 AND a.user_id = $1 ORDER BY t.provider, t.display_name, t.id`, userID)
	if err != nil {
		return nil, err
	}
	return scanTargets(rows)
}

// ListAssignedBuildTargets 仅返回指定用户获授权的运行目标；能力与健康筛选由调用方基于当前构建契约完成。
func (r *SQLRepository) ListAssignedBuildTargets(ctx context.Context, userID uint64) ([]Target, error) {
	if r == nil || r.db == nil || userID == 0 {
		return []Target{}, nil
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT t.id, t.provider, t.display_name, t.endpoint_label, t.connection_kind, t.capabilities_json, t.availability, t.last_error, t.checked_at
		FROM runtime_targets t INNER JOIN runtime_target_user_assignments a ON a.runtime_target_id = t.id
		WHERE t.deleted_at = 0 AND a.deleted_at = 0 AND a.user_id = $1 ORDER BY t.provider, t.display_name, t.id`, userID)
	if err != nil {
		return nil, err
	}
	return scanTargets(rows)
}

// HasActiveUserAssignment 判断有效部署使用授权是否存在。
func (r *SQLRepository) HasActiveUserAssignment(ctx context.Context, targetID, userID uint64) (bool, error) {
	if r == nil || r.db == nil || targetID == 0 || userID == 0 {
		return false, nil
	}
	var found bool
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM runtime_target_user_assignments a INNER JOIN runtime_targets t ON t.id = a.runtime_target_id WHERE a.runtime_target_id = $1 AND a.user_id = $2 AND a.deleted_at = 0 AND t.deleted_at = 0)`, targetID, userID).Scan(&found)
	return found, err
}

// ListUserAssignments 按稳定用户顺序返回运行目标的有效授权。
func (r *SQLRepository) ListUserAssignments(ctx context.Context, targetID uint64) ([]UserAssignment, error) {
	if r == nil || r.db == nil || targetID == 0 {
		return []UserAssignment{}, nil
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT runtime_target_id, user_id, created_at, created_by FROM runtime_target_user_assignments WHERE runtime_target_id = $1 AND deleted_at = 0 ORDER BY user_id`, targetID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []UserAssignment{}
	for rows.Next() {
		var item UserAssignment
		if err := rows.Scan(&item.TargetID, &item.UserID, &item.CreatedAt, &item.CreatedBy); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UserAssignmentRevision returns the current optimistic-concurrency revision for a target's assignments.
func (r *SQLRepository) UserAssignmentRevision(ctx context.Context, targetID uint64) (uint64, error) {
	if r == nil || r.db == nil || targetID == 0 {
		return 0, ErrNotFound
	}
	if _, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_target_assignment_revisions (runtime_target_id, revision) VALUES ($1, 1) ON CONFLICT (runtime_target_id) DO NOTHING`, targetID); err != nil {
		return 0, err
	}
	var revision uint64
	err := r.executor(ctx).QueryRowContext(ctx, `SELECT revision FROM runtime_target_assignment_revisions WHERE runtime_target_id = $1`, targetID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return revision, err
}

func (r *SQLRepository) lockAssignmentTarget(ctx context.Context, targetID uint64) error {
	if targetID == 0 {
		return ErrNotFound
	}
	if err := r.executor(ctx).QueryRowContext(ctx, `SELECT id FROM runtime_targets WHERE id = $1 AND deleted_at = 0 FOR UPDATE`, targetID).Scan(new(uint64)); err == nil {
		return nil
	} else if !strings.Contains(strings.ToLower(err.Error()), "syntax") {
		return err
	}
	return r.executor(ctx).QueryRowContext(ctx, `SELECT id FROM runtime_targets WHERE id = $1 AND deleted_at = 0`, targetID).Scan(new(uint64))
}

// GrantUserAssignment 幂等恢复或创建部署使用授权。
func (r *SQLRepository) GrantUserAssignment(ctx context.Context, targetID, userID, actorID uint64) (UserAssignment, error) {
	if r == nil || r.db == nil || targetID == 0 || userID == 0 {
		return UserAssignment{}, ErrNotFound
	}
	var item UserAssignment
	err := r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		if err := r.lockAssignmentTarget(txCtx, targetID); err != nil {
			return err
		}
		result, err := r.executor(txCtx).ExecContext(txCtx, `INSERT INTO runtime_target_user_assignments (runtime_target_id, user_id, created_by, updated_by, deleted_at, deleted_by) VALUES ($1, $2, $3, $3, 0, 0) ON CONFLICT (runtime_target_id, user_id) DO UPDATE SET deleted_at = 0, deleted_by = 0, updated_at = CURRENT_TIMESTAMP, updated_by = EXCLUDED.updated_by WHERE runtime_target_user_assignments.deleted_at <> 0`, targetID, userID, actorID)
		if err != nil {
			return err
		}
		if changed, _ := result.RowsAffected(); changed > 0 {
			if err := r.incrementAssignmentRevision(txCtx, targetID); err != nil {
				return err
			}
		}
		err = r.executor(txCtx).QueryRowContext(txCtx, `SELECT runtime_target_id, user_id, created_at, created_by FROM runtime_target_user_assignments WHERE runtime_target_id = $1 AND user_id = $2 AND deleted_at = 0`, targetID, userID).Scan(&item.TargetID, &item.UserID, &item.CreatedAt, &item.CreatedBy)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return item, err
}

// RevokeUserAssignment 软删除一条有效部署使用授权。
func (r *SQLRepository) RevokeUserAssignment(ctx context.Context, targetID, userID, actorID uint64) error {
	if r == nil || r.db == nil || targetID == 0 || userID == 0 {
		return ErrNotFound
	}
	return r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		if err := r.lockAssignmentTarget(txCtx, targetID); err != nil {
			return err
		}
		active, err := r.HasActiveUserAssignment(txCtx, targetID, userID)
		if err != nil {
			return err
		}
		if !active {
			return ErrNotFound
		}
		if _, err = r.executor(txCtx).ExecContext(txCtx, `UPDATE runtime_target_user_assignments SET deleted_at = $3, deleted_by = $4, updated_at = CURRENT_TIMESTAMP, updated_by = $4 WHERE runtime_target_id = $1 AND user_id = $2 AND deleted_at = 0`, targetID, userID, time.Now().UTC().Unix(), actorID); err != nil {
			return err
		}
		return r.incrementAssignmentRevision(txCtx, targetID)
	})
}

// ApplyAssignmentBatch 在一个事务中对所有运行目标执行授权或撤销，避免审计与 revision 同授权事实脱节。
func (r *SQLRepository) ApplyAssignmentBatch(ctx context.Context, targetIDs, userIDs []uint64, action AssignmentBatchAction, actorID uint64) error { //nolint:cyclop,gocognit // 同一事务负责校验、锁定、变更、revision 推进和回滚。
	if r == nil || r.db == nil || actorID == 0 || len(targetIDs) == 0 || len(userIDs) == 0 || (action != AssignmentBatchGrant && action != AssignmentBatchRevoke) {
		return ErrNotFound
	}
	targets := append([]uint64(nil), targetIDs...)
	sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })
	return r.RunInTransaction(ctx, func(txCtx context.Context, _ *sql.Tx) error {
		for _, targetID := range targets {
			if err := r.lockAssignmentTarget(txCtx, targetID); err != nil {
				return err
			}
		}
		for _, targetID := range targets {
			changed, err := r.applyAssignmentBatchTarget(txCtx, targetID, userIDs, action, actorID)
			if err != nil {
				return err
			}
			if changed {
				if err := r.incrementAssignmentRevision(txCtx, targetID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *SQLRepository) applyAssignmentBatchTarget(ctx context.Context, targetID uint64, userIDs []uint64, action AssignmentBatchAction, actorID uint64) (bool, error) {
	changed := false
	for _, userID := range userIDs {
		if userID == 0 {
			return false, ErrNotFound
		}
		if action == AssignmentBatchGrant {
			result, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_target_user_assignments (runtime_target_id, user_id, created_by, updated_by, deleted_at, deleted_by) VALUES ($1, $2, $3, $3, 0, 0) ON CONFLICT (runtime_target_id, user_id) DO UPDATE SET deleted_at = 0, deleted_by = 0, updated_at = CURRENT_TIMESTAMP, updated_by = EXCLUDED.updated_by WHERE runtime_target_user_assignments.deleted_at <> 0`, targetID, userID, actorID)
			if err != nil {
				return false, err
			}
			affected, _ := result.RowsAffected()
			changed = changed || affected > 0
			continue
		}
		result, err := r.executor(ctx).ExecContext(ctx, `UPDATE runtime_target_user_assignments SET deleted_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint, deleted_by = $3, updated_at = CURRENT_TIMESTAMP, updated_by = $3 WHERE runtime_target_id = $1 AND user_id = $2 AND deleted_at = 0`, targetID, userID, actorID)
		if err != nil {
			return false, err
		}
		affected, _ := result.RowsAffected()
		changed = changed || affected > 0
	}
	return changed, nil
}

// ReplaceUserAssignmentsTx 在调用方事务中替换授权集合，并由调用方决定提交其它事实。
//
//nolint:cyclop // revision CAS and replacement sequencing are intentionally kept together.
func (r *SQLRepository) ReplaceUserAssignmentsTx(ctx context.Context, targetID uint64, userIDs []uint64, expectedRevision, actorID uint64) ([]UserAssignment, uint64, error) {
	if r == nil || r.db == nil || targetID == 0 || actorID == 0 {
		return nil, 0, ErrNotFound
	}
	if _, err := r.Get(ctx, targetID); err != nil {
		return nil, 0, err
	}
	if err := r.lockAssignmentTarget(ctx, targetID); err != nil {
		return nil, 0, err
	}
	if err := r.advanceAssignmentRevision(ctx, targetID, expectedRevision); err != nil {
		return nil, 0, err
	}
	if _, err := r.executor(ctx).ExecContext(ctx, `UPDATE runtime_target_user_assignments SET deleted_at = $3, deleted_by = $2, updated_at = CURRENT_TIMESTAMP, updated_by = $2 WHERE runtime_target_id = $1 AND deleted_at = 0`, targetID, actorID, time.Now().UTC().Unix()); err != nil {
		return nil, 0, err
	}
	for _, userID := range userIDs {
		if _, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_target_user_assignments (runtime_target_id, user_id, created_by, updated_by, deleted_at, deleted_by) VALUES ($1, $2, $3, $3, 0, 0) ON CONFLICT (runtime_target_id, user_id) DO UPDATE SET deleted_at = 0, deleted_by = 0, updated_at = CURRENT_TIMESTAMP, updated_by = EXCLUDED.updated_by`, targetID, userID, actorID); err != nil {
			return nil, 0, err
		}
	}
	items, err := r.ListUserAssignments(ctx, targetID)
	return items, expectedRevision + 1, err
}

func (r *SQLRepository) advanceAssignmentRevision(ctx context.Context, targetID, expectedRevision uint64) error {
	if expectedRevision == 0 {
		return ErrAssignmentRevisionConflict
	}
	if _, err := r.UserAssignmentRevision(ctx, targetID); err != nil {
		return err
	}
	result, err := r.executor(ctx).ExecContext(ctx, `UPDATE runtime_target_assignment_revisions SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE runtime_target_id = $1 AND revision = $2`, targetID, expectedRevision)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrAssignmentRevisionConflict
	}
	return nil
}

func (r *SQLRepository) incrementAssignmentRevision(ctx context.Context, targetID uint64) error {
	if _, err := r.UserAssignmentRevision(ctx, targetID); err != nil {
		return err
	}
	_, err := r.executor(ctx).ExecContext(ctx, `UPDATE runtime_target_assignment_revisions SET revision = revision + 1, updated_at = CURRENT_TIMESTAMP WHERE runtime_target_id = $1`, targetID)
	return err
}

// UpsertLocalDocker 写入系统托管的本机 Docker 探测结果，并通过 provider 与 endpoint 的活跃记录唯一键更新已有记录。
func (r *SQLRepository) UpsertLocalDocker(ctx context.Context, probe LocalDockerProbe) error {
	if r == nil || r.db == nil {
		return nil
	}
	_, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO runtime_targets (provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at, system_managed, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by) VALUES ('docker', $1, 'Local Docker', 'unix:///var/run/docker.sock', 'unix_socket', '["containers","compose_execution","image_build","workspace_access","update_controller"]'::jsonb, $2, $3, $4, true, NOW(), 0, NOW(), 0, 0, 0) ON CONFLICT (provider, endpoint) WHERE deleted_at = 0 DO UPDATE SET capabilities_json = EXCLUDED.capabilities_json, availability = EXCLUDED.availability, last_error = EXCLUDED.last_error, checked_at = EXCLUDED.checked_at, updated_at = NOW(), updated_by = 0`, probe.Endpoint, probe.Available, probe.Error, probe.CheckedAt)
	return err
}
