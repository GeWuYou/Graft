const taskFailureCopy: Record<string, string> = {
  invalid_execution_intent: 'task.failureCodes.invalidExecutionIntent',
  unsupported_execution_operation: 'task.failureCodes.unsupportedExecutionOperation',
  docker_runtime_unavailable: 'task.failureCodes.runtimeUnavailable',
  docker_resource_not_found: 'task.failureCodes.resourceNotFound',
  docker_resource_conflict: 'task.failureCodes.resourceConflict',
  provider_operation_failed: 'task.failureCodes.providerOperationFailed',
  agent_execution_interrupted: 'task.failureCodes.agentExecutionInterrupted',
  external_execution_uncertain: 'task.failureCodes.externalExecutionUncertain',
  lease_expired: 'task.failureCodes.externalExecutionUncertain',
};

export function resolveTaskFailureMessage(
  code: string | null | undefined,
  fallback: string | null | undefined,
  translate: (key: string) => string,
) {
  const normalizedCode = code?.trim();
  if (normalizedCode && taskFailureCopy[normalizedCode]) return translate(taskFailureCopy[normalizedCode]);
  if (normalizedCode || fallback?.trim()) return translate('task.failureCodes.unknown');
  return '';
}
