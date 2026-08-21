import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

type DockerImagePullOperation = paths['/api/ops/docker/images/pull']['post'];
type DockerImageTagOperation = paths['/api/ops/docker/images/{id}/tag']['post'];
type DockerImageUntagOperation = paths['/api/ops/docker/images/{id}/untag']['post'];
type DockerImageRemoveOperation = paths['/api/ops/docker/images/{id}/remove']['post'];
type DockerImageBatchRemoveOperation = paths['/api/ops/docker/images/batch-remove']['post'];

export type DockerImagePullRequest = DockerImagePullOperation['requestBody']['content']['application/json'];
export type DockerImagePullReceipt = NonNullable<
  DockerImagePullOperation['responses'][202]['content']['application/json']['data']
>;
export type DockerImageTagRequest = DockerImageTagOperation['requestBody']['content']['application/json'];
export type DockerImageUntagRequest = DockerImageUntagOperation['requestBody']['content']['application/json'];
export type DockerImageRemoveRequest = NonNullable<
  DockerImageRemoveOperation['requestBody']
>['content']['application/json'];
export type DockerImageBatchRemoveRequest =
  DockerImageBatchRemoveOperation['requestBody']['content']['application/json'];
let mutationSequence = 0;

function createMutationIdempotencyKey(operation: string, resource: string) {
  mutationSequence += 1;
  return `container-${operation}-${resource}-${Date.now()}-${mutationSequence}`.slice(0, 128);
}

/** 拉取命令只提交 Task；后续日志与状态由 Task Runtime 统一提供。 */
export function pullDockerImage(payload: DockerImagePullRequest, idempotencyKey: string) {
  return request.post<DockerImagePullReceipt>({
    data: payload,
    headers: { 'Idempotency-Key': idempotencyKey },
    url: OPENAPI_RUNTIME_PATH.postDockerImagePull,
  });
}

export function tagDockerImage(imageId: string, payload: DockerImageTagRequest) {
  return request.post<DockerImagePullReceipt>({
    url: buildOpenApiRuntimePath('postDockerImageTag', { id: imageId }),
    data: payload,
    headers: { 'Idempotency-Key': createMutationIdempotencyKey('image-tag', imageId) },
  });
}

/** 标签移除固定按完整 Repository:Tag 调用，不能退化成按 Image ID 删除。 */
export function untagDockerImage(imageId: string, payload: DockerImageUntagRequest) {
  return request.post<DockerImagePullReceipt>({
    url: buildOpenApiRuntimePath('postDockerImageUntag', { id: imageId }),
    data: payload,
    headers: { 'Idempotency-Key': createMutationIdempotencyKey('image-untag', imageId) },
  });
}

export function removeDockerImage(imageId: string, payload: DockerImageRemoveRequest) {
  return request.post<DockerImagePullReceipt>({
    url: buildOpenApiRuntimePath('postDockerImageRemove', { id: imageId }),
    data: payload,
    headers: { 'Idempotency-Key': createMutationIdempotencyKey('image-remove', imageId) },
  });
}

/** 批量移除提交一个有序、fail-fast 的 Task，运行结果由 Task Runtime 解释。 */
export function batchRemoveDockerImages(payload: DockerImageBatchRemoveRequest) {
  return request.post<DockerImagePullReceipt>({
    url: OPENAPI_RUNTIME_PATH.postDockerImageBatchRemove,
    data: payload,
    headers: { 'Idempotency-Key': createMutationIdempotencyKey('image-batch-remove', String(payload.ids.length)) },
  });
}
