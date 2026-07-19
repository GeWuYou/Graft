import { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

type ComposeContextReferencePath = typeof OPENAPI_RUNTIME_PATH.postApplicationComposeContextReferences;
type ComposeContextReferenceOperation = paths[ComposeContextReferencePath]['post'];
type ComposeContextReferencePayload = ComposeContextReferenceOperation['requestBody']['content']['application/json'];
type ComposeContextReferenceData = NonNullable<
  ComposeContextReferenceOperation['responses'][200]['content']['application/json']['data']
>;

export type ComposeApplicationContext = components['schemas']['application-compose-context'];
export type ComposeApplicationReference = components['schemas']['application-compose-context-reference'];

/** 解析容器运行时 Compose 上下文对应的已登记 Application。 */
export function resolveComposeApplicationReferences(contexts: ComposeApplicationContext[]) {
  return request.post<ComposeContextReferenceData>({
    url: OPENAPI_RUNTIME_PATH.postApplicationComposeContextReferences,
    data: { contexts } satisfies ComposeContextReferencePayload,
  });
}
