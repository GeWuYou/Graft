import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  OutboundNetworkDiagnostic,
  OutboundNetworkPolicy,
  OutboundNetworkPolicyResponse,
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

export function getOutboundNetworkPolicy() {
  return request.get<GetOutboundData>({
    url: OPENAPI_RUNTIME_PATH.getPlatformNetworkOutbound,
  }) as Promise<OutboundNetworkPolicyResponse>;
}

export function updateOutboundNetworkPolicy(policy: OutboundNetworkPolicy) {
  return request.put<PutOutboundData>({
    url: OPENAPI_RUNTIME_PATH.putPlatformNetworkOutbound,
    data: policy as PutOutboundBody,
  }) as Promise<OutboundNetworkPolicyResponse>;
}

export function resetOutboundNetworkPolicy() {
  return request.post<ResetOutboundData>({
    url: OPENAPI_RUNTIME_PATH.resetPlatformNetworkOutbound,
  }) as Promise<OutboundNetworkPolicyResponse>;
}

export function diagnoseOutboundNetwork(targetId: string) {
  return request.post<DiagnosticData>({
    url: buildOpenApiRuntimePath('postPlatformNetworkDiagnostic', { targetId }),
  }) as Promise<OutboundNetworkDiagnostic>;
}
