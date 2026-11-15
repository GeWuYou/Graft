package task

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/dockerexecution"
	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
	taskstore "graft/server/modules/task/store"
)

const localExecutionVersion = "local-sdk/v1"
const localResourceProbeTimeout = 5 * time.Second

type localExecutionRuntime struct {
	mode        string
	environment dockerexecution.Environment
	connections moduleapi.RuntimeTargetProviderConnectionReader
	execute     func(context.Context, dockerexecution.Environment, dockerexecution.Request) dockerexecution.Result
}

func (r *Runtime) configureLocalExecution(cfg *config.Config) {
	r.localExecution = &localExecutionRuntime{mode: cfg.Execution.Mode, environment: dockerexecution.Environment{DockerSocket: cfg.Container.DockerEndpoint}, execute: dockerexecution.Execute}
	if r.localExecution.mode == "" {
		r.localExecution.mode = config.ExecutionModeAgent
	}
}

func (r *Runtime) freezeLocalExecution(ctx context.Context, plan moduleapi.TaskPlan) (moduleapi.TaskPlan, error) {
	if r.localExecution == nil || r.localExecution.mode != config.ExecutionModeLocal {
		return plan, nil
	}
	for index := range plan.Stages {
		if err := r.freezeLocalStage(ctx, &plan.Stages[index]); err != nil {
			return moduleapi.TaskPlan{}, err
		}
	}
	return plan, nil
}

func (r *Runtime) freezeLocalStage(ctx context.Context, stage *moduleapi.StagePlan) error {
	if stage.LocalExecution != nil {
		return errors.New("local execution binding is runtime-owned")
	}
	if stage.ExternalExecution == nil {
		return nil
	}
	expectation := *stage.ExternalExecution
	if expectation.ProviderID != "docker" || expectation.CapabilityVersion != "docker/v1" {
		return errors.New("local Docker execution capability is unsupported")
	}
	if err := r.validateLocalConnection(ctx, expectation.RuntimeTargetID); err != nil {
		return err
	}
	keys := append([]string(nil), expectation.ResourceKeys...)
	if expectation.Capability == "container_execution" {
		probeCtx, cancel := context.WithTimeout(ctx, localResourceProbeTimeout)
		resolved, resolvedKeys, err := dockerexecution.ResolveResources(probeCtx, r.localExecution.environment, sdkRequest(expectation, stage.Input))
		cancel()
		if err != nil {
			return fmt.Errorf("resolve execution resource: %w", err)
		}
		keys = resolvedKeys
		stage.Input = resolved.Input
		expectation.PayloadSHA256 = fmt.Sprintf("%x", sha256.Sum256(stage.Input))
	}
	if len(keys) == 0 {
		return errors.New("local execution resource coordination is missing")
	}
	sort.Strings(keys)
	expectation.ResourceKeys = keys
	stage.LocalExecution = &moduleapi.LocalExecutionExpectation{Version: localExecutionVersion, Operation: expectation}
	stage.ExternalExecution = nil
	return nil
}

func (r *Runtime) validateLocalConnection(ctx context.Context, targetID int64) error {
	if r.localExecution.connections == nil {
		return errors.New("local execution target reader is unavailable")
	}
	connection, err := r.localExecution.connections.GetProviderConnection(ctx, targetID)
	if err != nil || connection.Provider != "docker" || connection.ConnectionKind != "unix_socket" || connection.Endpoint != r.localExecution.environment.DockerSocket {
		return errors.New("local execution requires the configured local Docker target")
	}
	return nil
}

func sdkRequest(binding moduleapi.ExternalExecutionExpectation, input json.RawMessage) dockerexecution.Request {
	return dockerexecution.Request{Capability: binding.Capability, Protocol: binding.Protocol, OperationID: binding.OperationID, Input: input}
}

func (r *Runtime) executorForClaim(claim taskstore.StageClaim) (moduleapi.StageExecutor, bool) {
	stage, ok := localStage(claim.Task, claim.Stage)
	if ok {
		if r.localExecution == nil || r.localExecution.mode != config.ExecutionModeLocal || stage.LocalExecution.Version != localExecutionVersion {
			return nil, false
		}
		return localStageExecutor{runtime: r, stage: stage, task: claim.Task, claimed: claim.Stage}, true
	}
	return r.executorFor(claim.Stage.ExecutorType)
}

func localStage(task taskmodel.Task, stage taskmodel.Stage) (moduleapi.StagePlan, bool) {
	var plan moduleapi.TaskPlan
	if json.Unmarshal(task.Plan, &plan) != nil {
		return moduleapi.StagePlan{}, false
	}
	index := stage.Sequence - 1
	if index < 0 || index >= len(plan.Stages) {
		return moduleapi.StagePlan{}, false
	}
	frozen := plan.Stages[index]
	return frozen, frozen.LocalExecution != nil && frozen.Key == stage.Key && frozen.ExecutorType == stage.ExecutorType
}

type localStageExecutor struct {
	runtime *Runtime
	stage   moduleapi.StagePlan
	task    taskmodel.Task
	claimed taskmodel.Stage
}

func (e localStageExecutor) Type() moduleapi.StageExecutorType              { return e.stage.ExecutorType }
func (localStageExecutor) Cancel(context.Context, moduleapi.StageRun) error { return nil }

func (e localStageExecutor) checkAttempt(ctx context.Context) error {
	stages, err := e.runtime.repository.ListStages(ctx, e.task.ID)
	if err != nil {
		return err
	}
	for _, stage := range stages {
		if stage.ID == e.claimed.ID && stage.Status == moduleapi.StageStatusRunning && stage.Attempt == e.claimed.Attempt {
			return nil
		}
	}
	return taskstore.ErrStateConflict
}

func (e localStageExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	binding := e.stage.LocalExecution.Operation
	ctx, cancel := context.WithTimeout(ctx, binding.AbsoluteDeadline)
	defer cancel()
	if err := e.checkAttempt(ctx); err != nil {
		return err
	}
	request := moduleapi.ExternalExecutionMaterialRequest{TaskID: run.TaskID(), StageID: run.StageID(), Attempt: run.Attempt(), ExecutorType: e.stage.ExecutorType, RuntimeTargetID: binding.RuntimeTargetID, OperationID: binding.OperationID, Input: run.Input()}
	environment := e.runtime.localExecution.environment
	environment.ResolveMaterial = func(ctx context.Context, _ dockerexecution.Request) (dockerexecution.Material, error) {
		if err := e.checkAttempt(ctx); err != nil {
			return dockerexecution.Material{}, err
		}
		e.runtime.mu.RLock()
		resolver := e.runtime.materialResolvers[e.stage.ExecutorType]
		e.runtime.mu.RUnlock()
		if resolver == nil {
			return dockerexecution.Material{}, errors.New("local execution material resolver is unavailable")
		}
		material, err := resolver.ResolveExternalExecutionMaterial(ctx, request)
		return dockerexecution.Material{Protocol: material.Protocol, Payload: material.Payload}, err
	}
	result := e.runtime.localExecution.execute(ctx, environment, sdkRequest(binding, request.Input))
	if ctx.Err() != nil || result.Outcome == "needs_attention" {
		return localFailure("runner_interrupted", true)
	}
	if result.Outcome != "success" {
		code := result.FailureCode
		if result.FailureStage != "" {
			code = "local_" + result.FailureStage + "_failed"
		}
		return localFailure(code, false)
	}
	return e.recordResult(ctx, request, result)
}

func (e localStageExecutor) recordResult(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest, result dockerexecution.Result) error {
	if len(result.Payload) == 0 {
		return nil
	}
	if err := e.checkAttempt(ctx); err != nil {
		return localFailure("local_execution_result_unconfirmed", true)
	}
	e.runtime.mu.RLock()
	recorder := e.runtime.resultRecorders[e.stage.ExecutorType]
	e.runtime.mu.RUnlock()
	if recorder == nil {
		return localFailure("local_execution_result_unconfirmed", true)
	}
	err := recorder.RecordExternalExecutionResult(ctx, moduleapi.ExternalExecutionResultRequest{
		TaskID: request.TaskID, StageID: request.StageID, Attempt: request.Attempt,
		ExecutorType: request.ExecutorType, RuntimeTargetID: request.RuntimeTargetID, OperationID: request.OperationID,
		Input: request.Input, Protocol: result.Protocol, Result: result.Payload,
	})
	if err != nil {
		return localFailure("local_execution_result_unconfirmed", true)
	}
	return nil
}

func localFailure(code string, uncertain bool) error {
	disposition := moduleapi.RecoveryDispositionFailed
	class := moduleapi.ExecutionFailureClassProvider
	if uncertain {
		disposition = moduleapi.RecoveryDispositionNeedsAttention
		class = moduleapi.ExecutionFailureClassUnknown
	}
	return &moduleapi.ExecutionFailure{Code: code, Class: class, Disposition: disposition, Cause: errors.New(code)}
}

type executionOwnershipStore interface {
	AcquireExecutionOwnership(context.Context) (*taskstore.ExecutionOwnership, error)
	CheckExecutionMode(context.Context, string) error
}

func (r *Runtime) acquireExecutionOwnership(ctx context.Context) error {
	owner, ok := r.repository.(executionOwnershipStore)
	if !ok || r.localExecution == nil {
		return nil
	}
	ownership, err := owner.AcquireExecutionOwnership(ctx)
	if err != nil {
		return err
	}
	if err := owner.CheckExecutionMode(ctx, r.localExecution.mode); err != nil {
		_ = ownership.Close()
		return err
	}
	r.ownershipRelease = ownership.Close
	r.ownershipLost = ownership.Lost
	return nil
}

func (r *Runtime) releaseExecutionOwnership() error {
	if r.ownershipRelease == nil {
		return nil
	}
	release := r.ownershipRelease
	r.ownershipRelease = nil
	return release()
}

// ExecutionAdmission 返回 Task worker 的真实所有权状态，供 Runtime Target 投影执行就绪。
func (r *Runtime) ExecutionAdmission() moduleapi.TaskExecutionAdmission {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := moduleapi.TaskExecutionAdmission{Mode: config.ExecutionModeAgent, DiagnosticCode: "execution_runtime_stopped"}
	if r.localExecution == nil {
		return result
	}
	result.Mode = r.localExecution.mode
	if r.ownershipLost != nil {
		select {
		case <-r.ownershipLost:
			result.DiagnosticCode = "execution_ownership_lost"
			return result
		default:
		}
	}
	result.Ready = r.executionAdmissionReady
	if result.Ready {
		result.DiagnosticCode = "execution_ready"
	}
	return result
}

func (r *Runtime) requireExecutionAdmission() error {
	if r.localExecution == nil {
		return nil
	}
	admission := r.ExecutionAdmission()
	if !admission.Ready {
		return fmt.Errorf("%w: %s", moduleapi.ErrTaskExecutionUnavailable, admission.DiagnosticCode)
	}
	return nil
}

// submissionIntentPlan 冻结请求语义和执行版本，资源事实解析不改变客户端幂等身份。
func (r *Runtime) submissionIntentPlan(plan moduleapi.TaskPlan) (moduleapi.TaskPlan, error) {
	frozen, err := freezeExternalExecutionPayloads(plan)
	if err != nil {
		return moduleapi.TaskPlan{}, err
	}
	if r.localExecution == nil || r.localExecution.mode != config.ExecutionModeLocal {
		return frozen, nil
	}
	for index := range frozen.Stages {
		stage := &frozen.Stages[index]
		if stage.ExternalExecution != nil {
			stage.LocalExecution = &moduleapi.LocalExecutionExpectation{Version: localExecutionVersion, Operation: *stage.ExternalExecution}
			stage.ExternalExecution = nil
		}
	}
	return frozen, nil
}

type idempotentTaskLookup interface {
	LookupIdempotentTask(context.Context, taskmodel.Task) (taskmodel.Task, bool, error)
}

func (r *Runtime) replaySubmission(ctx context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, bool, error) {
	if input.IdempotencyKey == "" {
		return moduleapi.TaskReceipt{}, false, nil
	}
	lookup, ok := r.repository.(idempotentTaskLookup)
	if !ok {
		return moduleapi.TaskReceipt{}, false, nil
	}
	plan, err := r.submissionIntentPlan(input.Plan)
	if err != nil {
		return moduleapi.TaskReceipt{}, false, err
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return moduleapi.TaskReceipt{}, false, err
	}
	key, fingerprint, err := submissionIdentity(input, encoded)
	if err != nil {
		return moduleapi.TaskReceipt{}, false, err
	}
	existing, found, err := lookup.LookupIdempotentTask(ctx, taskmodel.Task{Type: input.Type, Owner: input.Owner, CreatedBy: nullableRequestedBy(input.RequestedBy), IdempotencyKeyHash: key})
	if err != nil || !found {
		return moduleapi.TaskReceipt{}, false, err
	}
	if !sameSubmissionFingerprint(fingerprint, existing.SubmissionFingerprint) {
		return moduleapi.TaskReceipt{}, false, moduleapi.ErrTaskSubmissionConflict
	}
	return moduleapi.TaskReceipt{TaskID: existing.ID, Status: existing.Status}, true, nil
}

func sameSubmissionFingerprint(left, right *string) bool {
	return left != nil && right != nil && *left == *right
}

func (r *Runtime) configuredExecutionMode() string {
	if r.localExecution == nil {
		return ""
	}
	return r.localExecution.mode
}
