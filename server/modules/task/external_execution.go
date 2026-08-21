package task

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"graft/server/internal/moduleapi"
	taskcontract "graft/server/modules/task/contract"
	taskmodel "graft/server/modules/task/model"
	taskstore "graft/server/modules/task/store"
)

const (
	externalExecutionCandidateLimit  = 100
	externalExecutionLogBatchLimit   = 100
	externalExecutionLogLineMaxRunes = 4096
)

var externalExecutionLogLines = map[string]struct{}{
	"execution lease accepted":     {},
	"cancellation observed":        {},
	"provider operation started":   {},
	"provider operation completed": {},
}

var _ moduleapi.RuntimeAgentExecutionGateway = (*Runtime)(nil)

// ClaimExternalExecution 为已认证的 Runtime Target capability 领取一个冻结 Stage attempt。
func (r *Runtime) ClaimExternalExecution(ctx context.Context, request moduleapi.ExternalExecutionClaimRequest) (moduleapi.ExternalExecutionLease, error) {
	if r == nil || r.repository == nil || !validExternalExecutionClaimRequest(request) {
		return moduleapi.ExternalExecutionLease{}, taskstore.ErrInvalidInput
	}
	for offset := 0; ; offset += externalExecutionCandidateLimit {
		candidates, err := r.repository.ListExternalExecutionCandidates(ctx, externalExecutionCandidateLimit, offset)
		if err != nil {
			return moduleapi.ExternalExecutionLease{}, err
		}
		lease, claimed, err := r.claimExternalExecutionCandidates(ctx, request, candidates)
		if claimed || err != nil {
			return lease, err
		}
		if len(candidates) < externalExecutionCandidateLimit {
			return moduleapi.ExternalExecutionLease{}, moduleapi.ErrExternalExecutionNotFound
		}
	}
}

func (r *Runtime) claimExternalExecutionCandidates(ctx context.Context, request moduleapi.ExternalExecutionClaimRequest, candidates []taskstore.StageClaim) (moduleapi.ExternalExecutionLease, bool, error) {
	for _, candidate := range candidates {
		lease, claimed, err := r.claimExternalExecutionCandidate(ctx, request, candidate)
		if claimed || err != nil {
			return lease, claimed, err
		}
	}
	return moduleapi.ExternalExecutionLease{}, false, nil
}

func (r *Runtime) claimExternalExecutionCandidate(ctx context.Context, request moduleapi.ExternalExecutionClaimRequest, candidate taskstore.StageClaim) (moduleapi.ExternalExecutionLease, bool, error) {
	expectation, ok := externalExecutionExpectation(candidate)
	if !ok || !externalExecutionBindingMatches(expectation, request) {
		return moduleapi.ExternalExecutionLease{}, false, nil
	}
	lease, err := r.createExternalExecutionLease(ctx, candidate, expectation)
	if errors.Is(err, taskstore.ErrStateConflict) {
		return moduleapi.ExternalExecutionLease{}, false, nil
	}
	return lease, true, err
}

func externalExecutionBindingMatches(expectation *moduleapi.ExternalExecutionExpectation, request moduleapi.ExternalExecutionClaimRequest) bool {
	return expectation.RuntimeTargetID == request.RuntimeTargetID && expectation.ProviderID == request.ProviderID &&
		expectation.Capability == request.Capability && expectation.CapabilityVersion == request.CapabilityVersion
}

func (r *Runtime) createExternalExecutionLease(ctx context.Context, candidate taskstore.StageClaim, expectation *moduleapi.ExternalExecutionExpectation) (moduleapi.ExternalExecutionLease, error) {
	leaseID, err := randomSubmissionSecret()
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	fenceToken, err := randomSubmissionSecret()
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	now := time.Now().UTC()
	model := taskmodel.ExternalExecutionLease{
		ID: leaseID, TaskID: candidate.Task.ID, StageID: candidate.Stage.ID, Attempt: candidate.Stage.Attempt + 1,
		ExecutorType: candidate.Stage.ExecutorType, RuntimeTargetID: expectation.RuntimeTargetID,
		ProviderID: expectation.ProviderID, Capability: expectation.Capability, Protocol: expectation.Protocol,
		OperationID: expectation.OperationID, PayloadSHA256: expectation.PayloadSHA256,
		FenceTokenHash: hashSubmissionSecret(fenceToken), State: moduleapi.ExternalExecutionLeaseStateClaimed,
		LeaseTTL: expectation.LeaseTTL, LeaseExpiresAt: now.Add(expectation.LeaseTTL),
		AbsoluteDeadlineAt: now.Add(expectation.AbsoluteDeadline),
	}
	created, err := r.repository.CreateExternalExecutionLease(ctx, taskstore.CreateExternalExecutionLeaseInput{Lease: model})
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	r.publishTask(created.TaskID, taskcontract.TaskRealtimeEventStageStarted)
	return externalExecutionLeaseView(created, candidate.Stage.Input, expectation.CapabilityVersion, fenceToken, candidate.Task.CancelRequestedAt != nil), nil
}

// RenewExternalExecution 延长同一 fenced attempt，并返回 Task 已持久化的取消请求观察值。
func (r *Runtime) RenewExternalExecution(ctx context.Context, handle moduleapi.ExternalExecutionLeaseHandle) (moduleapi.ExternalExecutionLease, error) {
	if r == nil || r.repository == nil || !validExternalExecutionHandle(handle) {
		return moduleapi.ExternalExecutionLease{}, taskstore.ErrInvalidInput
	}
	current, _, err := r.repository.GetExternalExecutionLease(ctx, handle.LeaseID)
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	updated, cancelRequested, err := r.repository.RenewExternalExecutionLease(ctx, taskstore.RenewExternalExecutionLeaseInput{
		ID: handle.LeaseID, FenceTokenHash: hashSubmissionSecret(handle.FenceToken),
		LeaseExpiresAt: time.Now().UTC().Add(current.LeaseTTL),
	})
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	stage, err := r.externalExecutionStage(ctx, updated.TaskID, updated.StageID)
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	return externalExecutionLeaseView(updated, stage.input, stage.expectation.CapabilityVersion, handle.FenceToken, cancelRequested), nil
}

// InspectExternalExecution 验证当前 fence 并返回冻结的执行绑定，不续租也不改变 Task 状态。
func (r *Runtime) InspectExternalExecution(ctx context.Context, handle moduleapi.ExternalExecutionLeaseHandle) (moduleapi.ExternalExecutionLease, error) {
	if r == nil || r.repository == nil || !validExternalExecutionHandle(handle) {
		return moduleapi.ExternalExecutionLease{}, taskstore.ErrInvalidInput
	}
	lease, cancelRequested, err := r.repository.GetExternalExecutionLease(ctx, handle.LeaseID)
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	now := time.Now().UTC()
	if lease.FenceTokenHash != hashSubmissionSecret(handle.FenceToken) || lease.State != moduleapi.ExternalExecutionLeaseStateClaimed ||
		!lease.LeaseExpiresAt.After(now) || !lease.AbsoluteDeadlineAt.After(now) {
		return moduleapi.ExternalExecutionLease{}, taskstore.ErrStateConflict
	}
	stage, err := r.externalExecutionStage(ctx, lease.TaskID, lease.StageID)
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	return externalExecutionLeaseView(lease, stage.input, stage.expectation.CapabilityVersion, handle.FenceToken, cancelRequested), nil
}

// ResolveExternalExecutionMaterial 在验证 lease fence 后调用领域解析器，并且不保存返回材料。
func (r *Runtime) ResolveExternalExecutionMaterial(ctx context.Context, handle moduleapi.ExternalExecutionLeaseHandle) (moduleapi.ExternalExecutionMaterial, error) {
	if r == nil || r.repository == nil || !validExternalExecutionHandle(handle) {
		return moduleapi.ExternalExecutionMaterial{}, taskstore.ErrInvalidInput
	}
	lease, _, err := r.repository.GetExternalExecutionLease(ctx, handle.LeaseID)
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, err
	}
	if !externalExecutionLeaseFenceValid(lease, handle, time.Now().UTC()) {
		return moduleapi.ExternalExecutionMaterial{}, taskstore.ErrStateConflict
	}
	stage, err := r.externalExecutionStage(ctx, lease.TaskID, lease.StageID)
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, err
	}
	r.mu.RLock()
	resolver := r.materialResolvers[stage.executorType]
	r.mu.RUnlock()
	if resolver == nil {
		return moduleapi.ExternalExecutionMaterial{}, taskstore.ErrNotFound
	}
	material, err := resolver.ResolveExternalExecutionMaterial(ctx, moduleapi.ExternalExecutionMaterialRequest{
		TaskID: lease.TaskID, StageID: lease.StageID, Attempt: lease.Attempt, ExecutorType: stage.executorType,
		RuntimeTargetID: lease.RuntimeTargetID, OperationID: lease.OperationID, Input: stage.input,
	})
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, err
	}
	if !validExternalExecutionMaterial(material) {
		return moduleapi.ExternalExecutionMaterial{}, taskstore.ErrInvalidInput
	}
	material.Payload = append(json.RawMessage(nil), material.Payload...)
	return material, nil
}

func externalExecutionLeaseFenceValid(lease taskmodel.ExternalExecutionLease, handle moduleapi.ExternalExecutionLeaseHandle, now time.Time) bool {
	return lease.FenceTokenHash == hashSubmissionSecret(handle.FenceToken) &&
		lease.State == moduleapi.ExternalExecutionLeaseStateClaimed && lease.LeaseExpiresAt.After(now) &&
		lease.AbsoluteDeadlineAt.After(now)
}

func validExternalExecutionMaterial(material moduleapi.ExternalExecutionMaterial) bool {
	return strings.TrimSpace(material.Protocol) != "" && len(material.Payload) > 0 &&
		len(material.Payload) <= 1<<20 && json.Valid(material.Payload)
}

// RecordExternalExecutionResult 在当前 lease fence 内把瞬时结果转交领域 owner。
// Task Runtime 不把协议或结果载荷写入 Task、Stage、日志、receipt 或数据库。
func (r *Runtime) RecordExternalExecutionResult(ctx context.Context, result moduleapi.ExternalExecutionResult) error {
	if r == nil || r.repository == nil || !validExternalExecutionHandle(result.Handle) || !validExternalExecutionResult(result) {
		return taskstore.ErrInvalidInput
	}
	lease, _, err := r.repository.GetExternalExecutionLease(ctx, result.Handle.LeaseID)
	if err != nil {
		return err
	}
	if !externalExecutionLeaseFenceValid(lease, result.Handle, time.Now().UTC()) {
		return taskstore.ErrStateConflict
	}
	resultDigest := sha256.Sum256(append(append([]byte(strings.TrimSpace(result.Protocol)), 0), result.Payload...))
	if err := r.repository.RecordExternalExecutionResultDigest(ctx, taskstore.RecordExternalExecutionResultDigestInput{
		LeaseID: result.Handle.LeaseID, FenceTokenHash: hashSubmissionSecret(result.Handle.FenceToken),
		ResultSHA256: fmt.Sprintf("%x", resultDigest), RecordedAt: time.Now().UTC(),
	}); err != nil {
		return err
	}
	stage, err := r.externalExecutionStage(ctx, lease.TaskID, lease.StageID)
	if err != nil {
		return err
	}
	r.mu.RLock()
	recorder := r.resultRecorders[stage.executorType]
	r.mu.RUnlock()
	if recorder == nil {
		return taskstore.ErrNotFound
	}
	return recorder.RecordExternalExecutionResult(ctx, moduleapi.ExternalExecutionResultRequest{
		TaskID: lease.TaskID, StageID: lease.StageID, Attempt: lease.Attempt, ExecutorType: stage.executorType,
		RuntimeTargetID: lease.RuntimeTargetID, OperationID: lease.OperationID,
		Input: append(json.RawMessage(nil), stage.input...), Protocol: strings.TrimSpace(result.Protocol),
		Result: append(json.RawMessage(nil), result.Payload...),
	})
}

func validExternalExecutionResult(result moduleapi.ExternalExecutionResult) bool {
	return strings.TrimSpace(result.Protocol) != "" && len(result.Payload) > 0 && len(result.Payload) <= 1<<20 && json.Valid(result.Payload)
}

// AppendExternalExecutionLogs 将一批受限日志写入 lease 绑定的 Stage；批次和单行均有固定上限。
func (r *Runtime) AppendExternalExecutionLogs(ctx context.Context, batch moduleapi.ExternalExecutionLogBatch) error {
	if r == nil || r.repository == nil || !validExternalExecutionLogBatch(batch) {
		return taskstore.ErrInvalidInput
	}
	lease, _, err := r.repository.GetExternalExecutionLease(ctx, batch.Handle.LeaseID)
	if err != nil {
		return err
	}
	if err := r.repository.AppendExternalExecutionLogs(ctx, taskstore.AppendExternalExecutionLogsInput{
		LeaseID: batch.Handle.LeaseID, FenceTokenHash: hashSubmissionSecret(batch.Handle.FenceToken),
		Entries: batch.Entries, OccurredAt: time.Now().UTC(),
	}); err != nil {
		return err
	}
	r.publishTask(lease.TaskID, taskcontract.TaskRealtimeEventLogAppended)
	return nil
}

func validExternalExecutionLogBatch(batch moduleapi.ExternalExecutionLogBatch) bool {
	if !validExternalExecutionHandle(batch.Handle) || len(batch.Entries) == 0 || len(batch.Entries) > externalExecutionLogBatchLimit {
		return false
	}
	for _, entry := range batch.Entries {
		line := strings.TrimSpace(entry.Line)
		if utf8.RuneCountInString(line) > externalExecutionLogLineMaxRunes {
			return false
		}
		if _, allowed := externalExecutionLogLines[line]; !allowed {
			return false
		}
	}
	return true
}

// SettleExternalExecution 以完全匹配的 lease fence 结算 Stage，并在非最终成功时唤醒下一 Stage。
func (r *Runtime) SettleExternalExecution(ctx context.Context, receipt moduleapi.ExternalExecutionReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	if r == nil || r.repository == nil || !validExternalExecutionHandle(receipt.Handle) || !lowercaseSHA256(receipt.IntegritySHA256) {
		return moduleapi.ExternalReceiptSettlement{}, taskstore.ErrInvalidInput
	}
	settlement, err := r.repository.SettleExternalExecution(ctx, taskstore.SettleExternalExecutionInput{
		LeaseID: receipt.Handle.LeaseID, FenceTokenHash: hashSubmissionSecret(receipt.Handle.FenceToken),
		Outcome: receipt.Outcome, FailureCode: receipt.FailureCode, IntegritySHA256: receipt.IntegritySHA256,
	})
	if err != nil {
		return moduleapi.ExternalReceiptSettlement{}, err
	}
	if !settlement.Idempotent {
		eventType := taskcontract.TaskRealtimeEventStageCompleted
		if receipt.Outcome != moduleapi.ExternalReceiptOutcomeSuccess {
			eventType = taskcontract.TaskRealtimeEventStageFailed
		}
		r.publishTask(settlement.TaskID, eventType)
		if settlement.Status == moduleapi.TaskStatusRunning {
			r.signalWake()
		}
	}
	return moduleapi.ExternalReceiptSettlement{TaskID: settlement.TaskID, StageID: settlement.StageID, Status: settlement.Status, Idempotent: settlement.Idempotent}, nil
}

// ExpireExternalExecutions 将超时 lease 收敛到 unknown/needs_attention，不触发自动重放。
func (r *Runtime) ExpireExternalExecutions(ctx context.Context, limit int) (int, error) {
	if r == nil || r.repository == nil {
		return 0, taskstore.ErrInvalidInput
	}
	return r.repository.ExpireExternalExecutionLeases(ctx, time.Now().UTC(), limit)
}

func externalExecutionExpectation(claim taskstore.StageClaim) (*moduleapi.ExternalExecutionExpectation, bool) {
	index := claim.Stage.Sequence - 1
	if index < 0 {
		return nil, false
	}
	var plan moduleapi.TaskPlan
	if err := json.Unmarshal(claim.Task.Plan, &plan); err != nil || index >= len(plan.Stages) {
		return nil, false
	}
	stage := plan.Stages[index]
	if stage.Key != claim.Stage.Key || stage.ExecutorType != claim.Stage.ExecutorType || stage.ExternalExecution == nil {
		return nil, false
	}
	return stage.ExternalExecution, true
}

func externalExecutionLeaseView(lease taskmodel.ExternalExecutionLease, input json.RawMessage, capabilityVersion, fenceToken string, cancelRequested bool) moduleapi.ExternalExecutionLease {
	return moduleapi.ExternalExecutionLease{
		ID: lease.ID, TaskID: lease.TaskID, StageID: lease.StageID, Attempt: lease.Attempt, ExecutorType: lease.ExecutorType,
		RuntimeTargetID: lease.RuntimeTargetID, ProviderID: lease.ProviderID, Capability: lease.Capability, CapabilityVersion: capabilityVersion,
		Protocol: lease.Protocol, OperationID: lease.OperationID, PayloadSHA256: lease.PayloadSHA256,
		Input: append(json.RawMessage(nil), input...), FenceToken: fenceToken, State: lease.State, LeaseTTL: lease.LeaseTTL,
		LeaseExpiresAt: lease.LeaseExpiresAt, AbsoluteDeadlineAt: lease.AbsoluteDeadlineAt,
		CancellationRequested: cancelRequested,
	}
}

type externalExecutionStageBinding struct {
	input        json.RawMessage
	executorType moduleapi.StageExecutorType
	expectation  *moduleapi.ExternalExecutionExpectation
}

func (r *Runtime) externalExecutionStage(ctx context.Context, taskID uint64, stageID uint64) (externalExecutionStageBinding, error) {
	task, err := r.repository.Get(ctx, taskID)
	if err != nil {
		return externalExecutionStageBinding{}, err
	}
	var plan moduleapi.TaskPlan
	if err := json.Unmarshal(task.Plan, &plan); err != nil {
		return externalExecutionStageBinding{}, err
	}
	stages, err := r.repository.ListStages(ctx, taskID)
	if err != nil {
		return externalExecutionStageBinding{}, err
	}
	for _, stage := range stages {
		if stage.ID == stageID {
			index := stage.Sequence - 1
			if index < 0 || index >= len(plan.Stages) || plan.Stages[index].ExternalExecution == nil {
				return externalExecutionStageBinding{}, taskstore.ErrStateConflict
			}
			return externalExecutionStageBinding{
				input: append(json.RawMessage(nil), stage.Input...), executorType: stage.ExecutorType,
				expectation: plan.Stages[index].ExternalExecution,
			}, nil
		}
	}
	return externalExecutionStageBinding{}, taskstore.ErrNotFound
}

func validExternalExecutionClaimRequest(request moduleapi.ExternalExecutionClaimRequest) bool {
	return request.RuntimeTargetID > 0 && strings.TrimSpace(request.ProviderID) != "" && strings.TrimSpace(request.Capability) != "" && strings.TrimSpace(request.CapabilityVersion) != ""
}

func validExternalExecutionHandle(handle moduleapi.ExternalExecutionLeaseHandle) bool {
	return strings.TrimSpace(handle.LeaseID) != "" && strings.TrimSpace(handle.FenceToken) != ""
}
