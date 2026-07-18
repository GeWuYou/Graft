import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { buildDockerImageRemoveApiPath, buildDockerImageTagApiPath, CONTAINER_API_PATH } from '../contract/paths';

type DockerImagePullOperation = paths['/api/ops/docker/images/pull']['post'];
type DockerImageTagOperation = paths['/api/ops/docker/images/{id}/tag']['post'];
type DockerImageRemoveOperation = paths['/api/ops/docker/images/{id}/remove']['post'];
type DockerImageBatchRemoveOperation = paths['/api/ops/docker/images/batch-remove']['post'];

export type DockerImagePullRequest = DockerImagePullOperation['requestBody']['content']['application/json'];
export type DockerImagePullEvent = DockerImagePullOperation['responses'][200]['content']['application/x-ndjson'];
export type DockerImageTagRequest = DockerImageTagOperation['requestBody']['content']['application/json'];
export type DockerImageRemoveRequest = NonNullable<
  DockerImageRemoveOperation['requestBody']
>['content']['application/json'];
export type DockerImageBatchRemoveRequest =
  DockerImageBatchRemoveOperation['requestBody']['content']['application/json'];
export type DockerImageBatchResult = NonNullable<
  DockerImageBatchRemoveOperation['responses'][200]['content']['application/json']['data']
>;
type DockerImageActionResponse = NonNullable<
  DockerImageTagOperation['responses'][200]['content']['application/json']['data']
>;

/** 拉取流的认证、locale 与错误语义由平台 request 边界统一处理。 */
export async function pullDockerImage(
  payload: DockerImagePullRequest,
  signal: AbortSignal,
  onEvent: (event: DockerImagePullEvent) => void,
) {
  let pending = '';
  await request.postNdjson({
    data: payload,
    onChunk: (chunk) => {
      pending += chunk;
      const lines = pending.split('\n');
      pending = lines.pop() ?? '';
      for (const line of lines) emitPullEvent(line, onEvent);
    },
    signal,
    url: CONTAINER_API_PATH.DOCKER_IMAGE_PULL,
  });
  emitPullEvent(pending, onEvent);
}

export function tagDockerImage(imageId: string, payload: DockerImageTagRequest) {
  return request.post<DockerImageActionResponse>({ url: buildDockerImageTagApiPath(imageId), data: payload });
}

export function removeDockerImage(imageId: string, payload: DockerImageRemoveRequest) {
  return request.post<DockerImageActionResponse>({ url: buildDockerImageRemoveApiPath(imageId), data: payload });
}

/** 批量移除由服务端按请求顺序返回逐项结果；调用方必须处理部分成功而不能只判断请求是否成功。 */
export function batchRemoveDockerImages(payload: DockerImageBatchRemoveRequest) {
  return request.post<DockerImageBatchResult>({
    url: CONTAINER_API_PATH.DOCKER_IMAGE_BATCH_REMOVE,
    data: payload,
  });
}

function emitPullEvent(line: string, onEvent: (event: DockerImagePullEvent) => void) {
  const normalized = line.trim();
  if (!normalized) return;
  try {
    onEvent(JSON.parse(normalized) as DockerImagePullEvent);
  } catch {
    onEvent({ status: normalized });
  }
}
