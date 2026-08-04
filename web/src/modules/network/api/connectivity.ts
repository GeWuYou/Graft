import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

type Target = components['schemas']['platform-network-connectivity-target'];
type Check = components['schemas']['platform-network-connectivity-check'];
type Aggregate = components['schemas']['platform-network-connectivity-aggregate'];
type Report = components['schemas']['platform-network-connectivity-report'];

export type ConnectivityTarget = Target;
export type ConnectivityCheck = Check;
export type ConnectivityAggregate = Aggregate;
export type ConnectivityReport = Report;

export function getConnectivityTargets() {
  return request.get<{ items: Target[] }>({ url: OPENAPI_RUNTIME_PATH.getPlatformConnectivityTargets });
}

export function getConnectivityLatest() {
  return request.get<{ items: Check[] }>({ url: OPENAPI_RUNTIME_PATH.getPlatformConnectivityLatest });
}

export function getConnectivityAggregate() {
  return request.get<Aggregate>({ url: OPENAPI_RUNTIME_PATH.getPlatformConnectivityAggregate });
}

export function runConnectivityBatch() {
  return request.post<{ items: Check[] }>({ url: OPENAPI_RUNTIME_PATH.postPlatformConnectivityBatchRun });
}

export function runConnectivityTarget(targetId: string) {
  return request.post<{ check: Check; report: Report }>({
    url: buildOpenApiRuntimePath('postPlatformConnectivityRun', { targetId }),
  });
}

export function getConnectivityHistory(targetId: string, limit = 20) {
  return request.get<{ items: Check[] }>({
    url: buildOpenApiRuntimePath('getPlatformConnectivityHistory', { targetId }),
    params: { limit },
  });
}

export function getConnectivityReport(targetId: string, checkId: number) {
  return request.get<Report>({
    url: buildOpenApiRuntimePath('getPlatformConnectivityReport', { targetId, checkId }),
  });
}
