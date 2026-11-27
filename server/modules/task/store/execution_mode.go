package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
)

const (
	executionOwnershipKey = "graft-task-runtime-owner/v1"
	// executionOwnershipPingInterval 是所有权监视连接的探测周期。
	executionOwnershipPingInterval = time.Second
	// ExecutionOwnershipDegradedAfter 是进入退化的连续探测失败次数；一次瞬时失败不锁定准入。
	// 导出供运行时回归按同一阈值驱动监视状态机。
	ExecutionOwnershipDegradedAfter = 5
	// executionOwnershipReacquireGap 是退化状态下两次所有权重取尝试之间的探测失败间隔。
	executionOwnershipReacquireGap = 10
)

// ExecutionOwnershipState 是执行所有权监视的稳定状态：ready、degraded（连接瞬时退化，自动恢复）
// 与 lost（另一个运行时持有执行权，需要人工重启本运行时）。
type ExecutionOwnershipState string

// 所有权监视的稳定状态值；diagnostic 码映射见 Admission。
const (
	ExecutionOwnershipReady    ExecutionOwnershipState = "ready"
	ExecutionOwnershipDegraded ExecutionOwnershipState = "degraded"
	ExecutionOwnershipLost     ExecutionOwnershipState = "lost"
)

// ExecutionOwnershipAdmissionDegraded 是退化状态下执行准入的稳定诊断码：
// 所有权连接瞬时退化，有界探测后自动恢复；与 execution_ownership_lost 的永久失去相区分。
const ExecutionOwnershipAdmissionDegraded = "execution_admission_degraded"

// ExecutionOwnership 持有单个 Task Runtime 的数据库会话锁，报告连接失效与瞬时退化。
type ExecutionOwnership struct {
	Lost       <-chan struct{}
	lost       chan struct{}
	connection *sql.Conn
	cancel     context.CancelFunc
	once       sync.Once
	conflict   sync.Once

	mu       sync.Mutex
	state    ExecutionOwnershipState
	failures int
}

// Admission 返回当前所有权状态对应的执行准入结果与诊断码。
// degraded 与 lost 都拒绝新的执行请求；degraded 在探测恢复后自动回到 ready。
func (o *ExecutionOwnership) Admission() (ready bool, diagnostic string) {
	if o == nil {
		return true, "execution_ready"
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	switch o.state {
	case ExecutionOwnershipDegraded:
		return false, ExecutionOwnershipAdmissionDegraded
	case ExecutionOwnershipLost:
		return false, "execution_ownership_lost"
	default:
		return true, "execution_ready"
	}
}

// ProbeFailed 由所有权监视循环在连接探测失败时调用；返回是否进入退化。
func (o *ExecutionOwnership) ProbeFailed() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == ExecutionOwnershipLost {
		return false
	}
	o.failures++
	if o.failures < ExecutionOwnershipDegradedAfter {
		return false
	}
	o.state = ExecutionOwnershipDegraded
	return true
}

// ProbeSucceeded 由所有权监视循环在连接探测成功时调用：清除退化并复位计数。
// 已判定 lost 的所有权不因探测成功而复活。
func (o *ExecutionOwnership) ProbeSucceeded() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state == ExecutionOwnershipLost {
		return
	}
	o.failures = 0
	o.state = ExecutionOwnershipReady
}

// ShouldReacquire 由监视循环在退化状态下决定是否立即尝试重取所有权（有界退避）。
func (o *ExecutionOwnership) ShouldReacquire() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state != ExecutionOwnershipDegraded {
		return false
	}
	if o.failures == ExecutionOwnershipDegradedAfter {
		return true
	}
	return o.failures > ExecutionOwnershipDegradedAfter &&
		(o.failures-ExecutionOwnershipDegradedAfter)%executionOwnershipReacquireGap == 0
}

// Reacquired 在新连接上重取所有权成功后由监视循环调用：恢复就绪并采纳新连接。
func (o *ExecutionOwnership) Reacquired(conn *sql.Conn) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connection = conn
	o.state = ExecutionOwnershipReady
	o.failures = 0
}

// OwnershipConflict 在重取所有权被另一运行时拒绝后由监视循环调用：执行权已归属他方，
// 准入按既有语义永久失去，等待人工重启本运行时。
func (o *ExecutionOwnership) OwnershipConflict() {
	o.mu.Lock()
	o.state = ExecutionOwnershipLost
	o.mu.Unlock()
	o.conflict.Do(func() { close(o.lost) })
}

// ping 探测当前持有的所有权连接。
func (o *ExecutionOwnership) ping(ctx context.Context) error {
	o.mu.Lock()
	conn := o.connection
	o.mu.Unlock()
	if conn == nil {
		return errors.New("execution ownership has no connection to probe")
	}
	return conn.PingContext(ctx)
}

// releaseConnection 尽力解锁并关闭当前连接；连接已失效时解锁失败属预期，所有权由数据库释放。
func (o *ExecutionOwnership) releaseConnection(ctx context.Context) error {
	o.mu.Lock()
	conn := o.connection
	o.connection = nil
	o.mu.Unlock()
	if conn == nil {
		return nil
	}
	unlockCtx, cancel := context.WithTimeout(ctx, executionOwnershipPingInterval)
	defer cancel()
	_, err := conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock(hashtextextended($1, 0))", executionOwnershipKey)
	closeErr := conn.Close()
	if err == nil {
		err = closeErr
	}
	return err
}

// Close 仅在 worker 停止后释放当前运行时的执行所有权。
func (o *ExecutionOwnership) Close() error {
	if o == nil {
		return nil
	}
	var err error
	o.once.Do(func() {
		if o.cancel != nil {
			o.cancel()
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = o.releaseConnection(ctx)
	})
	return err
}

// NewExecutionOwnership 构造带冲突通道的所有权句柄，供回归直接驱动监视状态机。
func NewExecutionOwnership() *ExecutionOwnership {
	lost := make(chan struct{})
	return &ExecutionOwnership{Lost: lost, lost: lost}
}

// AcquireExecutionOwnership 在任何崩溃恢复前排除另一活跃 server；连接退化时准入降级并自动恢复，
// 连接彻底失效且他方持有执行权时才永久失去所有权。
func (r *SQLRepository) AcquireExecutionOwnership(ctx context.Context) (*ExecutionOwnership, error) {
	ownership := NewExecutionOwnership()
	if r.placeholder == placeholderQuestion {
		return ownership, nil
	}
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	var acquired bool
	if err := connection.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtextextended($1, 0))", executionOwnershipKey).Scan(&acquired); err != nil || !acquired {
		_ = connection.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("another Task Runtime owns execution; stop it before starting this server")
	}
	monitorCtx, cancel := context.WithCancel(ctx)
	ownership.connection, ownership.cancel = connection, cancel
	go r.monitorExecutionOwnership(monitorCtx, ownership)
	return ownership, nil
}

// monitorExecutionOwnership 每秒探测所有权连接：瞬时失败在阈值内保持就绪，越过阈值进入退化；
// 退化期间按有界退避重取所有权（先尽力解锁旧会话再尝试新连接），同连接恢复或重取成功都回到就绪；
// 重取被他方拒绝时判定真实的所有权冲突并永久失去。探测错误（数据库仍不可达）保持退化继续重试。
func (r *SQLRepository) monitorExecutionOwnership(ctx context.Context, ownership *ExecutionOwnership) {
	ticker := time.NewTicker(executionOwnershipPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ownership.Lost:
			return
		case <-ticker.C:
			probeCtx, stop := context.WithTimeout(ctx, executionOwnershipPingInterval)
			err := ownership.ping(probeCtx)
			stop()
			if err == nil {
				ownership.ProbeSucceeded()
				continue
			}
			ownership.ProbeFailed()
			if !ownership.ShouldReacquire() {
				continue
			}
			if r.reacquireExecutionOwnership(ctx, ownership) {
				continue
			}
			return
		}
	}
}

// reacquireExecutionOwnership 在退化期间重取执行所有权：先尽力解锁并关闭失效连接，
// 再从连接池取新连接尝试咨询锁。返回 true 表示所有权已恢复或已确认永久失去（监视循环应退出），
// 返回 false 表示数据库仍不可达，保持退化等待下一轮。
func (r *SQLRepository) reacquireExecutionOwnership(ctx context.Context, ownership *ExecutionOwnership) bool {
	unlockCtx, cancel := context.WithTimeout(ctx, executionOwnershipPingInterval)
	defer cancel()
	_ = ownership.releaseConnection(unlockCtx)
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return false
	}
	var acquired bool
	if err := connection.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(hashtextextended($1, 0))", executionOwnershipKey).Scan(&acquired); err != nil {
		_ = connection.Close()
		return false
	}
	if !acquired {
		_ = connection.Close()
		ownership.OwnershipConflict()
		return true
	}
	ownership.Reacquired(connection)
	return true
}

// ExecutionModePreflight 是模式切换的只读报告，已有模式重启保留正在运行的 Controller。
type ExecutionModePreflight struct {
	CurrentMode         string `json:"current_mode"`
	RequestedMode       string `json:"requested_mode"`
	Ready               bool   `json:"ready"`
	BlockingTasks       int    `json:"blocking_tasks"`
	BlockingSubmissions int    `json:"blocking_submissions"`
	BlockingLeases      int    `json:"blocking_leases"`
	DiagnosticCode      string `json:"diagnostic_code"`
}

// PreflightExecutionMode 读取模式与未收敛事实，供部署启动前给出可操作的切换结果。
func (r *SQLRepository) PreflightExecutionMode(ctx context.Context, mode string) (ExecutionModePreflight, error) {
	if mode != "local" && mode != "agent" {
		return ExecutionModePreflight{}, errors.New("execution mode must be local or agent")
	}
	report := ExecutionModePreflight{RequestedMode: mode}
	var err error
	if err := r.db.QueryRowContext(ctx, "SELECT mode FROM task_runtime_execution_mode WHERE id = 1").Scan(&report.CurrentMode); err != nil {
		return report, err
	}
	if mode == report.CurrentMode {
		report.Ready = true
		report.DiagnosticCode = "execution_mode_unchanged"
		return report, nil
	}
	if report.BlockingTasks, err = countModeBoundTasks(ctx, r.db); err != nil {
		return report, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_submissions WHERE state = 'reserved'").Scan(&report.BlockingSubmissions); err != nil {
		return report, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_external_execution_leases WHERE state = 'claimed'").Scan(&report.BlockingLeases); err != nil {
		return report, err
	}
	report.Ready = report.BlockingTasks+report.BlockingSubmissions+report.BlockingLeases == 0
	report.DiagnosticCode = "execution_mode_ready"
	if !report.Ready {
		report.DiagnosticCode = "execution_mode_has_unsettled_work"
	}
	return report, nil
}

// CheckExecutionMode 在持有运行时独占权后再次验证并原子推进模式绑定。
func (r *SQLRepository) CheckExecutionMode(ctx context.Context, mode string) error {
	if mode != "local" && mode != "agent" {
		return errors.New("execution mode must be local or agent")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err := r.lockExecutionMode(ctx, tx); err != nil {
		return err
	}
	var current string
	if err := tx.QueryRowContext(ctx, "SELECT mode FROM task_runtime_execution_mode WHERE id = 1").Scan(&current); err != nil {
		return err
	}
	if current == mode {
		return tx.Commit()
	}
	if err := ensureExecutionModeDrained(ctx, tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind("UPDATE task_runtime_execution_mode SET mode = ?, updated_at = ? WHERE id = 1"), mode, time.Now().UTC()); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SQLRepository) lockExecutionMode(ctx context.Context, tx *sql.Tx) error {
	if r.placeholder == placeholderQuestion {
		return nil
	}
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", "graft-task-execution-mode/v1")
	return err
}

type executionResource struct {
	target int64
	key    string
}

func planExecutionResources(raw json.RawMessage) ([]executionResource, string, error) {
	var plan moduleapi.TaskPlan
	if json.Unmarshal(raw, &plan) != nil {
		if !bytes.Contains(raw, []byte("LocalExecution")) && !bytes.Contains(raw, []byte("ExternalExecution")) {
			return nil, "", nil
		}
		return nil, "", ErrInvalidInput
	}
	var resources []executionResource
	mode := ""
	for _, stage := range plan.Stages {
		keys, stageMode, err := stageExecutionResources(stage)
		if err != nil {
			return nil, "", err
		}
		if stageMode == "" {
			continue
		}
		if mode != "" && mode != stageMode {
			return nil, "", ErrInvalidInput
		}
		mode = stageMode
		resources = append(resources, keys...)
	}
	return resources, mode, nil
}

func stageExecutionResources(stage moduleapi.StagePlan) ([]executionResource, string, error) {
	binding, mode := stage.ExternalExecution, "agent"
	if stage.LocalExecution != nil {
		binding = &stage.LocalExecution.Operation
		mode = "local"
	}
	if binding == nil {
		return nil, "", nil
	}
	var resources []executionResource
	for _, key := range binding.ResourceKeys {
		if strings.TrimSpace(key) == "" || len(key) > maximumExecutionResourceKeyBytes {
			return nil, "", ErrInvalidInput
		}
		resources = append(resources, executionResource{target: binding.RuntimeTargetID, key: key})
	}
	return resources, mode, nil
}

const maximumExecutionResourceKeyBytes = 2048

// guardExecutionResources 以短事务锁原子检查持久化资源占用；未知结果的 Task 持续占用其原资源。
func (r *SQLRepository) guardExecutionResources(ctx context.Context, tx *sql.Tx, task taskmodel.Task) error {
	resources, mode, err := planExecutionResources(task.Plan)
	if err != nil || mode == "" {
		return err
	}
	if err := r.lockExecutionMode(ctx, tx); err != nil {
		return err
	}
	var current string
	if err := tx.QueryRowContext(ctx, "SELECT mode FROM task_runtime_execution_mode WHERE id = 1").Scan(&current); err != nil {
		return err
	}
	if current != mode {
		return moduleapi.ErrTaskExecutionUnavailable
	}
	if mode != "local" {
		return nil
	}
	if err := r.lockExecutionResources(ctx, tx, resources); err != nil {
		return err
	}
	return r.checkExecutionResourceConflicts(ctx, tx, resources, task.ID)
}

func (r *SQLRepository) checkExecutionResourceConflicts(ctx context.Context, tx *sql.Tx, resources []executionResource, ownTaskID uint64) error {
	rows, err := tx.QueryContext(ctx, r.placeholder.rebind("SELECT plan_json FROM tasks WHERE id <> ? AND status IN ('pending','ready','scheduled','running','needs_attention')"), ownTaskID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		existing, _, err := planExecutionResources(raw)
		if err != nil {
			return err
		}
		if executionResourcesConflict(resources, existing) {
			return moduleapi.ErrTaskOwnerBusy
		}
	}
	return rows.Err()
}

func executionResourcesConflict(left, right []executionResource) bool {
	for _, a := range left {
		for _, b := range right {
			if executionResourceConflict(a, b) {
				return true
			}
		}
	}
	return false
}

func executionResourceConflict(a, b executionResource) bool {
	if a.target == b.target && (a.key == b.key || a.key == "*" || b.key == "*" || a.key == "platform-update" || b.key == "platform-update") {
		return true
	}
	return strings.HasPrefix(a.key, "registry:") && a.key == b.key
}

func (r *SQLRepository) lockExecutionResources(ctx context.Context, tx *sql.Tx, resources []executionResource) error {
	keys := make([]string, 0, len(resources))
	for _, resource := range resources {
		keys = append(keys, fmt.Sprintf("graft-local-target:%d", resource.target))
	}
	sort.Strings(keys)
	for index, key := range keys {
		if index > 0 && keys[index-1] == key {
			continue
		}
		if r.placeholder == placeholderDollar {
			if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", key); err != nil {
				return err
			}
		}
	}
	return nil
}

// LookupIdempotentTask 在重新解析外部资源前读取既有 receipt，Create 仍在事务内复核并发提交。
func (r *SQLRepository) LookupIdempotentTask(ctx context.Context, task taskmodel.Task) (taskmodel.Task, bool, error) {
	return r.findIdempotentTask(ctx, r.db, task)
}

type taskPlanQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func countModeBoundTasks(ctx context.Context, queryer taskPlanQueryer) (int, error) {
	rows, err := queryer.QueryContext(ctx, "SELECT plan_json FROM tasks WHERE status IN ('pending','ready','scheduled','running','needs_attention')")
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	count := 0
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return 0, err
		}
		if modeBoundPlan(raw) {
			count++
		}
	}
	return count, rows.Err()
}

func modeBoundPlan(raw json.RawMessage) bool {
	var plan moduleapi.TaskPlan
	if json.Unmarshal(raw, &plan) != nil {
		return false
	}
	for _, stage := range plan.Stages {
		if stage.LocalExecution != nil || stage.ExternalExecution != nil || stage.ExternalReceipt != nil {
			return true
		}
	}
	return false
}

func (r *SQLRepository) guardSubmissionExecutionMode(ctx context.Context, tx *sql.Tx, mode string) error {
	if mode == "" {
		return nil
	}
	if err := r.lockExecutionMode(ctx, tx); err != nil {
		return err
	}
	var current string
	if err := tx.QueryRowContext(ctx, "SELECT mode FROM task_runtime_execution_mode WHERE id = 1").Scan(&current); err != nil {
		return err
	}
	if current != mode {
		return moduleapi.ErrTaskExecutionUnavailable
	}
	return nil
}

func ensureExecutionModeDrained(ctx context.Context, tx *sql.Tx) error {
	taskCount, err := countModeBoundTasks(ctx, tx)
	if err != nil {
		return err
	}
	var otherWork bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM task_submissions WHERE state = 'reserved') OR EXISTS(SELECT 1 FROM task_external_execution_leases WHERE state = 'claimed')").Scan(&otherWork); err != nil {
		return err
	}
	if taskCount > 0 || otherWork {
		return errors.New("execution mode switch requires settled tasks, submissions and leases")
	}
	return nil
}
