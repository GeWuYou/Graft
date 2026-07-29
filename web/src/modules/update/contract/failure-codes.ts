import type { components } from '@/contracts/openapi/generated/schema';

/** 更新启动失败码由 OpenAPI 定义；此处只提供模块内安全提示的派生映射。 */
export type UpdateOperationFailureCode = components['schemas']['platform-update-rollout-failure-code'];

/** 提供与 OpenAPI 失败码枚举一致的稳定值，供模块消费者避免重复书写 wire code。 */
export const UPDATE_OPERATION_FAILURE_CODE = {
  CATALOG_STALE: 'PLATFORM_UPDATE_CATALOG_STALE',
  INSTALLATION_UNAVAILABLE: 'PLATFORM_UPDATE_INSTALLATION_UNAVAILABLE',
  IMAGE_TAG_UNCONFIGURED: 'PLATFORM_UPDATE_IMAGE_TAG_UNCONFIGURED',
  IMAGE_TAG_INVALID: 'PLATFORM_UPDATE_IMAGE_TAG_INVALID',
  COMPOSE_CANDIDATE_CONFIRMATION_REQUIRED: 'PLATFORM_UPDATE_COMPOSE_CANDIDATE_CONFIRMATION_REQUIRED',
  NO_ELIGIBLE_NEWER_RELEASE: 'PLATFORM_UPDATE_NO_ELIGIBLE_NEWER_RELEASE',
  SOURCE_VERSION_UNSUPPORTED: 'PLATFORM_UPDATE_SOURCE_VERSION_UNSUPPORTED',
  INVALID_TARGET: 'PLATFORM_UPDATE_INVALID_TARGET',
  COMPOSE_CANDIDATE_INVALID: 'PLATFORM_UPDATE_COMPOSE_CANDIDATE_INVALID',
  COMPOSE_PREFLIGHT_FAILED: 'PLATFORM_UPDATE_COMPOSE_PREFLIGHT_FAILED',
  OPERATION_START_FAILED: 'PLATFORM_UPDATE_OPERATION_START_FAILED',
  RUNNER_TERMINAL_FAILED: 'PLATFORM_UPDATE_RUNNER_TERMINAL_FAILED',
} as const satisfies Record<string, UpdateOperationFailureCode>;

/** 将已知失败码映射为安全的本地化消息 key；未知服务端值不得通过此映射展示。 */
export const UPDATE_OPERATION_FAILURE_MESSAGE_KEY: Record<UpdateOperationFailureCode, string> = {
  [UPDATE_OPERATION_FAILURE_CODE.CATALOG_STALE]: 'update.center.confirmation.failure.catalogStale',
  [UPDATE_OPERATION_FAILURE_CODE.INSTALLATION_UNAVAILABLE]: 'update.center.confirmation.failure.executionUnavailable',
  [UPDATE_OPERATION_FAILURE_CODE.IMAGE_TAG_UNCONFIGURED]: 'update.center.confirmation.failure.imageTagUnconfigured',
  [UPDATE_OPERATION_FAILURE_CODE.IMAGE_TAG_INVALID]: 'update.center.confirmation.failure.imageTagInvalid',
  [UPDATE_OPERATION_FAILURE_CODE.COMPOSE_CANDIDATE_CONFIRMATION_REQUIRED]:
    'update.center.confirmation.failure.composeCandidateConfirmationRequired',
  [UPDATE_OPERATION_FAILURE_CODE.NO_ELIGIBLE_NEWER_RELEASE]:
    'update.center.confirmation.failure.noEligibleNewerRelease',
  [UPDATE_OPERATION_FAILURE_CODE.SOURCE_VERSION_UNSUPPORTED]: 'update.center.confirmation.failure.minimumSourceVersion',
  [UPDATE_OPERATION_FAILURE_CODE.INVALID_TARGET]: 'update.center.confirmation.failure.targetInvalid',
  [UPDATE_OPERATION_FAILURE_CODE.COMPOSE_CANDIDATE_INVALID]:
    'update.center.confirmation.failure.composeCandidateInvalid',
  [UPDATE_OPERATION_FAILURE_CODE.COMPOSE_PREFLIGHT_FAILED]: 'update.center.confirmation.failure.composePreflightFailed',
  [UPDATE_OPERATION_FAILURE_CODE.OPERATION_START_FAILED]: 'update.center.confirmation.failure.startFailed',
  [UPDATE_OPERATION_FAILURE_CODE.RUNNER_TERMINAL_FAILED]: 'update.center.confirmation.failure.runnerTerminalFailed',
};

/** 仅接受映射表自有的 OpenAPI 失败码，避免原型属性被误当作可展示的服务端错误。 */
export function isUpdateOperationFailureCode(code: string): code is UpdateOperationFailureCode {
  return Object.prototype.hasOwnProperty.call(UPDATE_OPERATION_FAILURE_MESSAGE_KEY, code);
}
