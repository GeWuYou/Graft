import { beforeEach, describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import { batchRemoveDockerImages, removeDockerImage, tagDockerImage, untagDockerImage } from './image-actions';

vi.mock('@/utils/request', () => ({
  request: {
    post: vi.fn(),
  },
}));

describe('container image mutation api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('submits every image mutation with an idempotency key and returns the Task receipt', async () => {
    const receipt = { task_id: 45, status: 'pending' };
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue(receipt as never);

    await expect(tagDockerImage('sha256:image', { target: 'graft/app:latest' })).resolves.toEqual(receipt);
    await expect(untagDockerImage('sha256:image', { reference: 'graft/app:latest' })).resolves.toEqual(receipt);
    await expect(removeDockerImage('sha256:image', { force: true })).resolves.toEqual(receipt);
    await expect(batchRemoveDockerImages({ ids: ['sha256:image'], force: false })).resolves.toEqual(receipt);

    expect(requestPost).toHaveBeenNthCalledWith(1, {
      data: { target: 'graft/app:latest' },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-image-tag-/) },
      url: buildOpenApiRuntimePath('postDockerImageTag', { id: 'sha256:image' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(2, {
      data: { reference: 'graft/app:latest' },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-image-untag-/) },
      url: buildOpenApiRuntimePath('postDockerImageUntag', { id: 'sha256:image' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(3, {
      data: { force: true },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-image-remove-/) },
      url: buildOpenApiRuntimePath('postDockerImageRemove', { id: 'sha256:image' }),
    });
    expect(requestPost).toHaveBeenNthCalledWith(4, {
      data: { ids: ['sha256:image'], force: false },
      headers: { 'Idempotency-Key': expect.stringMatching(/^container-image-batch-remove-/) },
      url: OPENAPI_RUNTIME_PATH.postDockerImageBatchRemove,
    });
  });
});
