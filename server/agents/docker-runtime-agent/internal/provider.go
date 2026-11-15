package agent

import (
	"context"
	"net/http"

	"graft/server/internal/dockerexecution"
)

const (
	failureInvalidIntent         = "invalid_execution_intent"
	failureUnsupportedAction     = "unsupported_execution_operation"
	failureRuntimeUnavailable    = "docker_runtime_unavailable"
	failureResourceNotFound      = "docker_resource_not_found"
	failureResourceConflict      = "docker_resource_conflict"
	failureProviderOperation     = "provider_operation_failed"
	failureInterrupted           = "agent_execution_interrupted"
	buildExecutionCapability     = "oci-build"
	buildExecutionResultProtocol = "build-execution-result/v1"
	buildExecutionProtocol       = "build-execution/v1"
	buildImageLocalOperation     = "build.image.local.v1"
	buildImagePublishOperation   = "build.image.publish.v1"
	containerExecutionProtocol   = "container-execution/v1"
)

func executeProviderOperation(ctx context.Context, client *http.Client, c config, lease executionLease) executionResult {
	return dockerexecution.Execute(ctx, dockerexecution.Environment{
		DockerSocket: c.DockerSocket,
		ResolveMaterial: func(ctx context.Context, _ dockerexecution.Request) (dockerexecution.Material, error) {
			return resolveExecutionMaterial(ctx, client, c.AgentURL, lease)
		},
	}, dockerexecution.Request{Capability: lease.Capability, Protocol: lease.Protocol, OperationID: lease.OperationID, Input: lease.Input})
}

func strictDecode(payload []byte, destination any) error {
	return dockerexecution.Decode(payload, destination)
}
