import { describe, expect, it } from 'vitest';

import router from './index';

describe('router static runtime surface', () => {
  it('does not register starter demo homepage or grouped result routes', () => {
    const registeredPaths = router.getRoutes().map((route) => route.path);

    expect(registeredPaths).not.toContain('/dashboard');
    expect(registeredPaths).not.toContain('/result');
    expect(registeredPaths).toContain('/result/404');
    expect(registeredPaths).toContain('/');
  });
});
