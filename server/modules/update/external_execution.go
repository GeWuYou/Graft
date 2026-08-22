package update

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	// composeUpdateLaunchExecutor 是 Task-owned 启动 Stage；Runtime Agent 拥有 Docker
	// 副作用，Update 只解析一次 fenced transient payload。
	composeUpdateLaunchExecutor         moduleapi.StageExecutorType = "platform.update.controller.launch"
	composeUpdateLaunchOperation        string                      = "platform.update.controller.launch.v1"
	composeUpdateCapability             string                      = "update_controller"
	composeUpdateCapabilityVersion      string                      = "docker/v1"
	composeUpdateLaunchProtocol         string                      = "platform-update-controller/v1"
	composeUpdateLaunchMaterialProtocol string                      = "platform-update-controller-material/v1"
)

const (
	composeUpdateLeaseTTL      = 2 * time.Minute
	composeUpdateLeaseDeadline = 10 * time.Minute
)

// updateControllerMaterial 是刻意保持瞬时的启动材料。它只在 Task Runtime 校验
// lease fence 后生成，且不会写入 Task、Update operation、Agent journal 或 receipt。
type updateControllerMaterial struct {
	ControllerReference string `json:"controller_reference"`
	InputBase64         string `json:"input_b64"`
	RecoveryStateBase64 string `json:"recovery_state_b64,omitempty"`
	RecoveryClaimID     string `json:"recovery_claim_id,omitempty"`
	Recovery            bool   `json:"recovery,omitempty"`
	ComposeRoot         string `json:"compose_root"`
	DockerSocket        string `json:"docker_socket"`
	StateVolume         string `json:"state_volume"`
	OperationID         string `json:"operation_id"`
}

type updateControllerLaunchSlot struct {
	ready    chan struct{}
	targetID int64
	input    RunnerInput
	recovery *RunnerRecoveryInput
	image    string
	claimID  string
	failed   bool
}

// Type 返回该 Update 材料解析器绑定的 Task Stage executor identity。
func (*RolloutService) Type() moduleapi.StageExecutorType { return composeUpdateLaunchExecutor }

// ResolveExternalExecutionMaterial 在 Task Runtime 验证 lease fence 后解析一次性 controller 启动材料。
//
//nolint:cyclop,gocyclo,gocognit // material 校验必须显式保留每个信任边界拒绝分支。
func (s *RolloutService) ResolveExternalExecutionMaterial(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest) (moduleapi.ExternalExecutionMaterial, error) {
	if s == nil || request.ExecutorType != composeUpdateLaunchExecutor || request.OperationID != composeUpdateLaunchOperation || request.TaskID == 0 || request.RuntimeTargetID < 1 {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller material request is invalid")
	}
	var intent struct {
		OperationID   string `json:"operation_id"`
		SourceVersion string `json:"source_version"`
		TargetVersion string `json:"target_version"`
	}
	if err := json.Unmarshal(request.Input, &intent); err != nil || !runnerOperationID.MatchString(intent.OperationID) || strings.TrimSpace(intent.SourceVersion) == "" || strings.TrimSpace(intent.TargetVersion) == "" {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller material request is invalid")
	}
	s.externalInputMu.Lock()
	slot, ok := s.externalLaunches[intent.OperationID]
	targetID := int64(0)
	if ok {
		targetID = slot.targetID
	}
	s.externalInputMu.Unlock()
	if !ok || targetID != request.RuntimeTargetID {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller material is unavailable")
	}
	select {
	case <-slot.ready:
	case <-ctx.Done():
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller material is unavailable")
	}
	s.externalInputMu.Lock()
	recovery := slot.recovery
	image, claimID := slot.image, slot.claimID
	input := slot.input
	failed := slot.failed
	s.externalInputMu.Unlock()
	stateVolume, err := runnerStateVolumeName()
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller state volume is invalid")
	}
	if recovery != nil {
		encoded, err := encodeRunnerRecoveryInput(*recovery)
		if err != nil {
			return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller recovery material encoding failed")
		}
		payload, err := json.Marshal(updateControllerMaterial{ControllerReference: image, RecoveryStateBase64: encoded, RecoveryClaimID: claimID, Recovery: true, StateVolume: stateVolume, OperationID: intent.OperationID})
		if err != nil {
			return moduleapi.ExternalExecutionMaterial{}, errors.New("encode update controller recovery material")
		}
		return moduleapi.ExternalExecutionMaterial{Protocol: composeUpdateLaunchMaterialProtocol, Payload: payload}, nil
	}
	if failed || input.TaskID != request.TaskID || input.OperationID != intent.OperationID || input.SourceVersion != intent.SourceVersion || input.TargetVersion != intent.TargetVersion {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller material is unavailable")
	}
	if err := ValidateRunnerInput(input); err != nil {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller material is invalid")
	}
	encoded, err := encodeRunnerInput(input)
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("update controller material encoding failed")
	}
	material := updateControllerMaterial{
		ControllerReference: input.Preflight.RunnerReference,
		InputBase64:         encoded,
		ComposeRoot:         input.Preflight.ComposeRoot,
		DockerSocket:        strings.TrimPrefix(input.Preflight.DockerSocket, "unix://"),
		StateVolume:         stateVolume,
		OperationID:         input.OperationID,
	}
	payload, err := json.Marshal(material)
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("encode update controller material")
	}
	return moduleapi.ExternalExecutionMaterial{Protocol: composeUpdateLaunchMaterialProtocol, Payload: payload}, nil
}

func encodeRunnerRecoveryInput(input RunnerRecoveryInput) (string, error) {
	contents, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode compose runner recovery state: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(contents), nil
}

func (s *RolloutService) prepareRecoveryLaunch(operationID string, targetID int64, recovery RunnerRecoveryInput, image, claimID string) error {
	s.externalInputMu.Lock()
	slot := s.externalLaunches[operationID]
	if slot == nil {
		slot = &updateControllerLaunchSlot{ready: make(chan struct{}), targetID: targetID}
		s.externalLaunches[operationID] = slot
	} else if slot.targetID != targetID || slot.failed {
		s.externalInputMu.Unlock()
		return errors.New("update controller recovery launch slot is unavailable")
	}
	slot.recovery, slot.image, slot.claimID = &recovery, image, claimID
	ready := slot.ready
	select {
	case <-ready:
	default:
		close(ready)
	}
	s.externalInputMu.Unlock()
	return nil
}

func (s *RolloutService) prepareExternalLaunch(operationID string, targetID int64) error {
	if s == nil || !runnerOperationID.MatchString(operationID) || targetID < 1 {
		return errors.New("update controller launch slot is invalid")
	}
	s.externalInputMu.Lock()
	defer s.externalInputMu.Unlock()
	if _, exists := s.externalLaunches[operationID]; exists {
		return errors.New("update controller launch slot already exists")
	}
	s.externalLaunches[operationID] = &updateControllerLaunchSlot{ready: make(chan struct{}), targetID: targetID}
	return nil
}

func (s *RolloutService) completeExternalLaunch(operationID string, targetID int64, input RunnerInput) error {
	s.externalInputMu.Lock()
	defer s.externalInputMu.Unlock()
	slot := s.externalLaunches[operationID]
	if slot == nil || slot.targetID != targetID || input.OperationID != operationID {
		return errors.New("update controller launch slot is unavailable")
	}
	select {
	case <-slot.ready:
		return errors.New("update controller launch slot is already completed")
	default:
	}
	slot.input = input
	close(slot.ready)
	return nil
}

func (s *RolloutService) failExternalLaunch(operationID string) {
	s.externalInputMu.Lock()
	defer s.externalInputMu.Unlock()
	slot := s.externalLaunches[operationID]
	if slot == nil {
		return
	}
	select {
	case <-slot.ready:
	default:
		slot.failed = true
		close(slot.ready)
	}
	delete(s.externalLaunches, operationID)
}

func (s *RolloutService) clearExternalLaunch(operationID string) {
	s.externalInputMu.Lock()
	delete(s.externalLaunches, operationID)
	s.externalInputMu.Unlock()
}

func updateControllerExecutionExpectation(targetID int64, payloadSHA string) *moduleapi.ExternalExecutionExpectation {
	return &moduleapi.ExternalExecutionExpectation{
		RuntimeTargetID: targetID, ProviderID: "docker", Capability: composeUpdateCapability,
		CapabilityVersion: composeUpdateCapabilityVersion, Protocol: composeUpdateLaunchProtocol,
		OperationID: composeUpdateLaunchOperation, PayloadSHA256: payloadSHA,
		LeaseTTL: composeUpdateLeaseTTL, AbsoluteDeadline: composeUpdateLeaseDeadline,
	}
}

func updateControllerTaskInput(operation ComposeUpdateOperation) ([]byte, error) {
	input, err := json.Marshal(struct {
		OperationID   string `json:"operation_id"`
		SourceVersion string `json:"source_version"`
		TargetVersion string `json:"target_version"`
	}{OperationID: operation.OperationID, SourceVersion: operation.SourceVersion, TargetVersion: operation.TargetVersion})
	if err != nil {
		return nil, fmt.Errorf("encode compose update task input: %w", err)
	}
	return input, nil
}
