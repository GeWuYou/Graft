import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  OutboundNetworkConfig,
  OutboundNetworkDiagnostic,
  OutboundNetworkDiagnosticHistory,
  OutboundNetworkOverview,
} from '../types/outbound';

type GetOutboundOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformNetworkOutbound]['get'];
type GetOutboundData = NonNullable<GetOutboundOperation['responses'][200]['content']['application/json']['data']>;
type PutOutboundOperation = paths[typeof OPENAPI_RUNTIME_PATH.putPlatformNetworkOutbound]['put'];
type PutOutboundData = NonNullable<PutOutboundOperation['responses'][200]['content']['application/json']['data']>;
type PutOutboundBody = PutOutboundOperation['requestBody']['content']['application/json'];
type ResetOutboundOperation = paths[typeof OPENAPI_RUNTIME_PATH.resetPlatformNetworkOutbound]['post'];
type ResetOutboundData = NonNullable<ResetOutboundOperation['responses'][200]['content']['application/json']['data']>;
type DiagnosticOperation = paths[typeof OPENAPI_RUNTIME_PATH.postPlatformNetworkDiagnostic]['post'];
type DiagnosticData = NonNullable<DiagnosticOperation['responses'][200]['content']['application/json']['data']>;
type DiagnosticHistoryOperation = paths[typeof OPENAPI_RUNTIME_PATH.getPlatformNetworkDiagnosticHistory]['get'];
type DiagnosticHistoryData = NonNullable<
  DiagnosticHistoryOperation['responses'][200]['content']['application/json']['data']
>;

function readETag(headers: unknown): string | null {
  if (!headers || typeof headers !== 'object') {
    return null;
  }

  const candidate = headers as { get?: (name: string) => unknown; etag?: unknown };
  const value = candidate.get?.('etag') ?? candidate.etag;
  return typeof value === 'string' && value.trim() ? value : null;
}

export function getOutboundNetworkPolicy() {
  return request
    .getWithResponse<GetOutboundData>({
      url: OPENAPI_RUNTIME_PATH.getPlatformNetworkOutbound,
    })
    .then(({ data, headers }) => ({ data: data as OutboundNetworkOverview, etag: readETag(headers) }));
}
export function updateOutboundNetworkPolicy(policy: OutboundNetworkConfig, etag: string) {
  return request
    .putWithResponse<PutOutboundData>({
      url: OPENAPI_RUNTIME_PATH.putPlatformNetworkOutbound,
      data: policy as PutOutboundBody,
      headers: { 'If-Match': etag },
    })
    .then(({ data, headers }) => ({ data: data as OutboundNetworkOverview, etag: readETag(headers) }));
}
export function resetOutboundNetworkPolicy(etag: string) {
  return request
    .postWithResponse<ResetOutboundData>({
      url: OPENAPI_RUNTIME_PATH.resetPlatformNetworkOutbound,
      headers: { 'If-Match': etag },
    })
    .then(({ data, headers }) => ({ data: data as OutboundNetworkOverview, etag: readETag(headers) }));
}
export function diagnoseOutboundNetwork(targetId: string) {
  return request.post<DiagnosticData>({
    url: buildOpenApiRuntimePath('postPlatformNetworkDiagnostic', { targetId }),
  }) as Promise<OutboundNetworkDiagnostic>;
}
export function getOutboundNetworkDiagnosticHistory(targetId: string, limit: number) {
  return request.get<DiagnosticHistoryData>({
    url: buildOpenApiRuntimePath('getPlatformNetworkDiagnosticHistory', { targetId }),
    params: { limit },
  }) as Promise<OutboundNetworkDiagnosticHistory>;
}
