import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}));

vi.mock('@/utils/request', () => ({ request: mocks }));

import {
  getApplicationTemplate,
  getApplicationTemplateCatalog,
  getPublishedApplicationTemplate,
  getPublishedApplicationTemplateVersion,
  postApplicationTemplateClone,
  postApplicationTemplateWithdraw,
} from './project';

describe('application template API client', () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.put.mockReset();
    mocks.delete.mockReset();
  });

  it('returns the request adapter payload without reading a second envelope data field', async () => {
    const template = { template_id: 'tpl_1' };
    mocks.get.mockResolvedValue(template);

    await expect(getApplicationTemplate('tpl_1')).resolves.toBe(template);
    expect(mocks.get).toHaveBeenCalledWith({ url: '/api/ops/applications/templates/tpl_1' });
  });

  it('uses the clone and withdraw endpoint contracts', async () => {
    mocks.post.mockResolvedValue({ template_id: 'tpl_2' });

    await postApplicationTemplateClone('tpl_1', 'Clone');
    await postApplicationTemplateWithdraw('tpl_1');

    expect(mocks.post).toHaveBeenNthCalledWith(1, {
      url: '/api/ops/applications/templates/tpl_1/clone',
      data: { display_name: 'Clone' },
    });
    expect(mocks.post).toHaveBeenNthCalledWith(2, {
      url: '/api/ops/applications/templates/tpl_1/withdraw',
    });
  });

  it('uses catalog and immutable published snapshot endpoints', async () => {
    mocks.get.mockResolvedValue({ items: [] });

    await getApplicationTemplateCatalog({ deployment_adapter_kind: 'compose', category: 'cache', page: 2 });
    await getPublishedApplicationTemplate('tpl_1');
    await getPublishedApplicationTemplateVersion('tplv_1');

    expect(mocks.get).toHaveBeenNthCalledWith(1, {
      url: '/api/ops/applications/templates',
      params: { deployment_adapter_kind: 'compose', category: 'cache', page: 2 },
    });
    expect(mocks.get).toHaveBeenNthCalledWith(2, { url: '/api/ops/applications/templates/tpl_1/published' });
    expect(mocks.get).toHaveBeenNthCalledWith(3, { url: '/api/ops/applications/template-versions/tplv_1' });
  });
});
