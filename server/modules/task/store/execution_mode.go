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

const executionOwnershipKey = "graft-task-runtime-owner/v1"

// ExecutionOwnership 持有单个 Task Runtime 的数据库会话锁，并报告连接失效。
type ExecutionOwnership struct {
	Lost       <-chan struct{}
	connection *sql.Conn
	cancel     context.CancelFunc
	once       sync.Once
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
		if o.connection != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, err = o.connection.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtextextended($1, 0))", executionOwnershipKey)
			closeErr := o.connection.Close()
			if err == nil {
				err = closeErr
			}
		}
	})
	return err
}

// AcquireExecutionOwnership 在任何崩溃恢复前排除另一活跃 server；连接失效时调用方必须停止全部 worker。
func (r *SQLRepository) AcquireExecutionOwnership(ctx context.Context) (*ExecutionOwnership, error) {
	lost := make(chan struct{})
	ownership := &ExecutionOwnership{Lost: lost}
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
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-monitorCtx.Done():
				return
			case <-ticker.C:
				probeCtx, stop := context.WithTimeout(monitorCtx, time.Second)
				err := connection.PingContext(probeCtx)
				stop()
				if err != nil {
					close(lost)
					return
				}
			}
		}
	}()
	return ownership, nil
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
		return errors.New("task execution mode differs from the active runtime binding")
	}
	if mode != "local" {
		return nil
	}
	if err := r.lockExecutionResources(ctx, tx, resources); err != nil {
		return err
	}
	return r.checkExecutionResourceConflicts(ctx, tx, resources)
}

func (r *SQLRepository) checkExecutionResourceConflicts(ctx context.Context, tx *sql.Tx, resources []executionResource) error {
	rows, err := tx.QueryContext(ctx, "SELECT plan_json FROM tasks WHERE status IN ('pending','ready','scheduled','running','needs_attention')")
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
