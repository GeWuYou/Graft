package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/moduleapi"
)

type dockerImageTagTaskInput struct {
	ImageRef  string `json:"image_ref"`
	TargetRef string `json:"target_ref"`
}

type dockerImageUntagTaskInput struct {
	ImageRef string `json:"image_ref"`
	TagRef   string `json:"tag_ref"`
}

type dockerImageRemoveTaskInput struct {
	ImageRef string `json:"image_ref"`
	Force    bool   `json:"force"`
}

type dockerNetworkCreateTaskInput struct {
	Name       string                      `json:"name"`
	Driver     string                      `json:"driver"`
	Internal   bool                        `json:"internal"`
	Attachable bool                        `json:"attachable"`
	IPAM       *dockerNetworkIPAMTaskInput `json:"ipam,omitempty"`
}

type dockerNetworkIPAMTaskInput struct {
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway,omitempty"`
}

type dockerNetworkRemoveTaskInput struct {
	NetworkRef string `json:"network_ref"`
}

type dockerVolumeRemoveTaskInput struct {
	VolumeRef string `json:"volume_ref"`
	Force     bool   `json:"force"`
}

func (s *service) SubmitDockerImageTag(ctx context.Context, imageRef, targetRef string, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if err := validateDockerImageReference(imageRef); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if err := validateDockerImageReference(targetRef); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	return s.submitContainerExternalTask(ctx, containerImageTagOperation, imageRef, dockerImageTagTaskInput{ImageRef: imageRef, TargetRef: targetRef}, requestedBy, idempotencyKey)
}

func (s *service) SubmitDockerImageUntag(ctx context.Context, imageRef, tagRef string, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	if err := validateDockerImageReference(imageRef); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if err := validateDockerImageReference(tagRef); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	image, err := s.DockerImage(ctx, imageRef)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if !dockerImageHasRepositoryTag(image, tagRef) {
		return moduleapi.TaskReceipt{}, errDockerImageTagNotAssociated
	}
	return s.submitContainerExternalTask(ctx, containerImageUntagOperation, imageRef, dockerImageUntagTaskInput{ImageRef: imageRef, TagRef: tagRef}, requestedBy, idempotencyKey)
}

func (s *service) SubmitDockerImageRemove(ctx context.Context, imageRef string, force bool, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	if err := validateDockerImageReference(imageRef); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	return s.submitContainerExternalTask(ctx, containerImageRemoveOperation, imageRef, dockerImageRemoveTaskInput{ImageRef: imageRef, Force: force}, requestedBy, idempotencyKey)
}

func (s *service) SubmitDockerImageBatchRemove(ctx context.Context, imageRefs []string, force bool, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	if len(imageRefs) == 0 || len(imageRefs) > maxContainerBatchActionIDs {
		return moduleapi.TaskReceipt{}, errInvalidListQuery
	}
	inputs := make([]any, 0, len(imageRefs))
	owners := make([]string, 0, len(imageRefs))
	seen := make(map[string]struct{}, len(imageRefs))
	for _, rawRef := range imageRefs {
		ref := strings.TrimSpace(rawRef)
		if err := validateDockerImageReference(ref); err != nil {
			return moduleapi.TaskReceipt{}, err
		}
		if _, exists := seen[ref]; exists {
			return moduleapi.TaskReceipt{}, errInvalidListQuery
		}
		seen[ref] = struct{}{}
		owners = append(owners, ref)
		inputs = append(inputs, dockerImageRemoveTaskInput{ImageRef: ref, Force: force})
	}
	return s.submitContainerExternalBatchTask(ctx, containerImageRemoveOperation, owners, inputs, requestedBy, idempotencyKey)
}

func (s *service) SubmitDockerNetworkCreate(ctx context.Context, command DockerNetworkCreateCommand, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	command.Name, command.Driver = strings.TrimSpace(command.Name), strings.TrimSpace(command.Driver)
	if command.Name == "" || !isDockerNetworkDriver(command.Driver) || len(command.Labels) != 0 {
		return moduleapi.TaskReceipt{}, errInvalidDockerNetworkRequest
	}
	payload := dockerNetworkCreateTaskInput{Name: command.Name, Driver: command.Driver, Internal: command.Internal, Attachable: command.Attachable}
	if command.IPAM != nil {
		if !validDockerNetworkIPAM(command.IPAM) {
			return moduleapi.TaskReceipt{}, errInvalidDockerNetworkRequest
		}
		payload.IPAM = &dockerNetworkIPAMTaskInput{Subnet: strings.TrimSpace(command.IPAM.Subnet), Gateway: strings.TrimSpace(command.IPAM.Gateway)}
	}
	return s.submitContainerExternalTask(ctx, containerNetworkCreateOperation, command.Name, payload, requestedBy, idempotencyKey)
}

func (s *service) SubmitDockerNetworkRemove(ctx context.Context, id, confirmation string, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	item, err := s.DockerNetwork(ctx, id)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if strings.TrimSpace(confirmation) != item.Name {
		return moduleapi.TaskReceipt{}, errDockerNetworkConfirmMismatch
	}
	if isDockerDefaultNetwork(item.Name) {
		return moduleapi.TaskReceipt{}, errDockerNetworkDefaultProtected
	}
	if item.ContainerCount > 0 {
		return moduleapi.TaskReceipt{}, errDockerNetworkInUse
	}
	return s.submitContainerExternalTask(ctx, containerNetworkRemoveOperation, item.ID, dockerNetworkRemoveTaskInput{NetworkRef: item.ID}, requestedBy, idempotencyKey)
}

func (s *service) SubmitDockerVolumeRemove(ctx context.Context, volumeRef string, force bool, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	volumeRef = strings.TrimSpace(volumeRef)
	if volumeRef == "" {
		return moduleapi.TaskReceipt{}, errInvalidRef
	}
	volume, err := s.DockerVolume(ctx, volumeRef)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	receipt, submitErr := s.submitContainerExternalTask(ctx, containerVolumeRemoveOperation, volumeRef, dockerVolumeRemoveTaskInput{VolumeRef: volumeRef, Force: force}, requestedBy, idempotencyKey)
	s.publishDockerVolumeAudit(ctx, volume, force, submitErr)
	return receipt, submitErr
}

func (s *service) SubmitDockerVolumeBatchRemove(ctx context.Context, volumeRefs []string, force bool, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	if len(volumeRefs) == 0 || len(volumeRefs) > maxDockerVolumeBatchRemoveIDs {
		return moduleapi.TaskReceipt{}, errInvalidListQuery
	}
	inputs := make([]any, 0, len(volumeRefs))
	owners := make([]string, 0, len(volumeRefs))
	seen := make(map[string]struct{}, len(volumeRefs))
	for _, rawRef := range volumeRefs {
		ref := strings.TrimSpace(rawRef)
		if ref == "" {
			return moduleapi.TaskReceipt{}, errInvalidListQuery
		}
		if _, exists := seen[ref]; exists {
			return moduleapi.TaskReceipt{}, errInvalidListQuery
		}
		seen[ref] = struct{}{}
		owners = append(owners, ref)
		inputs = append(inputs, dockerVolumeRemoveTaskInput{VolumeRef: ref, Force: force})
	}
	return s.submitContainerExternalBatchTask(ctx, containerVolumeRemoveOperation, owners, inputs, requestedBy, idempotencyKey)
}

func (s *service) submitContainerExternalTask(ctx context.Context, operation, ownerID string, payload any, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if s == nil || s.tasks == nil {
		return moduleapi.TaskReceipt{}, errors.New("task service is unavailable")
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	input, err := json.Marshal(payload)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal container external task input: %w", err)
	}
	execution, err := s.containerExternalExecution(ctx, operation, input)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type: moduleapi.TaskType(operation), Owner: moduleapi.TaskOwner{Type: containerExternalTaskOwnerType(operation, false), ID: ownerID},
		RequestedBy: requestedBy, IdempotencyKey: idempotencyKey, Input: input,
		Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{
			Key: operation, ExecutorType: moduleapi.StageExecutorType(operation), Input: input,
			RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile,
			ExternalExecution: execution,
		}}},
	})
}

func (s *service) submitContainerExternalBatchTask(ctx context.Context, operation string, ownerValues []string, payloads []any, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if s == nil || s.tasks == nil || len(ownerValues) == 0 || len(ownerValues) != len(payloads) {
		return moduleapi.TaskReceipt{}, errors.New("container external batch task is unavailable")
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	ownerID, err := containerExternalBatchOwnerID(ownerValues)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal container external batch owner: %w", err)
	}
	stages := make([]moduleapi.StagePlan, 0, len(payloads))
	for index, payload := range payloads {
		input, err := json.Marshal(payload)
		if err != nil {
			return moduleapi.TaskReceipt{}, fmt.Errorf("marshal container external batch input: %w", err)
		}
		execution, err := s.containerExternalExecution(ctx, operation, input)
		if err != nil {
			return moduleapi.TaskReceipt{}, err
		}
		stages = append(stages, moduleapi.StagePlan{
			Key: fmt.Sprintf("%s-%d", operation, index+1), ExecutorType: moduleapi.StageExecutorType(operation), Input: input,
			RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile,
			ExternalExecution: execution,
		})
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type:        moduleapi.TaskType(strings.TrimSuffix(operation, ".v1") + ".batch.v1"),
		Owner:       moduleapi.TaskOwner{Type: containerExternalTaskOwnerType(operation, true), ID: ownerID},
		RequestedBy: requestedBy, IdempotencyKey: idempotencyKey, Plan: moduleapi.TaskPlan{Stages: stages},
	})
}
