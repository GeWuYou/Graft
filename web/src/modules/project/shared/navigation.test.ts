import { describe, expect, it, vi } from 'vitest';

import { PROJECT_BOOTSTRAP_ROUTE } from '../contract/bootstrap';
import { navigateToApplicationCreateSource } from './navigation';

describe('project navigation', () => {
  it('returns to source selection with the current creation query', () => {
    const router = { push: vi.fn(), replace: vi.fn() };
    const query = { deployment: 'compose', runtime_target_id: '7' };

    navigateToApplicationCreateSource(router, query);

    expect(router.push).toHaveBeenCalledWith({
      name: PROJECT_BOOTSTRAP_ROUTE.CREATE_SOURCE.pageRouteName,
      query,
    });
  });
});
