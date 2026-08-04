import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

type Target = components['schemas']['platform-network-connectivity-target'];
type CustomTarget = components['schemas']['platform-network-connectivity-custom-target'];
type Check = components['schemas']['platform-network-connectivity-check'];
type Aggregate = components['schemas']['platform-network-connectivity-aggregate'];
type Report = components['schemas']['platform-network-connectivity-report'];
type Probe = components['schemas']['platform-network-connectivity-probe'];
type TargetsOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityTargets]['get'];
type CustomTargetsOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityCustomTargets]['get'];
type CreateCustomTargetOperation = paths[typeof OPENAPI_RUNTIME_PATH.postPlatformConnectivityCustomTarget]['post'];
type LatestOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityLatest]['get'];
type AggregateOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityAggregate]['get'];
type BatchRunOperation = paths[typeof OPENAPI_RUNTIME_PATH.postPlatformConnectivityBatchRun]['post'];
type RunOperation = paths[typeof OPENAPI_RUNTIME_PATH.postPlatformConnectivityRun]['post'];
type HistoryOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityHistory]['get'];
type ReportOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityReport]['get'];
type TraceOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityTrace]['get'];
type ExportOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformConnectivityExport]['get'];
type ResponseData<Operation> =
  NonNullable<
    Operation extends { responses: { 200: { content: { 'application/json': infer Response } } } } ? Response : never
  > extends infer Envelope
    ? Envelope extends { data: infer Data }
      ? Data
      : never
    : never;
type TargetsResponse = ResponseData<TargetsOperation>;
type CustomTargetsResponse = ResponseData<CustomTargetsOperation>;
type CreateCustomTargetResponse = NonNullable<
  CreateCustomTargetOperation['responses'][201]['content']['application/json']['data']
>;
type LatestResponse = ResponseData<LatestOperation>;
type AggregateResponse = ResponseData<AggregateOperation>;
type BatchRunResponse = ResponseData<BatchRunOperation>;
type RunResponse = ResponseData<RunOperation>;
type HistoryResponse = ResponseData<HistoryOperation>;
type ReportResponse = ResponseData<ReportOperation>;
type TraceResponse = ResponseData<TraceOperation>;
type ExportResponse = ResponseData<ExportOperation>;

export type ConnectivityTarget = Target;
export type ConnectivityCustomTarget = CustomTarget;
export type ConnectivityCheck = Check;
export type ConnectivityAggregate = Aggregate;
export type ConnectivityReport = Report;
export type ConnectivityProbe = Probe;
export type ConnectivityTrace = TraceResponse;

export function getConnectivityTargets() {
  return request.get<TargetsResponse>({ url: OPENAPI_RUNTIME_PATH.getPlatformConnectivityTargets });
}

export function getConnectivityCustomTargets() {
  return request.get<CustomTargetsResponse>({ url: OPENAPI_RUNTIME_PATH.getPlatformConnectivityCustomTargets });
}

export function createConnectivityCustomTarget(
  data: components['schemas']['create-platform-network-connectivity-custom-target-request'],
) {
  return request.post<CreateCustomTargetResponse>({
    url: OPENAPI_RUNTIME_PATH.postPlatformConnectivityCustomTarget,
    data,
  });
}

export function deleteConnectivityCustomTarget(targetId: string) {
  return request.delete<void>({
    url: buildOpenApiRuntimePath('deletePlatformConnectivityCustomTarget', { targetId }),
  });
}

export function getConnectivityLatest() {
  return request.get<LatestResponse>({ url: OPENAPI_RUNTIME_PATH.getPlatformConnectivityLatest });
}

export function getConnectivityAggregate() {
  return request.get<AggregateResponse>({ url: OPENAPI_RUNTIME_PATH.getPlatformConnectivityAggregate });
}

export function runConnectivityBatch() {
  return request.post<BatchRunResponse>({ url: OPENAPI_RUNTIME_PATH.postPlatformConnectivityBatchRun });
}

export function runConnectivityTarget(targetId: string) {
  return request.post<RunResponse>({
    url: buildOpenApiRuntimePath('postPlatformConnectivityRun', { targetId }),
  });
}

export function getConnectivityHistory(targetId: string, limit = 20) {
  return request.get<HistoryResponse>({
    url: buildOpenApiRuntimePath('getPlatformConnectivityHistory', { targetId }),
    params: { limit },
  });
}

export function getConnectivityReport(targetId: string, checkId: number) {
  return request.get<ReportResponse>({
    url: buildOpenApiRuntimePath('getPlatformConnectivityReport', { targetId, checkId }),
  });
}

export function getConnectivityTrace(targetId: string, checkId: number) {
  return request.get<TraceResponse>({
    url: buildOpenApiRuntimePath('getPlatformConnectivityTrace', { targetId, checkId }),
  });
}

export function getConnectivityExport(targetId: string, checkId: number) {
  return request.get<ExportResponse>({
    url: buildOpenApiRuntimePath('getPlatformConnectivityExport', { targetId, checkId }),
  });
}
