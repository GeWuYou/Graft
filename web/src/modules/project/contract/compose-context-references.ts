import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { APPLICATION_API_PATH } from './paths';

type ComposeContextReferencePath = (typeof APPLICATION_API_PATH)['COMPOSE_CONTEXT_REFERENCES'];
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
    url: APPLICATION_API_PATH.COMPOSE_CONTEXT_REFERENCES,
    data: { contexts } satisfies ComposeContextReferencePayload,
  });
}
