package container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

const (
	containerExecutionProvider          = "docker"
	containerExecutionCapability        = "container_execution"
	containerExecutionCapabilityVersion = "docker/v1"
	containerExecutionProtocol          = "container-execution/v1"
	containerExecutionLeaseTTL          = 30 * time.Second
	containerExecutionDeadline          = 5 * time.Minute
	containerImagePullDeadline          = 30 * time.Minute
	containerExternalOwnerPrefix        = "container_external_"
	containerExternalBatchOwnerPrefix   = "sha256:"
)

const (
	containerLifecycleStartOperation   = "container.lifecycle.start.v1"
	containerLifecycleStopOperation    = "container.lifecycle.stop.v1"
	containerLifecycleRestartOperation = "container.lifecycle.restart.v1"
	containerLifecycleRemoveOperation  = "container.lifecycle.remove.v1"
	containerImagePullOperation        = "container.image.pull.v1"
	containerImageTagOperation         = "container.image.tag.v1"
	containerImageUntagOperation       = "container.image.untag.v1"
	containerImageRemoveOperation      = "container.image.remove.v1"
	containerNetworkCreateOperation    = "container.network.create.v1"
	containerNetworkRemoveOperation    = "container.network.remove.v1"
	containerVolumeRemoveOperation     = "container.volume.remove.v1"
)

type containerExternalTaskOwnerAuthorizer struct {
	service    *service
	ownerType  string
	permission string
	batch      bool
}

func (a containerExternalTaskOwnerAuthorizer) OwnerType() string { return a.ownerType }

func (a containerExternalTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.service == nil || a.service.authorizer == nil {
		return moduleapi.ErrUnauthenticated
	}
	if owner.Type != a.ownerType || strings.TrimSpace(owner.ID) == "" {
		return errors.New("container external task owner is invalid")
	}
	if a.batch && !isContainerExternalBatchOwnerID(owner.ID) {
		return errors.New("container external batch task owner is invalid")
	}
	if !isContainerLifecycleTaskOwnerAction(action) {
		return errors.New("container external task owner action is unsupported")
	}
	return a.service.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, a.permission)
}

func registerContainerExternalTaskOwners(registrar moduleapi.TaskRuntimeRegistrar, service *service) error {
	if registrar == nil || service == nil {
		return errors.New("container external task dependencies are unavailable")
	}
	bindings := []struct {
		operation  string
		permission string
		batch      bool
	}{
		{containerImagePullOperation, containercontract.DockerImagePullPermission.String(), false},
		{containerImageTagOperation, containercontract.DockerImageTagPermission.String(), false},
		{containerImageUntagOperation, containercontract.DockerImageUntagPermission.String(), false},
		{containerImageRemoveOperation, containercontract.DockerImageRemovePermission.String(), false},
		{containerImageRemoveOperation, containercontract.DockerImageRemovePermission.String(), true},
		{containerNetworkCreateOperation, containercontract.DockerNetworkCreatePermission.String(), false},
		{containerNetworkRemoveOperation, containercontract.DockerNetworkRemovePermission.String(), false},
		{containerVolumeRemoveOperation, containercontract.ContainerVolumeRemovePermission.String(), false},
		{containerVolumeRemoveOperation, containercontract.ContainerVolumeRemovePermission.String(), true},
	}
	for _, binding := range bindings {
		if err := registrar.RegisterTaskOwnerAuthorizer(containerExternalTaskOwnerAuthorizer{
			service: service, ownerType: containerExternalTaskOwnerType(binding.operation, binding.batch), permission: binding.permission, batch: binding.batch,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) containerExternalExecution(ctx context.Context, operation string, input json.RawMessage) (*moduleapi.ExternalExecutionExpectation, error) {
	if s == nil || s.runtimeTargets == nil {
		return nil, errors.New("container execution runtime target reader is unavailable")
	}
	if !isContainerExternalOperation(operation) {
		return nil, errors.New("container external operation is unsupported")
	}
	target, err := s.runtimeTargets.ReadDockerTarget(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("resolve container execution runtime target: %w", err)
	}
	if target.ID < 1 || strings.TrimSpace(target.Provider) != containerExecutionProvider {
		return nil, errors.New("container execution runtime target is unsupported")
	}
	deadline := containerExecutionDeadline
	if operation == containerImagePullOperation {
		deadline = containerImagePullDeadline
	}
	digest := sha256.Sum256(input)
	return &moduleapi.ExternalExecutionExpectation{
		RuntimeTargetID: target.ID, ProviderID: containerExecutionProvider,
		Capability: containerExecutionCapability, CapabilityVersion: containerExecutionCapabilityVersion,
		Protocol: containerExecutionProtocol, OperationID: operation, PayloadSHA256: hex.EncodeToString(digest[:]),
		LeaseTTL: containerExecutionLeaseTTL, AbsoluteDeadline: deadline,
	}, nil
}

func isContainerExternalOperation(operation string) bool {
	switch operation {
	case containerLifecycleStartOperation, containerLifecycleStopOperation, containerLifecycleRestartOperation,
		containerLifecycleRemoveOperation, containerImagePullOperation, containerImageTagOperation,
		containerImageUntagOperation, containerImageRemoveOperation, containerNetworkCreateOperation,
		containerNetworkRemoveOperation, containerVolumeRemoveOperation:
		return true
	default:
		return false
	}
}

func containerLifecycleOperation(action string) string {
	switch action {
	case containerActionStart:
		return containerLifecycleStartOperation
	case containerActionStop:
		return containerLifecycleStopOperation
	case containerActionRestart:
		return containerLifecycleRestartOperation
	case containerActionRemove:
		return containerLifecycleRemoveOperation
	default:
		return ""
	}
}

func containerExternalTaskOwnerType(operation string, batch bool) string {
	value := strings.TrimSuffix(strings.TrimPrefix(operation, "container."), ".v1")
	value = strings.NewReplacer(".", "_").Replace(value)
	if batch {
		value += "_batch"
	}
	return containerExternalOwnerPrefix + value
}

func containerExternalBatchOwnerID(values []string) (string, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return containerExternalBatchOwnerPrefix + hex.EncodeToString(digest[:]), nil
}

func isContainerExternalBatchOwnerID(ownerID string) bool {
	if !strings.HasPrefix(ownerID, containerExternalBatchOwnerPrefix) {
		return false
	}
	digest := strings.TrimPrefix(ownerID, containerExternalBatchOwnerPrefix)
	decoded, err := hex.DecodeString(digest)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == digest
}
