const taskFailureCopy: Record<string, string> = {
  invalid_execution_intent: 'task.detail.failureCodes.invalidExecutionIntent',
  unsupported_execution_operation: 'task.detail.failureCodes.unsupportedExecutionOperation',
  docker_runtime_unavailable: 'task.detail.failureCodes.runtimeUnavailable',
  docker_resource_not_found: 'task.detail.failureCodes.resourceNotFound',
  docker_resource_conflict: 'task.detail.failureCodes.resourceConflict',
  provider_operation_failed: 'task.detail.failureCodes.providerOperationFailed',
  agent_execution_interrupted: 'task.detail.failureCodes.agentExecutionInterrupted',
  external_execution_uncertain: 'task.detail.failureCodes.externalExecutionUncertain',
  lease_expired: 'task.detail.failureCodes.externalExecutionUncertain',
  local_material_resolve_failed: 'task.detail.failureCodes.localMaterialResolveFailed',
  local_compose_load_failed: 'task.detail.failureCodes.localComposeLoadFailed',
  local_compose_apply_failed: 'task.detail.failureCodes.localComposeApplyFailed',
  local_container_apply_failed: 'task.detail.failureCodes.localContainerApplyFailed',
  local_build_execute_failed: 'task.detail.failureCodes.localBuildExecuteFailed',
  local_execution_result_unconfirmed: 'task.detail.failureCodes.localExecutionResultUnconfirmed',
  runner_interrupted: 'task.detail.failureCodes.localExecutionInterrupted',
  local_execution_not_ready: 'task.detail.failureCodes.localExecutionNotReady',
};

export function resolveTaskFailureMessage(
  code: string | null | undefined,
  fallback: string | null | undefined,
  translate: (key: string) => string,
) {
  const normalizedCode = code?.trim();
  if (normalizedCode && taskFailureCopy[normalizedCode]) return translate(taskFailureCopy[normalizedCode]);
  if (normalizedCode || fallback?.trim()) return translate('task.detail.failureCodes.unknown');
  return '';
}
