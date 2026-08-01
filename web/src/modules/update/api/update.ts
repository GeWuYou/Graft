import { buildOpenApiRuntimePath } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import {
  openRealtimeTopicEventStream,
  type RealtimeEventStreamState,
  type RealtimeTopicEventStreamController,
} from '@/shared/realtime/sse-client';
import { request } from '@/utils/request';

import { UPDATE_API_PATH } from '../contract/paths';
import { buildUpdateOperationTopicName } from '../contract/realtime';
import type {
  CreateUpdateOperationRequest,
  UpdateFailureDiagnostic,
  UpdateOperation,
  UpdateOperationLaunchAcknowledgement,
  UpdateStatus,
} from '../types/update';

type UpdateStatusEnvelope =
  paths[typeof UPDATE_API_PATH.STATUS]['get']['responses'][200]['content']['application/json'];
type UpdateStatusData = NonNullable<UpdateStatusEnvelope['data']>;

type CheckForUpdatesEnvelope =
  paths[typeof UPDATE_API_PATH.CHECK]['post']['responses'][200]['content']['application/json'];
type CheckForUpdatesData = NonNullable<CheckForUpdatesEnvelope['data']>;

export function getUpdateStatus() {
  return request.get<UpdateStatusData>({ url: UPDATE_API_PATH.STATUS }) as Promise<UpdateStatus>;
}

export function checkForUpdates() {
  return request.post<CheckForUpdatesData>({ url: UPDATE_API_PATH.CHECK }) as Promise<UpdateStatus>;
}

export function getUpdateOperations() {
  return request.get<UpdateOperation[]>({ url: UPDATE_API_PATH.OPERATIONS, params: { limit: 20 } }) as Promise<
    UpdateOperation[]
  >;
}

/** 读取单个升级操作，用于壳层升级进度会话轮询。 */
export function getUpdateOperation(operationID: string) {
  return request.get<UpdateOperation>({
    url: buildOpenApiRuntimePath('getPlatformUpdateOperation', { operationID }),
  }) as Promise<UpdateOperation>;
}

/** 订阅单次升级操作的服务端快照；票据和断线重连由统一实时 SSE 客户端拥有。 */
export function subscribeToUpdateOperation(
  operationID: string,
  options: Readonly<{
    onOperation: (operation: UpdateOperation) => void | Promise<void>;
    onStateChange?: (state: RealtimeEventStreamState) => void;
  }>,
): RealtimeTopicEventStreamController {
  return openRealtimeTopicEventStream<UpdateOperation>({
    topic: buildUpdateOperationTopicName(operationID),
    onMessage: options.onOperation,
    onStateChange: options.onStateChange,
  });
}

/** 操作结束后读取服务端映射出的脱敏失败原因，不透传 runner 原始输出。 */
export function getUpdateOperationDiagnostic(operationID: string) {
  return request.get<UpdateFailureDiagnostic>({
    url: UPDATE_API_PATH.OPERATION_DIAGNOSTIC.replace('{operationID}', encodeURIComponent(operationID)),
  }) as Promise<UpdateFailureDiagnostic>;
}

export function createUpdateOperation(payload: CreateUpdateOperationRequest) {
  return request.post<UpdateOperationLaunchAcknowledgement>({
    url: UPDATE_API_PATH.OPERATIONS,
    data: payload,
  }) as Promise<UpdateOperationLaunchAcknowledgement>;
}

/** 仅启动 runner 恢复流程；恢复后的真实状态仍须通过操作快照读取。 */
type UpdateFailureDiagnosticEnvelope =
  paths['/api/platform/updates/diagnostics/{requestId}']['get']['responses'][200]['content']['application/json'];
type UpdateFailureDiagnosticData = NonNullable<UpdateFailureDiagnosticEnvelope['data']>;

/** 读取服务器保存的脱敏启动失败诊断；调用方必须已具备更新管理权限。 */
export function getUpdateFailureDiagnostic(requestId: string) {
  return request.get<UpdateFailureDiagnosticData>({
    url: buildOpenApiRuntimePath('getPlatformUpdateFailureDiagnostic', { requestId }),
  }) as Promise<UpdateFailureDiagnostic>;
}
