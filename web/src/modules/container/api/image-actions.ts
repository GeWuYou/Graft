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
export type DockerImageBatchResult = NonNullable<
  DockerImageBatchRemoveOperation['responses'][200]['content']['application/json']['data']
>;
type DockerImageActionResponse = NonNullable<
  DockerImageTagOperation['responses'][200]['content']['application/json']['data']
>;

/** 拉取命令只提交 Task；后续日志与状态由 Task Runtime 统一提供。 */
export function pullDockerImage(payload: DockerImagePullRequest, idempotencyKey: string) {
  return request.post<DockerImagePullReceipt>({
    data: payload,
    headers: { 'Idempotency-Key': idempotencyKey },
    url: OPENAPI_RUNTIME_PATH.postDockerImagePull,
  });
}

export function tagDockerImage(imageId: string, payload: DockerImageTagRequest) {
  return request.post<DockerImageActionResponse>({
    url: buildOpenApiRuntimePath('postDockerImageTag', { id: imageId }),
    data: payload,
  });
}

/** 标签移除固定按完整 Repository:Tag 调用，不能退化成按 Image ID 删除。 */
export function untagDockerImage(imageId: string, payload: DockerImageUntagRequest) {
  return request.post<DockerImageActionResponse>({
    url: buildOpenApiRuntimePath('postDockerImageUntag', { id: imageId }),
    data: payload,
  });
}

export function removeDockerImage(imageId: string, payload: DockerImageRemoveRequest) {
  return request.post<DockerImageActionResponse>({
    url: buildOpenApiRuntimePath('postDockerImageRemove', { id: imageId }),
    data: payload,
  });
}

/** 批量移除由服务端按请求顺序返回逐项结果；调用方必须处理部分成功而不能只判断请求是否成功。 */
export function batchRemoveDockerImages(payload: DockerImageBatchRemoveRequest) {
  return request.post<DockerImageBatchResult>({
    url: OPENAPI_RUNTIME_PATH.postDockerImageBatchRemove,
    data: payload,
  });
}
