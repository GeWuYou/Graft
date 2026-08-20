package task

import (
	"context"
	"encoding/json"
	"errors"
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
			return moduleapi.ExternalExecutionLease{}, taskstore.ErrNotFound
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
	return expectation.RuntimeTargetID == request.RuntimeTargetID && expectation.ProviderID == request.ProviderID && expectation.Capability == request.Capability
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
	return externalExecutionLeaseView(created, candidate.Stage.Input, fenceToken, candidate.Task.CancelRequestedAt != nil), nil
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
	input, err := r.externalExecutionStageInput(ctx, updated.TaskID, updated.StageID)
	if err != nil {
		return moduleapi.ExternalExecutionLease{}, err
	}
	return externalExecutionLeaseView(updated, input, handle.FenceToken, cancelRequested), nil
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
		if strings.TrimSpace(entry.Line) == "" || utf8.RuneCountInString(entry.Line) > externalExecutionLogLineMaxRunes {
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

func externalExecutionLeaseView(lease taskmodel.ExternalExecutionLease, input json.RawMessage, fenceToken string, cancelRequested bool) moduleapi.ExternalExecutionLease {
	return moduleapi.ExternalExecutionLease{
		ID: lease.ID, TaskID: lease.TaskID, StageID: lease.StageID, Attempt: lease.Attempt, ExecutorType: lease.ExecutorType,
		RuntimeTargetID: lease.RuntimeTargetID, ProviderID: lease.ProviderID, Capability: lease.Capability,
		Protocol: lease.Protocol, OperationID: lease.OperationID, PayloadSHA256: lease.PayloadSHA256,
		Input: append(json.RawMessage(nil), input...), FenceToken: fenceToken, State: lease.State, LeaseTTL: lease.LeaseTTL,
		LeaseExpiresAt: lease.LeaseExpiresAt, AbsoluteDeadlineAt: lease.AbsoluteDeadlineAt,
		CancellationRequested: cancelRequested,
	}
}

func (r *Runtime) externalExecutionStageInput(ctx context.Context, taskID uint64, stageID uint64) (json.RawMessage, error) {
	stages, err := r.repository.ListStages(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for _, stage := range stages {
		if stage.ID == stageID {
			return append(json.RawMessage(nil), stage.Input...), nil
		}
	}
	return nil, taskstore.ErrNotFound
}

func validExternalExecutionClaimRequest(request moduleapi.ExternalExecutionClaimRequest) bool {
	return request.RuntimeTargetID > 0 && strings.TrimSpace(request.ProviderID) != "" && strings.TrimSpace(request.Capability) != ""
}

func validExternalExecutionHandle(handle moduleapi.ExternalExecutionLeaseHandle) bool {
	return strings.TrimSpace(handle.LeaseID) != "" && strings.TrimSpace(handle.FenceToken) != ""
}
